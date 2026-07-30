<!-- ORG SETTINGS -->

<script lang="ts">
  import type { Snippet } from "svelte";
  import { page } from "$app/stores";
  import LeftNav from "@rilldata/web-admin/components/nav/LeftNav.svelte";
  import ContentContainer from "@rilldata/web-common/components/layout/ContentContainer.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  let { children, data }: { children: Snippet; data: any } = $props();

  let organization = $derived($page.params.organization);
  let basePage = $derived(`/${organization}/-/settings`);
  let organizationPermissions = $derived(data.organizationPermissions);

  let navItems = $derived([
    { label: m.settings_nav_general(), route: "", hasPermission: true },
    { label: m.settings_nav_ai(), route: "/ai", hasPermission: true },
    {
      label: m.settings_nav_feature_access(),
      route: "/feature-access",
      hasPermission: organizationPermissions?.manageOrgMembers,
    },
  ]);
</script>

<ContentContainer title={m.settings_org_page_title()} maxWidth={1100}>
  <div class="container flex-col md:flex-row">
    <LeftNav
      {basePage}
      baseRoute="/[organization]/-/settings"
      {navItems}
      minWidth="180px"
    />
    <div class="flex flex-col gap-y-6 w-full">
      {@render children()}
    </div>
  </div>
</ContentContainer>

<style lang="postcss">
  .container {
    @apply flex pt-6 gap-6 max-w-full overflow-hidden;
  }
</style>
