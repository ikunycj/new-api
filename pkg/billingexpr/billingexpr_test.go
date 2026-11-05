package billingexpr_test

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Claude-style: fixed tiers, input > 200K changes both input & output price
// ---------------------------------------------------------------------------

const claudeExpr = `p <= 200000 ? tier("standard", p * 1.5 + c * 7.5) : tier("long_context", p * 3.0 + c * 11.25)`

func TestClaude_StandardTier(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(claudeExpr, billingexpr.TokenParams{P: 100000, C: 5000})
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "standard")
	}
}

func TestClaude_LongContextTier(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(claudeExpr, billingexpr.TokenParams{P: 300000, C: 10000})
	if err != nil {
		t.Fatal(err)
	}
	want := 300000*3.0 + 10000*11.25
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "long_context")
	}
}

func TestClaude_BoundaryExact(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(claudeExpr, billingexpr.TokenParams{P: 200000, C: 1000})
	if err != nil {
		t.Fatal(err)
	}
	want := 200000*1.5 + 1000*7.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "standard")
	}
}

// ---------------------------------------------------------------------------
// GLM-style: multi-condition tiers with both input and output dimensions
// ---------------------------------------------------------------------------

const glmExpr = `
(
	p < 32000 && c < 200 ? tier("tier1_short", (p)*2 + c*8) :
	p < 32000 && c >= 200 ? tier("tier2_long_output", (p)*3 + c*14) :
	tier("tier3_long_input", (p)*4 + c*16)
) / 1000000
`

func TestGLM_Tier1(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(glmExpr, billingexpr.TokenParams{P: 15000, C: 100})
	if err != nil {
		t.Fatal(err)
	}
	want := (15000.0*2 + 100.0*8) / 1000000
	if math.Abs(cost-want) > 1e-10 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "tier1_short" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "tier1_short")
	}
}

func TestGLM_Tier2(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(glmExpr, billingexpr.TokenParams{P: 15000, C: 500})
	if err != nil {
		t.Fatal(err)
	}
	want := (15000.0*3 + 500.0*14) / 1000000
	if math.Abs(cost-want) > 1e-10 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "tier2_long_output" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "tier2_long_output")
	}
}

func TestGLM_Tier3(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(glmExpr, billingexpr.TokenParams{P: 50000, C: 100})
	if err != nil {
		t.Fatal(err)
	}
	want := (50000.0*4 + 100.0*16) / 1000000
	if math.Abs(cost-want) > 1e-10 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "tier3_long_input" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "tier3_long_input")
	}
}

// ---------------------------------------------------------------------------
// Simple flat-rate (no tier() call)
// ---------------------------------------------------------------------------

func TestSimpleExpr_NoTier(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr("p * 0.5 + c * 1.0", billingexpr.TokenParams{P: 1000, C: 500})
	if err != nil {
		t.Fatal(err)
	}
	want := 1000*0.5 + 500*1.0
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "" {
		t.Errorf("tier should be empty, got %q", trace.MatchedTier)
	}
}

// ---------------------------------------------------------------------------
// Math helper functions
// ---------------------------------------------------------------------------

