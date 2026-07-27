<script lang="ts">
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import ConnectorExplorer from "@rilldata/web-common/features/connectors/explorer/ConnectorExplorer.svelte";
  import { connectorExplorerStore } from "@rilldata/web-common/features/connectors/explorer/connector-explorer-store";
  import StatusBadge from "@rilldata/web-common/components/status-badge/StatusBadge.svelte";
  import SectionHeader from "../../features/studio/SectionHeader.svelte";
  import {
    ResourceKind,
    useFilteredResources,
  } from "@rilldata/web-common/features/entity-management/resource-selectors";
  import type { V1Resource } from "@rilldata/web-common/runtime-client";
  import { runtimeServiceGetFile } from "@rilldata/web-common/runtime-client";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { createMetricsViewFromTable } from "../../../features/studio/create-metrics-view";
  import {
    countLabelCnCoverage,
    parseMetricsViewYaml,
  } from "../../../features/portal/metrics-view-yaml";

  const client = useRuntimeClient();
  const metricsViews = useFilteredResources(client, ResourceKind.MetricsView);

  let pickTableOpen = false;
  let creating = false;

  // Pick-table store: allow schema expansion, callback on table select
  const pickStore = connectorExplorerStore.duplicateStore(
    (connector, database, schema, table) => {
      if (!table || !schema) return;
      pickTableOpen = false;
      creating = true;
      void createMetricsViewFromTable(client, {
        connector,
        database: database ?? "",
        databaseSchema: schema,
        table,
      }).finally(() => (creating = false));
    },
  );

  $: rows = ($metricsViews.data ?? []).map((r) => {
    const name = r.meta?.name?.name ?? "";
    const spec = r.metricsView?.state?.validSpec;
    return {
      name,
      valid: !!spec,
      displayName: spec?.displayName || name,
      table: spec?.table || spec?.model || "—",
      measures: spec?.measures?.length ?? 0,
      dimensions: spec?.dimensions?.length ?? 0,
    };
  });

  let coverage: Record<string, { labeled: number; total: number }> = {};
  $: if ($metricsViews.isSuccess) void loadCoverage($metricsViews.data ?? []);

  async function loadCoverage(resources: V1Resource[]) {
    const next: typeof coverage = {};
    for (const r of resources) {
      const name = r.meta?.name?.name ?? "";
      const path = r.meta?.filePaths?.[0];
      if (!name || !path) continue;
      try {
        const file = await runtimeServiceGetFile(client, { path });
        const doc = parseMetricsViewYaml(String(file.blob ?? ""));
        if (doc) next[name] = countLabelCnCoverage(doc);
      } catch {
        // ignore single file read failure
      }
    }
    coverage = next;
  }
</script>

<svelte:head>
  <title>StarData Studio · 语义层</title>
</svelte:head>

<div class="flex items-start justify-between">
  <SectionHeader title="语义层" description="指标/维度定义 · 中文别名(label_cn) · 无代码编辑" />
  <button
    class="rounded-lg bg-primary-600 px-4 py-2 text-[13px] font-semibold text-white hover:bg-primary-700 disabled:opacity-50"
    disabled={creating}
    onclick={() => (pickTableOpen = true)}
  >
    {creating ? "正在生成…" : "＋ 从表新建指标集"}
  </button>
</div>

<div class="mt-5 card-basic overflow-hidden">
  <table class="w-full text-left text-[13px]">
    <thead class="border-b border-gray-200 bg-gray-50 text-xs text-gray-500">
      <tr>
        <th class="px-4 py-2.5 font-medium">指标集</th>
        <th class="px-4 py-2.5 font-medium">底层表</th>
        <th class="px-4 py-2.5 font-medium">指标 / 维度</th>
        <th class="px-4 py-2.5 font-medium">中文别名覆盖</th>
        <th class="px-4 py-2.5 font-medium">状态</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as row (row.name)}
        <tr
          class="cursor-pointer border-b border-gray-100 last:border-0 hover:bg-gray-50"
          onclick={() => (window.location.href = `/studio/semantics/${row.name}`)}
        >
          <td class="px-4 py-3">
            <div class="font-semibold text-gray-900">{row.displayName}</div>
            <div class="text-[11px] text-gray-400">{row.name}</div>
          </td>
          <td class="px-4 py-3 text-gray-600">{row.table}</td>
          <td class="px-4 py-3 text-gray-600">{row.measures} / {row.dimensions}</td>
          <td class="px-4 py-3 text-gray-600">
            {#if coverage[row.name]}
              {coverage[row.name].labeled}/{coverage[row.name].total}
            {:else}
              —
            {/if}
          </td>
          <td class="px-4 py-3">
            {#if row.valid}
              <StatusBadge variant="success">有效</StatusBadge>
            {:else}
              <StatusBadge variant="error">解析错误</StatusBadge>
            {/if}
          </td>
        </tr>
      {:else}
        <tr>
          <td colspan="5" class="px-4 py-10 text-center text-gray-400">
            还没有指标集。点击右上角「从表新建指标集」开始。
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<Dialog.Root bind:open={pickTableOpen}>
  <Dialog.Content class="max-h-[70vh] w-[440px] overflow-y-auto">
    <h3 class="text-sm font-bold text-gray-900">选择一张表</h3>
    <p class="mt-0.5 text-[12px] text-gray-400">
      将基于该表{creating ? "" : "由 AI "}生成指标集草稿，生成后可继续编辑
    </p>
    <div class="mt-3">
      <ConnectorExplorer store={pickStore} />
    </div>
  </Dialog.Content>
</Dialog.Root>
