# Current expression contract

This reference summarizes the executable contract in
`pkg/billingexpr/expr.md`, `pkg/billingexpr/compile.go`, and
`pkg/billingexpr/run.go`. If this file and the code disagree, verify the code
and update the skill rather than inventing compatibility behavior.

## Variables and units

Expression coefficients are provider prices in **USD per 1,000,000 tokens**.
The runtime later converts the expression result with:

```text
quota = expression_result / 1,000,000
       * QuotaPerUnit
       * BillingUSDToCNYRate
       * group_ratio
```

| Variable | Meaning | Notes |
| --- | --- | --- |
| `p` | input tokens not priced by a referenced sub-category | fallback input bucket |
| `c` | output tokens not priced by a referenced sub-category | fallback output bucket |
| `len` | complete input context length | use for tiers, not `p` |
| `cr` | cache-read/cache-hit input tokens | referencing it removes them from `p` for OpenAI-format usage |
| `cc` | cache-create tokens | generic or 5-minute cache bucket |
| `cc1h` | one-hour cache-create tokens | Claude-specific bucket |
| `img`, `img_o` | image input/output tokens | only use when upstream reports a separately priced dimension |
| `ai`, `ao` | audio input/output tokens | only use when upstream reports a separately priced dimension |

Supported functions are `tier`, `param`, `header`, `has`, `hour`, `minute`,
`weekday`, `month`, `day`, `max`, `min`, `abs`, `ceil`, and `floor`. Standard
arithmetic, comparisons, `&&`, `||`, and ternary `?:` are supported.

`weekday(tz)` is `0=Sunday` through `6=Saturday`. Time functions use the
current clock and `time.LoadLocation(tz)`. An empty or invalid timezone falls
back to UTC, so always emit a valid IANA timezone such as `Asia/Shanghai`.

## Cache and multimodal mapping

For OpenAI-format responses, `prompt_tokens`/`completion_tokens` include
sub-categories. The engine automatically subtracts a sub-category from `p` or
`c` only when that variable is referenced in the expression. For Claude
semantic usage, input text is already separate and no such subtraction is
needed. Do not write `p - cr`, `p - img`, or similar arithmetic yourself. Also
verify the channel adaptor's usage normalization: an expression variable does
not create missing upstream metadata.

If the provider says images are converted into ordinary tokens and gives no
separate image-token price, omit `img`; those tokens remain in the applicable
input bucket. Do not invent a coefficient for an unsupported dimension.

## Canonical schedule forms

Use a half-open interval. A two-window Monday-Friday peak schedule in Shanghai
is:

```text
(weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <= 5) &&
((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) ||
 (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18))
```

The frontend's request-rule parser recognizes a factor of the exact form
`(condition ? number : 1)` (one outer pair of parentheses) and the editor combines it with a base expression
as `(base) * rules`. For a half-price off-peak schedule, use the off-peak
price as the base and one mutually exclusive `? 2 : 1` factor per peak
window. A factor such as `(condition ? 1 : 0.5)` is executable but is not
recognized as a visual request rule. A ternary with `tier("peak", ...)` and
`tier("off_peak", ...)` gives distinct trace labels but should be entered in
raw mode.

For boundaries that include minutes, a raw expression can compare a
minutes-since-midnight value such as
`hour("Asia/Shanghai") * 60 + minute("Asia/Shanghai")` against integer minute
limits. The visual request-rule parser does not recognize that arithmetic
shape, so state the raw-mode limitation.

The old `|||` request-rule notation appears in historical documentation only;
it is not a separate backend parser in the current implementation.

## Verification checklist

- Recompile the exact expression with `billingexpr.CompileFromCache`; the
  skill includes `scripts/validate_expression.go` for a repeatable syntax and
  arithmetic check.
- Run non-negative, finite vectors including zero tokens, normal input/output,
  cache-hit input, and any image/audio dimensions used.
- Check every schedule boundary: start is included, end is excluded, each
  window is covered, and weekends/outside windows select the stated fallback.
- Check that each coefficient has the expected USD/1M unit and that currency
  conversion is not duplicated in the expression.
- Record that pre-consume and settlement call time functions at different
  moments; the current snapshot does not freeze the time tier.
