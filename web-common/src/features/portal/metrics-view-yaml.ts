import { parse } from "yaml";

export interface MetricsFieldYAML {
  name?: string;
  display_name?: string;
  label?: string;
  label_cn?: string;
}

export interface MetricsViewYAMLDoc {
  display_name?: string;
  title?: string;
  measures?: MetricsFieldYAML[];
  dimensions?: MetricsFieldYAML[];
}

/** 安全解析指标集原始 YAML,失败返回 null。 */
export function parseMetricsViewYaml(yamlText: string): MetricsViewYAMLDoc | null {
  try {
    return (parse(yamlText, { logLevel: "silent" }) as MetricsViewYAMLDoc) ?? null;
  } catch {
    return null;
  }
}

/** 字段展示名:label_cn 优先,回退 display_name/label/name(spec 2.3)。 */
export function fieldLabel(f: MetricsFieldYAML): string | undefined {
  return f.label_cn || f.display_name || f.label || f.name || undefined;
}

/** label_cn 覆盖统计(发布页用):labeled/total 含 measures + dimensions。 */
export function countLabelCnCoverage(doc: MetricsViewYAMLDoc): {
  labeled: number;
  total: number;
} {
  const fields = [...(doc.measures ?? []), ...(doc.dimensions ?? [])];
  return {
    labeled: fields.filter((f) => !!f.label_cn).length,
    total: fields.length,
  };
}