func TestMathHelpers(t *testing.T) {
	cost, _, err := billingexpr.RunExpr("max(p, c) * 0.5 + min(p, c) * 0.1", billingexpr.TokenParams{P: 300, C: 500})
	if err != nil {
		t.Fatal(err)
	}
	want := 500*0.5 + 300*0.1
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestRequestProbeHelpers(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`p * 0.5 + c * 1.0 * (param("service_tier") == "fast" ? 2 : 1)`,
		billingexpr.TokenParams{P: 1000, C: 500},
		billingexpr.RequestInput{
			Body: []byte(`{"service_tier":"fast"}`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 1000*0.5 + 500*1.0*2
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestHeaderProbeHelper(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`p * 0.5 + c * 1.0 * (has(header("anthropic-beta"), "fast-mode") ? 2 : 1)`,
		billingexpr.TokenParams{P: 1000, C: 500},
		billingexpr.RequestInput{
			Headers: map[string]string{
				"Anthropic-Beta": "fast-mode-2026-02-01",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 1000*0.5 + 500*1.0*2
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestParamProbeNestedBool(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`p * (param("stream_options.fast_mode") == true ? 1.5 : 1.0)`,
		billingexpr.TokenParams{P: 100},
		billingexpr.RequestInput{
			Body: []byte(`{"stream_options":{"fast_mode":true}}`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 150.0
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestParamProbeArrayLength(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`p * (param("messages.#") > 20 ? 1.2 : 1.0)`,
		billingexpr.TokenParams{P: 100},
		billingexpr.RequestInput{
			Body: []byte(`{"messages":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 120.0
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestRequestProbeMissingFieldReturnsNil(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`param("missing.value") == nil ? 2 : 1`,
		billingexpr.TokenParams{},
		billingexpr.RequestInput{
			Body: []byte(`{"service_tier":"standard"}`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if cost != 2 {
		t.Errorf("cost = %f, want 2", cost)
	}
}

func TestRequestProbeMultipleRulesMultiply(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`(param("service_tier") == "fast" ? 2 : 1) * (has(header("anthropic-beta"), "fast-mode-2026-02-01") ? 2.5 : 1)`,
		billingexpr.TokenParams{},
		billingexpr.RequestInput{
			Headers: map[string]string{
				"Anthropic-Beta": "fast-mode-2026-02-01",
			},
			Body: []byte(`{"service_tier":"fast"}`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(cost-5) > 1e-6 {
		t.Errorf("cost = %f, want 5", cost)
	}
}

func TestCeilFloor(t *testing.T) {
	cost, _, err := billingexpr.RunExpr("ceil(p / 1000) * 0.5", billingexpr.TokenParams{P: 1500})
	if err != nil {
		t.Fatal(err)
	}
	want := math.Ceil(1500.0/1000) * 0.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

// ---------------------------------------------------------------------------
// Zero tokens
// ---------------------------------------------------------------------------

func TestZeroTokens(t *testing.T) {
	cost, _, err := billingexpr.RunExpr(claudeExpr, billingexpr.TokenParams{})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 0 {
		t.Errorf("cost should be 0 for zero tokens, got %f", cost)
	}
}

// ---------------------------------------------------------------------------
// Rounding
// ---------------------------------------------------------------------------

func TestQuotaRound(t *testing.T) {
	tests := []struct {
		in   float64
		want int
	}{
		{0, 0},
		{0.4, 0},
		{0.5, 1},
		{0.6, 1},
		{1.5, 2},
		{-0.5, -1},
		{-0.6, -1},
		{999.4999, 999},
		{999.5, 1000},
		{1e9 + 0.5, 1e9 + 1},
		// Oversized expression results saturate at int32 (delegated to
		// common.QuotaRound); full saturation coverage lives in common.
		{3.6893488147419103e19, math.MaxInt32},
	}
	for _, tt := range tests {
		got := billingexpr.QuotaRound(tt.in)
		if got != tt.want {
			t.Errorf("QuotaRound(%f) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Settlement
// ---------------------------------------------------------------------------

func TestComputeTieredQuota_Basic(t *testing.T) {
	snap := &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                claudeExpr,
		ExprHash:                  billingexpr.ExprHashString(claudeExpr),
		GroupRatio:                1.0,
		EstimatedPromptTokens:     100000,
		EstimatedCompletionTokens: 5000,
		EstimatedQuotaBeforeGroup: (100000*1.5 + 5000*7.5) / 1_000_000 * 500_000,
		EstimatedQuotaAfterGroup:  billingexpr.QuotaRound((100000*1.5 + 5000*7.5) / 1_000_000 * 500_000),
		EstimatedTier:             "standard",
		QuotaPerUnit:              500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 300000, C: 10000})
	if err != nil {
		t.Fatal(err)
	}

	wantBefore := (300000*3.0 + 10000*11.25) / 1_000_000 * 500_000
	if math.Abs(result.ActualQuotaBeforeGroup-wantBefore) > 1e-6 {
		t.Errorf("before group: got %f, want %f", result.ActualQuotaBeforeGroup, wantBefore)
	}
	if result.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want %q", result.MatchedTier, "long_context")
	}
	if !result.CrossedTier {
		t.Error("expected crossed_tier=true (estimated standard, actual long_context)")
	}
}

func TestComputeTieredQuota_SameTier(t *testing.T) {
	snap := &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                claudeExpr,
		ExprHash:                  billingexpr.ExprHashString(claudeExpr),
		GroupRatio:                1.5,
		EstimatedPromptTokens:     50000,
		EstimatedCompletionTokens: 1000,
		EstimatedQuotaBeforeGroup: (50000*1.5 + 1000*7.5) / 1_000_000 * 500_000,
		EstimatedQuotaAfterGroup:  billingexpr.QuotaRound((50000*1.5 + 1000*7.5) / 1_000_000 * 500_000 * 1.5),
		EstimatedTier:             "standard",
		QuotaPerUnit:              500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 80000, C: 2000})
	if err != nil {
		t.Fatal(err)
	}

	wantBefore := (80000*1.5 + 2000*7.5) / 1_000_000 * 500_000
	wantAfter := billingexpr.QuotaRound(wantBefore * 1.5)
	if result.ActualQuotaAfterGroup != wantAfter {
		t.Errorf("after group: got %d, want %d", result.ActualQuotaAfterGroup, wantAfter)
	}
	if result.CrossedTier {
		t.Error("expected crossed_tier=false (both standard)")
	}
}

// ---------------------------------------------------------------------------
// Compile errors
// ---------------------------------------------------------------------------

func TestCompileError(t *testing.T) {
	_, _, err := billingexpr.RunExpr("invalid +-+ syntax", billingexpr.TokenParams{})
	if err == nil {
		t.Error("expected compile error")
	}
}

// ---------------------------------------------------------------------------
// Compile Cache
// ---------------------------------------------------------------------------

func TestCompileCache_SameResult(t *testing.T) {
	r1, _, err := billingexpr.RunExpr("p * 0.5", billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	r2, _, err := billingexpr.RunExpr("p * 0.5", billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Errorf("cached and uncached results differ: %f != %f", r1, r2)
	}
}

func TestInvalidateCache(t *testing.T) {
	billingexpr.InvalidateCache()
	r1, _, _ := billingexpr.RunExpr("p * 0.5", billingexpr.TokenParams{P: 100})
	billingexpr.InvalidateCache()
	r2, _, _ := billingexpr.RunExpr("p * 0.5", billingexpr.TokenParams{P: 100})
	if r1 != r2 {
		t.Errorf("post-invalidate results differ: %f != %f", r1, r2)
	}
}

// ---------------------------------------------------------------------------
// Hash
// ---------------------------------------------------------------------------

func TestExprHashString_Deterministic(t *testing.T) {
	h1 := billingexpr.ExprHashString("p * 0.5")
	h2 := billingexpr.ExprHashString("p * 0.5")
	if h1 != h2 {
		t.Error("hash should be deterministic")
	}
	h3 := billingexpr.ExprHashString("p * 0.6")
	if h1 == h3 {
		t.Error("different expressions should have different hashes")
	}
}

// ---------------------------------------------------------------------------
// Cache variables: present
// ---------------------------------------------------------------------------

const claudeWithCacheExpr = `p <= 200000 ? tier("standard", p * 1.5 + c * 7.5 + cr * 0.15 + cc * 1.875) : tier("long_context", p * 3.0 + c * 11.25 + cr * 0.3 + cc * 3.75)`

func TestCachePresent_StandardTier(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 5000, CR: 50000, CC: 10000}
	cost, trace, err := billingexpr.RunExpr(claudeWithCacheExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5 + 50000*0.15 + 10000*1.875
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "standard")
	}
}

func TestCachePresent_LongContextTier(t *testing.T) {
	params := billingexpr.TokenParams{P: 300000, C: 10000, CR: 100000, CC: 20000}
	cost, trace, err := billingexpr.RunExpr(claudeWithCacheExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 300000*3.0 + 10000*11.25 + 100000*0.3 + 20000*3.75
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "long_context")
	}
}

// ---------------------------------------------------------------------------
// Cache variables: absent (all zero) — same expression still works
// ---------------------------------------------------------------------------

func TestCacheAbsent_ZeroCacheTokens(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 5000}
	cost, trace, err := billingexpr.RunExpr(claudeWithCacheExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f (cache terms should be 0)", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "standard")
	}
}

// ---------------------------------------------------------------------------
// Mixed cache fields: cc and cc1h non-zero
// ---------------------------------------------------------------------------

const claudeCacheSplitExpr = `tier("default", p * 1.5 + c * 7.5 + cr * 0.15 + cc * 2.0 + cc1h * 3.0)`

func TestMixedCacheFields(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 5000, CR: 10000, CC: 5000, CC1h: 2000}
	cost, _, err := billingexpr.RunExpr(claudeCacheSplitExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5 + 10000*0.15 + 5000*2.0 + 2000*3.0
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestMixedCacheFields_AllCacheZero(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 5000}
	cost, _, err := billingexpr.RunExpr(claudeCacheSplitExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f (all cache zero)", cost, want)
	}
}

// ---------------------------------------------------------------------------
// Backward compatibility: p+c only expressions still work with TokenParams
// ---------------------------------------------------------------------------

func TestBackwardCompat_OldExprWithTokenParams(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 5000, CR: 99999, CC: 88888}
	cost, trace, err := billingexpr.RunExpr(claudeExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f (old expr ignores cache fields)", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "standard")
	}
}

// ---------------------------------------------------------------------------
// Settlement with cache tokens
// ---------------------------------------------------------------------------

func TestComputeTieredQuota_WithCache(t *testing.T) {
	snap := &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                claudeWithCacheExpr,
		ExprHash:                  billingexpr.ExprHashString(claudeWithCacheExpr),
		GroupRatio:                1.0,
		EstimatedPromptTokens:     100000,
		EstimatedCompletionTokens: 5000,
		EstimatedQuotaBeforeGroup: (100000*1.5 + 5000*7.5) / 1_000_000 * 500_000,
		EstimatedQuotaAfterGroup:  billingexpr.QuotaRound((100000*1.5 + 5000*7.5) / 1_000_000 * 500_000),
		EstimatedTier:             "standard",
		QuotaPerUnit:              500_000,
	}

	params := billingexpr.TokenParams{P: 100000, C: 5000, CR: 50000, CC: 10000}
	result, err := billingexpr.ComputeTieredQuota(snap, params)
	if err != nil {
		t.Fatal(err)
	}

	wantBefore := (100000*1.5 + 5000*7.5 + 50000*0.15 + 10000*1.875) / 1_000_000 * 500_000
	if math.Abs(result.ActualQuotaBeforeGroup-wantBefore) > 1e-6 {
		t.Errorf("before group: got %f, want %f", result.ActualQuotaBeforeGroup, wantBefore)
	}
	if result.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", result.MatchedTier, "standard")
	}
	if result.CrossedTier {
		t.Error("expected crossed_tier=false (same tier)")
	}
}

func TestComputeTieredQuota_WithCacheCrossTier(t *testing.T) {
	snap := &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                claudeWithCacheExpr,
		ExprHash:                  billingexpr.ExprHashString(claudeWithCacheExpr),
		GroupRatio:                2.0,
		EstimatedPromptTokens:     100000,
		EstimatedCompletionTokens: 5000,
		EstimatedQuotaBeforeGroup: (100000*1.5 + 5000*7.5) / 1_000_000 * 500_000,
		EstimatedQuotaAfterGroup:  billingexpr.QuotaRound((100000*1.5 + 5000*7.5) / 1_000_000 * 500_000 * 2.0),
		EstimatedTier:             "standard",
		QuotaPerUnit:              500_000,
	}

	params := billingexpr.TokenParams{P: 300000, C: 10000, CR: 50000, CC: 10000}
	result, err := billingexpr.ComputeTieredQuota(snap, params)
	if err != nil {
		t.Fatal(err)
	}

	wantBefore := (300000*3.0 + 10000*11.25 + 50000*0.3 + 10000*3.75) / 1_000_000 * 500_000
	wantAfter := billingexpr.QuotaRound(wantBefore * 2.0)
	if math.Abs(result.ActualQuotaBeforeGroup-wantBefore) > 1e-6 {
		t.Errorf("before group: got %f, want %f", result.ActualQuotaBeforeGroup, wantBefore)
	}
	if result.ActualQuotaAfterGroup != wantAfter {
		t.Errorf("after group: got %d, want %d", result.ActualQuotaAfterGroup, wantAfter)
	}
	if !result.CrossedTier {
		t.Error("expected crossed_tier=true (estimated standard, actual long_context)")
	}
}

// ---------------------------------------------------------------------------
// Settlement-level tests for ComputeTieredQuota
// ---------------------------------------------------------------------------

func TestComputeTieredQuota_BasicSettlement(t *testing.T) {
	exprStr := `tier("default", p + c)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 3000, C: 2000})
	if err != nil {
		t.Fatal(err)
	}
	// exprOutput = 5000; quota = 5000 / 1M * 500K = 2500
	if math.Abs(result.ActualQuotaBeforeGroup-2500) > 1e-6 {
		t.Errorf("before group = %f, want 2500", result.ActualQuotaBeforeGroup)
	}
	if result.ActualQuotaAfterGroup != 2500 {
		t.Errorf("after group = %d, want 2500", result.ActualQuotaAfterGroup)
	}
	if result.MatchedTier != "default" {
		t.Errorf("tier = %q, want default", result.MatchedTier)
	}
}

func TestComputeTieredQuota_WithGroupRatio(t *testing.T) {
	exprStr := `tier("default", p + c)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   2.0,
		QuotaPerUnit: 500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 1000, C: 500})
	if err != nil {
		t.Fatal(err)
	}
	// exprOutput = 1500; quotaBeforeGroup = 750; afterGroup = round(750 * 2.0) = 1500
	if result.ActualQuotaAfterGroup != 1500 {
		t.Errorf("after group = %d, want 1500", result.ActualQuotaAfterGroup)
	}
}

func TestComputeTieredQuota_ZeroTokens(t *testing.T) {
	exprStr := `tier("default", p * 2 + c * 10)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ActualQuotaAfterGroup != 0 {
		t.Errorf("after group = %d, want 0", result.ActualQuotaAfterGroup)
	}
}

func TestComputeTieredQuota_RoundingEdge(t *testing.T) {
	exprStr := `tier("default", p * 0.5)` // 3 * 0.5 = 1.5 (expr); 1.5 / 1M * 500K = 0.75; round(0.75) = 1
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 3})
	if err != nil {
		t.Fatal(err)
	}
	// 3 * 0.5 = 1.5 (expr); quota = 1.5 / 1M * 500K = 0.75; round(0.75) = 1
	if result.ActualQuotaAfterGroup != 1 {
		t.Errorf("after group = %d, want 1 (round 0.75 up)", result.ActualQuotaAfterGroup)
	}
}

func TestComputeTieredQuota_RoundingEdgeDown(t *testing.T) {
	exprStr := `tier("default", p * 0.4)` // 3 * 0.4 = 1.2 (expr); 1.2 / 1M * 500K = 0.6; round(0.6) = 1
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 3})
	if err != nil {
		t.Fatal(err)
	}
	// 3 * 0.4 = 1.2 (expr); quota = 1.2 / 1M * 500K = 0.6; round(0.6) = 1
	if result.ActualQuotaAfterGroup != 1 {
		t.Errorf("after group = %d, want 1 (round 0.6 up)", result.ActualQuotaAfterGroup)
	}
}

func TestComputeTieredQuotaWithRequest_ProbeAffectsQuota(t *testing.T) {
	exprStr := `param("fast") == true ? tier("fast", p * 4) : tier("normal", p * 2)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:   "tiered_expr",
		ExprString:    exprStr,
		ExprHash:      billingexpr.ExprHashString(exprStr),
		GroupRatio:    1.0,
		EstimatedTier: "normal",
		QuotaPerUnit:  500_000,
	}

	// Without request: normal tier
	r1, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 1000})
	if err != nil {
		t.Fatal(err)
	}
	// normal: p*2 = 2000; quota = 2000 / 1M * 500K = 1000
	if r1.ActualQuotaAfterGroup != 1000 {
		t.Errorf("normal = %d, want 1000", r1.ActualQuotaAfterGroup)
	}

	// With request: fast tier
	r2, err := billingexpr.ComputeTieredQuotaWithRequest(snap, billingexpr.TokenParams{P: 1000}, billingexpr.RequestInput{
		Body: []byte(`{"fast":true}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	// fast: p*4 = 4000; quota = 4000 / 1M * 500K = 2000
	if r2.ActualQuotaAfterGroup != 2000 {
		t.Errorf("fast = %d, want 2000", r2.ActualQuotaAfterGroup)
	}
	if !r2.CrossedTier {
		t.Error("expected CrossedTier = true when probe changes tier")
	}
}

func TestComputeTieredQuota_BoundaryTierCrossing(t *testing.T) {
	exprStr := `p <= 100000 ? tier("small", p * 1) : tier("large", p * 2)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:   "tiered_expr",
		ExprString:    exprStr,
		ExprHash:      billingexpr.ExprHashString(exprStr),
		GroupRatio:    1.0,
		EstimatedTier: "small",
		QuotaPerUnit:  500_000,
	}

	// At boundary: small, p*1 = 100000; quota = 100000 / 1M * 500K = 50000
	r1, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 100000})
	if err != nil {
		t.Fatal(err)
	}
	if r1.MatchedTier != "small" {
		t.Errorf("at boundary: tier = %s, want small", r1.MatchedTier)
	}
	if r1.ActualQuotaAfterGroup != 50000 {
		t.Errorf("at boundary: quota = %d, want 50000", r1.ActualQuotaAfterGroup)
	}

	// Past boundary: large, p*2 = 200002; quota = 200002 / 1M * 500K = 100001
	r2, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 100001})
	if err != nil {
		t.Fatal(err)
	}
	if r2.MatchedTier != "large" {
		t.Errorf("past boundary: tier = %s, want large", r2.MatchedTier)
	}
	if r2.ActualQuotaAfterGroup != 100001 {
		t.Errorf("past boundary: quota = %d, want 100001", r2.ActualQuotaAfterGroup)
	}
	if !r2.CrossedTier {
		t.Error("expected CrossedTier = true")
	}
}

// ---------------------------------------------------------------------------
// Time function tests
// ---------------------------------------------------------------------------

func TestTimeFunctions_ValidTimezone(t *testing.T) {
	exprStr := `tier("default", p) * (hour("UTC") >= 0 ? 1 : 1)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 100 {
		t.Errorf("cost = %f, want 100", cost)
	}
}

func TestTimeFunctions_AllFunctionsCompile(t *testing.T) {
	exprStr := `tier("default", p) * (hour("Asia/Shanghai") >= 0 ? 1 : 1) * (minute("UTC") >= 0 ? 1 : 1) * (weekday("UTC") >= 0 ? 1 : 1) * (month("UTC") >= 1 ? 1 : 1) * (day("UTC") >= 1 ? 1 : 1)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 500})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 500 {
		t.Errorf("cost = %f, want 500", cost)
	}
}

func TestTimeFunctions_InvalidTimezone(t *testing.T) {
	exprStr := `tier("default", p) * (hour("Invalid/Zone") >= 0 ? 1 : 2)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	// Invalid timezone falls back to UTC; hour is 0-23, so condition is always true
	if cost != 100 {
		t.Errorf("cost = %f, want 100 (fallback to UTC)", cost)
	}
}

func TestTimeFunctions_EmptyTimezone(t *testing.T) {
	exprStr := `tier("default", p) * (hour("") >= 0 ? 1 : 2)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 100 {
		t.Errorf("cost = %f, want 100 (empty tz -> UTC)", cost)
	}
}

func TestTimeFunctions_NightDiscountPattern(t *testing.T) {
	exprStr := `tier("default", p * 2 + c * 10) * (hour("UTC") >= 21 || hour("UTC") < 6 ? 0.5 : 1)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 1000, C: 500})
	if err != nil {
		t.Fatal(err)
	}
	// Base = 1000*2 + 500*10 = 7000; multiplier is either 0.5 or 1 depending on current UTC hour
	if cost != 7000 && cost != 3500 {
		t.Errorf("cost = %f, want 7000 or 3500", cost)
	}
}

func TestTimeFunctions_WeekdayRange(t *testing.T) {
	exprStr := `tier("default", p) * (weekday("UTC") >= 0 && weekday("UTC") <= 6 ? 1 : 999)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	// weekday is always 0-6, so multiplier is always 1
	if cost != 100 {
		t.Errorf("cost = %f, want 100", cost)
	}
}

func TestTimeFunctions_MonthDayPattern(t *testing.T) {
	exprStr := `tier("default", p) * (month("Asia/Shanghai") == 1 && day("Asia/Shanghai") == 1 ? 0.5 : 1)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 1000})
	if err != nil {
		t.Fatal(err)
	}
	// Either 1000 (not Jan 1) or 500 (Jan 1) — both are valid
	if cost != 1000 && cost != 500 {
		t.Errorf("cost = %f, want 1000 or 500", cost)
	}
}

// ---------------------------------------------------------------------------
// Image and audio token tests
// ---------------------------------------------------------------------------

func TestImageTokenVariable(t *testing.T) {
	exprStr := `tier("base", p * 2 + c * 10 + img * 5)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 1000, C: 500, Img: 200})
	if err != nil {
		t.Fatal(err)
	}
	// 1000*2 + 500*10 + 200*5 = 2000 + 5000 + 1000 = 8000
	if math.Abs(cost-8000) > 1e-6 {
		t.Errorf("cost = %f, want 8000", cost)
	}
}

func TestAudioTokenVariables(t *testing.T) {
	exprStr := `tier("base", p * 2 + c * 10 + ai * 50 + ao * 100)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 1000, C: 500, AI: 100, AO: 50})
	if err != nil {
		t.Fatal(err)
	}
	// 1000*2 + 500*10 + 100*50 + 50*100 = 2000 + 5000 + 5000 + 5000 = 17000
	if math.Abs(cost-17000) > 1e-6 {
		t.Errorf("cost = %f, want 17000", cost)
	}
}

