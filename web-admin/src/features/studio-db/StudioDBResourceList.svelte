<script lang="ts">
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import DelayedCircleOutlineSpinner from "@rilldata/web-common/components/spinner/DelayedCircleOutlineSpinner.svelte";
  import {
    createAdminServiceListSemanticResources,
    createAdminServiceSaveSemanticResource,
    createAdminServiceDeleteSemanticResource,
    createAdminServiceAcquireEditLock,
    getAdminServiceListSemanticResourcesQueryKey,
  } from "@rilldata/web-admin/client";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";

  let {
    organization,
    project,
    // Base path for the per-resource editor route.
    editorBase,
  }: {
    organization: string;
    project: string;
    editorBase: string;
  } = $props();

  // Phase 5.3: full-coverage resource list for DB-mode projects. Lists every
  // semantic resource kind, and supports create/delete. The per-kind editor is
  // the generic raw editor (StudioDBResourceEditor); a form-based editor per kind
  // is a later refinement.

  // The kinds a governor can create. Matches the server's validSemanticResourceKinds.
  const KINDS = [
    { value: "metrics_view", label: "语义视图 (metrics_view)" },
    { value: "model", label: "数据模型 (model)" },
    { value: "source", label: "数据源 (source)" },
    { value: "explore", label: "看板 (explore)" },
    { value: "canvas", label: "画布 (canvas)" },
    { value: "theme", label: "主题 (theme)" },
    { value: "api", label: "API (api)" },
    { value: "config", label: "项目配置 (config)" },
  ] as const;

  // Starter templates so a freshly-created resource parses cleanly instead of
  // tripping the dry-run gate on the first publish.
  const TEMPLATES: Record<string, (name: string) => string> = {
    metrics_view: (n) =>
      `type: metrics_view\ndisplay_name: ${n}\nmodel: CHANGE_ME\ntimeseries: CHANGE_ME\ndimensions: []\nmeasures:\n  - name: total\n    expression: COUNT(*)\n`,
    model: () => `SELECT 1 AS id\n`,
    source: (n) => `type: source\nconnector: duckdb\nsql: SELECT 1 AS id -- ${n}\n`,
    explore: (n) => `type: explore\ndisplay_name: ${n}\nmetrics_view: CHANGE_ME\n`,
    canvas: (n) => `type: canvas\ndisplay_name: ${n}\n`,
    theme: () => `type: theme\ncolors:\n  primary: hsl(220, 90%, 50%)\n`,
    api: (n) => `type: api\nname: ${n}\n`,
    config: () => `title: My Project\n`,
  };

  let listQuery = $derived(
    createAdminServiceListSemanticResources(organization, project),
  );
  let resources = $derived($listQuery.data?.resources ?? []);

  // Group by kind for a readable list.
  let grouped = $derived.by(() => {
    const m = new Map<string, typeof resources>();
    for (const r of resources) {
      const k = r.resourceKind ?? "";
      if (!m.has(k)) m.set(k, []);
      m.get(k)!.push(r);
    }
    return m;
  });

  const saveMutation = createAdminServiceSaveSemanticResource();
  const deleteMutation = createAdminServiceDeleteSemanticResource();
  const acquireLock = createAdminServiceAcquireEditLock();

  let showNew = $state(false);
  let newKind = $state<string>("metrics_view");
  let newName = $state("");
  let banner = $state<{ tone: "ok" | "err"; text: string } | null>(null);

  function errText(e: any): string {
    return e?.response?.data?.message ?? String(e);
  }

  async function refetch() {
    await queryClient.invalidateQueries({
      queryKey: getAdminServiceListSemanticResourcesQueryKey(organization, project),
    });
  }

  async function createResource() {
    banner = null;
    const name = newName.trim();
    if (!name) {
      banner = { tone: "err", text: "请填写资源名称。" };
      return;
    }
    try {
      // Creating a resource is a draft write, so it needs the editing lock like any
      // other save. Acquire it up front; if someone else holds it, tell the user.
      const lock = await $acquireLock.mutateAsync({ org: organization, project, data: {} });
      if (!lock.acquired) {
        banner = {
          tone: "err",
          text: `${lock.lock?.lockedByUserName || lock.lock?.lockedByUserEmail || "另一位用户"} 正在编辑，无法新建。`,
        };
        return;
      }

      const template = (TEMPLATES[newKind] ?? (() => ""))(name);
      await $saveMutation.mutateAsync({
        org: organization,
        project,
        data: {
          resourceKind: newKind,
          resourceName: name,
          definitionRaw: template,
          format: newKind === "model" ? "sql" : "",
        },
      });
      showNew = false;
      newName = "";
      banner = { tone: "ok", text: `已创建 ${newKind}/${name}。` };
      await refetch();
    } catch (e: any) {
      banner = { tone: "err", text: errText(e) };
    }
  }

  async function deleteResource(kind: string, name: string) {
    if (!confirm(`确定删除 ${kind}/${name} 吗？此操作会移除其全部草稿版本。`)) return;
    banner = null;
    try {
      await $deleteMutation.mutateAsync({
        org: organization,
        project,
        resourceKind: kind,
        resourceName: name,
      });
      banner = { tone: "ok", text: `已删除 ${kind}/${name}。` };
      await refetch();
    } catch (e: any) {
      banner = { tone: "err", text: errText(e) };
    }
  }

  function editHref(kind: string, name: string): string {
    return `${editorBase}/${kind}/${encodeURIComponent(name)}`;
  }
