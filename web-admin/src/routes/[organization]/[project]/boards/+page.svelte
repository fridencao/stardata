<script lang="ts">
  import { page } from "$app/stores";
  import { LayoutGrid, BarChart3 } from "lucide-svelte";
  import { useDashboards } from "@rilldata/web-common/features/dashboards/listing/selectors";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  const runtimeClient = useRuntimeClient();

  $: ({
    params: { organization, project },
  } = $page);
  $: basePath = `/${organization}/${project}`;

  $: dashboardsQuery = useDashboards(runtimeClient);
  // 业务门户只展示 canvas 看板；explore 属于技术侧概念
  $: boards = ($dashboardsQuery.data ?? []).filter((res) => res.canvas);
</script>

<svelte:head>
  <title>StarData · {m.portal_tabs_boards()}</title>
</svelte:head>

<div class="h-full overflow-y-auto">
  <div class="mx-auto max-w-[1100px] px-9 pb-16 pt-10">
    <h1 class="text-xl font-bold text-gray-900">{m.portal_home_my_boards()}</h1>
    <p class="mt-0.5 text-[13px] text-gray-400">
      {m.portal_boards_desc()}
    </p>

    {#if $dashboardsQuery.isLoading}
      <p class="mt-10 text-center text-sm text-gray-400">{m.common_loading()}</p>
    {:else if boards.length === 0}
      <div
        class="mt-8 grid place-items-center rounded-2xl border border-dashed border-gray-300 bg-surface-card py-16 text-center"
      >
        <div>
          <LayoutGrid class="size-8 text-gray-400" />
          <p class="mt-3 text-sm text-gray-500">
            {m.portal_boards_empty()}
          </p>
          <a
            href={`${basePath}/chat?new=true`}
            class="mt-4 inline-block rounded-lg bg-primary-600 px-4 py-2 text-[13px] font-semibold text-white no-underline hover:bg-primary-700"
          >
            {m.portal_boards_start_asking()}
          </a>
        </div>
      </div>
    {:else}
      <div class="mt-6 grid grid-cols-3 gap-4">
        {#each boards as board (board.meta?.name?.name)}
          {@const name = board.meta?.name?.name ?? ""}
          {@const displayName = board.canvas?.spec?.displayName || name}
          <a
            href={`${basePath}/boards/${name}`}
            class="rounded-2xl border border-gray-200 bg-surface-card p-5 no-underline shadow-sm hover:border-primary-300"
          >
            <BarChart3 class="size-5 text-gray-900" />
            <div class="mt-2 truncate font-semibold text-gray-900">
              {displayName}
            </div>
            <div class="mt-1 text-[12px] text-gray-400">{m.portal_boards_open_board()}</div>
          </a>
        {/each}
      </div>
    {/if}
  </div>
</div>