func TestImageAudioVariables(t *testing.T) {
	exprStr := `tier("base", p * 1 + img * 3 + ai * 5 + ao * 10)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 100, Img: 50, AI: 20, AO: 10})
	if err != nil {
		t.Fatal(err)
	}
	// 100*1 + 50*3 + 20*5 + 10*10 = 100 + 150 + 100 + 100 = 450
	if math.Abs(cost-450) > 1e-6 {
		t.Errorf("cost = %f, want 450", cost)
	}
}

func TestImageAudioZero(t *testing.T) {
	exprStr := `tier("base", p * 2 + img * 5 + ai * 50 + ao * 100)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 1000})
	if err != nil {
		t.Fatal(err)
	}
	// img, ai, ao default to 0
	if math.Abs(cost-2000) > 1e-6 {
		t.Errorf("cost = %f, want 2000", cost)
	}
}

// ---------------------------------------------------------------------------
// len variable tests — tier conditions based on context length
// ---------------------------------------------------------------------------

const lenTieredExpr = `len <= 200000 ? tier("standard", p * 3 + c * 15 + cr * 0.3) : tier("long_context", p * 6 + c * 22.5 + cr * 0.6)`

func TestLen_StandardTier(t *testing.T) {
	params := billingexpr.TokenParams{P: 80000, C: 5000, Len: 100000, CR: 20000}
	cost, trace, err := billingexpr.RunExpr(lenTieredExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 80000*3 + 5000*15 + 20000*0.3
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want standard", trace.MatchedTier)
	}
}

