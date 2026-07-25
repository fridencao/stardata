---
title: StarData ISO 8601 Extensions
description: Legacy stardata- prefixed time syntax extensions
sidebar_label: StarData ISO Extensions
sidebar_position: 10
---

StarData supports a set of legacy `stardata-` prefixed keywords for common time ranges. These are retained for backward compatibility with existing configurations.

:::tip New Syntax Available
For new configurations, use the modern [Time Range Syntax](/reference/time-syntax), which is more expressive and consistent across all contexts.
:::

## Time Range Extensions

| StarData Extension | Description |
|----------------|-------------|
| `inf` | All time |
| `stardata-TD` | Today |
| `stardata-WTD` | Week to Date |
| `stardata-MTD` | Month to Date |
| `stardata-QTD` | Quarter to Date |
| `stardata-YTD` | Year to Date |
| `stardata-PDC` | Yesterday (Previous Day Complete) |
| `stardata-PWC` | Previous Week Complete |
| `stardata-PMC` | Previous Month Complete |
| `stardata-PQC` | Previous Quarter Complete |
| `stardata-PYC` | Previous Year Complete |

:::note Reference point behavior
In a dashboard context, the reference point for these expressions is `latest` (most recent data timestamp). In alert contexts, the reference point is `watermark` (data completeness marker).
:::

## Time Comparison Extensions

These extensions are used specifically in comparison contexts (the "Comparing" feature in dashboards).

| StarData Extension | Description | Usage |
|----------------|-------------|-------|
| `stardata-PP` | Previous Period | Compares against the immediately preceding period of same duration |
| `stardata-PD` | Previous Day | Compares against the same time yesterday |
| `stardata-PW` | Previous Week | Compares against the same time last week |
| `stardata-PM` | Previous Month | Compares against the same time last month |
| `stardata-PQ` | Previous Quarter | Compares against the same time last quarter |
| `stardata-PY` | Previous Year | Compares against the same time last year |

## Usage Context

### As Time Range
Extensions ending in `TD` (to-date) or `C` (complete) are valid as primary time ranges:

```yaml
# In metrics view or explore configuration
default_time_range: "stardata-MTD"  # Month to date
```

### As Comparison
Extensions starting with `stardata-P` (previous) are typically used for comparisons:

```yaml
# In explore configuration
default_comparison:
  dimension: ""  # No dimension comparison
  mode: time
```

Then select "Previous Period", "Previous Day", etc. in the dashboard UI.

## ISO 8601 Duration Support

StarData also supports standard ISO 8601 duration format:

| Format | Description | Example |
|--------|-------------|---------|
| `P<n>Y` | n years | `P1Y` = 1 year |
| `P<n>M` | n months | `P6M` = 6 months |
| `P<n>W` | n weeks | `P2W` = 2 weeks |
| `P<n>D` | n days | `P7D` = 7 days |
| `PT<n>H` | n hours | `PT24H` = 24 hours |
| `PT<n>M` | n minutes | `PT30M` = 30 minutes |
| `PT<n>S` | n seconds | `PT60S` = 60 seconds |

Combined durations:
- `P1Y6M` = 1 year and 6 months
- `P1DT12H` = 1 day and 12 hours
- `PT1H30M` = 1 hour and 30 minutes

## Migration to Modern Syntax

The modern syntax provides equivalent functionality with more flexibility. One important distinction: `DTD` supports intraday ranges (e.g., `ref/D to ref/h+1h`) while `stardata-TD` does not.

| Legacy | Modern Equivalent |
|--------|-------------------|
| `stardata-TD` | `DTD` |
| `stardata-WTD` | `WTD` |
| `stardata-MTD` | `MTD` |
| `stardata-QTD` | `QTD` |
| `stardata-YTD` | `YTD` |
| `stardata-PDC` | `1D as of watermark/D` |
| `stardata-PWC` | `1W as of watermark/W` |
| `stardata-PMC` | `1M as of watermark/M` |
| `stardata-PQC` | `1Q as of watermark/Q` |
| `stardata-PYC` | `1Y as of watermark/Y` |
| `P7D` | `7D` |
| `P1M` | `1M` or `30D` |

See [Time Range Syntax](/reference/time-syntax) for the complete modern syntax reference.
