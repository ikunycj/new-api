package model

import (
	"strings"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// GetPricingGroupNames returns the configured pricing/routing groups. Persisted
// channel, ability, and token values are deliberately not part of this
// catalog: historical references must not create new selectable groups.
func GetPricingGroupNames() ([]string, error) {
	result := make([]string, 0)
	for _, rawName := range ratio_setting.GetPricingGroupOrder() {
		name := strings.TrimSpace(rawName)
		if name == "" || name == "auto" || name == setting.AllPricingGroups {
			continue
		}
		result = append(result, name)
	}
	return result, nil
}
