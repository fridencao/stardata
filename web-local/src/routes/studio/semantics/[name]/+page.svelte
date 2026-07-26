<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { fileArtifacts } from "@rilldata/web-common/features/entity-management/file-artifacts";
  import {
    ResourceKind,
    useResource,
  } from "@rilldata/web-common/features/entity-management/resource-selectors";
  import VisualMetrics from "@rilldata/web-common/features/workspaces/VisualMetrics.svelte";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";

  const client = useRuntimeClient();

  $: name = $page.params.name;
  $: resourceQuery = useResource(client, name, ResourceKind.MetricsView);
  $: filePath = $resourceQuery.data?.meta?.filePaths?.[0];

  $: fileArtifact = filePath ? fileArtifacts.getFileArtifact(filePath) : null;
  $: if (fileArtifact) void fileArtifact.fetchContent();
</script>

<svelte:head>
  <title>StarData Studio · {name}</title>
</svelte:head>

<div class="flex h-full min-h-0 flex-col">
  <div class="flex items-center gap-3 pb-3">
    <a href="/studio/semantics" class="text-[13px] text-gray-500 no-underline hover:text-gray-800">
      ← 返回语义层
    </a>
    <h2 class="text-base font-bold text-gray-900">{name}</h2>
    <span class="text-[11px] text-gray-400">编辑自动保存</span>
  </div>

  {#if fileArtifact}
    <div class="min-h-0 flex-1 overflow-hidden rounded-xl border border-gray-200 bg-white">
      {#key fileArtifact}
        <VisualMetrics
          {fileArtifact}
          switchView={() => {
            if (filePath) void goto(`/files${filePath}`);
          }}
        />
      {/key}
    </div>
  {:else if $resourceQuery.isError}
    <div class="rounded-xl border border-dashed border-gray-300 bg-white py-14 text-center text-sm text-gray-500">
      找不到指标集「{name}」。
      <a href="/studio/semantics" class="font-semibold text-primary-600 no-underline">返回列表</a>
    </div>
  {:else}
    <p class="text-sm text-gray-400">加载中…</p>
  {/if}
</div>
