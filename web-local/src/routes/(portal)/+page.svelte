<script lang="ts">
  import { Search, ArrowRight, MessageSquare, LayoutGrid } from "lucide-svelte";
  import { ResourceKind, useFilteredResources } from "@rilldata/web-common/features/entity-management/resource-selectors";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { generateRecommendedQuestions } from "../../features/portal/home/recommended-questions";
  import { portalRole } from "../../features/portal/portal-role-store";
  import { UNGATED, parsePublishYaml, usePublishFile } from "../../features/portal/publish/publish-store";

  const client = useRuntimeClient();
  const publishFile = usePublishFile(client);
  const metricsViews = useFilteredResources(client, ResourceKind.MetricsView);

  $: gate = $publishFile.isSuccess
    ? parsePublishYaml(String($publishFile.data?.blob ?? ""))
    : UNGATED;

  // 两个查询都出结果(成功或 404)后才生成,避免加载期误判空态
  $: ready =
    ($publishFile.isSuccess || $publishFile.isError) && $metricsViews.isSuccess;

  $: publishedResources = ($metricsViews.data ?? []).filter(
    (r) => !gate.gated || gate.published.has(r.meta?.name?.name ?? ""),
  );

  let questions: string[] = [];
  let questionsLoaded = false;
  $: if (ready) {
    void generateRecommendedQuestions(client, publishedResources).then((qs) => {
      questions = qs;
      questionsLoaded = true;
    });
  }

  $: placeholder = questions[0] ?? '比如："本月销售额怎么样？"';

  function chatHref(q: string) {
    return `/chat?new=true&q=${encodeURIComponent(q)}`;
  }
</script>

<svelte:head>
  <title>StarData · 智能问数</title>
</svelte:head>

<div class="h-full overflow-y-auto">
  <div class="mx-auto max-w-[880px] px-9 pb-20 pt-16">
    <h1 class="text-center text-3xl font-bold text-gray-900 dark:text-gray-100">
      你好，想了解点什么？
    </h1>
    <p class="mt-2 text-center text-sm text-gray-400 dark:text-gray-500">
      直接用一句话提问，StarData 会基于已发布的业务指标为你解答
    </p>

    <a
      href="/chat?new=true"
      class="mx-auto mt-7 flex max-w-[680px] items-center gap-3 rounded-2xl border-[1.5px] border-gray-200 bg-white px-5 py-4 no-underline shadow-sm transition-colors hover:border-primary-300 dark:border-gray-700 dark:bg-gray-900 dark:hover:border-primary-400"
    >
      <Search class="size-5 text-gray-400 dark:text-gray-500" />
      <span class="flex-1 text-[15px] text-gray-400 dark:text-gray-500">{placeholder}</span>
      <span
        class="grid size-9 place-items-center rounded-xl bg-primary-600"
      >
        <ArrowRight class="size-5 text-white" />
      </span>
    </a>

    {#if questionsLoaded && questions.length > 0}
      <div class="mx-auto mt-5 flex max-w-[680px] flex-wrap justify-center gap-2">
        {#each questions as q (q)}
          <a
            href={chatHref(q)}
            class="rounded-full border border-gray-200 bg-white px-3.5 py-1.5 text-[13px] text-gray-600 no-underline shadow-sm transition-colors hover:border-primary-300 hover:text-primary-700 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-400 dark:hover:border-primary-400"
          >
            {q}
          </a>
        {/each}
      </div>
    {:else if questionsLoaded}
      <div class="mx-auto mt-5 max-w-[680px] rounded-xl border border-dashed border-gray-300 bg-white px-5 py-4 text-center text-[13px] text-gray-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-400">
        尚无已发布的业务指标。请联系管理员在技术工作台完成指标发布。
        {#if $portalRole === "tech"}
          <a href="/studio/publish" class="ml-1 font-semibold text-primary-600 no-underline dark:text-primary-400">
            去发布 →
          </a>
        {/if}
      </div>
    {:else}
      <p class="mt-5 text-center text-xs text-gray-400 dark:text-gray-500">正在加载推荐问题…</p>
    {/if}

    <div class="mt-14 grid grid-cols-2 gap-4">
      <a
        href="/chat"
        class="rounded-2xl border border-gray-200 bg-white p-6 no-underline shadow-sm transition-colors hover:border-primary-300 dark:border-gray-700 dark:bg-gray-900 dark:hover:border-primary-400"
      >
        <MessageSquare class="size-6 text-gray-900 dark:text-gray-100" />
        <div class="mt-2 font-semibold text-gray-900 dark:text-gray-100">继续对话</div>
        <div class="mt-1 text-[13px] text-gray-500 dark:text-gray-400">
          查看历史对话,或开启新的提问
        </div>
      </a>
      <a
        href="/boards"
        class="rounded-2xl border border-gray-200 bg-white p-6 no-underline shadow-sm transition-colors hover:border-primary-300 dark:border-gray-700 dark:bg-gray-900 dark:hover:border-primary-400"
      >
        <LayoutGrid class="size-6 text-gray-900 dark:text-gray-100" />
        <div class="mt-2 font-semibold text-gray-900 dark:text-gray-100">我的看板</div>
        <div class="mt-1 text-[13px] text-gray-500 dark:text-gray-400">
          查看已保存的分析结果与共享看板
        </div>
      </a>
    </div>
  </div>
</div>
