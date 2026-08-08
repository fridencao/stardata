<script lang="ts">
  import { page } from "$app/stores";
  import { createRequestsBackend } from "@rilldata/web-admin/features/data-requests/data-requests";
  import { createPublishBackend } from "@rilldata/web-admin/features/publishes/publishes";
  import StudioPublishPage from "@rilldata/web-common/features/studio/StudioPublishPage.svelte";
  import {
    branchPathPrefix,
    extractBranchFromPath,
  } from "@rilldata/web-admin/features/branches/branch-utils";

  $: domain = $page.params.domain;
  $: organization = ($page.data?.organization as string | undefined) ?? "";
  $: project = domain;
  $: branchPrefix = branchPathPrefix(extractBranchFromPath($page.url.pathname));
  $: studioBase = `/studio/${domain}${branchPrefix}`;
  $: backend = createPublishBackend(organization, project);
  $: requestsBackend = createRequestsBackend(organization, project);
</script>

<StudioPublishPage
  {backend}
  {requestsBackend}
  semanticsBase={`${studioBase}/semantics`}
  requestsPageHref={`${studioBase}/requests`}
/>
