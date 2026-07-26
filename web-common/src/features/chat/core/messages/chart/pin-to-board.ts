import { convertExpressionToFilterParam } from "@rilldata/web-common/features/dashboards/url-state/filters/converters";
import { ExploreStateURLParams } from "@rilldata/web-common/features/dashboards/url-state/url-params";
import { mapResolverExpressionToV1Expression } from "@rilldata/web-common/features/explore-mappers/map-metrics-resolver-query-to-dashboard";
import { isSeq, parseDocument } from "yaml";
import type { ChartType } from "../../../../components/charts/types";

/** chat chart spec fields that are transcribed to component-level filters, not copied directly */
const DROPPED_KEYS = new Set([
  "time_range",
  "comparison_time_range",
  "where",
  "time_grain",
]);

/**
 * Transform a create_chart spec into canvas inline component props.
 * - x/y/color/measure/stage/metrics_view copied verbatim (same shape on both sides)
 * - time_range → component-level time_filters ("ISO to ISO" absolute interval via rilltime)
 * - where → component-level dimension_filters; fail-open (skip filter rather than corrupt YAML)
 */
export function chartSpecToCanvasItemProps(
  chartSpec: Record<string, unknown>,
): Record<string, unknown> {
  const props: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(chartSpec)) {
    if (DROPPED_KEYS.has(key)) continue;
    if (value === undefined || value === null) continue;
    props[key] = value;
  }

  const timeRange = chartSpec.time_range as
    | { start?: string; end?: string }
    | undefined;
  if (timeRange?.start && timeRange?.end) {
    const params = new URLSearchParams();
    params.set(
      ExploreStateURLParams.TimeRange,
      `${timeRange.start} to ${timeRange.end}`,
    );
    props.time_filters = params.toString();
  }

  if (chartSpec.where) {
    try {
      const expr = mapResolverExpressionToV1Expression(chartSpec.where);
      const filter = expr ? convertExpressionToFilterParam(expr) : "";
      if (filter) props.dimension_filters = filter;
    } catch {
      // fail-open: pin without the filter rather than corrupt the YAML
    }
  }

  if (!props.title) {
    props.title = String(chartSpec.metrics_view ?? "");
  }

  return props;
}

/**
 * Append a chart item as a new row at the end of an existing canvas YAML.
 * Preserves original comments and formatting (parseDocument mutates in place).
 * @throws Error if the YAML cannot be parsed
 */
export function appendChartToCanvasYaml(
  existingYaml: string,
  chartType: ChartType,
  chartSpec: Record<string, unknown>,
): string {
  const doc = parseDocument(existingYaml);
  if (doc.errors.length > 0) {
    throw new Error("看板文件格式有误，无法写入");
  }

  const row = {
    items: [{ [chartType]: chartSpecToCanvasItemProps(chartSpec), width: 12 }],
    height: "400px",
  };

  const rows = doc.get("rows");
  if (isSeq(rows)) {
    rows.add(doc.createNode(row));
  } else {
    doc.set("rows", doc.createNode([row]));
  }
  return doc.toString();
}

/** Generate a brand-new canvas YAML containing only the given chart */
export function newCanvasYaml(
  displayName: string,
  chartType: ChartType,
  chartSpec: Record<string, unknown>,
): string {
  const doc = parseDocument("type: canvas\n");
  doc.set("display_name", displayName);
  return appendChartToCanvasYaml(doc.toString(), chartType, chartSpec);
}
