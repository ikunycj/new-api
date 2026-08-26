package service

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
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
// BillingGroupChannel priority is used only as a final stable route-order
// tie-break; force priority is the only hard channel layer.
type RetryParam struct {
	Ctx         *gin.Context
	TokenGroup  string
	ModelName   string
	RequestPath string
	// CircuitRoute is the stable route key used by the circuit breaker. It is
	// separate from RequestPath because a request URL can include a version or
	// provider-specific suffix while the Gin route pattern remains stable.
	CircuitRoute string
	Retry        *int

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
	excludedProviders  map[int]struct{}
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
	pendingSchedule    *routingScheduleTicket
}

type dynamicChannelCandidate struct {
	channel         *model.Channel
	modelName       string
	group           string
	groupIndex      int
	routeConfigured bool
	routePriority   int
	routeOrder      int
	routeWeight     int
	routeCostFactor float64
	score           float64
	selectionWeight float64
	routingStrategy ratio_setting.PricingGroupRoutingStrategy
}

const (
	dynamicScoreTolerance      = 5.0
	dynamicPriceTolerance      = 0.03
	dynamicPricePenaltyCap     = 0.20
	dynamicAvailabilityPrior   = 0.95
	dynamicAvailabilitySamples = 100.0
	dynamicScoreTemperature    = 5.0
	dynamicGroupScoreTolerance = 5.0
)

type smoothScheduleState struct {
	sync.Mutex
	current map[string]float64
}

type routingScheduleTicket struct {
	sync.Mutex
	settled       bool
	channelID     int
	groupDeltas   map[string]float64
	channelDeltas map[string]float64
}

const routingScheduleTicketContextKey = "routing_schedule_ticket"

// dynamicScheduleState implements smooth weighted round-robin for normal
// requests. A selection reserves its scheduler credit immediately to avoid a
// concurrent herd, then rolls it back if no upstream attempt starts.
var dynamicScheduleState = smoothScheduleState{current: make(map[string]float64)}

// Group selection has its own smooth scheduler. Keeping it separate from the
// channel scheduler is important: a pricing group with more channels must not
// win simply because its inner scheduler has more entries.
var dynamicGroupScheduleState = smoothScheduleState{current: make(map[string]float64)}

func (ticket *routingScheduleTicket) addDeltas(group bool, deltas map[string]float64) {
	if ticket == nil || len(deltas) == 0 {
		return
	}
	ticket.Lock()
	defer ticket.Unlock()
	if ticket.settled {
		return
	}
	target := ticket.channelDeltas
	if group {
		target = ticket.groupDeltas
	}
	if target == nil {
		target = make(map[string]float64, len(deltas))
		if group {
			ticket.groupDeltas = target
		} else {
			ticket.channelDeltas = target
		}
	}
	for key, delta := range deltas {
		target[key] += delta
	}
}

func (ticket *routingScheduleTicket) setChannel(channelID int) {
	if ticket == nil {
		return
	}
	ticket.Lock()
	ticket.channelID = channelID
	ticket.Unlock()
}

func (ticket *routingScheduleTicket) commit(channelID int) bool {
	if ticket == nil {
		return false
	}
	ticket.Lock()
	defer ticket.Unlock()
	if ticket.settled || ticket.channelID != channelID {
		return false
	}
	ticket.settled = true
	return true
}

func (ticket *routingScheduleTicket) rollback() {
	if ticket == nil {
		return
	}
	ticket.Lock()
	if ticket.settled {
		ticket.Unlock()
		return
	}
	ticket.settled = true
	groupDeltas := ticket.groupDeltas
	channelDeltas := ticket.channelDeltas
	ticket.Unlock()

	rollbackScheduleDeltas(&dynamicGroupScheduleState, groupDeltas)
	rollbackScheduleDeltas(&dynamicScheduleState, channelDeltas)
}

func rollbackScheduleDeltas(state *smoothScheduleState, deltas map[string]float64) {
	if state == nil || len(deltas) == 0 {
		return
	}
	state.Lock()
	defer state.Unlock()
	for key, delta := range deltas {
		state.current[key] -= delta
		if math.Abs(state.current[key]) < 0.000000001 {
			delete(state.current, key)
		}
	}
}

func routingScheduleTicketFromContext(c *gin.Context) *routingScheduleTicket {
	if c == nil {
		return nil
	}
	value, exists := c.Get(routingScheduleTicketContextKey)
	if !exists {
		return nil
	}
	ticket, _ := value.(*routingScheduleTicket)
	return ticket
}

func clearRoutingScheduleTicketContext(c *gin.Context, ticket *routingScheduleTicket) {
	if c == nil || ticket == nil || routingScheduleTicketFromContext(c) != ticket {
		return
	}
	c.Set(routingScheduleTicketContextKey, nil)
}

func commitPendingRoutingSelection(c *gin.Context, channelID int) bool {
	ticket := routingScheduleTicketFromContext(c)
	if ticket == nil || !ticket.commit(channelID) {
		return false
	}
	clearRoutingScheduleTicketContext(c, ticket)
	return true
}

func (p *RetryParam) beginRoutingSelection() {
	if p == nil {
		return
	}
	p.CancelRoutingSelection()
	ticket := &routingScheduleTicket{}
	p.pendingSchedule = ticket
	if p.Ctx != nil {
		p.Ctx.Set(routingScheduleTicketContextKey, ticket)
	}
}

