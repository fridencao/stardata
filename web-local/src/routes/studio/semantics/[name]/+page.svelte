<script lang="ts">
  import { ChevronLeft } from "lucide-svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import SectionHeader from "../../../features/studio/SectionHeader.svelte";
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
  <SectionHeader title={name} description="编辑自动保存" />

  {#if fileArtifact}
    <div class="min-h-0 flex-1 overflow-hidden card-basic">
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
    <div class="card-hero py-14 text-center text-sm text-gray-500">
      找不到指标集「{name}」。
      <a href="/studio/semantics" class="ml-1 font-semibold text-accent-primary-action no-underline">返回列表</a>
    </div>
  {:else}
    <p class="text-sm text-gray-400">加载中…</p>
  {/if}
</div>
