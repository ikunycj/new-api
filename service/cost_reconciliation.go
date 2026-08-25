package service

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

const costReconciliationVersion = 1

type failedAttemptUsage struct {
	ChannelID int
	Group     string
	Usage     dto.Usage
}

const failedAttemptUsageContextKey = "cost_reconciliation_failed_attempts"

func RecordFailedAttemptUsage(c *gin.Context, group string, channelID int, usage *dto.Usage) {
	if c == nil || usage == nil || usage.PromptTokens+usage.CompletionTokens <= 0 {
		return
	}
	items, _ := c.Get(failedAttemptUsageContextKey)
	attempts, _ := items.([]failedAttemptUsage)
	attempts = append(attempts, failedAttemptUsage{ChannelID: channelID, Group: group, Usage: *usage})
	c.Set(failedAttemptUsageContextKey, attempts)
}

// BuildCostReconciliationSnapshot freezes the billing inputs needed to compare
// the user's final charge with an estimated provider cost. Retry attempts use
// the final observed usage as a conservative proxy when an upstream failure
// did not expose partial usage; the basis is recorded for auditability.
func BuildCostReconciliationSnapshot(c *gin.Context, info *relaycommon.RelayInfo, promptTokens, completionTokens, quota int, modelPrice, groupRatio, billingRate float64) map[string]interface{} {
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
	snapshot := map[string]interface{}{
		"version":                 costReconciliationVersion,
		"user_charge_usd_micros":  int64(chargeUSD * 1_000_000),
		"group_ratio":             groupRatio,
		"billing_usd_to_cny_rate": billingRate,
		"quota_per_unit":          common.QuotaPerUnit,
		"prompt_tokens":           promptTokens,
		"completion_tokens":       completionTokens,
		"model_price":             modelPrice,
	}
	if groupRatio <= 0 || math.IsNaN(groupRatio) || math.IsInf(groupRatio, 0) {
		snapshot["status"] = "unavailable"
		return snapshot
	}
	baseUSD := chargeUSD / groupRatio
	channels := c.GetStringSlice("use_channel")
	groups := c.GetStringSlice("use_channel_groups")
	if len(channels) == 0 {
		channels = []string{fmt.Sprintf("%d", info.ChannelId)}
	}
	var successfulCost, retryCost, failedPartialCost float64
	finalTokens := promptTokens + completionTokens
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
		cost := baseUSD * factor
		if i == len(channels)-1 {
			successfulCost += cost
		} else {
			attempts := failedByChannel[channelID]
			if len(attempts) > 0 && finalTokens > 0 {
				attempt := attempts[0]
				failedByChannel[channelID] = attempts[1:]
				partialTokens := attempt.Usage.PromptTokens + attempt.Usage.CompletionTokens
				failedPartialCost += cost * float64(partialTokens) / float64(finalTokens)
			} else {
				retryCost += cost
			}
		}
	}
	snapshot["successful_cost_usd_micros"] = int64(successfulCost * 1_000_000)
	snapshot["retry_cost_usd_micros"] = int64(retryCost * 1_000_000)
	snapshot["failed_partial_usage_cost_usd_micros"] = int64(failedPartialCost * 1_000_000)
	snapshot["estimated_cost_usd_micros"] = int64((successfulCost + retryCost + failedPartialCost) * 1_000_000)
	snapshot["status"] = "estimated"
	snapshot["retry_cost_basis"] = "final_usage_proxy"
	if len(failedAttempts) > 0 {
		snapshot["failed_partial_usage_basis"] = "observed_usage"
	}
	snapshot["channel_cost_factor_source"] = "billing_group_route_or_official_price_default"
	return snapshot
}
