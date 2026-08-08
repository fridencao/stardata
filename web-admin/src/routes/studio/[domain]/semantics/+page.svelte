<script lang="ts">
  import { page } from "$app/stores";
  import StudioSemanticsPage from "@rilldata/web-common/features/studio/StudioSemanticsPage.svelte";
  import {
    branchPathPrefix,
    extractBranchFromPath,
  } from "@rilldata/web-admin/features/branches/branch-utils";

  $: domain = $page.params.domain;
  $: branchPrefix = branchPathPrefix(extractBranchFromPath($page.url.pathname));
  $: semanticsBase = `/studio/${domain}${branchPrefix}/semantics`;
</script>

<StudioSemanticsPage {semanticsBase} />
