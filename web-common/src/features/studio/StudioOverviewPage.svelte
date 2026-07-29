<script lang="ts">
  import { Info } from "lucide-svelte";
  import SectionHeader from "./SectionHeader.svelte";
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
  } from "@rilldata/web-common/features/portal/publish/publish-store";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  /** Studio 路由前缀(web-local "/studio";web-admin "…/-/edit/studio") */
  export let studioBase = "/studio";

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
  <title>StarData Studio · {m.studio_tabs_overview()}</title>
</svelte:head>

<SectionHeader title={m.studio_tabs_overview()} description={m.studio_overview_desc()} />

<div class="mt-5 grid grid-cols-4 gap-3">
  <div class="card-basic px-4 py-4">
    <a href={`${studioBase}/sources`} class="block h-full">
      <div class="text-xs text-gray-500">{m.studio_overview_connected_sources()}</div>
      <div class="mt-1 text-2xl font-bold text-gray-900">{$connectors?.data?.connectors?.length ?? "—"}</div>
      <div class="mt-1 text-[11px] text-gray-400">{m.studio_overview_manage_in_sources()}</div>
    </a>
  </div>
  <div class="card-basic px-4 py-4">
    <a href={`${studioBase}/semantics`} class="block h-full">
      <div class="text-xs text-gray-500">{m.studio_overview_semantic_views()}</div>
      <div class="mt-1 text-2xl font-bold text-gray-900">{$metricsViews.data?.length ?? "—"}</div>
      <div class="mt-1 text-[11px] text-gray-400">{m.studio_overview_manage_in_semantics()}</div>
    </a>
  </div>
  <div class="card-basic px-4 py-4">
    <div class="text-xs text-gray-500">{m.studio_overview_hit_rate_7d()}</div>
    <div class="mt-1 text-2xl font-bold text-gray-300">—</div>
    <div class="mt-1 text-[11px] text-gray-400">{m.studio_overview_hit_rate_note()}</div>
  </div>
  <div class="card-basic px-4 py-4">
    <div class="text-xs text-gray-500">{m.studio_overview_pending_requests()}</div>
    <div class="mt-1 text-2xl font-bold text-gray-900">{openRequestCount}</div>
    <div class="mt-1 text-[11px] text-gray-400">{m.studio_overview_handle_in_publish()}</div>
  </div>
</div>

<div class="mt-4 flex items-center rounded-xl border border-blue-200 bg-blue-50 px-4 py-2.5 text-[12.5px] text-blue-800">
  <Info class="size-4 mr-1.5 flex-shrink-0 text-blue-600" />
  {m.studio_overview_banner()}
</div>
