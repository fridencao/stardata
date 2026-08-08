<script lang="ts">
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import Switch from "@rilldata/web-common/components/forms/Switch.svelte";
  import DelayedCircleOutlineSpinner from "@rilldata/web-common/components/spinner/DelayedCircleOutlineSpinner.svelte";
  import {
    createAdminServicePublishSemanticProject,
    createAdminServiceListSemanticVersions,
    createAdminServiceListResourceVisibility,
    createAdminServiceSetResourceVisibility,
    createAdminServiceListSemanticResources,
    getAdminServiceListSemanticVersionsQueryKey,
    getAdminServiceListResourceVisibilityQueryKey,
  } from "@rilldata/web-admin/client";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  export let organization: string;
  export let project: string;

  // The DB semantic layer's publish flow (Phase 5.2). This page is a parallel
  // surface to the archive-mode /publish page: only DB-mode projects reach it,
  // via the layout's semanticLayerMode check.

  let note = "";
  let banner: { tone: "ok" | "err"; text: string } | null = null;

  const publishMutation = createAdminServicePublishSemanticProject();
  const visibilityMutation = createAdminServiceSetResourceVisibility();

  $: versionsQuery = createAdminServiceListSemanticVersions(organization, project);
  $: visibilityQuery = createAdminServiceListResourceVisibility(organization, project);
  $: resourcesQuery = createAdminServiceListSemanticResources(organization, project);

  $: versions = $versionsQuery.data?.versions ?? [];
  $: visibilityRows = $visibilityQuery.data?.visibility ?? [];
  $: resources = $resourcesQuery.data?.resources ?? [];

  // Visibility state as a map for O(1) lookup while rendering the toggle list.
  $: visibleByKey = new Map<string, boolean>(
    visibilityRows.map((r) => [`${r.resourceKind}/${r.resourceName?.toLowerCase()}`, !!r.visible]),
  );

  // Only resources the runtime's publish gate cares about (metrics_view / explore /
  // canvas) show a visibility toggle. Everything else (model, source, theme...) is
  // shared infrastructure that business users never reach directly.
  const GATED_KINDS = new Set(["metrics_view", "explore", "canvas"]);
  $: gatedResources = resources.filter(
    (r) => r.resourceKind && GATED_KINDS.has(r.resourceKind),
  );

  function isVisible(kind: string, name: string): boolean {
    return !!visibleByKey.get(`${kind}/${name.toLowerCase()}`);
  }

  async function publish() {
    banner = null;
    try {
      const res = await $publishMutation.mutateAsync({
        org: organization,
        project,
        data: { note },
      });
      note = "";
      banner = {
        tone: "ok",
        text: `已发布为版本 ${res.version?.version ?? "?"}。运行时已收到通知。`,
      };
      await queryClient.invalidateQueries({
        queryKey: getAdminServiceListSemanticVersionsQueryKey(organization, project),
      });
    } catch (e: any) {
      banner = { tone: "err", text: errText(e) };
    }
  }

  async function toggleVisibility(kind: string, name: string, next: boolean) {
    banner = null;
    try {
      await $visibilityMutation.mutateAsync({
        org: organization,
        project,
        data: { resourceKind: kind, resourceName: name, visible: next },
      });
      await queryClient.invalidateQueries({
        queryKey: getAdminServiceListResourceVisibilityQueryKey(organization, project),
      });
    } catch (e: any) {
      banner = { tone: "err", text: errText(e) };
    }
  }

  function errText(e: any): string {
    return e?.response?.data?.message ?? String(e);
  }

  function statusLabel(s: string | undefined): string {
    switch (s) {
      case "published":
        return "已发布";
      case "validating":
        return "校验中";
      case "rejected":
        return "已拒绝";
      case "draft":
        return "草稿";
      default:
        return s ?? "";
    }
  }

  function statusClass(s: string | undefined): string {
    switch (s) {
      case "published":
        return "bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-200";
      case "rejected":
        return "bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-200";
      case "validating":
        return "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200";
      default:
        return "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200";
    }
  }

  function formatTime(iso: string | undefined): string {
    if (!iso) return "";
    const d = new Date(iso);
    return Number.isNaN(d.getTime()) ? iso : d.toLocaleString();
  }
</script>

