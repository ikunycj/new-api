# Source evidence and result format

## Evidence rules

Use the provider's official pricing page, API response, or a table pasted by
the user. Capture:

- exact URL or supplied document identifier;
- retrieval date/time and page locale;
- exact model column and model ID;
- input (cache hit and miss, if present), output, and any cache-create rows;
- the relay/channel usage fields that carry each separately priced dimension;
- footnotes describing time windows, token conversion, rounding, minimums,
  taxes, region/account differences, or price-change rights.

Quote the relevant row and footnote. Do not merge columns by visual proximity,
copy a neighboring model's price, or turn a marketing statement into a
numeric coefficient. If a page is dynamic, save the rendered/data response
or explain how the row was read.

## Currency conversion

The new-api expression language is USD/1M even when the provider publishes
another currency. Ask for or verify the effective project conversion setting.
For a CNY source and `B = BillingUSDToCNYRate`:

```text
USD/1M = CNY/1M / B
```

`B` is a billing conversion setting, not automatically the live foreign
exchange rate. A missing `B` means an exact executable expression is not yet
confirmed. Show the source-currency numbers and the symbolic formula, then
use `needs_confirmation`.

For a source currency other than CNY, require an explicit source-to-USD rate
and provenance. Do not reuse `BillingUSDToCNYRate` for EUR, JPY, or another
currency; it only expresses the project's USD-to-CNY billing conversion.

Do not embed `B`, `QuotaPerUnit`, a sales group ratio, or a wallet display rate
inside the provider-price coefficients. Those are applied by the billing
runtime after the expression result.

## Result template

```text
status: ready | needs_confirmation | cannot_verify
confidence: official_live | user_supplied | ambiguous
model: <exact upstream model ID>
source_url: <official URL>
retrieved_at: <timestamp and timezone>
source_evidence:
  - quote: "<input/cache/output row>"
  - quote: "<schedule or footnote>"
source_currency: <USD/CNY/...>
source_unit: <per 1M tokens / per request / ...>
conversion:
  billing_rate_name: BillingUSDToCNYRate
  billing_rate: <verified number or unresolved>
  formula: <source currency>/1M / billing rate -> USD/1M
normalized_prices:
  input_uncached_usd_per_million: <number>
  input_cached_usd_per_million: <number, if supplied>
  cache_create_usd_per_million: <number, if supplied>
  cache_create_1h_usd_per_million: <number, if supplied>
  output_usd_per_million: <number>
  image_input_usd_per_million: <number, if separately priced>
  image_output_usd_per_million: <number, if separately priced>
  audio_input_usd_per_million: <number, if separately priced>
  audio_output_usd_per_million: <number, if separately priced>
variable_mapping: <p/cr/c/...>
billing_mode: tiered_expr
billing_expr: |
  <one exact numeric expression when status=ready>
billing_expr_template: |
  <non-executable template when status is not ready>
assumptions: <explicit assumptions only>
boundaries: <half-open windows and fallback>
unresolved: <missing or unsupported terms>
validation: <compile, vectors, and caveats>
```

For multiple models, repeat the model-specific price and expression blocks;
do not use a wildcard model name unless the provider explicitly guarantees
identical pricing for that alias set.