func (p *RetryParam) recordRoutingScheduleDeltas(group bool, deltas map[string]float64) {
	if p == nil || p.pendingSchedule == nil {
		return
	}
	p.pendingSchedule.addDeltas(group, deltas)
}

func (p *RetryParam) finishRoutingSelection(channelID int) {
	if p == nil || p.pendingSchedule == nil {
		return
	}
	p.pendingSchedule.setChannel(channelID)
}

func (p *RetryParam) commitRoutingSelection(channelID int) {
	if p == nil {
		return
	}
	ticket := p.pendingSchedule
	if ticket == nil {
		commitPendingRoutingSelection(p.Ctx, channelID)
		return
	}
	if ticket == nil || !ticket.commit(channelID) {
		return
	}
	if p.pendingSchedule == ticket {
		p.pendingSchedule = nil
	}
	clearRoutingScheduleTicketContext(p.Ctx, ticket)
}

// CancelRoutingSelection restores scheduler credit when a selected channel
// never reaches an upstream attempt, including concurrency reservation races.
func (p *RetryParam) CancelRoutingSelection() {
	if p == nil {
		return
	}
	ticket := p.pendingSchedule
	contextTicket := routingScheduleTicketFromContext(p.Ctx)
	p.pendingSchedule = nil
	if ticket != nil {
		ticket.rollback()
	}
	if contextTicket != nil && contextTicket != ticket {
		contextTicket.rollback()
	}
	clearRoutingScheduleTicketContext(p.Ctx, contextTicket)
}

