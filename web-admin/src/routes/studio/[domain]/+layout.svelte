<script lang="ts">
  import { page } from "$app/stores";
  import {
    createAdminServiceGetProject,
    getAdminServiceGetProjectQueryKey,
    V1DeploymentStatus,
    type V1Organization,
  } from "@rilldata/web-admin/client";
  import AvatarButton from "@rilldata/web-admin/features/authentication/AvatarButton.svelte";
  import EditSessionLoading from "@rilldata/web-admin/features/edit-session/EditSessionLoading.svelte";
  import EditSessionTimeoutBanner from "@rilldata/web-admin/features/edit-session/EditSessionTimeoutBanner.svelte";
  import BranchDeploymentStopped from "@rilldata/web-admin/features/branches/BranchDeploymentStopped.svelte";
  import { baseGetProjectQueryOptions } from "@rilldata/web-admin/features/projects/project-query-options";
  import SlimProjectHeader from "@rilldata/web-admin/features/projects/SlimProjectHeader.svelte";
  import { getThemedLogoUrl } from "@rilldata/web-admin/features/themes/organization-logo";
  import ErrorPage from "@rilldata/web-common/components/ErrorPage.svelte";
  import FileAndResourceWatcher from "@rilldata/web-common/features/entity-management/FileAndResourceWatcher.svelte";
  import { themeControl } from "@rilldata/web-common/features/themes/theme-control";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";
  import PortalNav from "@rilldata/web-common/features/portal/PortalNav.svelte";
  import StudioTabs from "@rilldata/web-common/features/studio/StudioTabs.svelte";
  import RuntimeProvider from "@rilldata/web-common/runtime-client/v2/RuntimeProvider.svelte";
  import { useQueryClient } from "@tanstack/svelte-query";
  import { onDestroy } from "svelte";
  import { setCloudReadonlyNotice } from "@rilldata/web-common/features/entity-management/actions/protected-files.ts";
  import { InfoIcon } from "lucide-svelte";
  import { overlay } from "@rilldata/web-common/layout/overlay-store";
  import BlockingOverlayContainer from "@rilldata/web-common/layout/BlockingOverlayContainer.svelte";
  import { fileArtifacts } from "@rilldata/web-common/features/entity-management/file-artifacts.ts";
  import {
    branchPathPrefix,
    extractBranchFromPath,
  } from "@rilldata/web-admin/features/branches/branch-utils";
  import EditSessionGate from "@rilldata/web-admin/features/edit-session/EditSessionGate.svelte";

  // StarData Studio shell (top-level route, see design/phase4-enterprise-app.md
  // §3.1). This combines what used to be split across two layouts:
  //   - `[org]/[project]/-/edit/+layout.svelte`  → the edit session (runtime
  //     provider, dev deployment lifecycle, file/resource watcher)
  //   - `[org]/[project]/-/edit/studio/+layout.svelte` → the Studio chrome
  //     (PortalNav + StudioTabs)
  // The org segment is resolved in `+layout.ts` and hidden from the URL.
  $: domain = $page.params.domain;
  $: project = domain;
  $: organization = ($page.data?.organization as string | undefined) ?? "";

  // A dev session may pin an `@branch` segment onto the studio path
  // (`/studio/[domain]/@branch/...`). `hooks.ts`'s reroute strips it before
  // route matching, so it is only visible on `$page.url`.
  $: branch = extractBranchFromPath($page.url.pathname);
  $: branchPrefix = branchPathPrefix(branch);

  // Studio section root: every Studio tab hangs directly off this.
  $: studioBase = `/studio/${domain}${branchPrefix}`;
  // Advanced mode (IDE) root. `editorRoutePrefix` drives `getFileHref`,
  // `navigateToFile`, etc., so it must point at the IDE, not at Studio.
  $: ideBase = `${studioBase}/ide`;
  $: editorRoutePrefix.set(ideBase);
  $: ideActive = (path: string) => path.startsWith(ideBase);

  // Business portal base, for the "back to portal" and preview entries.
  $: portalBase = `/${organization}/${project}`;

  // Root layout data: org permissions, plan display name, organization object
  $: pageData = $page.data;
  $: organizationPermissions = pageData?.organizationPermissions ?? {};
  $: planDisplayName = pageData?.planDisplayName;
  $: organizationLogoUrl = getThemedLogoUrl(
    $themeControl,
    pageData?.organizationDetails as V1Organization | undefined,
  );

  // Polling and JWT-refresh cadence are governed by `baseGetProjectQueryOptions`,
  // shared with the project layout so both observers stay in sync.
  $: projectQuery = createAdminServiceGetProject(
    organization,
    project,
    branch ? { branch } : undefined,
    { query: { ...baseGetProjectQueryOptions, enabled: !!organization } },
  );
  $: projectPermissions = $projectQuery.data?.projectPermissions ?? {};
  $: primaryBranch = $projectQuery.data?.project?.primaryBranch;
  $: devTtlSeconds = $projectQuery.data?.project?.devTtlSeconds;

  $: primaryProjectQuery = createAdminServiceGetProject(
    organization,
    project,
    undefined,
    { query: { ...baseGetProjectQueryOptions, enabled: !!organization } },
  );
  $: hasPrimaryDeployment =
    !!$primaryProjectQuery.data?.project?.primaryDeploymentId;

  // Deployment data and credentials come from GetProject (no separate API needed)
  $: deployment = $projectQuery.data?.deployment;
  $: deploymentStatus = deployment?.status;
  $: runtimeHost = deployment?.runtimeHost ?? null;
  $: instanceId = deployment?.runtimeInstanceId ?? null;
  $: jwt = $projectQuery.data?.jwt ?? null;

  // Flipped when the user clicks "Start deployment" on a stopped deployment;
  // keeps the UI in loading state while the backend transitions STOPPED → PENDING → RUNNING.
  let starting = false;

  // Wait for `primaryProjectQuery` too: the cloud readonly notice is gated on
  // `hasPrimaryDeployment`, and it must be registered before `<slot />` renders
  // any file editor (getReadonlyNotice reads the notice non-reactively).
  $: isLoading =
    $projectQuery.isPending ||
    $primaryProjectQuery.isPending ||
    starting ||
    deploymentStatus === V1DeploymentStatus.DEPLOYMENT_STATUS_PENDING;

  $: isErrored =
    deploymentStatus === V1DeploymentStatus.DEPLOYMENT_STATUS_ERRORED;

  $: isStopped =
    !starting &&
    (deploymentStatus === V1DeploymentStatus.DEPLOYMENT_STATUS_STOPPED ||
      deploymentStatus === V1DeploymentStatus.DEPLOYMENT_STATUS_STOPPING);

  $: isReady =
    (deploymentStatus === V1DeploymentStatus.DEPLOYMENT_STATUS_RUNNING ||
      deploymentStatus === V1DeploymentStatus.DEPLOYMENT_STATUS_UPDATING) &&
    runtimeHost !== null &&
    instanceId !== null &&
    jwt !== null;

  // A non-editable deployment (e.g. the primary prod deployment resolved from
  // a branchless studio URL) mints a JWT without ReadRepo, so mounting the
  // editor would only surface SSE PermissionDenied errors as a 500. Gate it —
  // `EditSessionGate` resumes the latest dev session by pinning its `@branch`.
  //
  // StarData Phase 5: DB-mode projects have no dev deployment to resume. Their
  // draft lives in semantic_resources and is edited through the admin API, not
  // through the runtime's repo store, so the prod deployment being non-editable
  // is the expected steady state rather than a condition to recover from.
  $: isDBSemanticLayer =
    ($projectQuery.data?.project?.semanticLayerMode ?? "archive") === "db";
  $: isNotEditable = !isDBSemanticLayer && !!deployment && !deployment.editable;

  // Invalidating this query refetches a fresh JWT; `runtimeClient.getJwt()`
  // reads the updated value on the next call. Branch must be part of the
  // key or the invalidation misses the branch-scoped cache entry.
  const queryClient = useQueryClient();
  $: onBeforeReconnect = async () => {
    await queryClient.invalidateQueries({
      queryKey: getAdminServiceGetProjectQueryKey(
        organization,
        project,
        branch ? { branch } : undefined,
      ),
    });
  };

  // Only surface the env notice once the project has a primary deployment.
  // Fail closed: env becomes editable only once we've positively confirmed the
  // project has no primary deployment. A failed or otherwise inconclusive lookup
  // keeps the notice set, so a published project never exposes editable `.env`
  // files while the deployment state is unknown.
  $: if (!$primaryProjectQuery.isPending) {
    const envEditable = $primaryProjectQuery.isSuccess && !hasPrimaryDeployment;
    setCloudReadonlyNotice(envEditable ? undefined : envEditDisabled);
    fileArtifacts.recheckReadonlyStatus();
  }

  onDestroy(() => {
    $editorRoutePrefix = "";
  });
