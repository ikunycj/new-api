package controller

import (
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func filterVendorsByPricing(vendors []model.PricingVendor, pricing []model.Pricing) []model.PricingVendor {
	usedVendorIDs := make(map[int]struct{})
	for _, item := range pricing {
		if item.VendorID != 0 {
			usedVendorIDs[item.VendorID] = struct{}{}
		}
	}

	filtered := make([]model.PricingVendor, 0, len(usedVendorIDs))
	for _, vendor := range vendors {
		if _, ok := usedVendorIDs[vendor.ID]; ok {
			filtered = append(filtered, vendor)
		}
	}
	return filtered
}

func GetPricing(c *gin.Context) {
	pricing := model.GetPricing()

	// The model square is a public catalog, so its group list is based on the
	// default account group's configured visibility rather than the viewer's
	// permissions or each model's currently enabled abilities.
	usableGroup := service.GetUserGroupVisiblePricingGroups(model.DefaultUserGroup)
	groupRatio := ratio_setting.GetGroupRatioCopy()
	for group := range groupRatio {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}

	c.JSON(200, gin.H{
		"success":            true,
		"data":               pricing,
		"vendors":            filterVendorsByPricing(model.GetVendors(), pricing),
		"group_ratio":        groupRatio,
		"usable_group":       usableGroup,
		"supported_endpoint": model.GetSupportedEndpointMap(),
		"pricing_version":    "a42d372ccf0b5dd13ecf71203521f9d2",
	})
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
	err := model.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = ratio_setting.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "重置模型倍率成功",
	})
}
