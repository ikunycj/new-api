package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionUsagePercent(t *testing.T) {
	assert.Equal(t, 0.0, SubscriptionUsagePercent(10, 0))
	assert.Equal(t, 0.0, SubscriptionUsagePercent(-5, 100))
	assert.Equal(t, 33.33, SubscriptionUsagePercent(1, 3))
	assert.Equal(t, 100.0, SubscriptionUsagePercent(100, 100))
}

func TestApplyUserQuotaDisplayHidesAmountsForPercentPlans(t *testing.T) {
	truncateTables(t)

	now := GetDBTimestamp()
	percentPlan := &SubscriptionPlan{Id: 9301, Title: "Team 200", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 100000000, QuotaResetPeriod: SubscriptionResetMonthly, QuotaDisplay: SubscriptionQuotaDisplayPercent}
	amountPlan := &SubscriptionPlan{Id: 9302, Title: "Pro", DurationUnit: SubscriptionDurationMonth, DurationValue: 1, TotalAmount: 1000}
	require.NoError(t, DB.Create(percentPlan).Error)
	require.NoError(t, DB.Create(amountPlan).Error)
	InvalidateSubscriptionPlanCache(percentPlan.Id)
	InvalidateSubscriptionPlanCache(amountPlan.Id)

	percentSub := &UserSubscription{Id: 9401, UserId: 301, PlanId: percentPlan.Id, AmountTotal: 100000000, AmountUsed: 25000000, StartTime: now, EndTime: now + 3600, Status: "active"}
	amountSub := &UserSubscription{Id: 9402, UserId: 301, PlanId: amountPlan.Id, AmountTotal: 1000, AmountUsed: 100, StartTime: now, EndTime: now + 3600, Status: "active"}
	require.NoError(t, DB.Create(percentSub).Error)
	require.NoError(t, DB.Create(amountSub).Error)

	summaries, err := GetAllUserSubscriptions(301)
	require.NoError(t, err)
	result := ApplyUserQuotaDisplay(summaries)
	require.Len(t, result, 2)

	byId := map[int]SubscriptionSummary{}
	for _, summary := range result {
		byId[summary.Subscription.Id] = summary
	}

	percent := byId[percentSub.Id]
	assert.Equal(t, SubscriptionQuotaDisplayPercent, percent.QuotaDisplay)
	assert.Equal(t, 25.0, percent.UsagePercent)
	assert.Zero(t, percent.Subscription.AmountTotal)
	assert.Zero(t, percent.Subscription.AmountUsed)

	amount := byId[amountSub.Id]
	assert.Equal(t, SubscriptionQuotaDisplayAmount, amount.QuotaDisplay)
	assert.Equal(t, 10.0, amount.UsagePercent)
	assert.Equal(t, int64(1000), amount.Subscription.AmountTotal)
	assert.Equal(t, int64(100), amount.Subscription.AmountUsed)

	// The source summaries (used by admin APIs) must keep the raw amounts.
	for _, summary := range summaries {
		assert.NotZero(t, summary.Subscription.AmountTotal)
	}
}
