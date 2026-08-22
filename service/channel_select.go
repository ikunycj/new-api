package service

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

type RetryParam struct {
	Ctx             *gin.Context
	TokenGroup      string
	ModelName       string
	RequestPath     string
	RoutingStrategy string
	Retry           *int

	// Routing state is request-local. Keeping it off the Gin context prevents a
	// fresh RetryParam in the relay handler from skipping the first candidate.
	groupIndex           int
	attempted            bool
	concreteAttempted    bool
	attemptedChannels    map[int]struct{}
	channelAttemptCounts map[int]int
	channelAttemptLimits map[int]int
	channelGroups        map[int]string
	groupAttemptCounts   map[string]int
	excludedChannels     map[int]struct{}
	attemptedClusters    map[int]struct{}
	excludedClusters     map[int]struct{}
	poolAttemptCounts    map[int]map[int]int
	startedAt            time.Time
	runtimePolicy        *model.RuntimeFailoverPolicy
	basePolicy           *model.RuntimeFailoverPolicy
	rulePolicy           bool
	allowedClusters      map[int]struct{}
	clusterOrder         []int
	groups               []string
	groupsInit           bool
	resetNextTry         bool // retained for callers using the legacy RetryParam API
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry = *p.Retry + 1
}

// ResetRetryNextTry is kept for compatibility with older callers. Ordered
// routing now advances through AdvanceRetry, but the legacy no-op increment
// behavior remains available to code that still uses this method.
func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

// MarkAttempted distinguishes initial capability fallback from fallback after
// an upstream request. Initial selection may always skip groups that cannot
// serve the model; post-attempt group changes require cross-group retry.
func (p *RetryParam) MarkAttempted() {
	p.attempted = true
}

