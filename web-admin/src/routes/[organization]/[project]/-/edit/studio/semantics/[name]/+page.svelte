<script lang="ts">
  import { page } from "$app/stores";
  import StudioSemanticEditorPage from "@rilldata/web-common/features/studio/StudioSemanticEditorPage.svelte";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";

  $: editBase = $editorRoutePrefix;
  $: name = $page.params.name;
</script>

{#key name}
  <StudioSemanticEditorPage
    {name}
    semanticsBase={`${editBase}/studio/semantics`}
    ideFileHref={(filePath) => `${editBase}/files${filePath}`}
  />
{/key}
