<script lang="ts">
  import {
    parseRequestsYaml,
    writeRequests,
    REQUESTS_PATH,
    type RequestItem,
  } from "@rilldata/web-common/features/chat/requests/requests-file";
  import { createRuntimeServiceGetFile } from "@rilldata/web-common/runtime-client";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import type { RequestsBackend } from "./requests-backend";

  // 语义层入口，需求闭环用：web-admin 会传带路由前缀的地址
  export let semanticsHref = "/studio/semantics";
  // 需求清单后端：web-admin 注入 admin 通道（虚拟文件存在 Postgres，
  // runtime 的 /requests.yaml 读不到）；web-local 不传，读写 runtime 文件
  export let backend: RequestsBackend | null = null;
  /**
   * 独立需求列表页地址。传入时，嵌入版（总览/发布页）只展示前 limit 条并给出
   * 「查看全部」入口，避免需求变多之后把宿主页面挤爆。
   */
  export let allHref: string | undefined = undefined;
  /** 嵌入版展示上限；undefined 表示不限制（独立页使用）。 */
  export let limit: number | undefined = undefined;
  /** 独立页里隐藏区块标题，交由页面自己的标题承担。 */
  export let showHeading = true;

  const runtimeClient = useRuntimeClient();

  // retry=false — file missing (404) means no requests yet; disabled entirely when a backend is injected
  $: fileQuery = createRuntimeServiceGetFile(
    runtimeClient,
    { path: REQUESTS_PATH },
    { query: { retry: false, enabled: !backend } },
  );

  let backendItems: RequestItem[] = [];
  $: if (backend) void loadBackendItems(backend);
  async function loadBackendItems(b: RequestsBackend) {
    try {
      backendItems = await b.list();
    } catch {
      backendItems = [];
    }
  }

  $: items = backend
    ? backendItems
    : $fileQuery.isError
      ? []
      : parseRequestsYaml($fileQuery.data?.blob);
  $: openItems = items.filter((it) => it.status === "open");
  $: displayItems = limit != null ? openItems.slice(0, limit) : openItems;
  $: hasMore = limit != null && openItems.length > limit;
  $: doneItems = items.filter((it) => it.status === "done");

  let saving = false;

  async function markDone(item: RequestItem) {
    saving = true;
    const next = items.map((it) =>
      it === item ? { ...it, status: "done" as const } : it,
    );
    try {
      if (backend) {
        await backend.save(next);
        backendItems = next;
      } else {
        await writeRequests(runtimeClient, next);
      }
    } finally {
      saving = false;
    }
  }

  function formatTime(iso: string) {
    return iso ? iso.slice(0, 16).replace("T", " ") : "";
  }
</script>

<section class="mt-8">
  {#if showHeading}
    <h3 class="text-base font-bold text-gray-900">
      {m.studio_requests_title()}
      {#if openItems.length > 0}
        <span
          class="ml-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-semibold text-amber-700"
        >
          {openItems.length}
        </span>
      {/if}
    </h3>
    <p class="mt-0.5 text-[13px] text-gray-400">
      {m.studio_requests_desc()}
    </p>
  {/if}

  {#if openItems.length === 0}
    <div
      class="mt-3 rounded-xl border border-dashed border-gray-300 bg-surface-card px-4 py-8 text-center text-sm text-gray-400"
    >
      {m.studio_requests_empty()}
    </div>
  {:else}
    <ul class="mt-3 flex flex-col gap-2">
      {#each displayItems as item (item.created_at + item.question)}
        <li
          class="flex items-start justify-between gap-3 rounded-xl border border-gray-200 bg-surface-card px-4 py-3"
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
          <div class="flex shrink-0 items-center gap-2">
            <a
              href={semanticsHref}
              class="rounded-lg border border-primary-300 px-3 py-1 text-[12px] text-primary-600 hover:border-primary-400 hover:bg-primary-50"
            >
              {m.studio_requests_go_semantics()}
            </a>
            <button
              type="button"
              class="rounded-lg border border-gray-300 px-3 py-1 text-[12px] text-gray-600 hover:border-primary-400 hover:text-primary-600 disabled:opacity-50"
              disabled={saving}
              onclick={() => markDone(item)}
            >
              {m.studio_requests_mark_done()}
            </button>
          </div>
        </li>
      {/each}
    </ul>
    {#if hasMore && allHref}
      <a
        href={allHref}
        class="mt-2 inline-block text-sm font-medium text-primary-600 hover:text-primary-700"
      >
        {m.studio_requests_view_all({ count: openItems.length })} →
      </a>
    {/if}
  {/if}

  {#if doneItems.length > 0}
    <details class="mt-3">
      <summary class="cursor-pointer text-lg text-gray-400">
        {m.studio_requests_done_count({ count: doneItems.length })}
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
