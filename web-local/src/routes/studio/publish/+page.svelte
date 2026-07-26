<script lang="ts">
  import Switch from "@rilldata/web-common/components/forms/Switch.svelte";
  import {
    ResourceKind,
    useFilteredResources,
  } from "@rilldata/web-common/features/entity-management/resource-selectors";
  import type { V1Resource } from "@rilldata/web-common/runtime-client";
  import { runtimeServiceGetFile } from "@rilldata/web-common/runtime-client";
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
  <title>StarData Studio · 发布</title>
</svelte:head>

<h2 class="text-lg font-bold text-gray-900">发布</h2>
<p class="mt-0.5 text-[13px] text-gray-400">
  控制哪些指标集对业务门户(推荐问题 + Chat AI)可见
</p>

<div
  class="mt-4 rounded-xl border border-blue-200 bg-blue-50 px-4 py-2.5 text-[12.5px] text-blue-800"
>
  💡 门控规则:项目根目录 <code>publish.yaml</code> 列出的指标集才对业务可见;
  文件不存在或名单为空时不门控(全部可见)。
</div>

<div class="mt-4 overflow-hidden rounded-xl border border-gray-200 bg-white">
  <table class="w-full text-left text-[13px]">
    <thead class="border-b border-gray-200 bg-gray-50 text-xs text-gray-500">
      <tr>
        <th class="px-4 py-2.5 font-medium">指标集</th>
        <th class="px-4 py-2.5 font-medium">指标 / 维度</th>
        <th class="px-4 py-2.5 font-medium">中文别名覆盖</th>
        <th class="px-4 py-2.5 font-medium">状态</th>
        <th class="px-4 py-2.5 text-right font-medium">发布</th>
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
              <span class="rounded-md bg-red-50 px-2 py-0.5 text-[11px] font-semibold text-red-600">解析错误</span>
            {:else if row.published}
              <span class="rounded-md bg-green-50 px-2 py-0.5 text-[11px] font-semibold text-green-700">已发布</span>
            {:else}
              <span class="rounded-md bg-gray-100 px-2 py-0.5 text-[11px] font-semibold text-gray-500">未发布</span>
            {/if}
          </td>
          <td class="px-4 py-3 text-right">
            {#if !row.valid}
              <span class="text-[11px] text-gray-400">—</span>
            {:else}
              <label class="flex items-center justify-end gap-2 text-[13px] cursor-pointer">
                <input type="checkbox" checked={row.published} disabled={saving} on:change={(e) => togglePublish(row.name, (e.target as HTMLInputElement).checked)} />
                <span>{row.published ? "已发布" : "未发布"}</span>
              </label>
            {/if}
          </td>
        </tr>
      {:else}
        <tr>
          <td colspan="5" class="px-4 py-10 text-center text-gray-400">
            项目中还没有指标集。请先在「语义层」或高级模式(IDE)中创建。
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<RequestsTodo />
