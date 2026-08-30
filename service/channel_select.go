package service

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// IsMockLoadTestRequest reports whether the controller must use the internal
// managed load-test executor instead of entering channel routing.
func IsMockLoadTestRequest(c *gin.Context) bool {
	return c != nil && c.Request != nil && strings.EqualFold(strings.TrimSpace(c.GetHeader(constant.MockLoadTestHeader)), "true")
}

type RetryParam struct {
	Ctx         *gin.Context
	TokenGroup  string
	ModelName   string
	RequestPath string
	Retry       *int

	groupIndex              int
	attempted               bool
	attemptCounts           map[int]int
	attemptedChannels       map[int]struct{}
	excludedChannels        map[int]struct{}
	retryChannelID          int
	startedAt               time.Time
	runtimePolicy           *model.RuntimeRoutingPolicy
	routeChannels           []model.BillingGroupChannel
	routeConfigured         bool
	groups                  []string
	groupsInit              bool
	resetNextTry            bool
	projectedChannelCostUSD float64
}

const profitGuardAuditContextKey = "profit_guard_audit"

type ProfitGuardDecision struct {
	Mode                    string  `json:"mode"`
	Decision                string  `json:"decision"`
	ChannelID               int     `json:"channel_id"`
	MinimumProfitMargin     float64 `json:"minimum_profit_margin"`
	ProjectedProfitMargin   float64 `json:"projected_profit_margin"`
	EstimatedRevenueUSD     float64 `json:"estimated_revenue_usd"`
	ProjectedChannelCostUSD float64 `json:"projected_channel_cost_usd"`
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) { p.Retry = &retry }

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

func (p *RetryParam) ResetRetryNextTry() { p.resetNextTry = true }

func (p *RetryParam) MarkAttempted() { p.attempted = true }