</script>

<div class="flex flex-col gap-4">
  <div class="flex items-center justify-between">
    <div>
      <h2 class="text-lg font-semibold text-fg-primary">语义资源</h2>
      <p class="mt-1 text-sm text-fg-tertiary">数据模型、语义视图、看板等定义，均存储在数据库中并受版本管理。</p>
    </div>
    <Button type="primary" onClick={() => (showNew = !showNew)}>新建资源</Button>
  </div>

  {#if showNew}
    <div class="flex flex-wrap items-end gap-3 rounded-sm border border-gray-200 p-4 dark:border-gray-700">
      <div class="flex flex-col gap-1">
        <label class="text-sm text-fg-secondary" for="new-kind">类型</label>
        <select
          id="new-kind"
          class="rounded-sm border border-gray-200 bg-surface-background px-2 py-1.5 text-sm text-fg-primary dark:border-gray-700"
          bind:value={newKind}
        >
          {#each KINDS as k (k.value)}
            <option value={k.value}>{k.label}</option>
          {/each}
        </select>
      </div>
      <div class="flex flex-col gap-1">
        <label class="text-sm text-fg-secondary" for="new-name">名称</label>
        <input
          id="new-name"
          class="rounded-sm border border-gray-200 bg-surface-background px-2 py-1.5 text-sm text-fg-primary dark:border-gray-700"
          bind:value={newName}
          placeholder="revenue_mv"
        />
      </div>
      <Button type="primary" onClick={createResource} loading={$saveMutation.isPending}>创建</Button>
      <Button type="secondary" onClick={() => (showNew = false)}>取消</Button>
    </div>
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

  {#if $listQuery.isLoading}
    <DelayedCircleOutlineSpinner isLoading={true} />
  {:else if resources.length === 0}
    <p class="text-sm text-fg-tertiary">还没有任何资源。点「新建资源」开始。</p>
  {:else}
    {#each [...grouped.entries()] as [kind, items] (kind)}
      <div class="flex flex-col gap-1">
        <h3 class="text-sm font-semibold text-fg-secondary">{kind}</h3>
        <ul class="divide-y divide-gray-100 rounded-sm border border-gray-200 dark:divide-gray-800 dark:border-gray-700">
          {#each items as r (r.id)}
            <li class="flex items-center justify-between px-3 py-2">
              <a
                href={editHref(r.resourceKind ?? "", r.resourceName ?? "")}
                class="text-sm text-accent-primary-action no-underline hover:underline"
              >
                {r.resourceName}
              </a>
              <div class="flex items-center gap-3">
                <span class="text-xs text-fg-tertiary">v{r.version}</span>
                <button
                  class="text-xs text-red-600 hover:underline dark:text-red-400"
                  onclick={() => deleteResource(r.resourceKind ?? "", r.resourceName ?? "")}
                >
                  删除
                </button>
              </div>
            </li>
          {/each}
        </ul>
      </div>
    {/each}
  {/if}
</div>
