<script lang="ts">
  import { onDestroy } from "svelte";
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import DelayedCircleOutlineSpinner from "@rilldata/web-common/components/spinner/DelayedCircleOutlineSpinner.svelte";
  import {
    createAdminServiceAcquireEditLock,
    createAdminServiceHeartbeatEditLock,
    createAdminServiceReleaseEditLock,
    createAdminServiceSaveSemanticResource,
    createAdminServiceGetSemanticResource,
    getAdminServiceGetSemanticResourceQueryKey,
  } from "@rilldata/web-admin/client";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  // StarData Phase 5.1 tracer bullet: edits a semantic resource through the admin
  // API instead of the runtime's file store, proving the DB round-trip
  // (Studio -> semantic_resources -> rendered file -> parser -> reconcile).
  //
  // Deliberately a raw text editor. The form-based editor (VisualMetrics) is
  // wired to FileArtifact and the runtime repo, and porting it is Phase 5.3 work.
  // Keeping this path separate means archive-mode projects keep working untouched.

  export let organization: string;
  export let project: string;
  export let resourceKind: string;
  export let resourceName: string;

  // Autosave / lock heartbeat cadence. Matches the 60s figure in the Phase 5
  // design: long enough to be cheap, short enough that a crash loses little.
  const HEARTBEAT_MS = 60_000;

  let content = "";
  let loadedContent = "";
  let validationErrors: string[] = [];
  let banner: { tone: "ok" | "err"; text: string } | null = null;

  const acquireLock = createAdminServiceAcquireEditLock();
  const heartbeatLock = createAdminServiceHeartbeatEditLock();
  const releaseLock = createAdminServiceReleaseEditLock();
  const saveResource = createAdminServiceSaveSemanticResource();

  $: resourceQuery = createAdminServiceGetSemanticResource(
    organization,
    project,
    resourceKind,
    resourceName,
  );

  // Hydrate the editor once, then leave it alone so a background refetch cannot
  // discard what the governor is typing.
  let hydrated = false;
  $: if (!hydrated && $resourceQuery.isSuccess) {
    hydrated = true;
    loadedContent = $resourceQuery.data?.resource?.definitionRaw ?? "";
    content = loadedContent;
  }

  let lockHolder: string | null = null;
  let hasLock = false;
  let heartbeatTimer: ReturnType<typeof setInterval> | null = null;

  async function takeLock() {
    try {
      // org and project travel in the path, so the body is empty. Orval still
      // requires the `data` key to be present.
      const res = await $acquireLock.mutateAsync({
        org: organization,
        project,
        data: {},
      });
      hasLock = !!res.acquired;
      if (!hasLock) {
        lockHolder =
          res.lock?.lockedByUserName || res.lock?.lockedByUserEmail || "另一位用户";
      } else {
        lockHolder = null;
        startHeartbeat();
      }
    } catch (e: any) {
      banner = { tone: "err", text: errText(e) };
    }
  }

  function startHeartbeat() {
    stopHeartbeat();
    heartbeatTimer = setInterval(async () => {
      try {
        await $heartbeatLock.mutateAsync({
          org: organization,
          project,
          data: {},
        });
      } catch {
        // The lock lapsed or was taken over. Drop to read-only rather than letting
        // the governor keep typing into a session that can no longer save.
        hasLock = false;
        stopHeartbeat();
        banner = { tone: "err", text: "编辑锁已失效，请重新获取。" };
      }
    }, HEARTBEAT_MS);
  }

  function stopHeartbeat() {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer);
      heartbeatTimer = null;
    }
  }

  async function save() {
    validationErrors = [];
    banner = null;
    try {
      const res = await $saveResource.mutateAsync({
        org: organization,
        project,
        data: {
          resourceKind,
          resourceName,
          definitionRaw: content,
        },
      });
      if (res.validationErrors?.length) {
        validationErrors = res.validationErrors;
        return;
      }
      loadedContent = content;
      banner = {
        tone: "ok",
        text: `已保存为版本 ${res.resource?.version ?? "?"}。`,
      };
      void queryClient.invalidateQueries({
        queryKey: getAdminServiceGetSemanticResourceQueryKey(
          organization,
          project,
          resourceKind,
          resourceName,
        ),
      });
    } catch (e: any) {
      banner = { tone: "err", text: errText(e) };
    }
  }

  function errText(e: any): string {
    return e?.response?.data?.message ?? String(e);
  }

  function reset() {
    content = loadedContent;
    validationErrors = [];
    banner = null;
  }

  $: dirty = content !== loadedContent;

  onDestroy(() => {
    stopHeartbeat();
    if (hasLock) {
      // Fire-and-forget: the TTL would reclaim it anyway, but releasing promptly
      // means the next governor does not have to wait.
      void $releaseLock
        .mutateAsync({ org: organization, project, data: {} })
        .catch(() => {});
    }
  });
</script>

<div class="flex h-full min-h-0 flex-col gap-3">
  <div class="flex flex-wrap items-center gap-2">
    <span class="text-sm font-medium text-fg-primary">
      {resourceKind} / {resourceName}
    </span>
    {#if hasLock}
      <span class="rounded-sm bg-green-100 px-2 py-0.5 text-xs text-green-800 dark:bg-green-900/40 dark:text-green-200">
        编辑中
      </span>
    {:else if lockHolder}
      <span class="rounded-sm bg-amber-100 px-2 py-0.5 text-xs text-amber-800 dark:bg-amber-900/40 dark:text-amber-200">
        只读 — {lockHolder} 正在编辑
      </span>
    {:else}
      <Button type="secondary" onClick={takeLock} loading={$acquireLock.isPending}>
        获取编辑锁
      </Button>
    {/if}
    {#if dirty}
      <span class="text-xs text-fg-tertiary">未保存的修改</span>
    {/if}
  </div>

  {#if $resourceQuery.isLoading}
    <DelayedCircleOutlineSpinner isLoading={true} />
  {:else}
    <textarea
      class="min-h-0 flex-1 resize-none rounded-sm border border-gray-200 bg-surface-background p-3 font-mono text-xs text-fg-primary dark:border-gray-700"
      bind:value={content}
      readonly={!hasLock}
      spellcheck="false"
      aria-label="{resourceKind} {resourceName} 定义"
    ></textarea>

    {#if validationErrors.length}
      <ul class="rounded-sm border border-red-300 bg-red-50 p-3 text-sm text-red-800 dark:border-red-700 dark:bg-red-900/30 dark:text-red-200">
        {#each validationErrors as err (err)}
          <li>{err}</li>
        {/each}
      </ul>
    {/if}

    {#if banner}
      <div
        class="rounded-sm border px-3 py-2 text-sm {banner.tone === 'ok'
          ? 'border-green-300 bg-green-50 text-green-800 dark:border-green-700 dark:bg-green-900/30 dark:text-green-200'
          : 'border-red-300 bg-red-50 text-red-800 dark:border-red-700 dark:bg-red-900/30 dark:text-red-200'}"
      >
        {banner.text}
      </div>
    {/if}

    <div class="flex items-center gap-2">
      <Button
        type="primary"
        onClick={save}
        disabled={!hasLock || !dirty}
        loading={$saveResource.isPending}
      >
        保存
      </Button>
      <Button type="secondary" onClick={reset} disabled={!dirty}>撤销修改</Button>
    </div>
  {/if}
</div>
