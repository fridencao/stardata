<script lang="ts">
  import { useDashboards } from "@rilldata/web-common/features/dashboards/listing/selectors";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";

  const runtimeClient = useRuntimeClient();

  $: dashboardsQuery = useDashboards(runtimeClient);
  // 业务门户只展示 canvas 看板；explore 属于技术侧概念
  $: boards = ($dashboardsQuery.data ?? []).filter((res) => res.canvas);
</script>

<svelte:head>
  <title>StarData · 看板</title>
</svelte:head>

<div class="h-full overflow-y-auto">
  <div class="mx-auto max-w-[1100px] px-9 pb-16 pt-10">
    <h1 class="text-xl font-bold text-gray-900">我的看板</h1>
    <p class="mt-0.5 text-[13px] text-gray-400">
      在对话中生成图表后，可一键钉到看板，长期追踪
    </p>

    {#if $dashboardsQuery.isLoading}
      <p class="mt-10 text-center text-sm text-gray-400">加载中…</p>
    {:else if boards.length === 0}
      <div
        class="mt-8 grid place-items-center rounded-2xl border border-dashed border-gray-300 bg-white py-16 text-center"
      >
        <div>
          <div class="text-3xl">📌</div>
          <p class="mt-3 text-sm text-gray-500">
            还没有看板。去对话里问一个问题，把生成的图表钉过来吧
          </p>
          <a
            href="/chat?new=true"
            class="mt-4 inline-block rounded-lg bg-primary-600 px-4 py-2 text-[13px] font-semibold text-white no-underline hover:bg-primary-700"
          >
            开始提问 →
          </a>
        </div>
      </div>
    {:else}
      <div class="mt-6 grid grid-cols-3 gap-4">
        {#each boards as board (board.meta?.name?.name)}
          {@const name = board.meta?.name?.name ?? ""}
          {@const displayName = board.canvas?.spec?.displayName || name}
          <a
            href="/boards/{name}"
            class="rounded-2xl border border-gray-200 bg-white p-5 no-underline shadow-sm hover:border-primary-300"
          >
            <div class="text-2xl">📊</div>
            <div class="mt-2 truncate font-semibold text-gray-900">
              {displayName}
            </div>
            <div class="mt-1 text-[12px] text-gray-400">点击打开看板</div>
          </a>
        {/each}
      </div>
    {/if}
  </div>
</div>
