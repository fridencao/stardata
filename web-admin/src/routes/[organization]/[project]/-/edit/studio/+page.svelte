<script lang="ts">
  import { page } from "$app/stores";
  import { createRequestsBackend } from "@rilldata/web-admin/features/data-requests/data-requests";
  import StudioOverviewPage from "@rilldata/web-common/features/studio/StudioOverviewPage.svelte";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";

  $: ({ organization, project } = $page.params);
  $: requestsBackend = createRequestsBackend(organization, project);
</script>

<StudioOverviewPage studioBase={`${$editorRoutePrefix}/studio`} {requestsBackend} />
