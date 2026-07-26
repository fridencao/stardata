import { describe, expect, it } from "vitest";
import { parse } from "yaml";
import {
  appendChartToCanvasYaml,
  chartSpecToCanvasItemProps,
  newCanvasYaml,
} from "./pin-to-board";

const baseSpec = {
  metrics_view: "sales_metrics",
  time_range: { start: "2026-06-26T00:00:00Z", end: "2026-07-26T00:00:00Z" },
  time_grain: "TIME_GRAIN_DAY",
  x: { field: "order_date", type: "temporal" },
  y: { field: "total_sales", type: "quantitative" },
};

describe("chartSpecToCanvasItemProps", () => {
  it("copies fields and drops time_range/time_grain/where", () => {
    const props = chartSpecToCanvasItemProps(baseSpec);
    expect(props.metrics_view).toBe("sales_metrics");
    expect(props.x).toEqual(baseSpec.x);
    expect(props.time_range).toBeUndefined();
    expect(props.time_grain).toBeUndefined();
    expect((props as Record<string, unknown>).where).toBeUndefined();
  });

  it("serializes time_range to time_filters absolute interval", () => {
    const props = chartSpecToCanvasItemProps(baseSpec);
    const params = new URLSearchParams(props.time_filters as string);
    expect(params.get("tr")).toBe(
      "2026-06-26T00:00:00Z to 2026-07-26T00:00:00Z",
    );
  });

  it("defaults title to metrics_view when missing", () => {
    expect(chartSpecToCanvasItemProps(baseSpec).title).toBe("sales_metrics");
  });
});

describe("appendChartToCanvasYaml", () => {
  it("appends a row at the end and preserves existing content", () => {
    const existing = `type: canvas
display_name: "经营周报"
rows:
  - items:
      - markdown:
          content: "hello"
        width: 12
    height: 100px
`;
    const out = parse(appendChartToCanvasYaml(existing, "bar_chart", baseSpec));
    expect(out.rows).toHaveLength(2);
    expect(out.rows[0].items[0].markdown.content).toBe("hello");
    expect(out.rows[1].items[0].bar_chart.metrics_view).toBe("sales_metrics");
    expect(out.rows[1].items[0].width).toBe(12);
    expect(out.rows[1].height).toBe("400px");
  });

  it("creates rows when missing", () => {
    const out = parse(
      appendChartToCanvasYaml("type: canvas\n", "line_chart", baseSpec),
    );
    expect(out.rows).toHaveLength(1);
  });

  it("throws on invalid YAML", () => {
    expect(() =>
      appendChartToCanvasYaml("rows: [", "bar_chart", baseSpec),
    ).toThrow();
  });
});

describe("newCanvasYaml", () => {
  it("generates a canvas with display_name and first chart row", () => {
    const out = parse(newCanvasYaml("经营周报", "bar_chart", baseSpec));
    expect(out.type).toBe("canvas");
    expect(out.display_name).toBe("经营周报");
    expect(out.rows).toHaveLength(1);
  });
});
