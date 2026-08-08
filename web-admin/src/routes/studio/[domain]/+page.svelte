<script lang="ts">
  import { page } from "$app/stores";
  import { createRequestsBackend } from "@rilldata/web-admin/features/data-requests/data-requests";
  import StudioOverviewPage from "@rilldata/web-common/features/studio/StudioOverviewPage.svelte";
  import {
    branchPathPrefix,
    extractBranchFromPath,
  } from "@rilldata/web-admin/features/branches/branch-utils";

  $: domain = $page.params.domain;
  $: organization = ($page.data?.organization as string | undefined) ?? "";
  $: project = domain;
  $: branchPrefix = branchPathPrefix(extractBranchFromPath($page.url.pathname));
  $: studioBase = `/studio/${domain}${branchPrefix}`;
  $: requestsBackend = createRequestsBackend(organization, project);
</script>

<StudioOverviewPage {studioBase} {requestsBackend} />