func TestLen_LongContextTier(t *testing.T) {
	// p is low (cache subtracted), but len is high (full context)
	params := billingexpr.TokenParams{P: 50000, C: 5000, Len: 300000, CR: 250000}
	cost, trace, err := billingexpr.RunExpr(lenTieredExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 50000*6 + 5000*22.5 + 250000*0.6
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want long_context (len=300000 > 200000)", trace.MatchedTier)
	}
}

func TestLen_BoundaryExact(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 1000, Len: 200000, CR: 100000}
	_, trace, err := billingexpr.RunExpr(lenTieredExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want standard (len=200000 <= 200000)", trace.MatchedTier)
	}
}

func TestLen_BoundaryPlusOne(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 1000, Len: 200001, CR: 100001}
	_, trace, err := billingexpr.RunExpr(lenTieredExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	if trace.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want long_context (len=200001 > 200000)", trace.MatchedTier)
	}
}

func TestLen_ZeroDefaultsToZero(t *testing.T) {
	// len defaults to 0 when not set
	params := billingexpr.TokenParams{P: 1000, C: 500}
	_, trace, err := billingexpr.RunExpr(lenTieredExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want standard (len=0 <= 200000)", trace.MatchedTier)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks: compile vs cached execution
// ---------------------------------------------------------------------------

const benchComplexExpr = `len <= 200000 ? tier("standard", p * 3 + c * 15 + cr * 0.3 + cc * 3.75 + cc1h * 6 + img * 3 + img_o * 30 + ai * 10 + ao * 40) : tier("long_context", p * 6 + c * 22.5 + cr * 0.6 + cc * 7.5 + cc1h * 12 + img * 6 + img_o * 60 + ai * 20 + ao * 80)`

func BenchmarkExprCompile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		billingexpr.InvalidateCache()
		billingexpr.CompileFromCache(benchComplexExpr)
	}
}

