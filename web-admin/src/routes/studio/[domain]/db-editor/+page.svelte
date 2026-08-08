<script lang="ts">
  import { page } from "$app/stores";
  import StudioDBResourceList from "@rilldata/web-admin/features/studio-db/StudioDBResourceList.svelte";

  // Phase 5.3: full-coverage semantic resource list for DB-mode projects.
  $: organization = ($page.data?.organization as string | undefined) ?? "";
  $: project = $page.params.domain;
  $: editorBase = `/studio/${$page.params.domain}/db-editor`;
</script>

<StudioDBResourceList {organization} {project} {editorBase} />