// CancelPendingRoutingSelection covers middleware exits before a controller
// can either start an upstream attempt or cancel through its RetryParam.
func CancelPendingRoutingSelection(c *gin.Context) {
	ticket := routingScheduleTicketFromContext(c)
	if ticket == nil {
		return
	}
	ticket.rollback()
	clearRoutingScheduleTicketContext(c, ticket)
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
	p.commitRoutingSelection(channelID)
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
			activeChannels, err := p.activeRetryChannelCount(group)
			if err != nil {
				logger.LogError(p.Ctx, "failed to count active channels for retry budget: "+err.Error())
				limit = 1
			} else {
				limit = activeChannels + 1
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

// activeRetryChannelCount returns the number of channels that could actually
// receive a request right now. Retry budgets using "active channels" must not
// count a channel with no usable credential, a full concurrency slot, an open
// circuit, a provider excluded by the current failure, or a zero-weight route
// entry. Attempted channels are intentionally not excluded here: the budget is
// a group-wide upper bound, while candidate selection owns per-request
// exclusions.
func (p *RetryParam) activeRetryChannelCount(group string) (int, error) {
	if p == nil {
		return 0, nil
	}
	channels, err := model.GetEligibleChannels(group, p.ModelName, p.RequestPath, nil)
	if err != nil {
		return 0, err
	}
	_, entries, routeConfigured := p.loadGroupPolicy(group)
	entryByID := make(map[int]model.BillingGroupChannel, len(entries))
	if routeConfigured {
		for _, entry := range entries {
			if entry.Enabled && entry.ChannelId > 0 {
				entryByID[entry.ChannelId] = entry
			}
		}
	}
	route := p.circuitRoute()
	count := 0
	for _, channel := range channels {
		if channel == nil || !channel.HasEnabledKey() || p.isProviderExcluded(channel.Type) {
			continue
		}
		if CurrentChannelConcurrency(channel.Id) >= channel.GetMaxConcurrency() {
			continue
		}
		if route != "" && ChannelCircuitIsOpen(channel.Id, route) {
			continue
		}
		if routeConfigured {
			entry, exists := entryByID[channel.Id]
			if !exists || (entry.Weight == 0 && !channel.IsForcePriority()) {
				continue
			}
		}
		count++
	}
	return count, nil
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
	return len(p.availableCandidates(false)) > 0
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
	return len(p.availableCandidates(false)) > 0
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

func (p *RetryParam) circuitRoute() string {
	if p == nil {
		return ""
	}
	if route := strings.TrimSpace(p.CircuitRoute); route != "" {
		return route
	}
	return strings.TrimSpace(p.RequestPath)
}

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

func (p *RetryParam) ExcludeProvider(channelType int) {
	if p == nil || channelType <= 0 {
		return
	}
	if p.excludedProviders == nil {
		p.excludedProviders = make(map[int]struct{})
	}
	p.excludedProviders[channelType] = struct{}{}
}

func (p *RetryParam) isProviderExcluded(channelType int) bool {
	if p == nil || channelType <= 0 {
		return false
	}
	_, excluded := p.excludedProviders[channelType]
	return excluded
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

func (p *RetryParam) availableCandidates(commit ...bool) []dynamicChannelCandidate {
	commitState := len(commit) == 0 || commit[0]
	groups, err := p.candidateGroups()
	if err != nil || len(groups) == 0 {
		return nil
	}
	start := p.groupIndex
	if start < 0 {
		start = 0
	}
	if p.canCrossGroups() {
		// Cross-group routing evaluates all permitted groups. The selector first
		// chooses one group, then ranks channels inside that group.
		start = 0
	}
	if start >= len(groups) {
		return nil
	}
	end := len(groups)
	if !p.canCrossGroups() {
		end = start + 1
		if end > len(groups) {
			end = len(groups)
		}
	}
	indices := make([]int, 0, end-start)
	for i := start; i < end; i++ {
		indices = append(indices, i)
	}
	candidates, err := p.dynamicCandidates(groups[start:end], indices, commitState)
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
	param.beginRoutingSelection()
	groups, err := param.candidateGroups()
	if err != nil {
		param.CancelRoutingSelection()
		return nil, param.TokenGroup, err
	}
	candidates := param.availableCandidates(true)
	if len(candidates) == 0 {
		param.CancelRoutingSelection()
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
	param.finishRoutingSelection(selected.channel.Id)
	param.groupIndex = selected.groupIndex
	param.currentChannelID = selected.channel.Id
	param.RegisterSelectedChannel(selected.channel, selected.group)
	if param.Ctx != nil && param.IsAutoRouting() {
		common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, selected.group)
	}
	if param.Ctx != nil {
		common.SetContextKey(param.Ctx, constant.ContextKeyUsingGroup, selected.group)
	}
	UpdatePricingGroupActivity(param.Ctx, selected.group)
	logger.LogDebug(param.Ctx,
		"selected billing group %s channel %d score=%.2f selection_weight=%.2f concurrency=%d/%d force=%t",
		selected.group,
		selected.channel.Id,
		selected.score,
		selected.selectionWeight,
		CurrentChannelConcurrency(selected.channel.Id),
		selected.channel.GetMaxConcurrency(),
		selected.channel.IsForcePriority(),
	)
	return selected.channel, selected.group, nil
}

func (p *RetryParam) dynamicCandidates(groups []string, groupIndices []int, commit ...bool) ([]dynamicChannelCandidate, error) {
	commitState := len(commit) == 0 || commit[0]
	if len(groups) == 0 {
		return nil, nil
	}
	groupCandidates := make([][]dynamicChannelCandidate, len(groups))
	for index, group := range groups {
		if !p.groupHasBudget(group) {
			continue
		}
		groupIndex := groupIndices[index]
		policy, entries, configured := p.loadGroupPolicy(group)
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
					routePriority:   entry.Priority,
					routeOrder:      order,
					routeWeight:     entry.Weight,
					routeCostFactor: normalizeRouteCostFactor(entry.CostFactor),
				}
			}
		}
		candidatesForGroup := make([]dynamicChannelCandidate, 0, len(channels))
		for _, channel := range channels {
			if channel == nil || p.isExcluded(channel.Id) {
				continue
			}
			if p.isProviderExcluded(channel.Type) {
				continue
			}
			if !channel.HasEnabledKey() {
				continue
			}
			if CurrentChannelConcurrency(channel.Id) >= channel.GetMaxConcurrency() {
				continue
			}
			if route := p.circuitRoute(); route != "" && ChannelCircuitIsOpen(channel.Id, route) {
				continue
			}
			if previousGroup, attempted := p.channelGroups[channel.Id]; attempted && previousGroup != group {
				continue
			}
			candidate := dynamicChannelCandidate{
				channel:         channel,
				modelName:       p.ModelName,
				group:           group,
				groupIndex:      groupIndex,
				routeConfigured: configured,
				routeCostFactor: 1,
				routingStrategy: policy.RoutingStrategy,
			}
			if configured {
				entry, ok := entryByID[channel.Id]
				if !ok {
					continue
				}
				candidate.routePriority = entry.routePriority
				candidate.routeOrder = entry.routeOrder
				candidate.routeWeight = entry.routeWeight
				candidate.routeCostFactor = entry.routeCostFactor
				// A zero route weight explicitly opts a channel out of normal
				// traffic. A forced channel remains eligible so its hard layer can
				// still provide an intentional emergency path.
				if entry.routeWeight == 0 && !channel.IsForcePriority() {
					continue
				}
			}
			candidatesForGroup = append(candidatesForGroup, candidate)
		}
		groupCandidates[index] = candidatesForGroup
	}

	if p != nil && p.attempted && !groupCandidatesHaveCrossForce(groupCandidates) {
		for index, candidates := range groupCandidates {
			if len(candidates) == 0 || groupIndices[index] != p.groupIndex {
				continue
			}
			return p.rankDynamicCandidates(candidates, commitState), nil
		}
	}

	selectedGroup := selectDynamicGroupIndex(p, groupCandidates, groupIndices, commitState)
	if selectedGroup < 0 {
		return nil, nil
	}
	return p.rankDynamicCandidates(groupCandidates[selectedGroup], commitState), nil
}

func groupCandidatesHaveCrossForce(grouped [][]dynamicChannelCandidate) bool {
	for _, candidates := range grouped {
		for _, candidate := range candidates {
			if candidate.channel != nil && candidate.channel.IsForcePriority() &&
				candidate.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeCrossGroup {
				return true
			}
		}
	}
	return false
}

// selectDynamicGroupIndex chooses one pricing group before channel-level
// ranking. Group scores use a common price baseline and weighted aggregate
// availability/load, so a group with more channels does not receive more
// traffic merely because it has a larger candidate set.
func selectDynamicGroupIndex(p *RetryParam, grouped [][]dynamicChannelCandidate, _ []int, commit ...bool) int {
	commitState := len(commit) == 0 || commit[0]
	hasCrossForce := false
	globalCrossLevel := int(^uint(0) >> 1)
	for _, candidates := range grouped {
		for _, candidate := range candidates {
			if candidate.channel == nil {
				continue
			}
			if candidate.channel.IsForcePriority() && candidate.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeCrossGroup {
				hasCrossForce = true
				if level := forcePriorityLevel(candidate); level < globalCrossLevel {
					globalCrossLevel = level
				}
			}
		}
	}

	// Establish the hard force layer before calculating a shared price baseline.
	// A normal channel must never make a forced channel look artificially
	// expensive: force-priority candidates are the only candidates that can
	// participate in this group comparison when a force layer is active.
	filtered := make([][]dynamicChannelCandidate, len(grouped))
	globalMinPrice := math.Inf(1)
	for index, candidates := range grouped {
		if len(candidates) == 0 {
			continue
		}
		if hasCrossForce {
			candidates = filterForcePriorityLayerAtLevel(candidates, true, globalCrossLevel)
		} else {
			candidates = filterForcePriorityLayer(candidates, false)
		}
		if len(candidates) == 0 {
			continue
		}
		filtered[index] = candidates
		for _, candidate := range candidates {
			if price := channelComparableCost(candidate); price >= 0 && !math.IsNaN(price) && !math.IsInf(price, 0) && price < globalMinPrice {
				globalMinPrice = price
			}
		}
	}

	type scoredGroup struct {
		index int
		score float64
	}
	scored := make([]scoredGroup, 0, len(grouped))
	for index, candidates := range filtered {
		if len(candidates) == 0 {
			continue
		}
		grouped[index] = candidates
		strategy := routingStrategyForCandidates(candidates)
		score := dynamicGroupScore(candidates, globalMinPrice, strategy)
		if math.IsInf(score, -1) || math.IsNaN(score) {
			continue
		}
		scored = append(scored, scoredGroup{index: index, score: score})
	}
	if len(scored) == 0 {
		return -1
	}
	maxScore := scored[0].score
	for _, group := range scored[1:] {
		if group.score > maxScore {
			maxScore = group.score
		}
	}
	eligible := make([]scoredGroup, 0, len(scored))
	for _, group := range scored {
		if group.score >= maxScore-dynamicGroupScoreTolerance {
			eligible = append(eligible, group)
		}
	}
	if len(eligible) == 1 {
		return eligible[0].index
	}
	if !commitState {
		return eligible[0].index
	}

	// Smoothly rotate only among near-best groups. A clearly better group is
	// still selected deterministically, while equal-quality groups share
	// traffic without being biased by the number of channels they contain.
	totalWeight := 0.0
	selectedPosition := -1
	selectedCurrent := math.Inf(-1)
	deltas := make(map[string]float64, len(eligible))
	dynamicGroupScheduleState.Lock()
	keyParts := make([]string, 0, len(eligible))
	for _, group := range eligible {
		name := ""
		if group.index >= 0 && group.index < len(grouped) && len(grouped[group.index]) > 0 {
			name = grouped[group.index][0].group
		}
		keyParts = append(keyParts, name)
	}
	scheduleKey := strings.Join(keyParts, "\x00")
	for position, group := range eligible {
		weight := math.Exp((group.score - maxScore) / dynamicScoreTemperature)
		if weight < 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
			weight = 0
		}
		totalWeight += weight
		currentKey := scheduleKey + "\x00" + strconv.Itoa(group.index)
		dynamicGroupScheduleState.current[currentKey] += weight
		deltas[currentKey] += weight
		if dynamicGroupScheduleState.current[currentKey] > selectedCurrent {
			selectedCurrent = dynamicGroupScheduleState.current[currentKey]
			selectedPosition = position
		}
	}
	if selectedPosition < 0 || totalWeight <= 0 {
		dynamicGroupScheduleState.Unlock()
		return eligible[0].index
	}
	selected := eligible[selectedPosition]
	selectedKey := scheduleKey + "\x00" + strconv.Itoa(selected.index)
	dynamicGroupScheduleState.current[selectedKey] -= totalWeight
	deltas[selectedKey] -= totalWeight
	dynamicGroupScheduleState.Unlock()
	p.recordRoutingScheduleDeltas(true, deltas)
	return selected.index
}

