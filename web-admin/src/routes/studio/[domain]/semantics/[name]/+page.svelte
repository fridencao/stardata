<script lang="ts">
  import { page } from "$app/stores";
  import StudioSemanticEditorPage from "@rilldata/web-common/features/studio/StudioSemanticEditorPage.svelte";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";
  import {
    branchPathPrefix,
    extractBranchFromPath,
  } from "@rilldata/web-admin/features/branches/branch-utils";

  $: domain = $page.params.domain;
  $: name = $page.params.name;
  $: branchPrefix = branchPathPrefix(extractBranchFromPath($page.url.pathname));
  $: semanticsBase = `/studio/${domain}${branchPrefix}/semantics`;
  $: ideBase = $editorRoutePrefix;
</script>

{#key name}
  <StudioSemanticEditorPage
    {name}
    {semanticsBase}
    ideFileHref={(filePath) => `${ideBase}/files${filePath}`}
  />
{/key}
