package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
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

func TestGetPricingUsesDefaultGroupCatalogWithoutAbilityFiltering(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	previousRatios := ratio_setting.GroupRatio2JSONString()
	previousEnabled := ratio_setting.PricingGroupEnabled2JSONString()
	previousOrder := ratio_setting.PricingGroupOrder2JSONString()
	previousPermissions := setting.UserGroupPricingGroups2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatios))
		require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(previousEnabled))
		require.NoError(t, ratio_setting.UpdatePricingGroupOrderByJSONString(previousOrder))
		require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(previousPermissions))
		model.InvalidatePricingCache()
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"disabled":0.8,"viewer-only":0.6}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupEnabledByJSONString(`{"default":true,"disabled":false,"viewer-only":true}`))
	require.NoError(t, ratio_setting.UpdatePricingGroupOrderByJSONString(`["disabled","default","viewer-only"]`))
	require.NoError(t, setting.UpdateUserGroupPricingGroupsByJSONString(`{"default":["default","disabled"],"viewer":["viewer-only"]}`))

	require.NoError(t, db.Create(&model.User{
		Id:       9201,
		Username: "pricing-catalog-viewer",
		Password: "password",
		Group:    "viewer",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     "viewer-only",
		Model:     "pricing-catalog-model",
		ChannelId: 9201,
		Enabled:   true,
	}).Error)
	model.InvalidatePricingCache()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/pricing", nil)
	ctx.Set("id", 9201)

	GetPricing(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Success     bool               `json:"success"`
		Data        []model.Pricing    `json:"data"`
		GroupRatio  map[string]float64 `json:"group_ratio"`
		UsableGroup map[string]string  `json:"usable_group"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Equal(t, map[string]float64{"default": 1, "disabled": 0.8}, payload.GroupRatio)
	require.Equal(t, map[string]string{"default": "default", "disabled": "disabled"}, payload.UsableGroup)
	require.Len(t, payload.Data, 1)
	require.Equal(t, "pricing-catalog-model", payload.Data[0].ModelName)
}