// MarkChannelAttempted records a concrete upstream candidate for this request.
// It is intentionally request-local: channel health/auto-ban remains a
// separate policy handled after the attempt is classified.
func (p *RetryParam) MarkChannelAttempted(channelID int, clusterID int) {
	p.attempted = true
	p.concreteAttempted = true
	if channelID <= 0 {
		return
	}
	if p.attemptedChannels == nil {
		p.attemptedChannels = make(map[int]struct{})
	}
	p.attemptedChannels[channelID] = struct{}{}
	if p.channelAttemptCounts == nil {
		p.channelAttemptCounts = make(map[int]int)
	}
	p.channelAttemptCounts[channelID]++
	if p.channelGroups == nil {
		p.channelGroups = make(map[int]string)
	}
	group := p.channelGroups[channelID]
	if group == "" && p.Ctx != nil {
		group = common.GetContextKeyString(p.Ctx, constant.ContextKeyAutoGroup)
	}
	if group != "" {
		if _, exists := p.channelGroups[channelID]; !exists {
			p.channelGroups[channelID] = group
		}
	}
	if group := p.channelGroups[channelID]; group != "" {
		if p.groupAttemptCounts == nil {
			p.groupAttemptCounts = make(map[string]int)
		}
		p.groupAttemptCounts[group]++
	}
	maxAttempts := p.channelAttemptLimits[channelID]
	if maxAttempts <= 0 {
		maxAttempts = common.RetryTimes + 1
	}
	if configured := p.policy().ChannelAttemptsByID[channelID]; configured > 0 {
		maxAttempts = configured
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if p.channelAttemptCounts[channelID] >= maxAttempts {
		if p.excludedChannels == nil {
			p.excludedChannels = make(map[int]struct{})
		}
		p.excludedChannels[channelID] = struct{}{}
	}
	if clusterID > 0 {
		if p.attemptedClusters == nil {
			p.attemptedClusters = make(map[int]struct{})
		}
		p.attemptedClusters[clusterID] = struct{}{}
	}
}

// RegisterSelectedChannel records the metadata needed to enforce a channel's
// own retry budget. Selection and the actual upstream attempt are separate
// events because the distributor may select the first channel before Relay
// creates its request-local RetryParam.
func (p *RetryParam) RegisterSelectedChannel(channel *model.Channel, group string) {
	if channel == nil || channel.Id <= 0 {
		return
	}
	if p.channelGroups == nil {
		p.channelGroups = make(map[int]string)
	}
	if normalizedGroup := strings.TrimSpace(group); normalizedGroup != "" {
		p.channelGroups[channel.Id] = normalizedGroup
	}
	if p.channelAttemptLimits == nil {
		p.channelAttemptLimits = make(map[int]int)
	}
	maxAttempts := channel.GetUpstreamMaxRetries() + 1
	if configured := p.policy().ChannelAttemptsByID[channel.Id]; configured > 0 {
		maxAttempts = configured
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	p.channelAttemptLimits[channel.Id] = maxAttempts
}

func (p *RetryParam) AttemptedChannels() map[int]struct{} {
	if len(p.attemptedChannels) == 0 {
		return nil
	}
	result := make(map[int]struct{}, len(p.attemptedChannels))
	for channelID := range p.attemptedChannels {
		result[channelID] = struct{}{}
	}
	return result
}

// HasChannelRetry reports whether a selected channel can be attempted again.
// It is used by locked-channel task requests, which cannot switch to another
// candidate but still honor the channel's configured upstream retry budget.
func (p *RetryParam) HasChannelRetry(channelID int) bool {
	if channelID <= 0 {
		return false
	}
	limit := p.channelAttemptLimits[channelID]
	if limit <= 0 {
		limit = common.RetryTimes + 1
	}
	return p.channelAttemptCounts[channelID] < limit
}

func (p *RetryParam) IsAutoRouting() bool {
	return p.TokenGroup == "auto"
}

// HasNextRetry considers both the selected group's retry budget and the
// concrete channel's own retry allowance. Authenticated tokens carry an
// explicit per-group map; callers that construct RetryParam directly retain
// the legacy process-wide retry setting.
func (p *RetryParam) HasNextRetry() bool {
	if !p.withinFailoverBudget() {
		return false
	}
	if p.hasUntriedChannelInCurrentGroup() {
		return true
	}
	if !p.canCrossGroups() {
		return false
	}
	groups, err := p.candidateGroups()
	if err != nil {
		return false
	}
	for index := p.groupIndex + 1; index < len(groups); index++ {
		if p.groupHasAvailableChannels(groups[index], index) {
			return true
		}
	}
	return false
}

// AdvanceRetry exhausts the current group's dynamic candidates before moving
// to the next ordered group.
func (p *RetryParam) AdvanceRetry() bool {
	if p.hasUntriedChannelInCurrentGroup() {
		p.IncreaseRetry()
		return true
	}
	if !p.canCrossGroups() {
		return false
	}
	groups, err := p.candidateGroups()
	if err != nil {
		return false
	}
	for index := p.groupIndex + 1; index < len(groups); index++ {
		if !p.groupHasAvailableChannels(groups[index], index) {
			continue
		}
		p.groupIndex = index
		p.SetRetry(0)
		return true
	}
	return false
}

func (p *RetryParam) canCrossGroups() bool {
	if !p.IsAutoRouting() {
		return false
	}
	if p.Ctx == nil {
		return false
	}
	if common.GetContextKeyBool(p.Ctx, constant.ContextKeyTokenCrossGroupRetry) {
		return true
	}
	values, ok := common.GetContextKeyType[map[string]int](p.Ctx, constant.ContextKeyTokenGroupRetryTimes)
	return ok && len(values) > 0
}

func (p *RetryParam) groupRetryLimit(group string) int {
	if p.Ctx == nil {
		return common.RetryTimes + 1
	}
	values, ok := common.GetContextKeyType[map[string]int](p.Ctx, constant.ContextKeyTokenGroupRetryTimes)
	if ok {
		if retryTimes, exists := values[group]; exists {
			return retryTimes + 1
		}
	}
	return common.RetryTimes + 1
}

func (p *RetryParam) groupHasBudget(group string) bool {
	if p.Ctx == nil {
		return true
	}
	values, ok := common.GetContextKeyType[map[string]int](p.Ctx, constant.ContextKeyTokenGroupRetryTimes)
	if !ok || len(values) == 0 {
		// Legacy callers do not carry a per-group map. Keep their existing
		// channel-level failover behavior and let the global retry policy decide.
		return true
	}
	return p.groupAttemptCounts[group] < p.groupRetryLimit(group)
}

func (p *RetryParam) candidateGroups() ([]string, error) {
	if p.groupsInit {
		return p.groups, nil
	}
	p.groupsInit = true

	if p.TokenGroup != "auto" {
		// A caller such as Playground may explicitly override the token's
		// routing group for this request. In that case, do not reapply the
		// token's ordered candidates.
		p.groups = []string{p.TokenGroup}
		return p.groups, nil
	}

	// Explicit token candidates take precedence over the server-wide auto list.
	if p.Ctx != nil {
		if candidates := common.GetContextKeyStringSlice(p.Ctx, constant.ContextKeyTokenGroupCandidates); len(candidates) > 0 {
			p.groups = append([]string(nil), candidates...)
		} else {
			if len(setting.GetAutoGroups()) == 0 {
				return nil, errors.New("auto groups is not enabled")
			}
			userGroup := common.GetContextKeyString(p.Ctx, constant.ContextKeyUserGroup)
			p.groups = GetUserAutoGroup(userGroup)
		}
	} else {
		if len(setting.GetAutoGroups()) == 0 {
			return nil, errors.New("auto groups is not enabled")
		}
		p.groups = GetUserAutoGroup("")
	}

	// The distributor selects a concrete group before Relay creates its own
	// RetryParam. Resume from that group instead of re-running earlier candidates.
	selected := ""
	if p.Ctx != nil {
		selected = common.GetContextKeyString(p.Ctx, constant.ContextKeyAutoGroup)
	}
	if selected != "" {
		for i, group := range p.groups {
			if group == selected {
				p.groupIndex = i
				break
			}
		}
	}
	return p.groups, nil
}

func (p *RetryParam) hasUntriedChannelInCurrentGroup() bool {
	policy := p.policy()
	if len(p.attemptedChannels) >= policy.MaxTotalAttempts {
		return false
	}
	groups, err := p.candidateGroups()
	if err != nil || p.groupIndex < 0 || p.groupIndex >= len(groups) {
		return false
	}
	if p.attempted && !p.concreteAttempted && p.GetRetry() >= p.groupRetryLimit(groups[p.groupIndex])-1 {
		return false
	}
	if !p.groupHasBudget(groups[p.groupIndex]) {
		return false
	}
	return p.groupHasAvailableChannels(groups[p.groupIndex], p.groupIndex)
}

func (p *RetryParam) groupHasAvailableChannels(group string, groupIndex int) bool {
	policy := p.policy()
	if p.clusterOrder != nil {
		for _, clusterID := range p.clusterOrder {
			if _, excluded := p.excludedClusters[clusterID]; excluded {
				continue
			}
			candidates, err := p.dynamicCandidates([]string{group}, []int{groupIndex}, policy, p.clusterExclusionsFor(clusterID))
			if err == nil && len(candidates) > 0 {
				return true
			}
		}
		return false
	}
	candidates, err := p.dynamicCandidates([]string{group}, []int{groupIndex}, policy, p.excludedClusters)
	return err == nil && len(candidates) > 0
}

type dynamicChannelCandidate struct {
	channel    *model.Channel
	group      string
	groupIndex int
}

// CacheGetRandomSatisfiedChannel ranks the complete eligible candidate set for
// the currently reachable groups. A later cross-group force-priority channel
// may outrank an earlier ordinary channel; ordinary channels always retain
// their configured group order.
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	policy := param.policy()
	groups, err := param.candidateGroups()
	if err != nil {
		return nil, param.TokenGroup, err
	}
	if len(groups) == 0 {
		return nil, param.TokenGroup, errors.New("no usable groups for token")
	}

	start := param.groupIndex
	if start < 0 {
		start = 0
	}
	if start >= len(groups) {
		return nil, groups[len(groups)-1], nil
	}

	end := len(groups)
	if param.attempted && !param.canCrossGroups() {
		end = start + 1
		if end > len(groups) {
			end = len(groups)
		}
	}
	groupSubset := append([]string(nil), groups[start:end]...)
	indices := make([]int, len(groupSubset))
	for index := range groupSubset {
		indices[index] = start + index
	}

	var candidates []dynamicChannelCandidate
	if param.clusterOrder != nil {
		for _, clusterID := range param.clusterOrder {
			if _, excluded := param.excludedClusters[clusterID]; excluded {
				continue
			}
			candidates, err = param.dynamicCandidates(groupSubset, indices, policy, param.clusterExclusionsFor(clusterID))
			if err != nil || len(candidates) > 0 {
				break
			}
		}
	} else {
		candidates, err = param.dynamicCandidates(groupSubset, indices, policy, param.excludedClusters)
	}
	if err != nil {
		return nil, groups[start], err
	}
	if len(candidates) == 0 {
		group := groups[start]
		logger.LogDebug(param.Ctx, "No available channel in group %s for model %s", group, param.ModelName)
		return nil, group, nil
	}
	selected := candidates[0]
	param.groupIndex = selected.groupIndex
	if param.channelGroups == nil {
		param.channelGroups = make(map[int]string)
	}
	if param.channelAttemptLimits == nil {
		param.channelAttemptLimits = make(map[int]int)
	}
	param.channelGroups[selected.channel.Id] = selected.group
	param.channelAttemptLimits[selected.channel.Id] = selected.channel.GetUpstreamMaxRetries() + 1
	if param.IsAutoRouting() && param.Ctx != nil {
		common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, selected.group)
	}
	return selected.channel, selected.group, nil
}

