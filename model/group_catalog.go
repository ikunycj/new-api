package model

import (
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// GetPricingGroupNames returns the configured pricing/routing groups. Persisted
// channel, ability, and token values are deliberately not part of this
// catalog: historical references must not create new selectable groups.
func GetPricingGroupNames() ([]string, error) {
	names := make(map[string]struct{})
	for _, rawName := range ratio_setting.GetPricingGroupOrder() {
		name := strings.TrimSpace(rawName)
		if name == "" || name == "auto" || name == setting.AllPricingGroups {
			continue
		}
		names[name] = struct{}{}
	}
	// A routed billing group is a pricing group even before its ratio is
	// explicitly added to GroupRatio. This keeps newly-created route packages
	// selectable in user-group management instead of silently hiding them.
	if DB != nil && DB.Migrator().HasTable(&BillingGroupRoute{}) {
		var routedNames []string
		if err := DB.Model(&BillingGroupRoute{}).Pluck("billing_group", &routedNames).Error; err != nil {
			return nil, err
		}
		for _, rawName := range routedNames {
			name := strings.TrimSpace(rawName)
			if name == "" || name == "auto" || name == setting.AllPricingGroups {
				continue
			}
			names[name] = struct{}{}
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i] == "default" {
			return true
		}
		if result[j] == "default" {
			return false
		}
		return result[i] < result[j]
	})
	return result, nil
}
