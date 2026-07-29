<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { V1DeploymentStatus } from "@rilldata/web-admin/client";
  import {
    injectBranchIntoPath,
    requestSkipBranchInjection,
  } from "@rilldata/web-admin/features/branches/branch-utils";
  import { Button } from "@rilldata/web-common/components/button";
  import CtaContentContainer from "@rilldata/web-common/components/calls-to-action/CTAContentContainer.svelte";
  import CtaHeader from "@rilldata/web-common/components/calls-to-action/CTAHeader.svelte";
  import CtaLayoutContainer from "@rilldata/web-common/components/calls-to-action/CTALayoutContainer.svelte";
  import LoadingSpinner from "@rilldata/web-common/components/LoadingSpinner.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import EditBranchDialog from "./EditBranchDialog.svelte";
  import { useDevDeployments } from "./use-edit-session";

  /**
   * Gate for edit routes whose resolved deployment is not editable.
   *
   * Reaching an edit URL without an `@branch` segment resolves to the primary
   * prod deployment, whose JWT lacks `ReadRepo` — the SSE watcher then fails
   * with PermissionDenied and surfaces a 500 after the retry budget runs out.
   * Instead of mounting the editor against that deployment, this gate:
   * - auto-resumes the most recent editable dev session by rewriting the
   *   current URL onto its `@branch` (deep links keep their subpath), or
   * - offers to start a new edit session when none exists.
   */
  export let organization: string;
  export let project: string;
  /** The branch currently in the URL, if any. */
  export let activeBranch: string | undefined = undefined;
  /** The project's primary branch, used as the source for new branches. */
  export let primaryBranch: string | undefined = undefined;

  const devDeployments = useDevDeployments(organization, project);

  let dialogOpen = false;
  let redirecting = false;

  // Latest editable dev session; mirrors EditBranchDialog's resume list.
  $: candidates = ($devDeployments.data?.deployments ?? [])
    .filter(
      (d) =>
        d.editable &&
        d.branch &&
        d.status !== V1DeploymentStatus.DEPLOYMENT_STATUS_DELETING &&
        d.status !== V1DeploymentStatus.DEPLOYMENT_STATUS_DELETED,
    )
    .sort((a, b) => (b.updatedOn ?? "").localeCompare(a.updatedOn ?? ""));

  // Only auto-redirect from branchless URLs. When the URL already names a
  // (non-editable) branch, fall through to the CTA so the dialog can show
  // its read-only warning instead of silently switching branches.
  $: if (
    !activeBranch &&
    !$devDeployments.isPending &&
    candidates.length > 0 &&
    !redirecting
  ) {
    redirecting = true;
    const branch = candidates[0].branch!;
    requestSkipBranchInjection();
    void goto(
      injectBranchIntoPath($page.url.pathname, branch) + $page.url.search,
      { replaceState: true },
    );
  }
</script>

{#if $devDeployments.isPending || redirecting}
  <div class="flex flex-1 items-center justify-center">
    <LoadingSpinner />
  </div>
{:else}
  <CtaLayoutContainer>
    <CtaContentContainer>
      <CtaHeader variant="bold">{m.edit_no_active_session_title()}</CtaHeader>
      <p class="text-sm text-fg-secondary">
        {m.edit_no_active_session_body()}
      </p>
      <Button type="primary" onClick={() => (dialogOpen = true)}>
        {m.edit_start_editing()}
      </Button>
    </CtaContentContainer>
  </CtaLayoutContainer>
  <EditBranchDialog
    bind:open={dialogOpen}
    {organization}
    {project}
    {activeBranch}
    {primaryBranch}
  />
{/if}
