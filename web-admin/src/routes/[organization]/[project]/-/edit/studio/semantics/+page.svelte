<script lang="ts">
  import StudioSemanticsPage from "@rilldata/web-common/features/studio/StudioSemanticsPage.svelte";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";
</script>

<StudioSemanticsPage semanticsBase={`${$editorRoutePrefix}/studio/semantics`} />
