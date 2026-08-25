package service

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

const costReconciliationVersion = 2

type failedAttemptUsage struct {
	ChannelID           int
	Group               string
	ProviderBaseCostUSD float64
}

const failedAttemptUsageContextKey = "cost_reconciliation_failed_attempts"

func usdMicros(value float64) int64 {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value >= float64(math.MaxInt64)/1_000_000 {
		return math.MaxInt64
	}
	return int64(value * 1_000_000)
}

func RecordFailedAttemptUsage(c *gin.Context, info *relaycommon.RelayInfo, group string, channelID int, usage *dto.Usage) {
	if c == nil || info == nil || usage == nil {
		return
	}
	billingUsage := effectiveBillingUsage(usage)
	if billingUsage == nil || billingUsage.PromptTokens+billingUsage.CompletionTokens <= 0 {
		return
	}
	summary := calculateTextQuotaSummary(c, info, billingUsage)
	applyTieredProviderCost(info, billingUsage, &summary)
	if !summary.ProviderCostAvailable {
		return
	}
	items, _ := c.Get(failedAttemptUsageContextKey)
	attempts, _ := items.([]failedAttemptUsage)
	attempts = append(attempts, failedAttemptUsage{
		ChannelID:           channelID,
		Group:               group,
		ProviderBaseCostUSD: summary.ProviderBaseCostUSD,
	})
	c.Set(failedAttemptUsageContextKey, attempts)
}

// BuildCostReconciliationSnapshot freezes the billing inputs needed to compare
// the user's final charge with an estimated provider cost. Provider cost is
// calculated independently from actual usage and frozen model pricing, then
// multiplied by each attempted channel's cost factor. Retry attempts without
// usage use the successful attempt's actual-usage cost as a conservative proxy.
func BuildCostReconciliationSnapshot(c *gin.Context, info *relaycommon.RelayInfo, promptTokens, completionTokens, quota int, providerBaseCostUSD float64, providerCostAvailable bool, costBasis string, billingRate float64) map[string]interface{} {
	if info == nil {
		return map[string]interface{}{"version": costReconciliationVersion, "status": "unavailable"}
	}
	if billingRate <= 0 || math.IsNaN(billingRate) || math.IsInf(billingRate, 0) {
		billingRate = 1
	}
	chargeUSD := float64(quota) / common.QuotaPerUnit / billingRate
	if chargeUSD < 0 || math.IsNaN(chargeUSD) || math.IsInf(chargeUSD, 0) {
		chargeUSD = 0
	}
	providerCostValid := providerCostAvailable && providerBaseCostUSD >= 0 &&
		!math.IsNaN(providerBaseCostUSD) && !math.IsInf(providerBaseCostUSD, 0)
	snapshot := map[string]interface{}{
		"version":                       costReconciliationVersion,
		"user_charge_usd_micros":        usdMicros(chargeUSD),
		"provider_base_cost_usd_micros": usdMicros(providerBaseCostUSD),
		"billing_usd_to_cny_rate":       billingRate,
		"quota_per_unit":                common.QuotaPerUnit,
		"prompt_tokens":                 promptTokens,
		"completion_tokens":             completionTokens,
	}
	if !providerCostValid {
		snapshot["status"] = "unavailable"
		return snapshot
	}
	channels := c.GetStringSlice("use_channel")
	groups := c.GetStringSlice("use_channel_groups")
	if len(channels) == 0 {
		channels = []string{fmt.Sprintf("%d", info.ChannelId)}
	}
	var successfulCost, retryCost, failedPartialCost float64
	items, _ := c.Get(failedAttemptUsageContextKey)
	failedAttempts, _ := items.([]failedAttemptUsage)
	failedByChannel := make(map[int][]failedAttemptUsage)
	for _, attempt := range failedAttempts {
		failedByChannel[attempt.ChannelID] = append(failedByChannel[attempt.ChannelID], attempt)
	}
	for i, channelText := range channels {
		channelID := 0
		if _, err := fmt.Sscan(channelText, &channelID); err != nil {
			channelID = 0
		}
		group := info.UsingGroup
		if i < len(groups) && groups[i] != "" {
			group = groups[i]
		}
		factor := model.ResolveChannelCostFactor(group, channelID)
		cost := providerBaseCostUSD * factor
		if i == len(channels)-1 {
			successfulCost += cost
		} else {
			attempts := failedByChannel[channelID]
			if len(attempts) > 0 {
				attempt := attempts[0]
				failedByChannel[channelID] = attempts[1:]
				attemptFactor := factor
				if attempt.Group != "" {
					attemptFactor = model.ResolveChannelCostFactor(attempt.Group, channelID)
				}
				failedPartialCost += attempt.ProviderBaseCostUSD * attemptFactor
			} else {
				retryCost += cost
			}
		}
	}
	snapshot["successful_cost_usd_micros"] = usdMicros(successfulCost)
	snapshot["retry_cost_usd_micros"] = usdMicros(retryCost)
	snapshot["failed_partial_usage_cost_usd_micros"] = usdMicros(failedPartialCost)
	snapshot["estimated_cost_usd_micros"] = usdMicros(successfulCost + retryCost + failedPartialCost)
	snapshot["status"] = "estimated"
	if costBasis == "" {
		costBasis = "actual_usage_model_pricing_x_channel_cost_factor"
	}
	snapshot["cost_basis"] = costBasis
	snapshot["retry_cost_basis"] = "successful_actual_usage_proxy"
	if len(failedAttempts) > 0 {
		snapshot["failed_partial_usage_basis"] = "observed_usage"
	}
	snapshot["channel_cost_factor_source"] = "billing_group_route_or_official_price_default"
	return snapshot
}

func providerBaseCostFromTieredResult(info *relaycommon.RelayInfo, result *billingexpr.TieredResult) (float64, bool) {
	if info == nil || result == nil || info.TieredBillingSnapshot == nil {
		return 0, false
	}
	rate := info.TieredBillingSnapshot.EffectiveBillingUSDToCNYRate()
	cost := result.ActualQuotaBeforeGroup / common.QuotaPerUnit / rate
	return cost, cost >= 0 && !math.IsNaN(cost) && !math.IsInf(cost, 0)
}
