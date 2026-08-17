package service

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
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
	groupIndex        int
	attempted         bool
	attemptedChannels map[int]struct{}
	excludedChannels  map[int]struct{}
	attemptedClusters map[int]struct{}
	excludedClusters  map[int]struct{}
	poolAttemptCounts map[int]map[int]int
	startedAt         time.Time
	runtimePolicy     *model.RuntimeFailoverPolicy
	basePolicy        *model.RuntimeFailoverPolicy
	rulePolicy        bool
	allowedClusters   map[int]struct{}
	clusterOrder      []int
	groups            []string
	groupsInit        bool
	resetNextTry      bool // retained for callers using the legacy RetryParam API
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
	if channelID <= 0 {
		return
	}
	if p.attemptedChannels == nil {
		p.attemptedChannels = make(map[int]struct{})
	}
	p.attemptedChannels[channelID] = struct{}{}
	if p.excludedChannels == nil {
		p.excludedChannels = make(map[int]struct{})
	}
	p.excludedChannels[channelID] = struct{}{}
	if clusterID > 0 {
		if p.attemptedClusters == nil {
			p.attemptedClusters = make(map[int]struct{})
		}
		p.attemptedClusters[clusterID] = struct{}{}
	}
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

func (p *RetryParam) IsAutoRouting() bool {
	return p.TokenGroup == "auto"
}

// HasNextRetry is independent of common.RetryTimes for candidate transitions,
// so ordered failover still works when the global retry count is zero.
func (p *RetryParam) HasNextRetry() bool {
	if !p.withinFailoverBudget() {
		return false
	}
	if len(p.excludedChannels) > 0 || len(p.excludedClusters) > 0 {
		if p.hasUntriedChannelInCurrentGroup() {
			return true
		}
		if !p.IsAutoRouting() || !common.GetContextKeyBool(p.Ctx, constant.ContextKeyTokenCrossGroupRetry) {
			return false
		}
		groups, err := p.candidateGroups()
		return err == nil && p.groupIndex+1 < len(groups)
	}
	if p.GetRetry() < common.RetryTimes {
		return true
	}
	if !p.IsAutoRouting() || !common.GetContextKeyBool(p.Ctx, constant.ContextKeyTokenCrossGroupRetry) {
		return false
	}
	groups, err := p.candidateGroups()
	return err == nil && p.groupIndex+1 < len(groups)
}

// AdvanceRetry exhausts the current group's retry levels before moving to the
// next ordered candidate.
func (p *RetryParam) AdvanceRetry() bool {
	if len(p.excludedChannels) > 0 || len(p.excludedClusters) > 0 {
		if p.hasUntriedChannelInCurrentGroup() {
			p.IncreaseRetry()
			return true
		}
		if !p.IsAutoRouting() || !common.GetContextKeyBool(p.Ctx, constant.ContextKeyTokenCrossGroupRetry) {
			return false
		}
		groups, err := p.candidateGroups()
		if err != nil || p.groupIndex+1 >= len(groups) {
			return false
		}
		p.groupIndex++
		p.SetRetry(0)
		return true
	}
	if p.GetRetry() < common.RetryTimes {
		p.IncreaseRetry()
		return true
	}
	if !p.IsAutoRouting() || !common.GetContextKeyBool(p.Ctx, constant.ContextKeyTokenCrossGroupRetry) {
		return false
	}
	groups, err := p.candidateGroups()
	if err != nil || p.groupIndex+1 >= len(groups) {
		return false
	}
	p.groupIndex++
	p.SetRetry(0)
	return true
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
	if candidates := common.GetContextKeyStringSlice(p.Ctx, constant.ContextKeyTokenGroupCandidates); len(candidates) > 0 {
		p.groups = append([]string(nil), candidates...)
	} else {
		if len(setting.GetAutoGroups()) == 0 {
			return nil, errors.New("auto groups is not enabled")
		}
		userGroup := common.GetContextKeyString(p.Ctx, constant.ContextKeyUserGroup)
		p.groups = GetUserAutoGroup(userGroup)
	}

	// The distributor selects a concrete group before Relay creates its own
	// RetryParam. Resume from that group instead of re-running earlier candidates.
	if selected := common.GetContextKeyString(p.Ctx, constant.ContextKeyAutoGroup); selected != "" {
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
	if p.clusterOrder != nil {
		for _, clusterID := range p.clusterOrder {
			if _, excluded := p.excludedClusters[clusterID]; excluded {
				continue
			}
			available, err := selectChannelForPolicy(policy, groups[p.groupIndex], p.ModelName, 0, p.RequestPath, p.excludedChannels, p.clusterExclusionsFor(clusterID))
			if err == nil && available != nil {
				return true
			}
		}
		return false
	}
	available, err := selectChannelForPolicy(policy, groups[p.groupIndex], p.ModelName, 0, p.RequestPath, p.excludedChannels, p.excludedClusters)
	return err == nil && available != nil
}

// CacheGetRandomSatisfiedChannel selects within the first usable candidate.
// Candidate order is deterministic; channel selection within one group keeps
// the existing weighted/priority behavior.
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

	for i := start; i < len(groups); i++ {
		group := groups[i]
		priorityRetry := param.GetRetry()
		if i > start {
			priorityRetry = 0
		}
		if len(param.excludedChannels) > 0 || len(param.excludedClusters) > 0 {
			// Failover is candidate-based. Once a channel has failed, select the
			// highest-priority remaining channel instead of retrying the same
			// priority slot and potentially selecting it again.
			priorityRetry = 0
		}
		logger.LogDebug(param.Ctx, "Selecting group: %s, priorityRetry: %d", group, priorityRetry)

		var channel *model.Channel
		var channelErr error
		if param.clusterOrder != nil {
			for _, clusterID := range param.clusterOrder {
				if _, excluded := param.excludedClusters[clusterID]; excluded {
					continue
				}
				channel, channelErr = selectChannelForPolicy(policy, group, param.ModelName, priorityRetry, param.RequestPath, param.excludedChannels, param.clusterExclusionsFor(clusterID))
				if channelErr != nil || channel != nil {
					break
				}
			}
		} else {
			channel, channelErr = selectChannelForPolicy(policy, group, param.ModelName, priorityRetry, param.RequestPath, param.excludedChannels, param.excludedClusters)
		}
		if channelErr != nil {
			return nil, group, channelErr
		}
		if channel == nil {
			logger.LogDebug(param.Ctx, "No available channel in group %s for model %s at priorityRetry %d", group, param.ModelName, priorityRetry)
			if param.attempted && !common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry) {
				return nil, group, nil
			}
			param.groupIndex = i + 1
			param.SetRetry(0)
			continue
		}

		param.groupIndex = i
		if param.IsAutoRouting() {
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, group)
		}
		return channel, group, nil
	}
	return nil, groups[len(groups)-1], nil
}

func selectChannelForPolicy(policy model.RuntimeFailoverPolicy, group string, modelName string, retry int, requestPath string, excludedChannels map[int]struct{}, excludedClusters map[int]struct{}) (*model.Channel, error) {
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
