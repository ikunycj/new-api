# DeepSeek worked example

This is a dated example, not a price cache. Re-fetch the official page before
using it:

`https://api-docs.deepseek.com/zh-cn/quick_start/pricing`

The page was inspected on 2026-08-27 at 21:56 CST (UTC+08:00). Its table at that time listed
`deepseek-v4-flash`, `deepseek-v4-pro`, and
`deepseek-v4-flash-vision-exp`, with prices in CNY per 1M tokens. The relevant
rows were:

- cached input: off-peak `0.05 / 0.15 / 0.05`, peak `0.10 / 0.30 / 0.10`;
- uncached input: off-peak `1.5 / 4.5 / 1.5`, peak `3.0 / 9.0 / 3.0`;
- output: off-peak `4.5 / 13.5 / 4.5`, peak `9.0 / 27.0 / 9.0`.

The footnote states that peak time is Beijing time Monday-Friday 09:00-12:00
and 14:00-18:00, and that the remaining time is off-peak; off-peak is half
the peak price. The vision model's images are converted to tokens and billed
with text tokens, so no separate `img` price is invented here.

## Expression construction

Let `B` be the verified project `BillingUSDToCNYRate`. For
`deepseek-v4-flash`, the peak USD coefficients are:

```text
input_uncached = 3.0 / B
input_cached   = 0.10 / B
output         = 9.0 / B
```

At `2026-08-27T21:57:19+08:00`, the local `GET /api/status` returned
`BillingUSDToCNYRate = 7` and `QuotaPerUnit = 500000`. This is runtime evidence
for this checkout, not a portable default; recheck it before saving. With
`B = 7`, the off-peak USD coefficients for `deepseek-v4-flash` are
`0.21428571428571427`, `0.0071428571428571435`, and `0.6428571428571429` for
uncached input, cached input, and output. The exact expression should retain
enough decimal precision for the chosen conversion policy and must record `B`
in the result metadata.

The frontend-compatible expression shape, with symbolic coefficients shown for
clarity, uses the off-peak prices as the base and two disjoint peak factors:

```text
tier("base", p * (1.5 / B) + cr * (0.05 / B) + c * (4.5 / B)) * (weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6 && hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12 ? 2 : 1) * (weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6 && hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18 ? 2 : 1)
```

`B` is explanatory notation only and is not a valid runtime variable; this
code block must not be pasted into `billing_expr`. Replace each coefficient
with numbers after verifying the project rate, then run the validation script.
If `B` is unknown, this example is `needs_confirmation`, not a ready-to-save
expression. To make the matched tier distinguish peak and off-peak in logs,
use raw mode and expand it to the following template. `PEAK_INPUT`,
`PEAK_CACHE`, and `PEAK_OUTPUT` are placeholders, not runtime variables:

With the captured `B = 7`, the ready-to-validate `deepseek-v4-flash` expression
is:

```text
tier("base", p * 0.21428571428571427 + cr * 0.0071428571428571435 + c * 0.6428571428571429) * (weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6 && hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12 ? 2 : 1) * (weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6 && hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18 ? 2 : 1)
```

This numeric block is valid only while the source prices and effective `B = 7`
remain unchanged.

```text
(((weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5) && ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) || (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18))) ? tier("peak", p * PEAK_INPUT + cr * PEAK_CACHE + c * PEAK_OUTPUT) : tier("off_peak", p * PEAK_INPUT * 0.5 + cr * PEAK_CACHE * 0.5 + c * PEAK_OUTPUT * 0.5))
```

At 09:00 and 14:00 the peak branch applies; at 12:00 and 18:00 it does not.
Saturday and Sunday are off-peak. The current engine evaluates the clock
again during settlement, so a request crossing one of those boundaries can
change rate between pre-consume and settlement.