func BenchmarkExprRunCached(b *testing.B) {
	billingexpr.CompileFromCache(benchComplexExpr)
	params := billingexpr.TokenParams{P: 150000, C: 10000, Len: 188000, CR: 30000, CC: 5000, Img: 2000, AI: 1000, AO: 500}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		billingexpr.RunExpr(benchComplexExpr, params)
	}
}

func TestGPT56Terra12HourCachePricing(t *testing.T) {
	const expression = `len <= 272000 ? tier("0_272k", p * 2 + cr * 0.2 + cc * 2.5 + cc12h * 5 + c * 12) : tier("272k_plus", p * 4 + cr * 0.4 + cc * 5 + cc12h * 10 + c * 18)`
	for _, tc := range []struct {
		name    string
		prompt  float64
		length  float64
		cost    float64
		tier    string
		quota   int
		crossed bool
	}{
		{"below boundary", 199999, 271999, 473998, "0_272k", 947996, false},
		{"at boundary", 200000, 272000, 474000, "0_272k", 948000, false},
		{"above boundary", 200001, 272001, 942004, "272k_plus", 1884008, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			params := billingexpr.TokenParams{P: tc.prompt, C: 1000, Len: tc.length, CR: 60000, CC: 4000, CC12h: 8000}
			cost, trace, err := billingexpr.RunExpr(expression, params)
			require.NoError(t, err)
			assert.Equal(t, tc.cost, cost)
			assert.Equal(t, tc.cost, trace.Cost)
			assert.Equal(t, tc.tier, trace.MatchedTier)
			assert.Equal(t, billingexpr.BillingUnitToken, trace.BillingUnit)
			assert.Nil(t, trace.FixedPrice)
			assert.True(t, billingexpr.UsedVars(expression)["cc12h"])

			snapshot := &billingexpr.BillingSnapshot{
				ExprString: expression, ExprHash: billingexpr.ExprHashString(expression),
				QuotaPerUnit: 500000, BillingUSDToCNYRate: 8, GroupRatio: 0.5, EstimatedTier: "0_272k",
			}
			result, err := billingexpr.ComputeTieredQuota(snapshot, params)
			require.NoError(t, err)
			assert.Equal(t, tc.cost*4, result.ActualQuotaBeforeGroup)
			assert.Equal(t, tc.quota, result.ActualQuotaAfterGroup)
			assert.Equal(t, tc.crossed, result.CrossedTier)
			assert.Nil(t, result.Clamp)
		})
	}
}

