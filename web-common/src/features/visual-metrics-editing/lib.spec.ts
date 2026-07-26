import { describe, expect, it } from "vitest";
import { parseDocument, YAMLMap, YAMLSeq } from "yaml";
import { YAMLDimension, YAMLMeasure } from "./lib";

function firstItem(yaml: string, key: string): YAMLMap<string, string> {
  const doc = parseDocument(yaml);
  const seq = doc.get(key) as YAMLSeq;
  return seq.items[0] as YAMLMap<string, string>;
}

describe("label_cn support", () => {
  it("reads label_cn on dimensions", () => {
    const item = firstItem(
      "dimensions:\n  - name: region\n    display_name: Region\n    label_cn: 地区\n",
      "dimensions",
    );
    const d = new YAMLDimension(item);
    expect(d.label_cn).toBe("地区");
    expect(d.display_name).toBe("Region");
  });

  it("reads label_cn on measures", () => {
    const item = firstItem(
      "measures:\n  - name: total_sales\n    expression: sum(sales)\n    label_cn: 销售额\n",
      "measures",
    );
    const m = new YAMLMeasure(item);
    expect(m.label_cn).toBe("销售额");
  });

  it("defaults to empty string when absent", () => {
    const dimItem = firstItem("dimensions:\n  - name: region\n", "dimensions");
    expect(new YAMLDimension(dimItem).label_cn).toBe("");
    expect(new YAMLMeasure().label_cn).toBe("");
  });
});
