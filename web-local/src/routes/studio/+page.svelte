<svelte:head>
  <title>StarData Studio · 概览</title>
</svelte:head>

<h2 class="text-lg font-bold text-gray-900">概览 · 配置健康度</h2>
<p class="mt-0.5 text-[13px] text-gray-400">
  一屏了解:业务现在能问什么,还缺什么
</p>

<div class="mt-5 grid grid-cols-4 gap-3">
  {#each [{ k: "已接入数据源", hint: "在「数据源」中管理" }, { k: "已发布指标集", hint: "M2 接入发布门控" }, { k: "近 7 天提问命中率", hint: "M2 接入统计" }, { k: "待处理需求", hint: "M4 接入需求回流" }] as card (card.k)}
    <div class="rounded-xl border border-gray-200 bg-white px-4 py-4">
      <div class="text-xs text-gray-500">{card.k}</div>
      <div class="mt-1 text-2xl font-bold text-gray-300">—</div>
      <div class="mt-1 text-[11px] text-gray-400">{card.hint}</div>
    </div>
  {/each}
</div>

<div
  class="mt-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-2.5 text-[12.5px] text-amber-800"
>
  💡 M1 骨架版:健康度数据、发布状态与需求回流将在后续里程碑接入。日常配置请从左侧「数据源 / 语义层」进入,完整 IDE 在「高级」。
</div>