func (p *RetryParam) rankDynamicCandidates(candidates []dynamicChannelCandidate, commit ...bool) []dynamicChannelCandidate {
	commitState := len(commit) == 0 || commit[0]
	if len(candidates) == 0 {
		return candidates
	}
	candidates = filterZeroWeightCandidates(candidates)
	if len(candidates) == 0 {
		return nil
	}
	result := rankDynamicCandidateOrder(candidates)
	if p.prioritizeCurrentChannel(result) {
		return result
	}
	pool := dynamicCandidatePool(result)
	if len(pool) > 1 {
		totalWeight := 0.0
		for _, candidate := range pool {
			totalWeight += dynamicCandidateSelectionWeightFloat(candidate)
		}
		if totalWeight > 0 {
			var selectedIndex int
			if commitState && p != nil && p.randomIntn != nil {
				// The test hook and historical callers use integer draws. Keep the
				// integer scale while retaining fractional dynamic weights internally.
				draw := float64(p.randomIntn(maxIntWeight(totalWeight)))
				selectedIndex = weightedDynamicCandidateIndexFloat(pool, draw)
			} else if commitState {
				selectedIndex = smoothWeightedCandidateIndex(p, pool)
			}
			if commitState && selectedIndex > 0 {
				moveCandidateToFront(result, pool[selectedIndex])
			}
		}
	}
	return result
}

