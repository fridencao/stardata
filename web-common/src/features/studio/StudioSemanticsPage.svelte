<script lang="ts">
  import { goto } from "$app/navigation";
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import ConnectorExplorer from "@rilldata/web-common/features/connectors/explorer/ConnectorExplorer.svelte";
  import { connectorExplorerStore } from "@rilldata/web-common/features/connectors/explorer/connector-explorer-store";
  import StatusBadge from "@rilldata/web-common/components/status-badge/StatusBadge.svelte";
  import SectionHeader from "./SectionHeader.svelte";
  import {
    ResourceKind,
    useFilteredResources,
  } from "@rilldata/web-common/features/entity-management/resource-selectors";
  import type { V1Resource } from "@rilldata/web-common/runtime-client";
  import { runtimeServiceGetFile } from "@rilldata/web-common/runtime-client";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { createMetricsViewFromTable } from "./create-metrics-view";
  import {
    countLabelCnCoverage,
    parseMetricsViewYaml,
  } from "@rilldata/web-common/features/portal/metrics-view-yaml";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  /** Semantics 路由前缀(web-local "/studio/semantics";web-admin "…/-/edit/studio/semantics") */
  export let semanticsBase = "/studio/semantics";

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
      void createMetricsViewFromTable(
        client,
        {
          connector,
          database: database ?? "",
          databaseSchema: schema,
          table,
        },
        semanticsBase,
      ).finally(() => (creating = false));
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
  <title>StarData Studio · {m.studio_tabs_semantics()}</title>
</svelte:head>

<div class="flex items-start justify-between">
  <SectionHeader title={m.studio_tabs_semantics()} description={m.studio_semantics_desc()} />
  <button
    class="rounded-lg bg-primary-600 px-4 py-2 text-[13px] font-semibold text-white hover:bg-primary-700 disabled:opacity-50"
    disabled={creating}
    onclick={() => (pickTableOpen = true)}
  >
    {creating ? m.studio_semantics_generating() : m.studio_semantics_new_from_table()}
  </button>
</div>

<div class="mt-5 card-basic overflow-hidden">
  <table class="w-full text-left text-[13px]">
    <thead class="border-b border-gray-200 bg-gray-50 text-xs text-gray-500">
      <tr>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_metrics_view()}</th>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_table()}</th>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_measures_dims()}</th>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_label_coverage()}</th>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_status()}</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as row (row.name)}
        <tr
          class="cursor-pointer border-b border-gray-100 last:border-0 hover:bg-gray-50"
          onclick={() => void goto(`${semanticsBase}/${row.name}`)}
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
              <StatusBadge variant="success">{m.studio_semantics_status_valid()}</StatusBadge>
            {:else}
              <StatusBadge variant="error">{m.studio_semantics_status_error()}</StatusBadge>
            {/if}
          </td>
        </tr>
      {:else}
        <tr>
          <td colspan="5" class="px-4 py-10 text-center text-gray-400">
            {m.studio_semantics_empty()}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<Dialog.Root bind:open={pickTableOpen}>
  <Dialog.Content class="max-h-[70vh] w-[440px] overflow-y-auto">
    <h3 class="text-sm font-bold text-gray-900">{m.studio_semantics_pick_table()}</h3>
    <p class="mt-0.5 text-[12px] text-gray-400">
      {creating ? m.studio_semantics_pick_table_desc() : m.studio_semantics_pick_table_desc_ai()}
    </p>
    <div class="mt-3">
      <ConnectorExplorer store={pickStore} />
    </div>
  </Dialog.Content>
</Dialog.Root>
