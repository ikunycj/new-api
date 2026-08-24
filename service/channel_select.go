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
	Ctx         *gin.Context
	TokenGroup  string
	ModelName   string
	RequestPath string
	Retry       *int

	groupIndex        int
	attempted         bool
	attemptCounts     map[int]int
	attemptedChannels map[int]struct{}
	excludedChannels  map[int]struct{}
	retryChannelID    int
	startedAt         time.Time
	runtimePolicy     *model.RuntimeRoutingPolicy
	routeChannels     []model.BillingGroupChannel
	routeConfigured   bool
	groups            []string
	groupsInit        bool
	resetNextTry      bool
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

func (p *RetryParam) hasAvailableChannelInCurrentGroup() bool {
	group, err := p.currentGroup()
	if err != nil {
		return false
	}
	p.loadPolicy()
	if p.routeConfigured {
		channel, err := model.GetConfiguredRouteChannel(group, p.ModelName, p.RequestPath, p.routeChannels, p.excludedChannels)
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
				channel, err = model.GetConfiguredRouteChannel(group, param.ModelName, param.RequestPath, retryEntries, param.excludedChannels)
			}
			if channel == nil && err == nil {
				channel, err = model.GetConfiguredRouteChannel(group, param.ModelName, param.RequestPath, param.routeChannels, param.excludedChannels)
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