func (p *RetryParam) dynamicCandidates(groups []string, groupIndices []int, policy model.RuntimeFailoverPolicy, excludedClusters map[int]struct{}) ([]dynamicChannelCandidate, error) {
	if len(groups) == 0 {
		return nil, nil
	}
	if len(policy.ChannelIDs) > 0 {
		candidates := make([]dynamicChannelCandidate, 0, len(policy.ChannelIDs))
		group := groups[0]
		groupIndex := groupIndices[0]
		if !p.groupHasBudget(group) {
			return candidates, nil
		}
		for _, channelID := range policy.ChannelIDs {
			channel, err := model.GetConfiguredChannel(group, p.ModelName, channelID, p.excludedChannels, excludedClusters)
			if err != nil {
				return nil, err
			}
			if channel != nil {
				candidates = append(candidates, dynamicChannelCandidate{channel: channel, group: group, groupIndex: groupIndex})
			}
		}
		return candidates, nil
	}

	poolTiers := append([]int(nil), policy.PoolTiers...)
	if len(poolTiers) == 0 {
		poolTiers = []int{0}
	}
	poolTiers = append(poolTiers, -1)
	for _, poolTier := range poolTiers {
		all := make([]dynamicChannelCandidate, 0)
		for index, group := range groups {
			groupIndex := groupIndices[index]
			if !p.groupHasBudget(group) {
				continue
			}
			channels, err := model.GetEligibleChannels(group, p.ModelName, p.RequestPath, p.excludedChannels, excludedClusters, poolTier)
			if err != nil {
				return nil, err
			}
			for _, channel := range channels {
				if p.groupAttemptCounts[group] >= p.groupRetryLimit(group) {
					if _, alreadyAttempted := p.attemptedChannels[channel.Id]; alreadyAttempted {
						continue
					}
				}
				all = append(all, dynamicChannelCandidate{channel: channel, group: group, groupIndex: groupIndex})
			}
		}
		if len(all) > 0 {
			sort.SliceStable(all, func(i, j int) bool {
				return dynamicCandidateLess(all[i], all[j])
			})
			return all, nil
		}
		// A zero tier already includes legacy channels; the -1 fallback is only
		// needed for policies that explicitly listed pool tiers.
		if poolTier == 0 {
			break
		}
	}
	return nil, nil
}