</script>

<div class="edit-session">
  {#if isLoading}
    <EditSessionLoading status={deploymentStatus} href={`/${organization}`} />
  {:else if isErrored}
    <SlimProjectHeader
      {organization}
      {project}
      readProjects={organizationPermissions?.readProjects}
      {planDisplayName}
      {organizationLogoUrl}
    />
    <ErrorPage
      statusCode={500}
      header="Edit session failed"
      body={deployment?.statusMessage ||
        "The editing environment encountered an error. Please try again."}
    />
  {:else if isNotEditable}
    <SlimProjectHeader
      {organization}
      {project}
      readProjects={organizationPermissions?.readProjects}
      {planDisplayName}
      {organizationLogoUrl}
    />
    <EditSessionGate
      {organization}
      {project}
      activeBranch={branch}
      {primaryBranch}
    />
  {:else if isStopped && deployment?.id}
    <SlimProjectHeader
      {organization}
      {project}
      readProjects={organizationPermissions?.readProjects}
      {planDisplayName}
      {organizationLogoUrl}
    />
    <BranchDeploymentStopped
      {organization}
      {project}
      deploymentId={deployment.id}
      status={deploymentStatus}
      canManage={!!projectPermissions?.manageDev}
      {branch}
      bind:starting
    />
  {:else if isReady && deployment?.id && instanceId && runtimeHost && jwt}
    {#key `${runtimeHost}::${instanceId}::${hasPrimaryDeployment}`}
      <RuntimeProvider host={runtimeHost} {instanceId} {jwt}>
        <div class="flex h-full min-h-0 flex-1 flex-col bg-app-surface">
          <PortalNav
            brandHref={portalBase}
            portalHref={portalBase}
            adminHref={organizationPermissions?.manageOrg
              ? `/${organization}/-/settings`
              : null}
          >
            <svelte:fragment slot="user">
              <AvatarButton {projectPermissions} />
            </svelte:fragment>
          </PortalNav>
          <StudioTabs
            {studioBase}
            ideHref={ideBase}
            {ideActive}
            statusHref={`${portalBase}/-/status`}
            previewHref={`${portalBase}?preview=1`}
          />
          {#if !isDBSemanticLayer}
            <EditSessionTimeoutBanner
              usedOn={deployment.usedOn}
              {devTtlSeconds}
            />
          {/if}
          <FileAndResourceWatcher
            lifecycle="none"
            {onBeforeReconnect}
            errorBody="Lost connection to the editing environment. Try ending the session and starting a new one."
          >
            <main class="w-full flex-1 overflow-y-auto p-8 xl:max-w-6xl xl:mx-auto">
              <slot />
            </main>
          </FileAndResourceWatcher>
        </div>
      </RuntimeProvider>
    {/key}
  {:else}
    <SlimProjectHeader
      {organization}
      {project}
      readProjects={organizationPermissions?.readProjects}
      {planDisplayName}
      {organizationLogoUrl}
    />
    <ErrorPage
      statusCode={404}
      header="No active edit session"
      body="This editing session is no longer active. Use the Edit button to start a new one."
    />
  {/if}
</div>

{#if $overlay !== null}
  <BlockingOverlayContainer
    bg="linear-gradient(to right, rgba(0,0,0,.6), rgba(0,0,0,.8))"
  >
    <div slot="title" class="font-bold">
      {$overlay?.title}
    </div>
    <svelte:fragment slot="detail">
      {#if $overlay?.detail}
        <svelte:component
          this={$overlay.detail.component}
          {...$overlay.detail.props}
        />
      {/if}
    </svelte:fragment>
  </BlockingOverlayContainer>
{/if}

{#snippet envEditDisabled()}
  <div class="flex flex-row gap-2 items-center w-fit text-sm">
    <InfoIcon size={14} /> Manage environment variables in
    <a
      href="/{organization}/{project}/-/settings/environment-variables"
      target="_blank"
      rel="noopener"
    >
      Settings →
    </a>
  </div>
{/snippet}

<style lang="postcss">
  .edit-session {
    @apply flex flex-col h-full;
  }
</style>
