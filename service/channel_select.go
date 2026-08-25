package service

import (
	"errors"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

// RetryParam is request-local routing state. Candidate ranking is performed
// over the complete capability set.
// BillingGroupChannel priority remains a route-level ordering field.
type RetryParam struct {
	Ctx         *gin.Context
	TokenGroup  string
	ModelName   string
	RequestPath string
	Retry       *int

	groupIndex         int
	attempted          bool
	concreteAttempted  bool
	attemptCounts      map[int]int
	attemptedChannels  map[int]struct{}
	channelGroups      map[int]string
	channelLimits      map[int]int
	groupAttemptCounts map[string]int
	groupRetryLimits   map[string]int
	excludedChannels   map[int]struct{}
	currentChannelID   int
	startedAt          time.Time
	runtimePolicy      *model.RuntimeRoutingPolicy
	runtimePolicyGroup string
	routeChannels      []model.BillingGroupChannel
	routeConfigured    bool
	groupPolicies      map[string]model.RuntimeRoutingPolicy
	groupRoutes        map[string][]model.BillingGroupChannel
	groupConfigured    map[string]bool
	groups             []string
	groupsInit         bool
	randomIntn         func(int) int
}

type dynamicChannelCandidate struct {
	channel         *model.Channel
	group           string
	groupIndex      int
	routeConfigured bool
	routePriority   int
	routeOrder      int
	routeWeight     int
}

func (p *RetryParam) GetRetry() int {
	if p == nil || p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	if p == nil {
		return
	}
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p == nil {
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry = *p.Retry + 1
}

// ResetRetryNextTry is retained for callers that used the old selector. The
// new selector does not skip an increment.
func (p *RetryParam) ResetRetryNextTry() {}

func (p *RetryParam) MarkAttempted() {
	if p != nil {
		p.attempted = true
	}
}

// RegisterSelectedChannel records the effective attempt budget before an
// upstream request starts. A route entry can narrow the channel's own budget.
func (p *RetryParam) RegisterSelectedChannel(channel *model.Channel, group string) {
	if p == nil || channel == nil || channel.Id <= 0 {
		return
	}
	if p.channelGroups == nil {
		p.channelGroups = make(map[int]string)
	}
	group = strings.TrimSpace(group)
	if group != "" {
		p.channelGroups[channel.Id] = group
	}
	if p.channelLimits == nil {
		p.channelLimits = make(map[int]int)
	}
	limit := channel.GetUpstreamMaxRetries() + 1
	if entries, configured := p.routeForGroup(group); configured {
		for _, entry := range entries {
			if entry.ChannelId == channel.Id && entry.MaxAttempts > 0 && entry.MaxAttempts < limit {
				limit = entry.MaxAttempts
			}
		}
	}
	if limit < 1 {
		limit = 1
	}
	p.channelLimits[channel.Id] = limit
	p.currentChannelID = channel.Id
}

// MarkChannelAttempted records one concrete upstream attempt. A channel is
// excluded only after its own budget is exhausted, keeping retries contiguous.
func (p *RetryParam) MarkChannelAttempted(channelID int) {
	if p == nil {
		return
	}
	p.attempted = true
	p.concreteAttempted = true
	if p.attemptCounts == nil {
		p.attemptCounts = make(map[int]int)
	}
	if p.attemptedChannels == nil {
		p.attemptedChannels = make(map[int]struct{})
	}
	if channelID <= 0 {
		return
	}
	if p.channelLimits == nil {
		p.channelLimits = make(map[int]int)
	}
	if p.channelLimits[channelID] <= 0 {
		p.channelLimits[channelID] = model.DefaultChannelUpstreamMaxRetries + 1
	}
	p.attemptCounts[channelID]++
	p.attemptedChannels[channelID] = struct{}{}
	p.currentChannelID = channelID
	if p.channelGroups == nil {
		p.channelGroups = make(map[int]string)
	}
	group := p.channelGroups[channelID]
	if group == "" {
		if p.TokenGroup != "auto" {
			group = strings.TrimSpace(p.TokenGroup)
		} else if p.Ctx != nil {
			group = common.GetContextKeyString(p.Ctx, constant.ContextKeyAutoGroup)
		}
		if group != "" {
			p.channelGroups[channelID] = group
		}
	}
	if group != "" {
		if p.groupAttemptCounts == nil {
			p.groupAttemptCounts = make(map[string]int)
		}
		p.groupAttemptCounts[group]++
	}
	if p.attemptCounts[channelID] >= p.channelLimits[channelID] {
		p.ExcludeChannel(channelID)
	}
}

func (p *RetryParam) AttemptedChannels() map[int]struct{} {
	if p == nil || len(p.attemptedChannels) == 0 {
		return nil
	}
	result := make(map[int]struct{}, len(p.attemptedChannels))
	for channelID := range p.attemptedChannels {
		result[channelID] = struct{}{}
	}
	return result
}

// HasChannelRetry is used by task requests whose channel is intentionally
// locked. Such requests cannot switch candidates, but still honor the same
// per-channel budget as normal relay requests.
func (p *RetryParam) HasChannelRetry(channelID int) bool {
	if p == nil || channelID <= 0 || p.isExcluded(channelID) {
		return false
	}
	limit := p.channelLimits[channelID]
	if limit <= 0 {
		limit = model.DefaultChannelUpstreamMaxRetries + 1
	}
	return p.attemptCounts[channelID] < limit &&
		p.channelGroupHasBudget(channelID) &&
		p.channelGroupWithinRoutingAttemptBudget(channelID) &&
		p.withinRoutingBudget()
}

func (p *RetryParam) IsAutoRouting() bool { return p != nil && p.TokenGroup == "auto" }

func (p *RetryParam) canCrossGroups() bool {
	return p != nil && p.IsAutoRouting() && common.GetContextKeyBool(p.Ctx, constant.ContextKeyTokenCrossGroupRetry)
}

func (p *RetryParam) groupRetryLimit(group string) (int, bool) {
	if p == nil {
		return 0, false
	}
	group = strings.TrimSpace(group)
	if p.groupRetryLimits == nil {
		p.groupRetryLimits = make(map[string]int)
	}
	if limit, exists := p.groupRetryLimits[group]; exists {
		return limit, true
	}

	limit := 0
	configured := false
	if policy, exists := ratio_setting.GetPricingGroupRetryPolicy(group); exists {
		configured = true
		switch policy.Mode {
		case ratio_setting.PricingGroupRetryModeFixed:
			limit = policy.RetryTimes + 1
		case ratio_setting.PricingGroupRetryModeActiveChannels:
			channels, err := model.GetEligibleChannels(group, p.ModelName, p.RequestPath, nil)
			if err != nil {
				logger.LogError(p.Ctx, "failed to count active channels for retry budget: "+err.Error())
				limit = 1
			} else {
				if _, entries, routeConfigured := p.loadGroupPolicy(group); routeConfigured {
					routeChannelIDs := make(map[int]struct{}, len(entries))
					for _, entry := range entries {
						if entry.Enabled && entry.ChannelId > 0 {
							routeChannelIDs[entry.ChannelId] = struct{}{}
						}
					}
					channels = slices.DeleteFunc(channels, func(channel *model.Channel) bool {
						if channel == nil {
							return true
						}
						_, allowed := routeChannelIDs[channel.Id]
						return !allowed
					})
				}
				limit = len(channels) + 1
			}
		default:
			limit = 1
		}
	}
	if p.Ctx != nil {
		if values, ok := common.GetContextKeyType[map[string]int](p.Ctx, constant.ContextKeyTokenGroupRetryTimes); ok {
			if retryTimes, exists := values[group]; exists {
				configured = true
				tokenLimit := retryTimes + 1
				if retryTimes < 0 || retryTimes > MaxTokenGroupRetryTimes {
					tokenLimit = 1
				}
				if limit == 0 || tokenLimit < limit {
					limit = tokenLimit
				}
			}
		}
	}
	if !configured {
		return 0, false
	}
	if limit < 1 {
		limit = 1
	}
	p.groupRetryLimits[group] = limit
	return limit, true
}

func (p *RetryParam) groupHasBudget(group string) bool {
	limit, configured := p.groupRetryLimit(group)
	return !configured || p.groupAttemptCounts[group] < limit
}

func (p *RetryParam) channelGroupHasBudget(channelID int) bool {
	if p == nil {
		return false
	}
	return p.groupHasBudget(p.groupForChannel(channelID))
}

func (p *RetryParam) groupForChannel(channelID int) string {
	group := p.channelGroups[channelID]
	if group == "" && p.TokenGroup != "auto" {
		return strings.TrimSpace(p.TokenGroup)
	}
	return group
}

func (p *RetryParam) groupWithinRoutingAttemptBudget(group string) bool {
	if p == nil {
		return false
	}
	policy, _, configured := p.loadGroupPolicy(group)
	return !configured || policy.MaxTotalAttempts <= 0 || p.groupAttemptCounts[group] < policy.MaxTotalAttempts
}

func (p *RetryParam) channelGroupWithinRoutingAttemptBudget(channelID int) bool {
	if p == nil {
		return false
	}
	return p.groupWithinRoutingAttemptBudget(p.groupForChannel(channelID))
}

func (p *RetryParam) HasNextRetry() bool {
	if p == nil || !p.withinRoutingBudget() {
		return false
	}
	return len(p.availableCandidates()) > 0
}

func (p *RetryParam) AdvanceRetry() bool {
	if p == nil || !p.withinRoutingBudget() {
		return false
	}
	// Keep the current channel contiguous until its configured budget is used.
	if p.currentChannelID > 0 && !p.isExcluded(p.currentChannelID) &&
		p.attemptCounts[p.currentChannelID] < p.channelLimits[p.currentChannelID] &&
		p.channelGroupHasBudget(p.currentChannelID) &&
		p.channelGroupWithinRoutingAttemptBudget(p.currentChannelID) {
		p.IncreaseRetry()
		return true
	}
	p.SetRetry(0)
	return len(p.availableCandidates()) > 0
}

func (p *RetryParam) candidateGroups() ([]string, error) {
	if p == nil {
		return nil, errors.New("retry parameter is nil")
	}
	if p.groupsInit {
		return p.groups, nil
	}
	p.groupsInit = true
	if p.TokenGroup != "auto" {
		group := strings.TrimSpace(p.TokenGroup)
		if group == "" {
			return nil, errors.New("no usable groups for token")
		}
		p.groups = []string{group}
		return p.groups, nil
	}
	var groups []string
	if candidates := common.GetContextKeyStringSlice(p.Ctx, constant.ContextKeyTokenGroupCandidates); len(candidates) > 0 {
		groups = candidates
	} else {
		return nil, errors.New("auto group requires explicit candidates")
	}
	seen := make(map[string]struct{}, len(groups))
	for _, raw := range groups {
		group := strings.TrimSpace(raw)
		if group == "" {
			continue
		}
		if _, exists := seen[group]; exists {
			continue
		}
		seen[group] = struct{}{}
		p.groups = append(p.groups, group)
	}
	if selected := common.GetContextKeyString(p.Ctx, constant.ContextKeyAutoGroup); selected != "" {
		for i, group := range p.groups {
			if group == selected {
				p.groupIndex = i
				break
			}
		}
	}
	if len(p.groups) == 0 {
		return nil, errors.New("no usable groups for token")
	}
	return p.groups, nil
}

func (p *RetryParam) currentGroup() (string, error) {
	groups, err := p.candidateGroups()
	if err != nil {
		return "", err
	}
	if p.groupIndex < 0 || p.groupIndex >= len(groups) {
		return "", errors.New("no usable groups for token")
	}
	return groups[p.groupIndex], nil
}

func (p *RetryParam) loadGroupPolicy(group string) (model.RuntimeRoutingPolicy, []model.BillingGroupChannel, bool) {
	group = strings.TrimSpace(group)
	if p.groupPolicies == nil {
		p.groupPolicies = make(map[string]model.RuntimeRoutingPolicy)
		p.groupRoutes = make(map[string][]model.BillingGroupChannel)
		p.groupConfigured = make(map[string]bool)
	}
	if policy, ok := p.groupPolicies[group]; ok {
		return policy, p.groupRoutes[group], p.groupConfigured[group]
	}
	policy, entries, configured := model.ResolveBillingGroupRoute(group)
	p.groupPolicies[group] = policy
	p.groupRoutes[group] = append([]model.BillingGroupChannel(nil), entries...)
	p.groupConfigured[group] = configured
	return policy, entries, configured
}

func (p *RetryParam) loadPolicy() model.RuntimeRoutingPolicy {
	group, err := p.currentGroup()
	if err != nil {
		policy := model.DefaultRuntimeRoutingPolicy()
		p.runtimePolicy = &policy
		p.runtimePolicyGroup = ""
		p.routeChannels = nil
		p.routeConfigured = false
		return policy
	}
	if p.runtimePolicy != nil && p.runtimePolicyGroup == group {
		return *p.runtimePolicy
	}
	policy, entries, configured := p.loadGroupPolicy(group)
	p.runtimePolicy = &policy
	p.runtimePolicyGroup = group
	p.routeChannels = entries
	p.routeConfigured = configured
	if p.startedAt.IsZero() {
		p.startedAt = time.Now()
	}
	return policy
}

func (p *RetryParam) RuntimePolicy() model.RuntimeRoutingPolicy { return p.loadPolicy() }

func (p *RetryParam) RouteConfigured() bool {
	p.loadPolicy()
	return p.routeConfigured
}

func (p *RetryParam) routeForGroup(group string) ([]model.BillingGroupChannel, bool) {
	if p == nil {
		return nil, false
	}
	_, entries, configured := p.loadGroupPolicy(group)
	return entries, configured
}

func (p *RetryParam) isExcluded(channelID int) bool {
	_, excluded := p.excludedChannels[channelID]
	return excluded
}

func (p *RetryParam) ExcludeChannel(channelID int) {
	if p == nil || channelID <= 0 {
		return
	}
	if p.excludedChannels == nil {
		p.excludedChannels = make(map[int]struct{})
	}
	p.excludedChannels[channelID] = struct{}{}
}

// HandleChannelFailure applies the error mapping action. retry_channel leaves
// the candidate eligible; switching actions exclude it.
func (p *RetryParam) HandleChannelFailure(channelID int, action string) {
	if p == nil || channelID <= 0 {
		return
	}
	if action == "retry_channel" && !p.isExcluded(channelID) {
		return
	}
	p.ExcludeChannel(channelID)
}

func (p *RetryParam) withinRoutingBudget() bool {
	if p == nil {
		return false
	}
	policy := p.loadPolicy()
	if p.startedAt.IsZero() {
		p.startedAt = time.Now()
	}
	// Route attempt budgets belong to one pricing group. Cross-group routing
	// checks them while building candidates so an exhausted group cannot block
	// the next group.
	group, err := p.currentGroup()
	if !p.canCrossGroups() && err == nil && !p.groupWithinRoutingAttemptBudget(group) {
		return false
	}
	if policy.TotalTimeoutMs > 0 && time.Since(p.startedAt) >= time.Duration(policy.TotalTimeoutMs)*time.Millisecond {
		return false
	}
	return true
}

func (p *RetryParam) availableCandidates() []dynamicChannelCandidate {
	groups, err := p.candidateGroups()
	if err != nil || len(groups) == 0 {
		return nil
	}
	start := p.groupIndex
	if start < 0 {
		start = 0
	}
	if p.canCrossGroups() {
		// Cross-group routing uses one dynamic candidate set. The currently selected
		// group is only a tie-break preference, not a permanent lower bound.
		start = 0
	}
	if start >= len(groups) {
		return nil
	}
	end := len(groups)
	if p.attempted && !p.canCrossGroups() {
		end = start + 1
		if end > len(groups) {
			end = len(groups)
		}
	}
	indices := make([]int, 0, end-start)
	for i := start; i < end; i++ {
		indices = append(indices, i)
	}
	candidates, err := p.dynamicCandidates(groups[start:end], indices)
	if err != nil {
		logger.LogError(p.Ctx, "failed to build channel candidates: "+err.Error())
		return nil
	}
	return candidates
}

// CacheGetRandomSatisfiedChannel ranks the complete eligible candidate set.
// The historical function name is retained for callers.
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	if param == nil {
		return nil, "", errors.New("retry parameter is nil")
	}
	groups, err := param.candidateGroups()
	if err != nil {
		return nil, param.TokenGroup, err
	}
	candidates := param.availableCandidates()
	if len(candidates) == 0 {
		if len(groups) == 0 {
			return nil, param.TokenGroup, nil
		}
		index := param.groupIndex
		if index < 0 {
			index = 0
		}
		if index >= len(groups) {
			index = len(groups) - 1
		}
		return nil, groups[index], nil
	}
	selected := candidates[0]
	param.groupIndex = selected.groupIndex
	param.currentChannelID = selected.channel.Id
	param.RegisterSelectedChannel(selected.channel, selected.group)
	if param.IsAutoRouting() {
		common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, selected.group)
	}
	common.SetContextKey(param.Ctx, constant.ContextKeyUsingGroup, selected.group)
	UpdatePricingGroupActivity(param.Ctx, selected.group)
	logger.LogDebug(param.Ctx, "selected billing group %s channel %d", selected.group, selected.channel.Id)
	return selected.channel, selected.group, nil
}