func dynamicCandidateLess(left, right dynamicChannelCandidate) bool {
	leftCrossForce := left.channel.IsForcePriority() && left.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeCrossGroup
	rightCrossForce := right.channel.IsForcePriority() && right.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeCrossGroup
	if leftCrossForce != rightCrossForce {
		return leftCrossForce
	}
	if left.groupIndex != right.groupIndex {
		return left.groupIndex < right.groupIndex
	}
	leftForce := left.channel.IsForcePriority()
	rightForce := right.channel.IsForcePriority()
	if leftForce != rightForce {
		return leftForce
	}
	leftCost := channelComparableCost(left.channel)
	rightCost := channelComparableCost(right.channel)
	if leftCost != rightCost {
		return leftCost < rightCost
	}
	leftRate := left.channel.PreviousDayProbeSuccessRate
	rightRate := right.channel.PreviousDayProbeSuccessRate
	if leftRate != rightRate {
		return leftRate > rightRate
	}
	if left.channel.GetWeight() != right.channel.GetWeight() {
		return left.channel.GetWeight() > right.channel.GetWeight()
	}
	return left.channel.Id < right.channel.Id
}

func channelComparableCost(channel *model.Channel) float64 {
	cost := channel.GetPriceMultiplier()
	if channel.GetPriceMultiplierMode() == model.ChannelPriceMultiplierModeCNY {
		rate := operation_setting.GetBillingUSDToCNYRate()
		if rate > 0 {
			return cost / rate
		}
	}
	return cost
}