func TestFixedImagePriceQuantityAndCNYSettlement(t *testing.T) {
	const expression = `tier("image", fixed(0.0035)) * image_count`
	for _, tc := range []struct {
		name   string
		count  *int
		images int
		cost   float64
		before float64
		quota  int
	}{
		{"default one", nil, 1, 3500, 14000, 7000},
		{"explicit one", common.GetPointer(1), 1, 3500, 14000, 7000},
		{"three images", common.GetPointer(3), 3, 10500, 42000, 21000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := billingexpr.RequestInput{ImageCount: tc.count}
			cost, trace, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{P: 99999}, request)
			require.NoError(t, err)
			assert.Equal(t, tc.cost, cost)
			assert.Equal(t, cost, trace.Cost, "trace includes the image multiplier outside tier")
			assert.Equal(t, billingexpr.BillingUnitRequest, trace.BillingUnit)
			require.NotNil(t, trace.FixedPrice)
			assert.Equal(t, 0.0035, *trace.FixedPrice)
			require.NotNil(t, trace.ImageCount)
			assert.Equal(t, tc.images, *trace.ImageCount)

			snapshot := &billingexpr.BillingSnapshot{
				ExprString: expression, ExprHash: billingexpr.ExprHashString(expression),
				QuotaPerUnit: 500000, BillingUSDToCNYRate: 8, GroupRatio: 0.5, EstimatedTier: "image",
				EstimatedImageCount: common.GetPointer(1), EstimatedBillingUnit: billingexpr.BillingUnitRequest,
				EstimatedFixedPrice: common.GetPointer(0.0035),
			}
			result, err := billingexpr.ComputeTieredQuotaWithRequest(snapshot, billingexpr.TokenParams{}, request)
			require.NoError(t, err)
			assert.Equal(t, tc.before, result.ActualQuotaBeforeGroup)
			assert.Equal(t, tc.quota, result.ActualQuotaAfterGroup)
			assert.Equal(t, trace.BillingUnit, result.BillingUnit)
			assert.Equal(t, trace.FixedPrice, result.FixedPrice)
			assert.Equal(t, trace.ImageCount, result.ImageCount)
			assert.Equal(t, "image", result.MatchedTier)
			assert.False(t, result.CrossedTier)
			assert.Nil(t, result.Clamp)
			assert.Equal(t, 1, *snapshot.EstimatedImageCount, "actual count must not overwrite the frozen estimate")
		})
	}
}

func TestFixedAndTokenBranchBillingMetadata(t *testing.T) {
	const expression = `(len <= 32000 ? tier("request", fixed(0.0035)) : tier("tokens", p * 2 + c * 8)) * image_count * (param("fast") == true ? 2 : 1)`
	for _, tc := range []struct {
		name   string
		params billingexpr.TokenParams
		cost   float64
		tier   string
		unit   billingexpr.BillingUnit
		fixed  *float64
	}{
		{"zero usage still charges", billingexpr.TokenParams{}, 21000, "request", billingexpr.BillingUnitRequest, common.GetPointer(0.0035)},
		{"fixed ignores tokens", billingexpr.TokenParams{P: 32000, C: 9000, Len: 32000}, 21000, "request", billingexpr.BillingUnitRequest, common.GetPointer(0.0035)},
		{"token branch uses tokens", billingexpr.TokenParams{P: 32001, C: 100, Len: 32001}, 388812, "tokens", billingexpr.BillingUnitToken, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := billingexpr.RequestInput{Body: []byte(`{"fast":true}`), ImageCount: common.GetPointer(3)}
			cost, trace, err := billingexpr.RunExprWithRequest(expression, tc.params, request)
			require.NoError(t, err)
			assert.Equal(t, tc.cost, cost)
			assert.Equal(t, cost, trace.Cost, "trace includes all request multipliers")
			assert.Equal(t, tc.tier, trace.MatchedTier)
			assert.Equal(t, tc.unit, trace.BillingUnit)
			assert.Equal(t, tc.fixed, trace.FixedPrice)

			snapshot := &billingexpr.BillingSnapshot{
				ExprString: expression, ExprHash: billingexpr.ExprHashString(expression),
				QuotaPerUnit: 1000000, GroupRatio: 1, EstimatedTier: "request",
			}
			result, err := billingexpr.ComputeTieredQuotaWithRequest(snapshot, tc.params, request)
			require.NoError(t, err)
			assert.Equal(t, tc.unit, result.BillingUnit)
			assert.Equal(t, tc.fixed, result.FixedPrice)
			assert.Equal(t, tc.cost, result.ActualQuotaBeforeGroup)
			assert.Equal(t, tc.tier != "request", result.CrossedTier)
		})
	}
}