func rankDynamicCandidateOrder(candidates []dynamicChannelCandidate) []dynamicChannelCandidate {
	if len(candidates) == 0 {
		return candidates
	}
	candidates = filterZeroWeightCandidates(candidates)
	if len(candidates) == 0 {
		return nil
	}
	candidates = filterForcePriorityLayer(candidates, false)
	if len(candidates) == 0 {
		return candidates
	}
	annotateDynamicCandidateScores(candidates)
	if len(candidates) == 1 {
		return candidates
	}
	sort.SliceStable(candidates, func(i, j int) bool { return dynamicCandidateLess(candidates[i], candidates[j]) })
	result := candidates[:0]
	type candidateKey struct {
		group string
		id    int
	}
	seen := make(map[candidateKey]struct{}, len(candidates))
	for _, candidate := range candidates {
		key := candidateKey{group: candidate.group, id: candidate.channel.Id}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
	}
	return result
}

// filterZeroWeightCandidates applies the route-weight contract at the edge of
// the selector as well as in the database-backed candidate builder. Keeping it
// here protects direct callers and tests that construct candidates in memory.
func filterZeroWeightCandidates(candidates []dynamicChannelCandidate) []dynamicChannelCandidate {
	if len(candidates) == 0 {
		return candidates
	}
	result := make([]dynamicChannelCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.routeConfigured && dynamicCandidateSelectionWeight(candidate) <= 0 &&
			(candidate.channel == nil || !candidate.channel.IsForcePriority()) {
			continue
		}
		result = append(result, candidate)
	}
	return result
}

// filterForcePriorityLayer keeps the hard force-priority layer ahead of all
// dynamic scoring. Cross-group force candidates are global; group-scoped force
// candidates are filtered within the group passed to this function.
func filterForcePriorityLayer(candidates []dynamicChannelCandidate, crossGroupOnly bool) []dynamicChannelCandidate {
	return filterForcePriorityLayerAtLevel(candidates, crossGroupOnly, 0)
}

func filterForcePriorityLayerAtLevel(candidates []dynamicChannelCandidate, crossGroupOnly bool, requiredLevel int) []dynamicChannelCandidate {
	if len(candidates) == 0 {
		return candidates
	}
	hasCrossForce := false
	hasGroupForce := false
	bestCrossLevel := int(^uint(0) >> 1)
	bestGroupLevel := int(^uint(0) >> 1)
	for _, candidate := range candidates {
		if candidate.channel == nil || !candidate.channel.IsForcePriority() {
			continue
		}
		if candidate.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeCrossGroup {
			hasCrossForce = true
			if level := forcePriorityLevel(candidate); level < bestCrossLevel {
				bestCrossLevel = level
			}
		} else {
			hasGroupForce = true
			if level := forcePriorityLevel(candidate); level < bestGroupLevel {
				bestGroupLevel = level
			}
		}
	}
	if crossGroupOnly || hasCrossForce {
		if !hasCrossForce {
			return nil
		}
		result := make([]dynamicChannelCandidate, 0, len(candidates))
		for _, candidate := range candidates {
			level := bestCrossLevel
			if requiredLevel > 0 {
				level = requiredLevel
			}
			if candidate.channel != nil && candidate.channel.IsForcePriority() && candidate.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeCrossGroup && forcePriorityLevel(candidate) == level {
				result = append(result, candidate)
			}
		}
		return result
	}
	if !hasGroupForce {
		return candidates
	}
	result := make([]dynamicChannelCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.channel != nil && candidate.channel.IsForcePriority() && candidate.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeGroup && forcePriorityLevel(candidate) == bestGroupLevel {
			result = append(result, candidate)
		}
	}
	return result
}

// forcePriorityLevel reuses the route's numeric priority as the force layer.
// This keeps the existing schema and UI contract while providing the design's
// 1 -> 2 -> 3 -> normal ordering for routes that opt into ForcePriority.
func forcePriorityLevel(candidate dynamicChannelCandidate) int {
	if candidate.routeConfigured && candidate.routePriority > 0 {
		return candidate.routePriority
	}
	return 1
}

func routingStrategyForCandidates(candidates []dynamicChannelCandidate) ratio_setting.PricingGroupRoutingStrategy {
	if len(candidates) == 0 {
		return ratio_setting.DefaultPricingGroupRoutingStrategy()
	}
	strategy := candidates[0].routingStrategy
	if strategy.Strategy == "" {
		var exists bool
		strategy, exists = ratio_setting.GetPricingGroupRoutingStrategy(candidates[0].group)
		if !exists {
			strategy = ratio_setting.DefaultPricingGroupRoutingStrategy()
		}
	}
	return normalizeRoutingStrategy(strategy)
}

func normalizeRoutingStrategy(strategy ratio_setting.PricingGroupRoutingStrategy) ratio_setting.PricingGroupRoutingStrategy {
	weightTotal := strategy.PriceWeight + strategy.AvailabilityWeight + strategy.LoadWeight
	// Strategy IDs are administrator-defined catalog keys. They are not a
	// closed enum; only an empty ID or invalid weights should fall back.
	if strings.TrimSpace(strategy.Strategy) == "" ||
		strategy.PriceWeight < 0 || strategy.AvailabilityWeight < 0 || strategy.LoadWeight < 0 ||
		math.IsNaN(weightTotal) || math.IsInf(weightTotal, 0) || math.Abs(weightTotal-100) > 0.0001 {
		return ratio_setting.DefaultPricingGroupRoutingStrategy()
	}
	return strategy
}

