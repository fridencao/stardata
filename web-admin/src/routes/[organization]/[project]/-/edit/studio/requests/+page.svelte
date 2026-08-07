<script lang="ts">
  import { page } from "$app/stores";
  import { createRequestsBackend } from "@rilldata/web-admin/features/data-requests/data-requests";
  import RequestsTodo from "@rilldata/web-common/features/studio/RequestsTodo.svelte";
  import SectionHeader from "@rilldata/web-common/features/studio/SectionHeader.svelte";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  $: ({ organization, project } = $page.params);
  $: backend = createRequestsBackend(organization, project);
  $: semanticsBase = `${$editorRoutePrefix}/studio/semantics`;
</script>

<svelte:head>
  <title>StarData · {m.studio_requests_page_title()}</title>
</svelte:head>

<SectionHeader
  title={m.studio_requests_page_title()}
  description={m.studio_requests_page_desc()}
/>

<RequestsTodo {backend} semanticsHref={semanticsBase} showHeading={false} />