func (p *RetryParam) dynamicCandidates(groups []string, groupIndices []int) ([]dynamicChannelCandidate, error) {
	if len(groups) == 0 {
		return nil, nil
	}
	all := make([]dynamicChannelCandidate, 0)
	for index, group := range groups {
		if !p.groupHasBudget(group) {
			continue
		}
		groupIndex := groupIndices[index]
		_, entries, configured := p.loadGroupPolicy(group)
		if !p.groupWithinRoutingAttemptBudget(group) {
			continue
		}
		channels, err := model.GetEligibleChannels(group, p.ModelName, p.RequestPath, p.excludedChannels)
		if err != nil {
			return nil, err
		}
		entryByID := make(map[int]dynamicChannelCandidate, len(entries))
		if configured {
			for order, entry := range entries {
				if !entry.Enabled || entry.ChannelId <= 0 || p.isExcluded(entry.ChannelId) {
					continue
				}
				entryByID[entry.ChannelId] = dynamicChannelCandidate{
					routePriority: entry.Priority,
					routeOrder:    order,
					routeWeight:   entry.Weight,
				}
			}
		}
		groupCandidates := make([]dynamicChannelCandidate, 0, len(channels))
		for _, channel := range channels {
			if channel == nil || p.isExcluded(channel.Id) {
				continue
			}
			if previousGroup, attempted := p.channelGroups[channel.Id]; attempted && previousGroup != group {
				continue
			}
			candidate := dynamicChannelCandidate{
				channel:         channel,
				group:           group,
				groupIndex:      groupIndex,
				routeConfigured: configured,
			}
			if configured {
				entry, ok := entryByID[channel.Id]
				if !ok {
					continue
				}
				candidate.routePriority = entry.routePriority
				candidate.routeOrder = entry.routeOrder
				candidate.routeWeight = entry.routeWeight
			}
			groupCandidates = append(groupCandidates, candidate)
		}
		if configured && len(groupCandidates) > 0 {
			// A configured route exposes only its highest currently eligible
			// priority tier. This preserves the route hierarchy without comparing
			// group-local priority numbers across different billing groups.
			bestPriority := groupCandidates[0].routePriority
			for _, candidate := range groupCandidates[1:] {
				if candidate.routePriority > bestPriority {
					bestPriority = candidate.routePriority
				}
			}
			for _, candidate := range groupCandidates {
				if candidate.routePriority == bestPriority {
					all = append(all, candidate)
				}
			}
			continue
		}
		all = append(all, groupCandidates...)
	}
	return p.rankDynamicCandidates(all), nil
}

