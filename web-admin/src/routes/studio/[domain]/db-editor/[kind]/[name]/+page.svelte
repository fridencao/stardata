<script lang="ts">
  import { page } from "$app/stores";
  import StudioDBResourceEditor from "@rilldata/web-admin/features/studio-db/StudioDBResourceEditor.svelte";

  // Phase 5.1 tracer-bullet route: edits a DB-mode project's semantic resource
  // through the admin API. Coexists with the archive-mode editor under
  // /studio/[domain]/semantics/[name], which still goes through the runtime's
  // file store.
  $: organization = $page.data?.organization as string;
  $: project = $page.params.domain;
  $: resourceKind = $page.params.kind;
  $: resourceName = $page.params.name;
</script>

<div class="flex h-full min-h-0 flex-col gap-4">
  <div>
    <h2 class="text-lg font-semibold text-fg-primary">语义定义（数据库版本）</h2>
    <p class="mt-1 text-sm text-fg-tertiary">
      直接编辑存储在数据库中的语义定义。保存时会做语法与引用校验；真实数据需要在发布预览中查看。
    </p>
  </div>

  {#key `${resourceKind}/${resourceName}`}
    <StudioDBResourceEditor
      {organization}
      {project}
      {resourceKind}
      {resourceName}
    />
  {/key}
</div>
