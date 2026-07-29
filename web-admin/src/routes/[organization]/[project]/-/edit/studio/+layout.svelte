<script lang="ts">
  // Studio shell (StarData): PortalNav + StudioTabs replace the technical
  // ProjectHeader inside the edit route group (see the edit layout's
  // inStudioPage branch). Links are branch-aware via editorRoutePrefix.
  import { page } from "$app/stores";
  import AvatarButton from "@rilldata/web-admin/features/authentication/AvatarButton.svelte";
  import PortalNav from "@rilldata/web-common/features/portal/PortalNav.svelte";
  import StudioTabs from "@rilldata/web-common/features/studio/StudioTabs.svelte";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";

  $: ({ organization, project } = $page.params);
  $: portalBase = `/${organization}/${project}`;
  // Set by the edit layout: "/[org]/[project](@branch)/-/edit"
  $: editBase = $editorRoutePrefix;

  // IDE tab: the edit workspace root; active on any edit page outside /studio
  $: ideActive = (path: string) =>
    path.startsWith(editBase) && !path.startsWith(`${editBase}/studio`);
</script>

<div class="flex h-full min-h-0 flex-1 flex-col bg-app-surface">
  <PortalNav
    brandHref={portalBase}
    portalHref={portalBase}
    adminHref={$page.data?.organizationPermissions?.manageOrg
      ? `/${organization}/-/settings`
      : null}
  >
    <svelte:fragment slot="user">
      <AvatarButton projectPermissions={$page.data?.projectPermissions} />
    </svelte:fragment>
  </PortalNav>
  <StudioTabs
    basePath={editBase}
    ideHref={editBase || "/"}
    {ideActive}
    statusHref={`${portalBase}/-/status`}
    settingsHref={`${portalBase}/-/settings`}
  />
  <main class="w-full flex-1 overflow-y-auto p-8 xl:max-w-6xl xl:mx-auto">
    <slot />
  </main>
</div>
