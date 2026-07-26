<svelte:head>
  <title>StarData · 智能问数</title>
</svelte:head>

<div class="h-full overflow-y-auto">
  <div class="mx-auto max-w-[880px] px-9 pb-20 pt-16">
    <h1 class="text-center text-3xl font-bold text-gray-900">
      你好,想了解点什么?
    </h1>
    <p class="mt-2 text-center text-sm text-gray-400">
      直接用一句话提问,StarData 会基于已发布的业务指标为你解答
    </p>

    <a
      href="/chat?new=true"
      class="mx-auto mt-7 flex max-w-[680px] items-center gap-3 rounded-2xl border-[1.5px] border-gray-200 bg-white px-5 py-4 no-underline shadow-sm hover:border-primary-300"
    >
      <span class="text-lg">🔍</span>
      <span class="flex-1 text-[15px] text-gray-400">
        比如:"本月销售额怎么样?"
      </span>
      <span
        class="grid size-9 place-items-center rounded-xl bg-primary-600 text-white"
      >
        ➤
      </span>
    </a>

    <!-- M2 接入发布门控后由已发布指标生成;M1 静态占位 -->
    <p class="mt-5 text-center text-xs text-gray-400">
      发布指标后,这里会出现为你推荐的问题
    </p>

    <div class="mt-14 grid grid-cols-2 gap-4">
      <a
        href="/chat"
        class="rounded-2xl border border-gray-200 bg-white p-6 no-underline shadow-sm hover:border-primary-300"
      >
        <div class="text-2xl">💬</div>
        <div class="mt-2 font-semibold text-gray-900">继续对话</div>
        <div class="mt-1 text-[13px] text-gray-500">
          查看历史对话,或开启新的提问
        </div>
      </a>
      <a
        href="/boards"
        class="rounded-2xl border border-gray-200 bg-white p-6 no-underline shadow-sm hover:border-primary-300"
      >
        <div class="text-2xl">📌</div>
        <div class="mt-2 font-semibold text-gray-900">我的看板</div>
        <div class="mt-1 text-[13px] text-gray-500">
          查看已保存的分析结果与共享看板
        </div>
      </a>
    </div>
  </div>
</div>
