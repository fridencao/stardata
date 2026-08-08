<script lang="ts">
  import { page } from "$app/stores";
  import DangerZone from "@rilldata/web-admin/components/danger-zone/DangerZone.svelte";
  import DeleteOrg from "@rilldata/web-admin/features/organizations/settings/DeleteOrg.svelte";
  import FaviconSettings from "@rilldata/web-admin/features/organizations/settings/FaviconSettings.svelte";
  import LogoSettings from "@rilldata/web-admin/features/organizations/settings/LogoSettings.svelte";
  import OrgNameSettings from "@rilldata/web-admin/features/organizations/settings/OrgNameSettings.svelte";
  import type { PageData } from "./$types";

  export let data: PageData;

  $: ({ organization: organizationObj } = data);

  $: ({
    logoUrl: organizationLogoUrl,
    logoDarkUrl: organizationLogoDarkUrl,
    faviconUrl: organizationFaviconUrl,
  } = organizationObj ?? {});

  $: organization = $page.params.organization;
</script>

<OrgNameSettings {organization} />
<LogoSettings {organization} {organizationLogoUrl} {organizationLogoDarkUrl} />
<FaviconSettings {organization} {organizationFaviconUrl} />

<DangerZone>
  <DeleteOrg {organization} />
</DangerZone>