func dynamicAvailabilityScore(channel *model.Channel) float64 {
	if channel == nil {
		return 0
	}
	availability := channel.PreviousDayProbeSuccessRate / 100
	if math.IsNaN(availability) || math.IsInf(availability, 0) || availability < 0 {
		availability = 0
	}
	if availability > 1 {
		availability = 1
	}
	samples := float64(channel.PreviousDayProbeSampleCount)
	if samples < 0 || math.IsNaN(samples) || math.IsInf(samples, 0) {
		samples = 0
	}
	confidence := math.Min(samples/dynamicAvailabilitySamples, 1)
	return confidence*availability + (1-confidence)*dynamicAvailabilityPrior
}

func dynamicCandidateFeatures(candidate dynamicChannelCandidate, minPrice float64) (float64, float64, float64) {
	if candidate.channel == nil {
		return 0, 0, 0
	}
	price := channelComparableCost(candidate)
	priceScore := 1.0
	if math.IsInf(minPrice, 1) || math.IsNaN(minPrice) {
		// A missing price baseline means no candidate has a usable configured
		// price. Keep price neutral and let availability/load decide.
		priceScore = 1
	} else if minPrice <= 0 {
		// Zero is a valid explicitly free model price. It is the best possible
		// price, while every positive price receives the maximum penalty.
		if price > 0 {
			priceScore = 0
		}
	} else if price >= 0 && !math.IsNaN(price) && !math.IsInf(price, 0) {
		priceGap := price/minPrice - 1
		if priceGap > dynamicPriceTolerance {
			priceScore = 1 - math.Min(1, math.Max(0, (priceGap-dynamicPriceTolerance)/(dynamicPricePenaltyCap-dynamicPriceTolerance)))
		}
	}
	return priceScore, dynamicAvailabilityScore(candidate.channel), 1 - dynamicCandidateUtilization(candidate)
}

func dynamicGroupScore(candidates []dynamicChannelCandidate, minPrice float64, strategy ratio_setting.PricingGroupRoutingStrategy) float64 {
	if len(candidates) == 0 {
		return 0
	}
	strategy = normalizeRoutingStrategy(strategy)
	priceSum, availabilitySum, weightSum := 0.0, 0.0, 0.0
	totalCapacity, totalCurrent := 0.0, 0.0
	for _, candidate := range candidates {
		if candidate.channel == nil {
			continue
		}
		price, availability, _ := dynamicCandidateFeatures(candidate, minPrice)
		weight := float64(dynamicCandidateSelectionWeight(candidate))
		if weight <= 0 {
			// A zero route weight does not participate in normal traffic. A
			// forced channel is the deliberate exception, and still needs a
			// finite group score so its hard layer can be selected.
			if candidate.channel == nil || !candidate.channel.IsForcePriority() {
				continue
			}
			weight = 1
		}
		priceSum += price * weight
		availabilitySum += availability * weight
		weightSum += weight

		// At the group level, load represents aggregate remaining capacity.
		// Route weight is a traffic preference, not a capacity declaration.
		if maxConcurrency := candidate.channel.GetMaxConcurrency(); maxConcurrency > 0 {
			totalCapacity += float64(maxConcurrency)
			current := CurrentChannelConcurrency(candidate.channel.Id)
			if current < 0 {
				current = 0
			}
			if current > maxConcurrency {
				current = maxConcurrency
			}
			totalCurrent += float64(current)
		}
	}
	if weightSum == 0 {
		return math.Inf(-1)
	}
	loadScore := 1.0
	if totalCapacity > 0 {
		loadScore = 1 - totalCurrent/totalCapacity
		loadScore = math.Min(1, math.Max(0, loadScore))
	}
	return 100 * ((strategy.PriceWeight/100)*(priceSum/weightSum) +
		(strategy.AvailabilityWeight/100)*(availabilitySum/weightSum) +
		(strategy.LoadWeight/100)*loadScore)
}

func smoothWeightedCandidateIndex(p *RetryParam, candidates []dynamicChannelCandidate) int {
	if len(candidates) == 0 {
		return -1
	}
	totalWeight := 0.0
	selected := -1
	selectedCurrent := math.Inf(-1)
	deltas := make(map[string]float64, len(candidates))
	dynamicScheduleState.Lock()
	for _, candidate := range candidates {
		weight := dynamicCandidateSelectionWeightFloat(candidate)
		if weight < 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
			weight = 0
		}
		totalWeight += weight
		key := dynamicScheduleKey(candidate)
		dynamicScheduleState.current[key] += weight
		deltas[key] += weight
		if dynamicScheduleState.current[key] > selectedCurrent {
			selectedCurrent = dynamicScheduleState.current[key]
			selected = candidate.channel.Id
		}
	}
	if selected < 0 || totalWeight <= 0 {
		dynamicScheduleState.Unlock()
		return -1
	}
	for _, candidate := range candidates {
		if candidate.channel.Id == selected {
			key := dynamicScheduleKey(candidate)
			dynamicScheduleState.current[key] -= totalWeight
			deltas[key] -= totalWeight
			break
		}
	}
	dynamicScheduleState.Unlock()
	p.recordRoutingScheduleDeltas(false, deltas)
	for index, candidate := range candidates {
		if candidate.channel.Id == selected {
			return index
		}
	}
	return -1
}

func dynamicScheduleKey(candidate dynamicChannelCandidate) string {
	if candidate.channel == nil {
		return candidate.group + ":nil"
	}
	scope := "normal"
	if candidate.channel != nil && candidate.channel.IsForcePriority() {
		scope = candidate.channel.GetForcePriorityScope() + ":" + strconv.Itoa(forcePriorityLevel(candidate))
	}
	return candidate.group + ":" + scope + ":" + strconv.Itoa(candidate.channel.Id)
}