func (p *RetryParam) MarkChannelAttempted(channelID int) {
	p.attempted = true
	if channelID <= 0 {
		return
	}
	if p.attemptCounts == nil {
		p.attemptCounts = make(map[int]int)
	}
	if p.attemptedChannels == nil {
		p.attemptedChannels = make(map[int]struct{})
	}
	p.attemptCounts[channelID]++
	p.attemptedChannels[channelID] = struct{}{}
	maxAttempts := 1
	for _, entry := range p.routeChannels {
		if entry.ChannelId == channelID {
			maxAttempts = entry.MaxAttempts
			break
		}
	}
	if p.attemptCounts[channelID] >= maxAttempts {
		p.ExcludeChannel(channelID)
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

func (p *RetryParam) IsAutoRouting() bool { return p.TokenGroup == "auto" }

func (p *RetryParam) HasNextRetry() bool {
	if !p.withinRoutingBudget() {
		return false
	}
	if p.hasAvailableChannelInCurrentGroup() {
		return true
	}
	if !p.IsAutoRouting() || !common.GetContextKeyBool(p.Ctx, constant.ContextKeyTokenCrossGroupRetry) {
		return false
	}
	groups, err := p.candidateGroups()
	return err == nil && p.groupIndex+1 < len(groups)
}

func (p *RetryParam) AdvanceRetry() bool {
	if !p.withinRoutingBudget() {
		return false
	}
	if p.hasAvailableChannelInCurrentGroup() {
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
	p.runtimePolicy = nil
	p.routeChannels = nil
	p.routeConfigured = false
	p.SetRetry(0)
	return true
}

func (p *RetryParam) candidateGroups() ([]string, error) {
	if p.groupsInit {
		return p.groups, nil
	}
	p.groupsInit = true
	if p.TokenGroup != "auto" {
		p.groups = []string{p.TokenGroup}
		return p.groups, nil
	}
	if candidates := common.GetContextKeyStringSlice(p.Ctx, constant.ContextKeyTokenGroupCandidates); len(candidates) > 0 {
		p.groups = append([]string(nil), candidates...)
	} else {
		if len(setting.GetAutoGroups()) == 0 {
			return nil, errors.New("auto groups is not enabled")
		}
		p.groups = GetUserAutoGroup(common.GetContextKeyString(p.Ctx, constant.ContextKeyUserGroup))
	}
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

func (p *RetryParam) loadPolicy() model.RuntimeRoutingPolicy {
	if p.runtimePolicy != nil {
		return *p.runtimePolicy
	}
	group, err := p.currentGroup()
	if err != nil {
		policy := model.DefaultRuntimeRoutingPolicy(model.RoutingModeBalanced)
		p.runtimePolicy = &policy
		return policy
	}
	policy, entries, configured := model.ResolveBillingGroupRoute(group)
	p.runtimePolicy = &policy
	p.routeChannels = entries
	p.routeConfigured = configured
	if configured {
		var filteredEntries []model.BillingGroupChannel
		var err error
		// Managed load-test mocks are terminated by controller.Relay before a
		// RetryParam is created. Any request that reaches routing is therefore a
		// real request and must use only real channels; retaining a mock-channel
		// fallback here could accidentally send an unauthenticated request to a
		// provider configured solely for legacy mocks.
		filteredEntries, err = model.FilterRealRouteChannels(entries)
		if err != nil {
			logger.LogError(p.Ctx, fmt.Sprintf("filter billing route channels: %s", err.Error()))
			filteredEntries = nil
		}
		p.routeChannels = filteredEntries
	}
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

// EvaluateProfitGuard compares estimated user revenue with the cumulative
// estimated channel cost before an upstream attempt starts. Off and warn modes
// never block traffic; enforce mode blocks only when the configured margin
// would be breached.
func (p *RetryParam) EvaluateProfitGuard(channelID int, priceData types.PriceData) ProfitGuardDecision {
	policy := p.loadPolicy()
	decision := ProfitGuardDecision{
		Mode:                policy.ProfitGuardMode,
		Decision:            "unavailable",
		ChannelID:           channelID,
		MinimumProfitMargin: policy.MinimumProfitMargin,
	}
	if policy.ProfitGuardMode == model.ProfitGuardModeOff {
		decision.Decision = "off"
		return decision
	}
	providerBaseCost := priceData.EstimatedProviderBaseCostUSD
	groupRatio := priceData.GroupRatioInfo.GroupRatio
	if providerBaseCost <= 0 || groupRatio <= 0 || math.IsNaN(providerBaseCost) ||
		math.IsInf(providerBaseCost, 0) || math.IsNaN(groupRatio) || math.IsInf(groupRatio, 0) {
		return decision
	}
	group, err := p.currentGroup()
	if err != nil {
		return decision
	}
	nextCost := providerBaseCost * model.ResolveChannelCostFactor(group, channelID)
	projectedCost := p.projectedChannelCostUSD + nextCost
	revenue := providerBaseCost * groupRatio
	margin := (revenue - projectedCost) / revenue * 100
	decision.EstimatedRevenueUSD = revenue
	decision.ProjectedChannelCostUSD = projectedCost
	decision.ProjectedProfitMargin = margin
	if margin+1e-9 >= policy.MinimumProfitMargin {
		decision.Decision = "allow"
		p.projectedChannelCostUSD = projectedCost
		return decision
	}
	decision.Decision = "warn"
	if policy.ProfitGuardMode == model.ProfitGuardModeEnforce {
		decision.Decision = "block"
		return decision
	}
	p.projectedChannelCostUSD = projectedCost
	return decision
}

func RecordProfitGuardDecision(c *gin.Context, decision ProfitGuardDecision) {
	if c == nil || decision.Mode == model.ProfitGuardModeOff {
		return
	}
	c.Set(profitGuardAuditContextKey, decision)
}

func profitGuardAudit(c *gin.Context) (ProfitGuardDecision, bool) {
	if c == nil {
		return ProfitGuardDecision{}, false
	}
	value, ok := c.Get(profitGuardAuditContextKey)
	if !ok {
		return ProfitGuardDecision{}, false
	}
	decision, ok := value.(ProfitGuardDecision)
	return decision, ok
}

func (p *RetryParam) hasAvailableChannelInCurrentGroup() bool {
	group, err := p.currentGroup()
	if err != nil {
		return false
	}
	p.loadPolicy()
	if p.routeConfigured {
		policy := p.loadPolicy()
		channel, err := model.GetConfiguredRouteChannelWithStrategyConfig(group, p.ModelName, p.RequestPath, p.routeChannels, p.excludedChannels, policy.StrategyConfig)
		return err == nil && channel != nil
	}
	available, err := model.HasSatisfiedChannelExcluding(group, p.ModelName, p.RequestPath, p.excludedChannels)
	return err == nil && available
}

func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	groups, err := param.candidateGroups()
	if err != nil {
		return nil, param.TokenGroup, err
	}
	for i := param.groupIndex; i < len(groups); i++ {
		param.groupIndex = i
		param.runtimePolicy = nil
		param.loadPolicy()
		group := groups[i]
		var channel *model.Channel
		if param.routeConfigured {
			if param.retryChannelID > 0 {
				retryEntries := make([]model.BillingGroupChannel, 0, 1)
				for _, entry := range param.routeChannels {
					if entry.ChannelId == param.retryChannelID {
						retryEntries = append(retryEntries, entry)
						break
					}
				}
				param.retryChannelID = 0
				policy := param.loadPolicy()
				channel, err = model.GetConfiguredRouteChannelWithStrategyConfig(group, param.ModelName, param.RequestPath, retryEntries, param.excludedChannels, policy.StrategyConfig)
			}
			if channel == nil && err == nil {
				policy := param.loadPolicy()
				channel, err = model.GetConfiguredRouteChannelWithStrategyConfig(group, param.ModelName, param.RequestPath, param.routeChannels, param.excludedChannels, policy.StrategyConfig)
			}
		} else {
			priorityRetry := param.GetRetry()
			if len(param.excludedChannels) > 0 {
				priorityRetry = 0
			}
			channel, err = model.GetRandomSatisfiedChannelExcluding(group, param.ModelName, priorityRetry, param.RequestPath, param.excludedChannels)
		}
		if err != nil {
			return nil, group, err
		}
		if channel == nil {
			if param.attempted && !common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry) {
				return nil, group, nil
			}
			continue
		}
		logger.LogDebug(param.Ctx, "selected billing group %s channel %d", group, channel.Id)
		if param.IsAutoRouting() {
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, group)
		}
		return channel, group, nil
	}
	if len(groups) == 0 {
		return nil, param.TokenGroup, nil
	}
	return nil, groups[len(groups)-1], nil
}

// HandleChannelFailure applies the configured error action to the next
// selection. retry_channel keeps the current channel while its per-channel
// attempt budget remains; every other action advances to another candidate.
func (p *RetryParam) HandleChannelFailure(channelID int, action string) {
	if channelID <= 0 {
		return
	}
	if action == "retry_channel" {
		if _, excluded := p.excludedChannels[channelID]; !excluded {
			p.retryChannelID = channelID
		}
		return
	}
	p.ExcludeChannel(channelID)
}

func (p *RetryParam) ExcludeChannel(channelID int) {
	if channelID <= 0 {
		return
	}
	if p.excludedChannels == nil {
		p.excludedChannels = make(map[int]struct{})
	}
	p.excludedChannels[channelID] = struct{}{}
	if p.retryChannelID == channelID {
		p.retryChannelID = 0
	}
}

func (p *RetryParam) withinRoutingBudget() bool {
	policy := p.loadPolicy()
	count := 0
	for _, attempts := range p.attemptCounts {
		count += attempts
	}
	if count >= policy.MaxTotalAttempts {
		return false
	}
	return time.Since(p.startedAt) < time.Duration(policy.TotalTimeoutMs)*time.Millisecond
}
