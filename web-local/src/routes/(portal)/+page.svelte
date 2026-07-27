<script lang="ts">
  import { Search, ArrowRight, MessageSquare, LayoutGrid } from "lucide-svelte";
  import { ResourceKind, useFilteredResources } from "@rilldata/web-common/features/entity-management/resource-selectors";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { generateRecommendedQuestions } from "../../features/portal/home/recommended-questions";
  import { canViewTech } from "../../features/portal/user-spaces";
  import { UNGATED, parsePublishYaml, usePublishFile } from "../../features/portal/publish/publish-store";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

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

  $: placeholder = questions[0] ?? m.portal_home_placeholder_example();

  function chatHref(q: string) {
    return `/chat?new=true&q=${encodeURIComponent(q)}`;
  }
</script>

<svelte:head>
  <title>StarData · {m.app_header_ask_ai()}</title>
</svelte:head>

<div class="h-full overflow-y-auto">
  <div class="mx-auto max-w-[880px] px-9 pb-20 pt-16">
    <h1 class="text-center text-3xl font-bold text-gray-900">
      {m.portal_home_greeting()}
    </h1>
    <p class="mt-2 text-center text-sm text-gray-400">
      {m.portal_home_subtitle()}
    </p>

    <a
      href="/chat?new=true"
      class="mx-auto mt-7 flex max-w-[680px] items-center gap-3 rounded-2xl border-[1.5px] border-gray-200 bg-surface-card px-5 py-4 no-underline shadow-sm transition-colors hover:border-primary-300"
    >
      <Search class="size-5 text-gray-400" />
      <span class="flex-1 text-[15px] text-gray-400">{placeholder}</span>
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
            class="rounded-full border border-gray-200 bg-surface-card px-3.5 py-1.5 text-[13px] text-gray-600 no-underline shadow-sm transition-colors hover:border-primary-300 hover:text-primary-700"
          >
            {q}
          </a>
        {/each}
      </div>
    {:else if questionsLoaded}
      <div class="mx-auto mt-5 max-w-[680px] rounded-xl border border-dashed border-gray-300 bg-surface-card px-5 py-4 text-center text-[13px] text-gray-500">
        {m.portal_home_no_published()}
        {#if canViewTech()}
          <a href="/studio/publish" class="ml-1 font-semibold text-primary-600 no-underline">
            {m.portal_home_go_publish()}
          </a>
        {/if}
      </div>
    {:else}
      <p class="mt-5 text-center text-xs text-gray-400">{m.portal_home_loading_questions()}</p>
    {/if}

    <div class="mt-14 grid grid-cols-2 gap-4">
      <a
        href="/chat"
        class="rounded-2xl border border-gray-200 bg-surface-card p-6 no-underline shadow-sm transition-colors hover:border-primary-300"
      >
        <MessageSquare class="size-6 text-gray-900" />
        <div class="mt-2 font-semibold text-gray-900">{m.portal_home_continue_chat()}</div>
        <div class="mt-1 text-[13px] text-gray-500">
          {m.portal_home_continue_chat_desc()}
        </div>
      </a>
      <a
        href="/boards"
        class="rounded-2xl border border-gray-200 bg-surface-card p-6 no-underline shadow-sm transition-colors hover:border-primary-300"
      >
        <LayoutGrid class="size-6 text-gray-900" />
        <div class="mt-2 font-semibold text-gray-900">{m.portal_home_my_boards()}</div>
        <div class="mt-1 text-[13px] text-gray-500">
          {m.portal_home_my_boards_desc()}
        </div>
      </a>
    </div>
  </div>
</div>
