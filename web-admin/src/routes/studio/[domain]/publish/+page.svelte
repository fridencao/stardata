<script lang="ts">
  import { page } from "$app/stores";
  import { createRequestsBackend } from "@rilldata/web-admin/features/data-requests/data-requests";
  import { createPublishBackend } from "@rilldata/web-admin/features/publishes/publishes";
  import StudioPublishPage from "@rilldata/web-common/features/studio/StudioPublishPage.svelte";
  import StudioDBPublishPage from "@rilldata/web-admin/features/studio-db/StudioDBPublishPage.svelte";
  import {
    branchPathPrefix,
    extractBranchFromPath,
  } from "@rilldata/web-admin/features/branches/branch-utils";
  import {
    createAdminServiceGetProject,
  } from "@rilldata/web-admin/client";

  $: domain = $page.params.domain;
  $: organization = ($page.data?.organization as string | undefined) ?? "";
  $: project = domain;
  $: branchPrefix = branchPathPrefix(extractBranchFromPath($page.url.pathname));
  $: studioBase = `/studio/${domain}${branchPrefix}`;

  // Decide which publish surface to show based on the project's semantic layer mode.
  $: projectQuery = createAdminServiceGetProject(organization, project);
  $: semanticLayerMode =
    $projectQuery.data?.project?.semanticLayerMode ?? "archive";

  // Archive-mode backends (unchanged).
  $: backend = createPublishBackend(organization, project);
  $: requestsBackend = createRequestsBackend(organization, project);
</script>

{#if semanticLayerMode === "db"}
  <StudioDBPublishPage {organization} {project} />
{:else}
  <StudioPublishPage
    {backend}
    {requestsBackend}
    semanticsBase={`${studioBase}/semantics`}
    requestsPageHref={`${studioBase}/requests`}
  />
{/if}