func annotateDynamicCandidateScores(candidates []dynamicChannelCandidate) {
	byGroup := make(map[string][]int)
	for index := range candidates {
		byGroup[candidates[index].group] = append(byGroup[candidates[index].group], index)
	}
	for _, indices := range byGroup {
		minPrice := math.Inf(1)
		for _, index := range indices {
			price := channelComparableCost(candidates[index])
			if price >= 0 && !math.IsNaN(price) && !math.IsInf(price, 0) && price < minPrice {
				minPrice = price
			}
		}
		strategy := routingStrategyForCandidates(candidatesForIndices(candidates, indices))
		maxScore := 0.0
		for _, index := range indices {
			candidate := &candidates[index]
			candidate.score = dynamicCandidateScore(*candidate, minPrice, strategy)
			if candidate.score > maxScore {
				maxScore = candidate.score
			}
		}
		for _, index := range indices {
			candidate := &candidates[index]
			baseWeight := float64(dynamicCandidateSelectionWeight(*candidate))
			utilization := dynamicCandidateUtilization(*candidate)
			loadFactor := math.Pow(math.Max(0, 1-utilization), 1.5)
			if loadFactor < 0.25 {
				loadFactor = 0.25
			}
			scoreFactor := math.Exp((candidate.score - maxScore) / dynamicScoreTemperature)
			candidate.selectionWeight = baseWeight * loadFactor * scoreFactor
		}
	}
}

func candidatesForIndices(candidates []dynamicChannelCandidate, indices []int) []dynamicChannelCandidate {
	result := make([]dynamicChannelCandidate, 0, len(indices))
	for _, index := range indices {
		if index >= 0 && index < len(candidates) {
			result = append(result, candidates[index])
		}
	}
	return result
}

func dynamicCandidateScore(candidate dynamicChannelCandidate, minPrice float64, strategy ratio_setting.PricingGroupRoutingStrategy) float64 {
	if candidate.channel == nil {
		return 0
	}
	strategy = normalizeRoutingStrategy(strategy)
	priceScore, availability, load := dynamicCandidateFeatures(candidate, minPrice)
	return 100 * ((strategy.PriceWeight/100)*priceScore +
		(strategy.AvailabilityWeight/100)*availability +
		(strategy.LoadWeight/100)*load)
}

func dynamicCandidateUtilization(candidate dynamicChannelCandidate) float64 {
	if candidate.channel == nil {
		return 1
	}
	maxConcurrency := candidate.channel.GetMaxConcurrency()
	if maxConcurrency <= 0 {
		return 0
	}
	current := CurrentChannelConcurrency(candidate.channel.Id)
	utilization := float64(current) / float64(maxConcurrency)
	return math.Min(1, math.Max(0, utilization))
}

func dynamicCandidatePool(candidates []dynamicChannelCandidate) []dynamicChannelCandidate {
	if len(candidates) == 0 {
		return nil
	}
	first := candidates[0]
	if first.channel == nil {
		return nil
	}
	pool := make([]dynamicChannelCandidate, 0, len(candidates))
	maxScore := first.score
	for _, candidate := range candidates {
		if candidate.channel == nil {
			continue
		}
		if candidate.groupIndex != first.groupIndex || candidate.channel.IsForcePriority() != first.channel.IsForcePriority() ||
			(candidate.channel.IsForcePriority() && (candidate.channel.GetForcePriorityScope() != first.channel.GetForcePriorityScope() ||
				forcePriorityLevel(candidate) != forcePriorityLevel(first))) {
			continue
		}
		if candidate.score >= maxScore-dynamicScoreTolerance {
			pool = append(pool, candidate)
		}
	}
	if len(pool) == 0 {
		return []dynamicChannelCandidate{first}
	}
	return pool
}

func dynamicCandidateSelectionWeightFloat(candidate dynamicChannelCandidate) float64 {
	if candidate.selectionWeight > 0 {
		return candidate.selectionWeight
	}
	return float64(dynamicCandidateSelectionWeight(candidate))
}

func weightedDynamicCandidateIndexFloat(candidates []dynamicChannelCandidate, randomWeight float64) int {
	if len(candidates) == 0 {
		return -1
	}
	for index, candidate := range candidates {
		randomWeight -= dynamicCandidateSelectionWeightFloat(candidate)
		if randomWeight < 0 {
			return index
		}
	}
	return len(candidates) - 1
}

func maxIntWeight(weight float64) int {
	if weight <= 0 || math.IsNaN(weight) {
		return 1
	}
	if weight > float64(int(^uint(0)>>1)) {
		return int(^uint(0) >> 1)
	}
	return int(math.Ceil(weight))
}

func moveCandidateToFront(candidates []dynamicChannelCandidate, selected dynamicChannelCandidate) {
	for index, candidate := range candidates {
		if candidate.channel.Id != selected.channel.Id || candidate.group != selected.group {
			continue
		}
		if index > 0 {
			copy(candidates[1:index+1], candidates[:index])
			candidates[0] = selected
		}
		return
	}
}

func (p *RetryParam) prioritizeCurrentChannel(candidates []dynamicChannelCandidate) bool {
	if p == nil || !p.concreteAttempted || p.currentChannelID <= 0 || len(candidates) == 0 || p.isExcluded(p.currentChannelID) {
		return false
	}
	if p.attemptCounts[p.currentChannelID] >= p.channelLimits[p.currentChannelID] {
		return false
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
		return true
	}
	return false
}