func TestFixedPriceAllowsRequestMultipliers(t *testing.T) {
	for _, tc := range []struct {
		name       string
		expression string
		cost       float64
	}{
		{"header and body", `(param("fast") == true ? 2 : 1) * (tier("request", fixed(0.0035)) * (has(header("x-mode"), "fast") ? 2.5 : 1))`, 17500},
		{"unmatched rule", `tier("request", fixed(0.0035)) * (param("missing") == true ? 2 : 1)`, 3500},
		{"zero multiplier", `tier("request", fixed(0.0035)) * (param("fast") == true ? 0 : 1)`, 0},
		{"time conditions", `tier("request", fixed(0.0035)) * (hour("UTC") >= 0 ? 2 : 1) * (minute("UTC") >= 0 ? 2 : 1) * (weekday("UTC") >= 0 ? 2 : 1) * (month("UTC") >= 1 ? 2 : 1) * (day("UTC") >= 1 ? 2 : 1)`, 112000},
		{"quantity on left", `image_count * tier("request", fixed(0.0035))`, 10500},
		{"nested pricing branches", `param("fast") == true ? (len <= 1 ? tier("short", fixed(0.0035)) : tier("long", fixed(0.007))) : tier("tokens", p)`, 3500},
		{"integer literal", `tier("request", fixed(1))`, 1000000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cost, trace, err := billingexpr.RunExprWithRequest(tc.expression, billingexpr.TokenParams{}, billingexpr.RequestInput{
				Body: []byte(`{"fast":true}`), Headers: map[string]string{"X-Mode": "fast"}, ImageCount: common.GetPointer(3),
			})
			require.NoError(t, err)
			assert.Equal(t, tc.cost, cost)
			assert.Equal(t, cost, trace.Cost)
			assert.Equal(t, billingexpr.BillingUnitRequest, trace.BillingUnit)
		})
	}
}

func TestFixedPriceRejectsInvalidOriginalAST(t *testing.T) {
	for _, expression := range []string{
		`true ? tier("ok", p) : tier("bad", fixed(-0.01))`,
		`false ? tier("bad", fixed(p)) : tier("ok", p)`,
		`tier("bad", fixed(0.01 + 0.02))`,
		`tier("bad", fixed(param("price")))`,
		`tier("bad", fixed(1e308))`,
		`tier("bad", fixed(1e309))`,
		`tier("bad", fixed(0.01) + p * 2)`,
		`tier("bad", p * fixed(0.01))`,
		`tier("a", fixed(0.01)) + tier("b", p * 2)`,
		`tier("a", fixed(0.01)) * tier("b", fixed(0.02))`,
		`tier("outer", tier("inner", fixed(0.01)))`,
		`fixed(0.01)`,
		`tier("bad", fixed(0.01)) * p`,
		`tier("bad", fixed(0.01)) * 2`,
		`tier("bad", fixed(0.01)) * (param("fast") == true ? -2 : 1)`,
		`tier("bad", fixed(0.01)) * (param("fast") == true ? 2 : 0)`,
		`tier("bad", fixed(0.01)) * (param("fast") == true ? p : 1)`,
		`tier("bad", fixed(0.01)) * (len > 1 ? 2 : 1)`,
		`tier("bad", fixed(0.01)) * (param("fast") == true ? 1e309 : 1)`,
		`tier("condition", p) > 0 ? tier("a", fixed(0.01)) : tier("b", p)`,
		`fixed(0.01) > 0 ? tier("a", p) : tier("b", p)`,
		`tier("bad", fixed(0.01)) * (param("fast") == true && tier("condition", p) > 0 ? 2 : 1)`,
		`tier("bad", param("fast") == true ? fixed(0.01) : p)`,
		`tier("bad", fixed())`,
		`tier("bad", fixed(0.01, 0.02))`,
	} {
		t.Run(expression, func(t *testing.T) {
			_, err := billingexpr.CompileFromCache(expression)
			require.Error(t, err, "invalid leaves must be rejected before optimizer removes unreachable branches")
		})
	}
}

func TestImageQuantityValidationUsesOriginalAST(t *testing.T) {
	for _, expression := range []string{
		`tier("image", fixed(0.0035)) * image_count`,
		`tier("tokens", p * 2) * image_count`,
		`true ? tier("tokens", p) : tier("image", fixed(0.0035)) * image_count`,
	} {
		t.Run(expression, func(t *testing.T) {
			hash := billingexpr.ExprHashString(expression)
			_, err := billingexpr.CompileFromCacheByHash(expression, hash)
			require.NoError(t, err)
			assert.True(t, billingexpr.UsedVarsByHash(expression, hash)["image_count"])
			for _, count := range []int{0, -1, dto.MaxImageN + 1} {
				request := billingexpr.RequestInput{ImageCount: &count}
				cost, trace, err := billingexpr.RunExprByHashWithRequest(expression, hash, billingexpr.TokenParams{P: 100}, request)
				require.ErrorContains(t, err, "image_count")
				assert.Zero(t, cost, "invalid count must not participate in billing")
				assert.Empty(t, trace.MatchedTier, "validation precedes execution")
				assert.Nil(t, trace.FixedPrice)

				_, err = billingexpr.ComputeTieredQuotaWithRequest(&billingexpr.BillingSnapshot{
					ExprString: expression, ExprHash: hash, QuotaPerUnit: 500000, GroupRatio: 1,
				}, billingexpr.TokenParams{P: 100}, request)
				require.ErrorContains(t, err, "image_count")
			}
		})
	}
	count := dto.MaxImageN
	cost, trace, err := billingexpr.RunExprWithRequest(`tier("image", fixed(0.0035)) * image_count`, billingexpr.TokenParams{}, billingexpr.RequestInput{ImageCount: &count})
	require.NoError(t, err)
	assert.Equal(t, float64(dto.MaxImageN)*3500, cost)
	require.NotNil(t, trace.ImageCount)
	assert.Equal(t, dto.MaxImageN, *trace.ImageCount)
}

