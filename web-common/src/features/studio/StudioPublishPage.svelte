<script lang="ts">
  import { Info } from "lucide-svelte";
  import StatusBadge from "@rilldata/web-common/components/status-badge/StatusBadge.svelte";
  import SectionHeader from "./SectionHeader.svelte";
  import {
    ResourceKind,
    useFilteredResources,
  } from "@rilldata/web-common/features/entity-management/resource-selectors";
  import type { V1Resource } from "@rilldata/web-common/runtime-client";
  import { runtimeServiceGetFile } from "@rilldata/web-common/runtime-client";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import {
    countLabelCnCoverage,
    parseMetricsViewYaml,
  } from "@rilldata/web-common/features/portal/metrics-view-yaml";
  import {
    UNGATED,
    parsePublishYaml,
    usePublishFile,
    writePublishYaml,
  } from "@rilldata/web-common/features/portal/publish/publish-store";
  import RequestsTodo from "./RequestsTodo.svelte";
  import type { PublishBackend, PublishEntry } from "./publish-backend";
  import type { RequestsBackend } from "./requests-backend";
  import { eventBus } from "@rilldata/web-common/lib/event-bus/event-bus";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  // 发布模型后端（web-admin 注入；web-local 不传则隐藏发布/历史区块）
  export let backend: PublishBackend | null = null;
  export let semanticsBase = "/studio/semantics";
  // 需求清单后端（web-admin 注入 admin 通道；web-local 不传则读写 runtime 文件）
  export let requestsBackend: RequestsBackend | null = null;
  // 独立需求列表页地址（web-admin 传入；不传则嵌入版不显示「查看全部」）
  export let requestsPageHref: string | undefined = undefined;

  const client = useRuntimeClient();
  const publishFile = usePublishFile(client);
  const metricsViews = useFilteredResources(client, ResourceKind.MetricsView);

  $: gate = $publishFile.isSuccess
    ? parsePublishYaml(String($publishFile.data?.blob ?? ""))
    : UNGATED;

  $: rows = ($metricsViews.data ?? []).map((r) => {
    const name = r.meta?.name?.name ?? "";
    return {
      name,
      resource: r,
      valid: !!r.metricsView?.state?.validSpec,
      displayName: r.metricsView?.state?.validSpec?.displayName || name,
      published: !gate.gated || gate.published.has(name),
    };
  });

  let coverage: Record<string, { labeled: number; total: number }> = {};
  $: if ($metricsViews.isSuccess) void loadCoverage($metricsViews.data ?? []);

  async function loadCoverage(resources: V1Resource[]) {
    const next: typeof coverage = {};
    for (const r of resources) {
      const name = r.meta?.name?.name ?? "";
      const path = r.meta?.filePaths?.[0];
      if (!name || !path) continue;
      try {
        const file = await runtimeServiceGetFile(client, { path });
        const doc = parseMetricsViewYaml(String(file.blob ?? ""));
        if (doc) next[name] = countLabelCnCoverage(doc);
      } catch {
        // 读取失败时该行不显示覆盖数
      }
    }
    coverage = next;
  }

  let saving = false;
  async function togglePublish(name: string, next: boolean) {
    const effective = new Set(
      gate.gated
        ? gate.published
        : rows.filter((row) => row.valid).map((row) => row.name),
    );
    if (next) effective.add(name);
    else effective.delete(name);
    saving = true;
    try {
      await writePublishYaml(client, [...effective]);
    } finally {
      saving = false;
    }
  }

  // —— 发布模型（打包发布 + 历史 + 回滚）——
  let history: PublishEntry[] = [];
  let publishNote = "";
  let publishing = false;
  let rollingBack: number | null = null;

  $: if (backend) void loadHistory(backend);

  async function loadHistory(b: PublishBackend) {
    try {
      history = await b.list();
    } catch {
      history = [];
    }
  }

  function errMessage(e: unknown): string {
    const err = e as {
      response?: { data?: { message?: string } };
      message?: string;
    };
    return err?.response?.data?.message ?? err?.message ?? String(e);
  }

  async function doPublish() {
    if (!backend || publishing) return;
    if (!confirm(m.studio_publish_confirm())) return;
    publishing = true;
    try {
      const entry = await backend.publish(publishNote);
      publishNote = "";
      eventBus.emit("notification", {
        type: "success",
        message: m.studio_publish_success({ version: entry.version }),
      });
      await loadHistory(backend);
    } catch (e) {
      eventBus.emit("notification", {
        type: "error",
        message: m.studio_publish_error({ message: errMessage(e) }),
      });
    } finally {
      publishing = false;
    }
  }

  async function doRollback(version: number) {
    if (!backend || rollingBack !== null) return;
    if (!confirm(m.studio_publish_rollback_confirm({ version }))) return;
    rollingBack = version;
    try {
      await backend.rollback(version);
      eventBus.emit("notification", {
        type: "success",
        message: m.studio_publish_rollback_success({ version }),
      });
      await loadHistory(backend);
    } catch (e) {
      eventBus.emit("notification", {
        type: "error",
        message: m.studio_publish_rollback_error({ message: errMessage(e) }),
      });
    } finally {
      rollingBack = null;
    }
  }

  function formatTime(iso: string) {
    return iso ? iso.slice(0, 16).replace("T", " ") : "";
  }
