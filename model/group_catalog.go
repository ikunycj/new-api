package model

import (
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func addPricingGroupName(names map[string]struct{}, rawName string) {
	name := strings.TrimSpace(rawName)
	if name == "" || name == "auto" {
		return
	}
	names[name] = struct{}{}
}

// GetPricingGroupNames returns the configured pricing/routing groups. Persisted
// channel, ability, and token values are deliberately not part of this
// catalog: historical references must not create new selectable groups.
func GetPricingGroupNames() ([]string, error) {
	names := make(map[string]struct{})
	for name := range ratio_setting.GetGroupRatioCopy() {
		addPricingGroupName(names, name)
	}
	for name := range setting.GetUserUsableGroupsCopy() {
		addPricingGroupName(names, name)
	}

	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}
