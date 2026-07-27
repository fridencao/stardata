import type { V1Resource } from "@rilldata/web-common/runtime-client";
import { runtimeServiceGetFile } from "@rilldata/web-common/runtime-client";
import type { RuntimeClient } from "@rilldata/web-common/runtime-client/v2";
import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
import {
  fieldLabel,
  parseMetricsViewYaml,
} from "../metrics-view-yaml";

const MAX_QUESTIONS = 6;

/** 单个指标集 → 最多 3 条模板问题(确定性,零 LLM)。 */
export function buildQuestionsFromYaml(yamlText: string): string[] {
  const doc = parseMetricsViewYaml(yamlText);
  if (!doc) return [];
  const measures = (doc.measures ?? [])
    .slice(0, 2)
    .map(fieldLabel)
    .filter((l): l is string => !!l);
  const dimension = (doc.dimensions ?? [])
    .map(fieldLabel)
    .find((l): l is string => !!l);
  const questions: string[] = [];
  if (measures[0])
    questions.push(m.portal_home_q_trend({ measure: measures[0] }));
  if (measures[0] && dimension)
    questions.push(m.portal_home_q_dist({ measure: measures[0], dimension }));
  const monthly = measures[1] ?? measures[0];
  if (monthly) questions.push(m.portal_home_q_month({ measure: monthly }));
  return questions;
}

/** 按已发布指标集顺序汇总,全局最多 6 条;单个文件读取失败跳过不阻塞。 */
export async function generateRecommendedQuestions(
  client: RuntimeClient,
  resources: V1Resource[],
): Promise<string[]> {
  const questions: string[] = [];
  for (const r of resources) {
    if (questions.length >= MAX_QUESTIONS) break;
    const path = r.meta?.filePaths?.[0];
    if (!path) continue;
    try {
      const file = await runtimeServiceGetFile(client, { path });
      for (const q of buildQuestionsFromYaml(String(file.blob ?? ""))) {
        if (questions.length >= MAX_QUESTIONS) break;
        questions.push(q);
      }
    } catch {
      // 单文件读取失败跳过该指标集
    }
  }
  return questions;
}