func (p *RetryParam) rankDynamicCandidates(candidates []dynamicChannelCandidate) []dynamicChannelCandidate {
	if len(candidates) < 2 {
		return candidates
	}
	sort.SliceStable(candidates, func(i, j int) bool { return dynamicCandidateLess(candidates[i], candidates[j]) })
	result := candidates[:0]
	seen := make(map[int]struct{}, len(candidates))
	for _, candidate := range candidates {
		if _, exists := seen[candidate.channel.Id]; exists {
			continue
		}
		seen[candidate.channel.Id] = struct{}{}
		result = append(result, candidate)
	}
	bestCount := 1
	for bestCount < len(result) && dynamicCandidatesHaveEqualRank(result[0], result[bestCount]) {
		bestCount++
	}
	if bestCount > 1 {
		totalWeight := 0
		for _, candidate := range result[:bestCount] {
			totalWeight += dynamicCandidateSelectionWeight(candidate)
		}
		randomIntn := rand.Intn
		if p != nil && p.randomIntn != nil {
			randomIntn = p.randomIntn
		}
		selectedIndex := weightedDynamicCandidateIndex(result[:bestCount], randomIntn(totalWeight))
		if selectedIndex > 0 {
			selected := result[selectedIndex]
			copy(result[1:selectedIndex+1], result[:selectedIndex])
			result[0] = selected
		}
	}
	p.prioritizeCurrentChannel(result)
	return result
}

