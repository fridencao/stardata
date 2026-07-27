<script lang="ts">
  import { Info } from "lucide-svelte";
  import Switch from "@rilldata/web-common/components/forms/Switch.svelte";
  import StatusBadge from "@rilldata/web-common/components/status-badge/StatusBadge.svelte";
  import SectionHeader from "../../../features/studio/SectionHeader.svelte";
  import {
    ResourceKind,
    useFilteredResources,
  } from "@rilldata/web-common/features/entity-management/resource-selectors";
  import type { V1Resource } from "@rilldata/web-common/runtime-client";
  import { runtimeServiceGetFile } from "@rilldata/web-common/runtime-client";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import {
    countLabelCnCoverage,
    parseMetricsViewYaml,
  } from "../../../features/portal/metrics-view-yaml";
  import {
    UNGATED,
    parsePublishYaml,
    usePublishFile,
    writePublishYaml,
  } from "../../../features/portal/publish/publish-store";
  import RequestsTodo from "../../../features/studio/RequestsTodo.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  const client = useRuntimeClient();
  const publishFile = usePublishFile(client);
  const metricsViews = useFilteredResources(client, ResourceKind.MetricsView);

  $: gate = $publishFile.isSuccess
    ? parsePublishYaml(String($publishFile.data?.blob ?? ""))
    : UNGATED;

  $: rows = ($metricsViews.data ?? []).map((r) => {
    const name = r.meta?.name?.name ?? "";
    return {
      name,
      resource: r,
      valid: !!r.metricsView?.state?.validSpec,
      displayName: r.metricsView?.state?.validSpec?.displayName || name,
      published: !gate.gated || gate.published.has(name),
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
        // 读取失败时该行不显示覆盖数
      }
    }
    coverage = next;
  }

  let saving = false;
  async function togglePublish(name: string, next: boolean) {
    const effective = new Set(
      gate.gated
        ? gate.published
        : rows.filter((row) => row.valid).map((row) => row.name),
    );
    if (next) effective.add(name);
    else effective.delete(name);
    saving = true;
    try {
      await writePublishYaml(client, [...effective]);
    } finally {
      saving = false;
    }
  }
</script>

<svelte:head>
  <title>StarData Studio · {m.studio_tabs_publish()}</title>
</svelte:head>

<SectionHeader title={m.studio_tabs_publish()} description={m.studio_publish_desc()} />

<div class="mt-4 flex items-center rounded-xl border border-blue-200 bg-blue-50 px-4 py-2.5 text-[12.5px] text-blue-800">
  <Info class="size-4 mr-1.5 flex-shrink-0 text-blue-600" />
  {m.studio_publish_gate_before()} <code>publish.yaml</code> {m.studio_publish_gate_after()}
</div>

<div class="mt-4 card-basic overflow-hidden">
  <table class="w-full text-left text-[13px]">
    <thead class="border-b border-gray-200 bg-gray-50 text-xs text-gray-500">
      <tr>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_metrics_view()}</th>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_measures_dims()}</th>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_label_coverage()}</th>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_status()}</th>
        <th class="px-4 py-2.5 text-right font-medium">{m.studio_tabs_publish()}</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as row (row.name)}
        {@const spec = row.resource.metricsView?.state?.validSpec}
        <tr class="border-b border-gray-100 last:border-0">
          <td class="px-4 py-3">
            <div class="font-semibold text-gray-900">{row.displayName}</div>
            <div class="text-[11px] text-gray-400">{row.name}</div>
          </td>
          <td class="px-4 py-3 text-gray-600">
            {spec?.measures?.length ?? "—"} / {spec?.dimensions?.length ?? "—"}
          </td>
          <td class="px-4 py-3 text-gray-600">
            {#if coverage[row.name]}
              {coverage[row.name].labeled}/{coverage[row.name].total}
            {:else}
              —
            {/if}
          </td>
          <td class="px-4 py-3">
            {#if !row.valid}
              <StatusBadge variant="error">{m.studio_semantics_status_error()}</StatusBadge>
            {:else if row.published}
              <StatusBadge variant="success">{m.studio_publish_status_published()}</StatusBadge>
            {:else}
              <StatusBadge variant="neutral">{m.studio_publish_status_unpublished()}</StatusBadge>
            {/if}
          </td>
          <td class="px-4 py-3 text-right">
            {#if !row.valid}
              <span class="text-[11px] text-gray-400">—</span>
            {:else}
              <label class="flex items-center justify-end gap-2 text-[13px] cursor-pointer">
                <input type="checkbox" checked={row.published} disabled={saving} on:change={(e) => togglePublish(row.name, (e.target as HTMLInputElement).checked)} />
                <span>{row.published ? m.studio_publish_status_published() : m.studio_publish_status_unpublished()}</span>
              </label>
            {/if}
          </td>
        </tr>
      {:else}
        <tr>
          <td colspan="5" class="px-4 py-10 text-center text-gray-400">
            {m.studio_publish_empty()}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<RequestsTodo />