func selectChannelForPolicy(policy model.RuntimeFailoverPolicy, group string, modelName string, retry int, requestPath string, excludedChannels map[int]struct{}, excludedClusters map[int]struct{}) (*model.Channel, error) {
	if len(policy.ChannelIDs) > 0 {
		for _, channelID := range policy.ChannelIDs {
			channel, err := model.GetConfiguredChannel(group, modelName, channelID, excludedChannels, excludedClusters)
			if err != nil || channel != nil {
				return channel, err
			}
		}
		return nil, nil
	}
	poolTiers := policy.PoolTiers
	if len(poolTiers) == 0 {
		poolTiers = []int{0}
	}
	for _, poolTier := range poolTiers {
		channel, err := model.GetRandomSatisfiedChannelWithExclusionsAndPoolTier(group, modelName, retry, requestPath, excludedChannels, excludedClusters, poolTier)
		if err != nil || channel != nil {
			return channel, err
		}
	}
	return model.GetRandomSatisfiedChannelWithExclusionsAndPoolTier(group, modelName, retry, requestPath, excludedChannels, excludedClusters, -1)
}

func (p *RetryParam) policy() model.RuntimeFailoverPolicy {
	if p.runtimePolicy != nil {
		return *p.runtimePolicy
	}
	if p.basePolicy != nil {
		return *p.basePolicy
	}
	mode := ""
	if p.Ctx != nil && p.Ctx.Request != nil {
		mode = p.Ctx.GetHeader("X-Alltoken-Failover-Mode")
	}
	userGroup := ""
	billingGroup := p.TokenGroup
	if p.Ctx != nil {
		userGroup = common.GetContextKeyString(p.Ctx, constant.ContextKeyUserGroup)
		if selected := common.GetContextKeyString(p.Ctx, constant.ContextKeyAutoGroup); selected != "" {
			billingGroup = selected
		}
	}
	policy, clusterOrder, rulePolicy := model.ResolveRuntimeFailoverWithStrategy(mode, p.RoutingStrategy, p.ModelName, p.RequestPath, userGroup, billingGroup)
	p.basePolicy = &policy
	p.rulePolicy = rulePolicy
	p.clusterOrder = clusterOrder
	if clusterOrder != nil {
		p.allowedClusters = make(map[int]struct{}, len(clusterOrder))
		for _, clusterID := range clusterOrder {
			p.allowedClusters[clusterID] = struct{}{}
		}
	}
	if p.startedAt.IsZero() {
		p.startedAt = time.Now()
	}
	return policy
}