<div class="flex flex-col gap-6">
  <div>
    <h2 class="text-lg font-semibold text-fg-primary">发布</h2>
    <p class="mt-1 text-sm text-fg-tertiary">
      发布会为当前草稿定义打一个原子快照并作为新版本上线。业务用户仅能看到「已开放」的资源。
    </p>
  </div>

  <!-- Publish action -->
  <div class="flex flex-col gap-3 rounded-sm border border-gray-200 p-4 dark:border-gray-700">
    <label class="text-sm font-medium text-fg-secondary" for="publish-note">发布说明（可选）</label>
    <input
      id="publish-note"
      class="rounded-sm border border-gray-200 bg-surface-background px-2 py-1.5 text-sm text-fg-primary dark:border-gray-700"
      bind:value={note}
      placeholder="本次发布的目的、影响范围等"
    />
    <div>
      <Button type="primary" onClick={publish} loading={$publishMutation.isPending}>
        发布当前草稿
      </Button>
    </div>
    {#if banner}
      <div
        class="rounded-sm border px-3 py-2 text-sm {banner.tone === 'ok'
          ? 'border-green-300 bg-green-50 text-green-800 dark:border-green-700 dark:bg-green-900/30 dark:text-green-200'
          : 'border-red-300 bg-red-50 text-red-800 dark:border-red-700 dark:bg-red-900/30 dark:text-red-200'}"
      >
        {banner.text}
      </div>
    {/if}
  </div>

  <!-- Resource visibility -->
  <div class="flex flex-col gap-3">
    <div>
      <h3 class="text-base font-semibold text-fg-primary">业务侧可见性</h3>
      <p class="text-sm text-fg-tertiary">
        默认全部不可见。业务用户只能看到显式开启的看板与语义视图。数据源、模型等基础资源始终对业务隐藏。
      </p>
    </div>

    {#if $resourcesQuery.isLoading || $visibilityQuery.isLoading}
      <DelayedCircleOutlineSpinner isLoading={true} />
    {:else if gatedResources.length === 0}
      <p class="text-sm text-fg-tertiary">当前草稿里没有可见性可控的资源。</p>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full border-collapse text-sm">
          <thead>
            <tr class="border-b border-gray-200 text-left dark:border-gray-700">
              <th class="py-2 pr-4 font-medium text-fg-secondary">资源</th>
              <th class="py-2 pr-4 font-medium text-fg-secondary">类型</th>
              <th class="py-2 pr-4 font-medium text-fg-secondary">对业务可见</th>
            </tr>
          </thead>
          <tbody>
            {#each gatedResources as r (r.id)}
              <tr class="border-b border-gray-100 dark:border-gray-800">
                <td class="py-2 pr-4 text-fg-primary">{r.resourceName}</td>
                <td class="py-2 pr-4 text-fg-tertiary">{r.resourceKind}</td>
                <td class="py-2 pr-4">
                  <Switch
                    checked={isVisible(r.resourceKind ?? "", r.resourceName ?? "")}
                    onclick={() =>
                      toggleVisibility(
                        r.resourceKind ?? "",
                        r.resourceName ?? "",
                        !isVisible(r.resourceKind ?? "", r.resourceName ?? ""),
                      )}
                  />
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <!-- Version history -->
  <div class="flex flex-col gap-3">
    <h3 class="text-base font-semibold text-fg-primary">发布历史</h3>
    {#if $versionsQuery.isLoading}
      <DelayedCircleOutlineSpinner isLoading={true} />
    {:else if versions.length === 0}
      <p class="text-sm text-fg-tertiary">尚未发布任何版本。</p>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full border-collapse text-sm">
          <thead>
            <tr class="border-b border-gray-200 text-left dark:border-gray-700">
              <th class="py-2 pr-4 font-medium text-fg-secondary whitespace-nowrap">版本</th>
              <th class="py-2 pr-4 font-medium text-fg-secondary">状态</th>
              <th class="py-2 pr-4 font-medium text-fg-secondary whitespace-nowrap">时间</th>
              <th class="py-2 pr-4 font-medium text-fg-secondary">发布人</th>
              <th class="py-2 pr-4 font-medium text-fg-secondary">备注</th>
            </tr>
          </thead>
          <tbody>
            {#each versions as v (v.id)}
              <tr class="border-b border-gray-100 align-top dark:border-gray-800">
                <td class="py-2 pr-4 text-fg-primary whitespace-nowrap">
                  v{v.version}
                  {#if v.isCurrent}
                    <span class="ml-1 rounded-sm bg-primary-100 px-1.5 py-0.5 text-xs text-primary-800 dark:bg-primary-900/40 dark:text-primary-200">
                      当前
                    </span>
                  {/if}
                </td>
                <td class="py-2 pr-4">
                  <span class="rounded-sm px-2 py-0.5 text-xs {statusClass(v.status)}">
                    {statusLabel(v.status)}
                  </span>
                </td>
                <td class="py-2 pr-4 text-fg-tertiary whitespace-nowrap">
                  {formatTime(v.publishedOn) || formatTime(v.createdOn)}
                </td>
                <td class="py-2 pr-4 text-fg-secondary whitespace-nowrap">
                  {v.publishedByUserEmail || "—"}
                </td>
                <td class="py-2 pr-4 text-fg-secondary">
                  {v.note || "—"}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
