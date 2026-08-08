<script lang="ts">
  import { page } from "$app/stores";
  import { createRequestsBackend } from "@rilldata/web-admin/features/data-requests/data-requests";
  import RequestsTodo from "@rilldata/web-common/features/studio/RequestsTodo.svelte";
  import SectionHeader from "@rilldata/web-common/features/studio/SectionHeader.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import {
    branchPathPrefix,
    extractBranchFromPath,
  } from "@rilldata/web-admin/features/branches/branch-utils";

  $: domain = $page.params.domain;
  $: organization = ($page.data?.organization as string | undefined) ?? "";
  $: project = domain;
  $: branchPrefix = branchPathPrefix(extractBranchFromPath($page.url.pathname));
  $: semanticsBase = `/studio/${domain}${branchPrefix}/semantics`;
  $: backend = createRequestsBackend(organization, project);
</script>

<svelte:head>
  <title>StarData · {m.studio_requests_page_title()}</title>
</svelte:head>

<SectionHeader
  title={m.studio_requests_page_title()}
  description={m.studio_requests_page_desc()}
/>

<RequestsTodo {backend} semanticsHref={semanticsBase} showHeading={false} />