func (p *RetryParam) RuntimePolicy() model.RuntimeFailoverPolicy {
	return p.policy()
}

func (p *RetryParam) RuntimePolicyForCluster(clusterID int) model.RuntimeFailoverPolicy {
	p.policy()
	basePolicy := *p.basePolicy
	if p.Ctx != nil && p.Ctx.Request != nil && p.Ctx.GetHeader("X-Alltoken-Failover-Mode") != "" {
		p.runtimePolicy = &basePolicy
		return basePolicy
	}
	if model.IsRoutingStrategy(p.RoutingStrategy) {
		p.runtimePolicy = &basePolicy
		return basePolicy
	}
	if p.rulePolicy {
		p.runtimePolicy = &basePolicy
		return basePolicy
	}
	clusterPolicy := model.GetRuntimeFailoverPolicyForCluster(clusterID, basePolicy)
	p.runtimePolicy = &clusterPolicy
	return clusterPolicy
}

func (p *RetryParam) ExcludeCluster(clusterID int) {
	if clusterID <= 0 {
		return
	}
	if p.excludedClusters == nil {
		p.excludedClusters = make(map[int]struct{})
	}
	p.excludedClusters[clusterID] = struct{}{}
}

func (p *RetryParam) CanAttemptCluster(clusterID int) bool {
	if clusterID <= 0 {
		return true
	}
	p.policy()
	if p.allowedClusters != nil {
		if _, allowed := p.allowedClusters[clusterID]; !allowed {
			return false
		}
	}
	if _, attempted := p.attemptedClusters[clusterID]; attempted {
		return true
	}
	return len(p.attemptedClusters) < p.policy().MaxClusterAttempts
}

func (p *RetryParam) CanAttemptPool(clusterID int, poolTier int) bool {
	if clusterID <= 0 || poolTier <= 0 {
		return true
	}
	policy := p.policy()
	clusterAttempts := p.poolAttemptCounts[clusterID]
	maxAttempts := policy.SamePoolRetries + 1
	if configured, ok := policy.PoolAttemptsByTier[poolTier]; ok && configured > 0 {
		maxAttempts = configured
	}
	if clusterAttempts[poolTier] >= maxAttempts {
		return false
	}
	if clusterAttempts[poolTier] == 0 && len(clusterAttempts) >= policy.MaxPoolAttempts {
		return false
	}
	return true
}

func (p *RetryParam) MarkPoolAttempted(clusterID int, poolTier int) {
	if clusterID <= 0 || poolTier <= 0 {
		return
	}
	if p.poolAttemptCounts == nil {
		p.poolAttemptCounts = make(map[int]map[int]int)
	}
	if p.poolAttemptCounts[clusterID] == nil {
		p.poolAttemptCounts[clusterID] = make(map[int]int)
	}
	p.poolAttemptCounts[clusterID][poolTier]++
}

func (p *RetryParam) ExcludeChannel(channelID int) {
	if channelID <= 0 {
		return
	}
	if p.excludedChannels == nil {
		p.excludedChannels = make(map[int]struct{})
	}
	p.excludedChannels[channelID] = struct{}{}
}

func (p *RetryParam) clusterExclusionsFor(clusterID int) map[int]struct{} {
	exclusions := make(map[int]struct{}, len(p.excludedClusters)+len(p.clusterOrder))
	for excludedClusterID := range p.excludedClusters {
		exclusions[excludedClusterID] = struct{}{}
	}
	for _, candidateClusterID := range p.clusterOrder {
		if candidateClusterID != clusterID {
			exclusions[candidateClusterID] = struct{}{}
		}
	}
	return exclusions
}

func (p *RetryParam) withinFailoverBudget() bool {
	policy := p.policy()
	if len(p.attemptedChannels) >= policy.MaxTotalAttempts {
		return false
	}
	return time.Since(p.startedAt) < time.Duration(policy.TotalFailoverBudgetMs)*time.Millisecond
}
