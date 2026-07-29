<script lang="ts">
  import StudioOverviewPage from "@rilldata/web-common/features/studio/StudioOverviewPage.svelte";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";
</script>

<StudioOverviewPage studioBase={`${$editorRoutePrefix}/studio`} />