func dynamicCandidateLess(left, right dynamicChannelCandidate) bool {
	if left.channel == nil || right.channel == nil {
		return left.channel != nil
	}
	leftCrossForce := left.channel.IsForcePriority() && left.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeCrossGroup
	rightCrossForce := right.channel.IsForcePriority() && right.channel.GetForcePriorityScope() == model.ChannelForcePriorityScopeCrossGroup
	if leftCrossForce != rightCrossForce {
		return leftCrossForce
	}
	if leftCrossForce && rightCrossForce {
		if leftLevel, rightLevel := forcePriorityLevel(left), forcePriorityLevel(right); leftLevel != rightLevel {
			return leftLevel < rightLevel
		}
	}
	// Force priority is a hard layer inside one pricing group. Group-level
	// selection happens before this comparator when cross-group routing is on.
	if left.groupIndex == right.groupIndex {
		leftForce := left.channel.IsForcePriority()
		rightForce := right.channel.IsForcePriority()
		if leftForce != rightForce {
			return leftForce
		}
		if leftForce && rightForce {
			if leftLevel, rightLevel := forcePriorityLevel(left), forcePriorityLevel(right); leftLevel != rightLevel {
				return leftLevel < rightLevel
			}
		}
	} else if left.groupIndex != right.groupIndex {
		return left.groupIndex < right.groupIndex
	}
	if left.score > 0 || right.score > 0 {
		if math.Abs(left.score-right.score) > 0.000001 {
			return left.score > right.score
		}
	}
	leftCost := channelComparableCost(left)
	rightCost := channelComparableCost(right)
	if leftCost != rightCost {
		return leftCost < rightCost
	}
	leftAvailability := dynamicAvailabilityScore(left.channel)
	rightAvailability := dynamicAvailabilityScore(right.channel)
	if math.Abs(leftAvailability-rightAvailability) > 0.000001 {
		return leftAvailability > rightAvailability
	}
	leftLoad := 1 - dynamicCandidateUtilization(left)
	rightLoad := 1 - dynamicCandidateUtilization(right)
	if math.Abs(leftLoad-rightLoad) > 0.000001 {
		return leftLoad > rightLoad
	}
	if math.Abs(left.selectionWeight-right.selectionWeight) > 0.000001 {
		return left.selectionWeight > right.selectionWeight
	}
	if left.routeWeight != right.routeWeight {
		return left.routeWeight > right.routeWeight
	}
	if left.routePriority != right.routePriority {
		return left.routePriority < right.routePriority
	}
	if left.routeOrder != right.routeOrder {
		return left.routeOrder < right.routeOrder
	}
	return left.channel.Id < right.channel.Id
}

func dynamicCandidateSelectionWeight(candidate dynamicChannelCandidate) int {
	if candidate.routeConfigured {
		if candidate.routeWeight < 0 {
			return 1
		}
		return candidate.routeWeight
	}
	return 1
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
	if candidate.channel == nil {
		return math.Inf(1)
	}
	cost := candidate.channel.GetPriceMultiplier()
	// Billing uses the requested/origin model. A channel mapping changes the
	// upstream payload, but it must not silently change the model price used to
	// rank candidates. Only fall back to the mapped name when the requested
	// model has no configured price or ratio.
	if modelFactor, configured := configuredModelCost(candidate.modelName, candidate.channel); configured {
		cost *= modelFactor
	}
	if candidate.channel.GetPriceMultiplierMode() == model.ChannelPriceMultiplierModeCNY {
		rate := operation_setting.GetBillingUSDToCNYRate()
		if rate > 0 {
			cost /= rate
		}
	}
	return cost * normalizeRouteCostFactor(candidate.routeCostFactor)
}

func configuredModelCost(requested string, channel *model.Channel) (float64, bool) {
	names := make([]string, 0, 2)
	requested = strings.TrimSpace(requested)
	if requested != "" {
		names = append(names, requested)
	}
	if mapped := mappedRoutingModelName(channel, requested); mapped != "" && mapped != requested {
		names = append(names, mapped)
	}
	for _, name := range names {
		if price, ok := ratio_setting.GetModelPrice(name, false); ok && price >= 0 && !math.IsNaN(price) && !math.IsInf(price, 0) {
			return price, true
		}
		if ratio, ok, _ := ratio_setting.GetModelRatio(name); ok && ratio >= 0 && !math.IsNaN(ratio) && !math.IsInf(ratio, 0) {
			return ratio, true
		}
	}
	return 1, false
}

func mappedRoutingModelName(channel *model.Channel, requested string) string {
	requested = strings.TrimSpace(requested)
	if channel == nil || requested == "" {
		return requested
	}
	raw := strings.TrimSpace(channel.GetModelMapping())
	if raw == "" || raw == "{}" {
		return requested
	}
	mapping := make(map[string]string)
	if err := common.UnmarshalJsonStr(raw, &mapping); err != nil {
		return requested
	}
	current := requested
	visited := map[string]struct{}{current: {}}
	for range mapping {
		next := strings.TrimSpace(mapping[current])
		if next == "" {
			break
		}
		if _, seen := visited[next]; seen {
			break
		}
		visited[next] = struct{}{}
		current = next
	}
	return current
}

func normalizeRouteCostFactor(value float64) float64 {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 1
	}
	return value
}