</script>

<svelte:head>
  <!-- i18n-ignore: brand name -->
  <title>StarData Studio · {m.studio_tabs_publish()}</title>
</svelte:head>

<SectionHeader title={m.studio_tabs_publish()} description={m.studio_publish_desc()} />

{#if backend}
  <div class="mt-4 card-basic p-4">
    <h3 class="text-sm font-bold text-gray-900">{m.studio_publish_release_title()}</h3>
    <p class="mt-0.5 text-lg text-gray-500">{m.studio_publish_release_desc()}</p>
    <div class="mt-3 flex items-center gap-2">
      <input
        type="text"
        class="h-9 flex-1 rounded-lg border border-gray-300 px-3 text-[13px] focus:border-primary-400 focus:outline-none"
        placeholder={m.studio_publish_note_placeholder()}
        maxlength="500"
        bind:value={publishNote}
        disabled={publishing}
      />
      <button
        type="button"
        class="h-9 shrink-0 rounded-lg bg-primary-600 px-4 text-[13px] font-semibold text-white hover:bg-primary-700 disabled:opacity-50"
        disabled={publishing}
        on:click={doPublish}
      >
        {publishing ? m.studio_publish_publishing() : m.studio_publish_action()}
      </button>
    </div>
  </div>
{/if}

{#if backend}
  <div class="mt-4 card-basic p-4">
    <h3 class="text-sm font-bold text-gray-900">{m.studio_publish_release_title()}</h3>
    <p class="mt-0.5 text-lg text-gray-500">{m.studio_publish_release_desc()}</p>
    <div class="mt-3 flex items-center gap-2">
      <input
        type="text"
        class="h-9 flex-1 rounded-lg border border-gray-300 px-3 text-[13px] focus:border-primary-400 focus:outline-none"
        placeholder={m.studio_publish_note_placeholder()}
        maxlength="500"
        bind:value={publishNote}
        disabled={publishing}
      />
      <button
        type="button"
        class="h-9 shrink-0 rounded-lg bg-primary-600 px-4 text-[13px] font-semibold text-white hover:bg-primary-700 disabled:opacity-50"
        disabled={publishing}
        on:click={doPublish}
      >
        {publishing ? m.studio_publish_publishing() : m.studio_publish_action()}
      </button>
    </div>
  </div>
{/if}

<div class="mt-4 flex items-center rounded-xl border border-blue-200 bg-blue-50 px-4 py-2.5 text-lg text-blue-800">
  <Info class="size-4 mr-1.5 flex-shrink-0 text-blue-600" />
  {m.studio_publish_gate_before()} <code>publish.yaml</code> {m.studio_publish_gate_after()}
</div>

<div class="mt-4 card-basic overflow-hidden">
  <table class="w-full text-left text-[13px]">
    <thead class="border-b border-gray-200 bg-gray-50 text-xs text-gray-500">
      <tr>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_metrics_view()}</th>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_measures_dims()}</th>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_label_coverage()}</th>
        <th class="px-4 py-2.5 font-medium">{m.studio_semantics_col_status()}</th>
        <th class="px-4 py-2.5 text-right font-medium">{m.studio_tabs_publish()}</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as row (row.name)}
        {@const spec = row.resource.metricsView?.state?.validSpec}
        <tr class="border-b border-gray-100 last:border-0">
          <td class="px-4 py-3">
            <div class="font-semibold text-gray-900">{row.displayName}</div>
            <div class="text-[11px] text-gray-400">{row.name}</div>
          </td>
          <td class="px-4 py-3 text-gray-600">
            {spec?.measures?.length ?? "—"} / {spec?.dimensions?.length ?? "—"}
          </td>
          <td class="px-4 py-3 text-gray-600">
            {#if coverage[row.name]}
              {coverage[row.name].labeled}/{coverage[row.name].total}
            {:else}
              —
            {/if}
          </td>
          <td class="px-4 py-3">
            {#if !row.valid}
              <StatusBadge variant="error">{m.studio_semantics_status_error()}</StatusBadge>
            {:else if row.published}
              <StatusBadge variant="success">{m.studio_publish_status_published()}</StatusBadge>
            {:else}
              <StatusBadge variant="neutral">{m.studio_publish_status_unpublished()}</StatusBadge>
            {/if}
          </td>
          <td class="px-4 py-3 text-right">
            {#if !row.valid}
              <span class="text-[11px] text-gray-400">—</span>
            {:else}
              <label class="flex items-center justify-end gap-2 text-[13px] cursor-pointer">
                <input type="checkbox" checked={row.published} disabled={saving} on:change={(e) => togglePublish(row.name, (e.target as HTMLInputElement).checked)} />
                <span>{row.published ? m.studio_publish_status_published() : m.studio_publish_status_unpublished()}</span>
              </label>
            {/if}
          </td>
        </tr>
      {:else}
        <tr>
          <td colspan="5" class="px-4 py-10 text-center text-gray-400">
            {m.studio_publish_empty()}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

{#if backend}
  <h3 class="mt-8 text-base font-bold text-gray-900">{m.studio_publish_history_title()}</h3>
  <div class="mt-3 card-basic overflow-hidden">
    <table class="w-full text-left text-[13px]">
      <thead class="border-b border-gray-200 bg-gray-50 text-xs text-gray-500">
        <tr>
          <th class="px-4 py-2.5 font-medium">{m.studio_publish_history_col_version()}</th>
          <th class="px-4 py-2.5 font-medium">{m.studio_publish_history_col_time()}</th>
          <th class="px-4 py-2.5 font-medium">{m.studio_publish_history_col_publisher()}</th>
          <th class="px-4 py-2.5 font-medium">{m.studio_publish_history_col_note()}</th>
          <th class="px-4 py-2.5 text-right font-medium"></th>
        </tr>
      </thead>
      <tbody>
        {#each history as entry (entry.version)}
          <tr class="border-b border-gray-100 last:border-0">
            <td class="px-4 py-3">
              <span class="font-semibold text-gray-900">v{entry.version}</span>
              {#if entry.current}
                <span class="ml-1.5 inline-block align-middle">
                  <StatusBadge variant="success">{m.studio_publish_current()}</StatusBadge>
                </span>
              {/if}
            </td>
            <td class="px-4 py-3 text-gray-600">{formatTime(entry.created_at)}</td>
            <td class="px-4 py-3 text-gray-600">{entry.published_by || "—"}</td>
            <td class="px-4 py-3 text-gray-600">{entry.note || "—"}</td>
            <td class="px-4 py-3 text-right">
              {#if !entry.current}
                <button
                  type="button"
                  class="rounded-lg border border-gray-300 px-3 py-1 text-[12px] text-gray-600 hover:border-primary-400 hover:text-primary-600 disabled:opacity-50"
                  disabled={rollingBack !== null}
                  on:click={() => doRollback(entry.version)}
                >
                  {m.studio_publish_rollback()}
                </button>
              {/if}
            </td>
          </tr>
        {:else}
          <tr>
            <td colspan="5" class="px-4 py-10 text-center text-gray-400">
              {m.studio_publish_history_empty()}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

<RequestsTodo semanticsHref={semanticsBase} backend={requestsBackend} limit={3} allHref={requestsPageHref} />
