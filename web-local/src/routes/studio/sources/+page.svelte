<script lang="ts">
  import AddDataModal from "@rilldata/web-common/features/add-data/AddDataModal.svelte";
  import ConnectorExplorer from "@rilldata/web-common/features/connectors/explorer/ConnectorExplorer.svelte";
  import { ConnectorExplorerStore } from "@rilldata/web-common/features/connectors/explorer/connector-explorer-store";
  import { getAnalyzedConnectors } from "@rilldata/web-common/features/connectors/selectors";
  import { BehaviourEventMedium } from "@rilldata/web-common/metrics/service/BehaviourEventTypes";
  import {
    MetricsEventScreenName,
    MetricsEventSpace,
  } from "@rilldata/web-common/metrics/service/MetricsTypes";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";

  const client = useRuntimeClient();
  const connectors = getAnalyzedConnectors(client, false);

  // Independent explorer store: allow schema expansion, disable IDE nav/context menu, no localStorage
  const explorerStore = new ConnectorExplorerStore({
    allowNavigateToTable: false,
    allowContextMenu: false,
    allowShowSchema: true,
    localStorage: false,
  });

  let addDataOpen = false;
</script>

<svelte:head>
  <title>StarData Studio · 数据源</title>
</svelte:head>

<div class="flex items-start justify-between">
  <div>
    <h2 class="text-lg font-bold text-gray-900">数据源</h2>
    <p class="mt-0.5 text-[13px] text-gray-400">
      已接入连接器 · 向导式新增 · 表结构浏览
    </p>
  </div>
  <button
    class="rounded-lg bg-primary-600 px-4 py-2 text-[13px] font-semibold text-white hover:bg-primary-700"
    on:click={() => (addDataOpen = true)}
  >
    ＋ 新增数据源
  </button>
</div>

{#if $connectors.data?.connectors?.length}
  <div class="mt-5 grid grid-cols-3 gap-3">
    {#each $connectors.data.connectors as connector (connector.name)}
      <div class="rounded-xl border border-gray-200 bg-white px-4 py-4">
        <div class="flex items-center justify-between">
          <div class="font-semibold text-gray-900">{connector.name}</div>
          {#if connector.driver?.implementsOlap}
            <span class="rounded-md bg-primary-50 px-1.5 py-0.5 text-[10.5px] font-semibold text-primary-700">OLAP</span>
          {/if}
        </div>
        <div class="mt-1 text-[12px] text-gray-500">
          驱动：{connector.driver?.name ?? "未知"}
        </div>
      </div>
    {/each}
  </div>
{:else if $connectors.isLoading}
  <p class="mt-5 text-sm text-gray-400">正在分析连接器…</p>
{:else}
  <div class="mt-5 rounded-xl border border-dashed border-gray-300 bg-white py-10 text-center text-sm text-gray-500">
    还没有接入数据源，点击右上角「新增数据源」开始
  </div>
{/if}

<h3 class="mt-8 text-sm font-bold text-gray-700">表浏览</h3>
<div class="mt-2 overflow-hidden rounded-xl border border-gray-200 bg-white">
  <div class="bg-white p-2">
    <ConnectorExplorer store={explorerStore} />
  </div>
</div>

<p class="mt-6 text-[12px] text-gray-400">
  需要编辑/删除连接器或写 SQL 模型？
  <a href="/files" class="font-semibold text-primary-600 no-underline">前往高级模式(IDE) →</a>
</p>

<AddDataModal
  config={{
    medium: BehaviourEventMedium.Button,
    space: MetricsEventSpace.Workspace,
    screen: MetricsEventScreenName.Home,
  }}
  bind:open={addDataOpen}
/>
