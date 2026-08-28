---
name: upstream-pricing-expression
description: Convert an AI provider's official model pricing into a verified new-api v1 billing expression, including cache prices, tiers, and time-window rules. Use when a pricing URL or source table must become an executable `billing_expr`.
---

# Upstream pricing to billing expression

Turn an authoritative provider price table into an expression that the current
`new-api` billing engine can execute. This skill produces a proposed
configuration artifact; it does not write database options or deploy anything
unless the user separately asks for that change.

## Read the contract first

In this repository, read `pkg/billingexpr/expr.md` before designing an
expression. Read [references/expression-contract.md](references/expression-contract.md)
for the compact rules and [references/source-evidence.md](references/source-evidence.md)
for source and output requirements. Treat the example in
[references/deepseek-example.md](references/deepseek-example.md) as a worked
example only; provider prices and model names must be fetched again.

## Workflow

1. **Identify the exact target.** Record the provider, exact model ID (not an
   alias), source URL, retrieval time, source currency, price unit, and the
   requested billing timezone. If the page contains several model variants,
   produce one expression per variant or ask which one is intended.

2. **Collect evidence before calculating.** Fetch the official provider page
   with the available browser or HTTP tooling, or use source text supplied by
   the user. If live access is unavailable, say so and do not present a dated
   example as current. Quote the model's input/output and
   cache rows and every footnote that changes their meaning. For a dynamic or
   JavaScript-rendered page, inspect the rendered table or its data response;
   never infer a value from a search snippet or a neighboring model column.
   Record the retrieval date because provider prices can change.

3. **Normalize the price dimensions.** Map source dimensions explicitly:
   normal input -> `p`, output -> `c`, cache read/hit -> `cr`, cache creation ->
   `cc` or `cc1h`, image/audio input -> `img`/`ai`, and image/audio output ->
   `img_o`/`ao`. Use `len` only for context-length conditions. If the
   expression references a sub-category, the engine removes that category
   from `p`/`c` for OpenAI-format usage; do not subtract it a second time.
   Verify that the selected channel adapter actually populates each referenced
   usage field (for example, DeepSeek cache-hit metadata is normalized into
   `PromptTokensDetails.CachedTokens`). If the response field is unsupported,
   the expression cannot recover it; mark the mapping unresolved rather than
   silently charging all tokens at the uncached rate.

4. **Resolve currency and units.** The expression contract is actual
   **USD per 1M tokens**. For a CNY source, obtain the effective project
   `BillingUSDToCNYRate` from an explicitly supplied value or a current,
   verified project status, then use
   `usd_per_million = cny_per_million / B`. For any other non-USD currency,
   require an explicit, attributable source-to-USD conversion policy; the CNY
   billing setting is not a generic FX converter. Convert coefficients before
   generating the expression and do not put the exchange rate into it. Never
   assume `B = 7.3`, use the repository's historical `USD2RMB`, or treat CNY
   as USD. If the required rate is unavailable, return `needs_confirmation`
   and a symbolic conversion table instead of claiming an executable exact
   expression.

5. **Translate conditions exactly.** Use half-open intervals `[start,end)`.
   For a provider schedule, state the timezone, weekday numbering (the engine
   uses Sunday `0` through Saturday `6`), inclusive/exclusive boundaries, and
   what happens outside the listed windows. Use `minute(tz)` when a boundary
   is not on an hour. For a common workday schedule, express weekdays and
   each non-overlapping time window explicitly. Validate the timezone as an
   IANA name; do not rely on the runtime's silent invalid-timezone fallback to
   UTC.

6. **Choose the expression shape.** Every priced branch must be wrapped in
   `tier("name", cost)`. For a time multiplier that applies to every price,
   use the current frontend-compatible shape: make the base price the
   off-peak price, then add one factor per disjoint peak window:

   ```text
   tier("base", OFFPEAK_COST)
     * (PEAK_WINDOW_1 ? 2 : 1)
     * (PEAK_WINDOW_2 ? 2 : 1)
   ```

   Each factor must have the exact one-outer-pair `(condition ? number : 1)` form so the
   visual editor and pricing display can parse it. The peak windows must be
   mutually exclusive; otherwise the factors multiply twice. If distinct
   `matched_tier` values are required in logs, use a raw-editor-only ternary:

   ```text
   (SCHEDULE_CONDITION) ? tier("peak", PEAK_COST) : tier("off_peak", OFFPEAK_COST)
   ```

   Explain that this second shape is less discoverable by the visual rule
   parser. Do not use the old documented `|||` separator; the current code
   evaluates one combined expression and the frontend combines rules as
   `(base) * rules`.

7. **Validate before delivery.** Check that all coefficients are finite and
   non-negative, every source price appears exactly once, and no unsupported
   function or variable was introduced. Run
   `go run .codex/skills/upstream-pricing-expression/scripts/validate_expression.go`
   with the exact expression (or compile it directly with
   `billingexpr.CompileFromCache`) and, where practical, run the repository's
   billing smoke tests. Inspect the expression for the schedule boundaries,
   weekday coverage, cache mapping, and unit conversion. Test representative
   token vectors and document boundary cases. Do not treat the frontend local
   cost preview as proof for expressions using time, `param`, or `header`:
   that preview does not provide those runtime functions.

## Time-based billing warning

`hour`, `minute`, `weekday`, `month`, and `day` read `time.Now()` at evaluation
time. The pre-consume and settlement paths evaluate the expression separately,
and the billing snapshot does not freeze the matched time tier. A request that
crosses a peak/off-peak boundary can therefore be estimated at one rate and
settled at another. State this limitation in every time-based result and do
not claim that the provider's request-start-time semantics are implemented.
If fixed request-start pricing is required, report that the current engine
needs a timestamp input/freeze change before this expression alone can provide
that guarantee.

## Required result

Return a concise result with these fields (plain text or JSON is acceptable):

- `status`: `ready`, `needs_confirmation`, or `cannot_verify`;
  use `needs_confirmation` when the source is clear but a project setting or
  business choice (such as currency conversion) is missing; use
  `cannot_verify` when the source itself is unavailable, contradictory, or
  not attributable to the provider;
- exact `model`, `source_url`, `retrieved_at`, and quoted source evidence;
- source confidence (`official_live`, `user_supplied`, or `ambiguous`);
- source currency/unit and normalized USD/1M coefficients, including the
  conversion rate and its provenance;
- the exact numeric `billing_expr` in one code block when `status=ready`;
  otherwise provide a clearly labeled non-executable `billing_expr_template`
  and keep `billing_mode: tiered_expr` as the intended mode;
- variable mapping, schedule boundaries, assumptions, and unresolved items;
- validation evidence: compilation result, token-vector checks, and any known
  pre-consume/settlement time caveat.

Do not silently fill missing cache, image, audio, minimum-fee, request-fee,
region, account, tax, promotional, or concurrency terms. Mark a term as
unsupported or unresolved and ask for confirmation when it cannot be
represented by the current token expression contract.
