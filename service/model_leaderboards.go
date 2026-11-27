package service

import (
	"context"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/QuantumNous/new-api/model"
)

func SyncModelLeaderboards(ctx context.Context) error {
	boards, err := fetchModelLeaderboards(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return model.SaveModelLeaderboardSnapshot(ctx, model.ModelLeaderboardSnapshot{
		UpdatedAt: &now,
		Boards:    boards,
	})
}

func MatchRecommendedModels(recommendedModels []string, pricing []model.Pricing) []string {
	matched := make([]string, 0, len(recommendedModels))
	if len(recommendedModels) == 0 {
		return matched
	}
	catalog := make(map[string]struct{}, len(pricing))
	for _, item := range pricing {
		catalog[item.ModelName] = struct{}{}
	}
	for _, name := range recommendedModels {
		if _, ok := catalog[name]; ok {
			matched = append(matched, name)
		}
	}
	return matched
}

// MatchModelLeaderboardEntries connects Arena display names to the current
// catalog. Exact effort variants keep their rank. An unqualified catalog model
// takes the best rank published for its base model.
func MatchModelLeaderboardEntries(entries []model.ModelLeaderboardEntry, pricing []model.Pricing) []model.ModelLeaderboardEntry {
	if len(entries) == 0 {
		return entries
	}
	catalog := make(map[string][]string, len(pricing))
	for _, item := range pricing {
		key := modelLeaderboardNameKey(item.ModelName)
		catalog[key] = append(catalog[key], item.ModelName)
	}
	seenIDs := make(map[string]bool, len(pricing))
	for i := range entries {
		entry := &entries[i]
		display := entry.ModelName
		keys := []string{modelLeaderboardNameKey(display)}
		if prefix, effort, found := strings.Cut(display, " ("); found && strings.HasSuffix(effort, ")") {
			switch strings.ToLower(strings.TrimSuffix(effort, ")")) {
			case "max", "xhigh", "high", "medium", "low", "minimal", "none", "thinking":
				keys = append(keys, modelLeaderboardNameKey(prefix))
			}
		}
		for _, key := range keys {
			for _, id := range catalog[key] {
				if seenIDs[id] {
					continue
				}
				seenIDs[id] = true
				entry.ModelIDs = append(entry.ModelIDs, id)
			}
		}
		slices.Sort(entry.ModelIDs)
	}
	return entries
}

func modelLeaderboardNameKey(name string) string {
	parts := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.Join(parts, "-")
}
