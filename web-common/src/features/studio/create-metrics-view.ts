import { goto } from "$app/navigation";
import { fileArtifacts } from "@rilldata/web-common/features/entity-management/file-artifacts";
import { getName } from "@rilldata/web-common/features/entity-management/name-utils";
import { ResourceKind } from "@rilldata/web-common/features/entity-management/resource-selectors";
import { featureFlags } from "@rilldata/web-common/features/feature-flags";
import { runtimeServiceGenerateMetricsViewFile } from "@rilldata/web-common/runtime-client";
import type { RuntimeClient } from "@rilldata/web-common/runtime-client/v2";
import { get } from "svelte/store";

export interface TableRef {
  connector: string;
  database: string;
  databaseSchema: string;
  table: string;
}

/**
 * Studio版"从表生成指标集":与IDE"Generate metrics with AI"同RPC
 * (AI不可用时runtime自动回退模板骨架),生成后跳Studio编辑页。
 * semanticsBase: 编辑页路由前缀(web-local "/studio/semantics";
 * web-admin "/[org]/[project]/-/edit/studio/semantics")。
 */
export async function createMetricsViewFromTable(
  client: RuntimeClient,
  ref: TableRef,
  semanticsBase = "/studio/semantics",
): Promise<string> {
  const name = getName(
    `${ref.table}_metrics`,
    fileArtifacts.getNamesForKind(ResourceKind.MetricsView),
  );
  await runtimeServiceGenerateMetricsViewFile(client, {
    connector: ref.connector,
    database: ref.database,
    databaseSchema: ref.databaseSchema,
    table: ref.table,
    path: `/metrics/${name}.yaml`,
    useAi: get(featureFlags.ai),
  });
  await goto(`${semanticsBase}/${name}`);
  return name;
}
