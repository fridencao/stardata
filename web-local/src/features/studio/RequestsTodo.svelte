<script lang="ts">
  import {
    parseRequestsYaml,
    writeRequests,
    REQUESTS_PATH,
    type RequestItem,
  } from "@rilldata/web-common/features/chat/requests/requests-file";
  import { createRuntimeServiceGetFile } from "@rilldata/web-common/runtime-client";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";

  const runtimeClient = useRuntimeClient();

  // retry=false — file missing (404) means no requests yet
  $: fileQuery = createRuntimeServiceGetFile(
    runtimeClient,
    { path: REQUESTS_PATH },
    { query: { retry: false } },
  );
  $: items = $fileQuery.isError ? [] : parseRequestsYaml($fileQuery.data?.blob);
  $: openItems = items.filter((it) => it.status === "open");
  $: doneItems = items.filter((it) => it.status === "done");

  let saving = false;

  async function markDone(item: RequestItem) {
    saving = true;
    const next = items.map((it) =>
      it === item ? { ...it, status: "done" as const } : it,
    );
    try {
      await writeRequests(runtimeClient, next);
    } finally {
      saving = false;
    }
  }

  function formatTime(iso: string) {
    return iso ? iso.slice(0, 16).replace("T", " ") : "";
  }
</script>

<section class="mt-8">
  <h3 class="text-base font-bold text-gray-900">
    需求待办
    {#if openItems.length > 0}
      <span
        class="ml-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-semibold text-amber-700"
      >
        {openItems.length}
      </span>
    {/if}
  </h3>
  <p class="mt-0.5 text-[13px] text-gray-400">
    业务人员在对话中提出的数据需求（来自 /requests.yaml）
  </p>

  {#if openItems.length === 0}
    <div
      class="mt-3 rounded-xl border border-dashed border-gray-300 bg-white px-4 py-8 text-center text-sm text-gray-400"
    >
      暂无待处理需求
    </div>
  {:else}
    <ul class="mt-3 flex flex-col gap-2">
      {#each openItems as item (item.created_at + item.question)}
        <li
          class="flex items-start justify-between gap-3 rounded-xl border border-gray-200 bg-white px-4 py-3"
        >
          <div class="min-w-0">
            <p class="text-sm font-medium text-gray-900">{item.question}</p>
            {#if item.note}
              <p class="mt-0.5 text-[12px] text-gray-500">{item.note}</p>
            {/if}
            <p class="mt-0.5 text-[11px] text-gray-400">
              {formatTime(item.created_at)}
            </p>
          </div>
          <button
            type="button"
            class="shrink-0 rounded-lg border border-gray-300 px-3 py-1 text-[12px] text-gray-600 hover:border-primary-400 hover:text-primary-600 disabled:opacity-50"
            disabled={saving}
            onclick={() => markDone(item)}
          >
            ✓ 标记完成
          </button>
        </li>
      {/each}
    </ul>
  {/if}

  {#if doneItems.length > 0}
    <details class="mt-3">
      <summary class="cursor-pointer text-[12.5px] text-gray-400">
        已完成（{doneItems.length}）
      </summary>
      <ul class="mt-2 flex flex-col gap-1">
        {#each doneItems as item (item.created_at + item.question)}
          <li class="px-4 py-1.5 text-[13px] text-gray-400 line-through">
            {item.question}
          </li>
        {/each}
      </ul>
    </details>
  {/if}
</section>
