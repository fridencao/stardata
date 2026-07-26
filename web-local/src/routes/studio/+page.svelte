<script lang="ts">
  import { Info } from "lucide-svelte";
  import SectionHeader from "../../features/studio/SectionHeader.svelte";
  import {
    ResourceKind,
    useFilteredResources,
  } from "@rilldata/web-common/features/entity-management/resource-selectors";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { getAnalyzedConnectors } from "@rilldata/web-common/features/connectors/selectors";
  import {
    parseRequestsYaml,
    REQUESTS_PATH,
  } from "@rilldata/web-common/features/chat/requests/requests-file";
  import { createRuntimeServiceGetFile } from "@rilldata/web-common/runtime-client";
  import {
    UNGATED,
    parsePublishYaml,
    usePublishFile,
  } from "../../features/portal/publish/publish-store";

  const client = useRuntimeClient();
  const publishFile = usePublishFile(client);
  const metricsViews = useFilteredResources(client, ResourceKind.MetricsView);
  const connectors = getAnalyzedConnectors(client, false);
  const requestsFileQuery = createRuntimeServiceGetFile(
    client,
    { path: REQUESTS_PATH },
    { query: { retry: false } },
  );
  $: openRequestCount = $requestsFileQuery.isError
    ? 0
    : parseRequestsYaml($requestsFileQuery.data?.blob).filter(
        (it) => it.status === "open",
      ).length;

  $: gate = $publishFile.isSuccess
    ? parsePublishYaml(String($publishFile.data?.blob ?? ""))
    : UNGATED;

  $: publishedCount = ($metricsViews.data ?? []).filter(
    (r) =>
      !!r.metricsView?.state?.validSpec &&
      (!gate.gated || gate.published.has(r.meta?.name?.name ?? "")),
  ).length;
</script>

<svelte:head>
  <title>StarData Studio · 概览</title>
</svelte:head>

<SectionHeader title="概览" description="一屏了解:业务现在能问什么,还缺什么" />

<div class="mt-5 grid grid-cols-4 gap-3">
  <div class="card-basic px-4 py-4">
    <a href="/studio/sources" class="block h-full">
      <div class="text-xs text-gray-500">已接入数据源</div>
      <div class="mt-1 text-2xl font-bold text-gray-900">{$connectors?.data?.connectors?.length ?? "—"}</div>
      <div class="mt-1 text-[11px] text-gray-400">在「数据源」中管理</div>
    </a>
  </div>
  <div class="card-basic px-4 py-4">
    <a href="/studio/semantics" class="block h-full">
      <div class="text-xs text-gray-500">语义层指标集</div>
      <div class="mt-1 text-2xl font-bold text-gray-900">{$metricsViews.data?.length ?? "—"}</div>
      <div class="mt-1 text-[11px] text-gray-400">在「语义层」中管理</div>
    </a>
  </div>
  <div class="card-basic px-4 py-4">
    <div class="text-xs text-gray-500">近 7 天提问命中率</div>
    <div class="mt-1 text-2xl font-bold text-gray-300">—</div>
    <div class="mt-1 text-[11px] text-gray-400">M3 接入统计</div>
  </div>
  <div class="card-basic px-4 py-4">
    <div class="text-xs text-gray-500">待处理需求</div>
    <div class="mt-1 text-2xl font-bold text-gray-900">{openRequestCount}</div>
    <div class="mt-1 text-[11px] text-gray-400">在「发布」页处理</div>
  </div>
</div>

<div class="mt-4 flex items-center rounded-xl border border-blue-200 bg-blue-50 px-4 py-2.5 text-[12.5px] text-blue-800">
  <Info class="size-4 mr-1.5 flex-shrink-0 text-blue-600" />
  M3/4 已上线数据源管理、语义层向导、看板与钉图能力。近 7 天提问命中率仍在规划中，完整版功能可在「高级模式(IDE)」继续配置。
</div>
