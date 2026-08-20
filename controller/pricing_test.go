package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestFilterVendorsByPricingUsesDistinctReferencedVendors(t *testing.T) {
	vendors := []model.PricingVendor{
		{ID: 1, Name: "OpenAI"},
		{ID: 2, Name: "xAI"},
		{ID: 3, Name: "讯飞"},
	}
	pricing := []model.Pricing{
		{ModelName: "gpt-5", VendorID: 1, EnableGroup: []string{"default", "vip"}},
		{ModelName: "gpt-5-mini", VendorID: 1, EnableGroup: []string{"default"}},
		{ModelName: "grok-4", VendorID: 2, EnableGroup: []string{"vip"}},
	}

	filtered := filterVendorsByPricing(vendors, pricing)

	require.Equal(t, []model.PricingVendor{
		{ID: 1, Name: "OpenAI"},
		{ID: 2, Name: "xAI"},
	}, filtered)
}

func TestFilterVendorsByPricingReturnsEmptyWithoutVendorReferences(t *testing.T) {
	vendors := []model.PricingVendor{{ID: 1, Name: "OpenAI"}}
	pricing := []model.Pricing{{ModelName: "unmapped-model", VendorID: 0}}

	require.Empty(t, filterVendorsByPricing(vendors, pricing))
}
