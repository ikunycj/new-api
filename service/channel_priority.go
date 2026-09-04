package service

import (
	"math"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// CalculateChannelPriorityScores annotates result channels with the score
// calculated from the complete reference set for one billing group. The
// reference set is supplied by the caller so search/name/type filters cannot
// change the normalization baseline.
func CalculateChannelPriorityScores(result, reference []*model.Channel, group string) {
	for _, channel := range result {
		if channel != nil {
			channel.PriorityScore = nil
		}
	}
	if len(result) == 0 || len(reference) == 0 || group == "" {
		return
	}

	candidates := buildChannelPriorityCandidates(reference, group)
	if len(candidates) == 0 {
		return
	}
	annotateDynamicCandidateScores(candidates)

	byID := make(map[int]*model.Channel, len(result))
	for _, channel := range result {
		if channel != nil {
			byID[channel.Id] = channel
		}
	}
	for _, candidate := range candidates {
		if candidate.channel == nil {
			continue
		}
		if channel := byID[candidate.channel.Id]; channel != nil {
			score := normalizePriorityScore(candidate.score)
			channel.PriorityScore = &score
		}
	}
}

// SortChannelsByPriority sorts result channels after scoring against the full
// reference set. Dynamic scores are not persisted, so pagination must happen
// after this function returns.
func SortChannelsByPriority(result, reference []*model.Channel, group string, descending bool) {
	if len(result) == 0 || len(reference) == 0 || group == "" {
		return
	}

	candidates := buildChannelPriorityCandidates(reference, group)
	if len(candidates) == 0 {
		return
	}
	annotateDynamicCandidateScores(candidates)
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		leftEnabled := left.channel != nil && left.channel.Status == common.ChannelStatusEnabled
		rightEnabled := right.channel != nil && right.channel.Status == common.ChannelStatusEnabled
		if leftEnabled != rightEnabled {
			return leftEnabled
		}
		if descending {
			return dynamicCandidateLess(left, right)
		}
		return dynamicCandidateLess(right, left)
	})

	byID := make(map[int]*model.Channel, len(result))
	for _, channel := range result {
		if channel != nil {
			byID[channel.Id] = channel
		}
	}
	ordered := make([]*model.Channel, 0, len(result))
	seen := make(map[int]struct{}, len(result))
	for _, candidate := range candidates {
		if candidate.channel == nil {
			continue
		}
		channel := byID[candidate.channel.Id]
		if channel == nil {
			continue
		}
		score := normalizePriorityScore(candidate.score)
		channel.PriorityScore = &score
		if _, exists := seen[channel.Id]; exists {
			continue
		}
		seen[channel.Id] = struct{}{}
		ordered = append(ordered, channel)
	}
	for _, channel := range result {
		if channel == nil {
			ordered = append(ordered, nil)
			continue
		}
		if _, exists := seen[channel.Id]; !exists {
			ordered = append(ordered, channel)
		}
	}
	copy(result, ordered)
}

func buildChannelPriorityCandidates(channels []*model.Channel, group string) []dynamicChannelCandidate {
	policy, entries, configured := model.ResolveBillingGroupRoute(group)
	entryByID := make(map[int]dynamicChannelCandidate, len(entries))
	if configured {
		for order, entry := range entries {
			if !entry.Enabled || entry.ChannelId <= 0 {
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

	candidates := make([]dynamicChannelCandidate, 0, len(channels))
	for _, channel := range channels {
		if channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		candidate := dynamicChannelCandidate{
			channel:         channel,
			group:           group,
			groupIndex:      0,
			routeConfigured: configured,
			routeCostFactor: 1,
			routingStrategy: policy.RoutingStrategy,
		}
		if configured {
			entry, ok := entryByID[channel.Id]
			if !ok || (entry.routeWeight == 0 && !channel.IsForcePriority()) {
				continue
			}
			candidate.routePriority = entry.routePriority
			candidate.routeOrder = entry.routeOrder
			candidate.routeWeight = entry.routeWeight
			candidate.routeCostFactor = entry.routeCostFactor
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func normalizePriorityScore(score float64) float64 {
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0
	}
	return math.Min(100, math.Max(0, score))
}