func (p *RetryParam) prioritizeCurrentChannel(candidates []dynamicChannelCandidate) {
	if p == nil || !p.concreteAttempted || p.currentChannelID <= 0 || len(candidates) < 2 || p.isExcluded(p.currentChannelID) {
		return
	}
	if p.attemptCounts[p.currentChannelID] >= p.channelLimits[p.currentChannelID] {
		return
	}
	group := p.channelGroups[p.currentChannelID]
	for i, candidate := range candidates {
		if candidate.channel.Id != p.currentChannelID || candidate.group != group {
			continue
		}
		if i > 0 {
			copy(candidates[1:i+1], candidates[:i])
			candidates[0] = candidate
		}
		return
	}
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
	leftCost := channelComparableCost(left)
	rightCost := channelComparableCost(right)
	if leftCost != rightCost {
		return leftCost < rightCost
	}
	if left.channel.PreviousDayProbeSuccessRate != right.channel.PreviousDayProbeSuccessRate {
		return left.channel.PreviousDayProbeSuccessRate > right.channel.PreviousDayProbeSuccessRate
	}
	if left.routeOrder != right.routeOrder {
		return left.routeOrder < right.routeOrder
	}
	return left.channel.Id < right.channel.Id
}

func dynamicCandidatesHaveEqualRank(left, right dynamicChannelCandidate) bool {
	leftCrossForce := left.channel.IsForcePriority() && left.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeCrossGroup
	rightCrossForce := right.channel.IsForcePriority() && right.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeCrossGroup
	return leftCrossForce == rightCrossForce &&
		left.groupIndex == right.groupIndex &&
		left.channel.IsForcePriority() == right.channel.IsForcePriority() &&
		channelComparableCost(left) == channelComparableCost(right) &&
		left.channel.PreviousDayProbeSuccessRate == right.channel.PreviousDayProbeSuccessRate
}

func dynamicCandidateSelectionWeight(candidate dynamicChannelCandidate) int {
	if candidate.routeConfigured {
		if candidate.routeWeight > 0 {
			return candidate.routeWeight
		}
		return 100
	}
	return candidate.channel.GetWeight() + 10
}

func weightedDynamicCandidateIndex(candidates []dynamicChannelCandidate, randomWeight int) int {
	if len(candidates) == 0 {
		return -1
	}
	for index, candidate := range candidates {
		randomWeight -= dynamicCandidateSelectionWeight(candidate)
		if randomWeight < 0 {
			return index
		}
	}
	return len(candidates) - 1
}

func channelComparableCost(candidate dynamicChannelCandidate) float64 {
	cost := candidate.channel.GetPriceMultiplier()
	if candidate.channel.GetPriceMultiplierMode() == model.ChannelPriceMultiplierModeCNY {
		rate := operation_setting.GetBillingUSDToCNYRate()
		if rate > 0 {
			cost /= rate
		}
	}
	return cost
}
