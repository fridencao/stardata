<script lang="ts">
  import { page } from "$app/stores";
  import { createPublishBackend } from "@rilldata/web-admin/features/publishes/publishes";
  import StudioPublishPage from "@rilldata/web-common/features/studio/StudioPublishPage.svelte";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";

  $: ({ organization, project } = $page.params);
  $: backend = createPublishBackend(organization, project);
  $: editBase = $editorRoutePrefix;
</script>

<StudioPublishPage {backend} semanticsBase={`${editBase}/studio/semantics`} />