func TestOriginalFixedBranchesRemainVisibleThroughCache(t *testing.T) {
	const expression = `v1:true ? tier("tokens", p) : tier("request", fixed(0.0035)) * image_count`
	hash := billingexpr.ExprHashString(expression)
	for _, invalidate := range []bool{false, true} {
		if invalidate {
			billingexpr.InvalidateCache()
		}
		assert.True(t, billingexpr.UsesFixedPricing(expression))
		assert.True(t, billingexpr.UsesFixedPricingByHash(expression, hash))
		assert.True(t, billingexpr.UsedVarsByHash(expression, hash)["image_count"])
		cost, trace, err := billingexpr.RunExprByHash(expression, hash, billingexpr.TokenParams{P: 10})
		require.NoError(t, err)
		assert.Equal(t, float64(10), cost)
		assert.Equal(t, billingexpr.BillingUnitToken, trace.BillingUnit)
		assert.Nil(t, trace.FixedPrice, "unselected fixed branch must not change the actual unit")
		require.NotNil(t, trace.ImageCount)
		assert.Equal(t, 1, *trace.ImageCount)
	}
	assert.False(t, billingexpr.UsesFixedPricing(`tier("tokens", p)`))
}

func TestCacheUnawarePricingIgnoresNewOptionalInputs(t *testing.T) {
	const expression = `tier("tokens", p * 2 + c * 8)`
	cost, trace, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{
		P: 100, C: 10, CR: 1000, CC: 2000, CC1h: 3000, CC12h: 4000,
	}, billingexpr.RequestInput{ImageCount: common.GetPointer(-1)})
	require.NoError(t, err, "unused image_count must not validate unrelated request data")
	assert.Equal(t, float64(280), cost)
	assert.Equal(t, cost, trace.Cost)
	assert.Equal(t, billingexpr.BillingUnitToken, trace.BillingUnit)
	assert.Nil(t, trace.ImageCount)
	assert.Nil(t, trace.FixedPrice)
	assert.False(t, billingexpr.UsedVars(expression)["cc12h"])
}

func TestZeroFixedPriceJSONContract(t *testing.T) {
	const expression = `tier("free", fixed(0)) * image_count`
	cost, trace, err := billingexpr.RunExprWithRequest(expression, billingexpr.TokenParams{P: 1000}, billingexpr.RequestInput{ImageCount: common.GetPointer(3)})
	require.NoError(t, err)
	assert.Zero(t, cost)
	require.NotNil(t, trace.FixedPrice)
	assert.Zero(t, *trace.FixedPrice)
	data, err := common.Marshal(trace)
	require.NoError(t, err)
	var traceJSON map[string]interface{}
	require.NoError(t, common.Unmarshal(data, &traceJSON))
	assert.Equal(t, "request", traceJSON["billing_unit"])
	assert.Equal(t, float64(0), traceJSON["fixed_price"], "explicit zero must not be omitted")
	assert.Equal(t, float64(3), traceJSON["image_count"])

	snapshot := &billingexpr.BillingSnapshot{
		ExprString: expression, ExprHash: billingexpr.ExprHashString(expression),
		QuotaPerUnit: 500000, GroupRatio: 1, BillingUSDToCNYRate: 8, EstimatedTier: "free",
		EstimatedImageCount: trace.ImageCount, EstimatedBillingUnit: trace.BillingUnit, EstimatedFixedPrice: trace.FixedPrice,
	}
	data, err = common.Marshal(snapshot)
	require.NoError(t, err)
	var snapshotJSON map[string]interface{}
	require.NoError(t, common.Unmarshal(data, &snapshotJSON))
	assert.Equal(t, "request", snapshotJSON["estimated_billing_unit"])
	assert.Equal(t, float64(0), snapshotJSON["estimated_fixed_price"])
	assert.Equal(t, float64(3), snapshotJSON["estimated_image_count"])
	assert.Equal(t, float64(8), snapshotJSON["billing_usd_to_cny_rate"])
	var restored billingexpr.BillingSnapshot
	require.NoError(t, common.Unmarshal(data, &restored))
	assert.Equal(t, *snapshot, restored)

	result, err := billingexpr.ComputeTieredQuotaWithRequest(&restored, billingexpr.TokenParams{}, billingexpr.RequestInput{ImageCount: common.GetPointer(3)})
	require.NoError(t, err)
	assert.Zero(t, result.ActualQuotaAfterGroup)
	assert.Equal(t, billingexpr.BillingUnitRequest, result.BillingUnit)
	require.NotNil(t, result.FixedPrice)
	assert.Zero(t, *result.FixedPrice)
	assert.Nil(t, result.Clamp)
	data, err = common.Marshal(result)
	require.NoError(t, err)
	var resultJSON map[string]interface{}
	require.NoError(t, common.Unmarshal(data, &resultJSON))
	assert.Equal(t, "request", resultJSON["billing_unit"])
	assert.Equal(t, float64(0), resultJSON["fixed_price"])
	assert.Equal(t, float64(3), resultJSON["image_count"])
}

func TestFixedPriceSaturationRemainsAuditable(t *testing.T) {
	const expression = `tier("huge", fixed(1e20)) * image_count`
	result, err := billingexpr.ComputeTieredQuotaWithRequest(&billingexpr.BillingSnapshot{
		ExprString: expression, ExprHash: billingexpr.ExprHashString(expression),
		QuotaPerUnit: 500000, GroupRatio: 1, BillingUSDToCNYRate: 8,
	}, billingexpr.TokenParams{}, billingexpr.RequestInput{ImageCount: common.GetPointer(3)})
	require.NoError(t, err)
	assert.Equal(t, math.MaxInt32, result.ActualQuotaAfterGroup, "oversized fixed prices must never wrap into credits")
	require.NotNil(t, result.Clamp)
	assert.Equal(t, common.QuotaClampOverflow, result.Clamp.Kind)
	assert.Equal(t, billingexpr.BillingUnitRequest, result.BillingUnit)
}
