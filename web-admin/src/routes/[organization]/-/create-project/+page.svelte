<script lang="ts">
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { listProjectsForOrgQueryOptions } from "@rilldata/web-admin/features/projects/list-projects-query-options";
  import { createQuery } from "@tanstack/svelte-query";
  import CreateProjectForm from "@rilldata/web-admin/features/projects/CreateProjectForm.svelte";
  import { getName } from "@rilldata/web-common/features/entity-management/name-utils.ts";
  import { EntityStatus } from "@rilldata/web-common/features/entity-management/types.ts";
  import Spinner from "@rilldata/web-common/features/entity-management/Spinner.svelte";
  import StarDataLogoWordmark from "@rilldata/web-common/components/icons/StarDataLogoWordmark.svelte";
  import {
    type DeployError,
    isQuotaDeployError,
  } from "@rilldata/web-common/features/project/deploy/deploy-errors.ts";
  import { Button } from "@rilldata/web-common/components/button";
  import CTAHeader from "@rilldata/web-common/components/calls-to-action/CTAHeader.svelte";

  let organization = $derived(page.params.organization);

  let projectsQuery = $derived(
    createQuery(listProjectsForOrgQueryOptions(organization)),
  );
  let hasProjects = $derived($projectsQuery.data?.projects?.length > 0);

  let defaultProjectName = $derived(
    getName(
      "new_project",
      $projectsQuery.data?.projects?.map((p) => p.name) ?? [],
    ),
  );

  let deployError: DeployError | undefined = $state(undefined);

  async function handleCreate(projectName: string) {
    return goto(`/${organization}/${projectName}`);
  }
</script>

<div class="background">
  <div class="flex flex-col items-center gap-4 mx-auto w-fit pt-48">
    {#if deployError && isQuotaDeployError(deployError)}
      <CTAHeader variant="bold">{deployError.title}</CTAHeader>
      <p class="text-base text-fg-secondary text-left w-[500px]">
        {deployError.message}
      </p>
      <Button type="secondary" noStroke href="/{organization}"
        >{m.common_back()}</Button
      >
    {:else}
      <StarDataLogoWordmark size="lg" />
      <div class="auth-title text-center">
        {hasProjects ? m.project_create_first() : m.project_create_new()}
      </div>

      <div class="flex flex-col gap-6 text-left auth-card">
        <div>
          <div class="auth-card__title">
            {hasProjects ? m.project_name_first() : m.project_name_new()}
          </div>
          <div class="auth-card__subtitle">
            {m.project_rename_anytime()}
          </div>
        </div>

        {#if $projectsQuery.isPending}
          <div class="h-36 w-[500px]">
            <Spinner status={EntityStatus.Running} size="2rem" duration={725} />
          </div>
        {:else}
          <CreateProjectForm
            {organization}
            defaultName={defaultProjectName}
            onCreate={handleCreate}
            onDeployError={(e) => (deployError = e)}
          />
        {/if}
      </div>
    {/if}
  </div>
</div>

<style lang="postcss">
  .background {
    @apply flex flex-col w-full h-fit min-h-screen bg-no-repeat bg-cover;
    background-image: url("/img/welcome-bg-art.jpg");
  }

  :global(.dark) .background {
    background-image: url("/img/welcome-bg-art-dark.jpg");
  }
</style>
