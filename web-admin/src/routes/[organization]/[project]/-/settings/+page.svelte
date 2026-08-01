<script lang="ts">
  import { page } from "$app/stores";
  import DangerZone from "@rilldata/web-admin/components/settings/DangerZone.svelte";
  import DeleteProject from "@rilldata/web-admin/features/projects/settings/DeleteProject.svelte";
  import HibernateProject from "@rilldata/web-admin/features/projects/settings/HibernateProject.svelte";
  import ProjectNameSettings from "@rilldata/web-admin/features/projects/settings/ProjectNameSettings.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  let organization = $derived($page.params.organization);
  let project = $derived($page.params.project);
</script>

<ProjectNameSettings {organization} {project} />

<div class="danger-zone-section">
  <h3 class="danger-zone-title">{m.settings_danger_zone_title()}</h3>
  <DangerZone>
    <!-- StarData: public-project visibility is a multi-tenant cloud feature; hidden in the enterprise deployment -->
    <HibernateProject {organization} {project} />
    <DeleteProject {organization} {project} />
  </DangerZone>
</div>

<style lang="postcss">
  .danger-zone-section {
    @apply flex flex-col gap-3;
  }

  .danger-zone-title {
    @apply text-lg font-semibold text-red-600;
  }
</style>
