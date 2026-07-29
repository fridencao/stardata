<script lang="ts">
  import AddDataModal from "@rilldata/web-common/features/add-data/AddDataModal.svelte";
  import ConnectorExplorer from "@rilldata/web-common/features/connectors/explorer/ConnectorExplorer.svelte";
  import { ConnectorExplorerStore } from "@rilldata/web-common/features/connectors/explorer/connector-explorer-store";
  import { getAnalyzedConnectors } from "@rilldata/web-common/features/connectors/selectors";
  import StatusBadge from "@rilldata/web-common/components/status-badge/StatusBadge.svelte";
  import SectionHeader from "./SectionHeader.svelte";
  import { BehaviourEventMedium } from "@rilldata/web-common/metrics/service/BehaviourEventTypes";
  import {
    MetricsEventScreenName,
    MetricsEventSpace,
  } from "@rilldata/web-common/metrics/service/MetricsTypes";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  /** 高级模式(IDE)入口(web-local "/files";web-admin edit 工作区根) */
  export let ideHref = "/files";

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
  <title>StarData Studio · {m.studio_tabs_sources()}</title>
</svelte:head>

<div class="flex items-start justify-between">
  <SectionHeader title={m.studio_tabs_sources()} description={m.studio_sources_desc()}>
    <button
      slot="actions"
      class="rounded-lg bg-primary-600 px-4 py-2 text-[13px] font-semibold text-white hover:bg-primary-700"
      onclick={() => (addDataOpen = true)}
    >
      {m.studio_sources_add()}
    </button>
  </SectionHeader>
</div>

{#if $connectors.data?.connectors?.length}
  <div class="mt-5 grid grid-cols-3 gap-3">
    {#each $connectors.data.connectors as connector (connector.name)}
      <div class="card-basic px-4 py-4">
        <div class="flex items-center justify-between">
          <div class="font-semibold text-gray-900">{connector.name}</div>
          {#if connector.driver?.implementsOlap}
            <StatusBadge variant="info" size="sm">OLAP</StatusBadge>
          {/if}
        </div>
        <div class="mt-1 text-[12px] text-gray-500">
          {m.studio_sources_driver({ name: connector.driver?.name ?? m.studio_sources_driver_unknown() })}
        </div>
      </div>
    {/each}
  </div>
{:else if $connectors.isLoading}
  <p class="mt-5 text-sm text-gray-400">{m.studio_sources_analyzing()}</p>
{:else}
  <div class="mt-5 card-hero py-10 text-center text-sm text-gray-500">
    {m.studio_sources_empty()}
  </div>
{/if}

<h3 class="mt-8 text-lg font-bold text-fg-primary">{m.studio_sources_table_browser()}</h3>
<div class="mt-2 card-basic overflow-hidden">
  <div class="p-2">
    <ConnectorExplorer store={explorerStore} />
  </div>
</div>

<p class="mt-6 text-[12px] text-gray-400">
  {m.studio_sources_ide_hint()}
  <a href={ideHref} class="font-semibold text-primary-600 no-underline">{m.studio_sources_ide_link()}</a>
</p>

<AddDataModal
  config={{
    medium: BehaviourEventMedium.Button,
    space: MetricsEventSpace.Workspace,
    screen: MetricsEventScreenName.Home,
  }}
  bind:open={addDataOpen}
/>
