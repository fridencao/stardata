<script lang="ts">
  import StudioSourcesPage from "@rilldata/web-common/features/studio/StudioSourcesPage.svelte";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";
</script>

<StudioSourcesPage ideHref={$editorRoutePrefix || "/"} />
