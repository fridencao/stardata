import { describe, expect, it } from "vitest";
import { parseRequestsYaml } from "./requests-file";

describe("parseRequestsYaml", () => {
  it("parses valid list", () => {
    const items = parseRequestsYaml(`requests:
  - question: "能不能按门店查退货率？"
    note: "月会要用"
    created_at: "2026-07-26T10:00:00Z"
    status: open
  - question: "毛利率"
    created_at: "2026-07-25T10:00:00Z"
    status: done
`);
    expect(items).toHaveLength(2);
    expect(items[0].status).toBe("open");
    expect(items[0].note).toBe("月会要用");
    expect(items[1].status).toBe("done");
    expect(items[1].note).toBeUndefined();
  });

  it("returns [] for empty/undefined/corrupt/non-array YAML", () => {
    expect(parseRequestsYaml(undefined)).toEqual([]);
    expect(parseRequestsYaml("")).toEqual([]);
    expect(parseRequestsYaml("requests: [")).toEqual([]);
    expect(parseRequestsYaml("requests: 3")).toEqual([]);
  });

  it("filters entries without question; unknown status normalizes to open", () => {
    const items = parseRequestsYaml(`requests:
  - note: "无问题字段"
  - question: "有效"
    status: whatever
`);
    expect(items).toHaveLength(1);
    expect(items[0].status).toBe("open");
  });
});
