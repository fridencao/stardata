<script lang="ts">
  import { goto } from "$app/navigation";
  import { Button } from "@rilldata/web-common/components/button";
  import CreateNewOrgForm, {
    CreateNewOrgFormId,
  } from "@rilldata/web-common/features/organization/CreateNewOrgForm.svelte";
  import StarDataLogoWordmark from "@rilldata/web-common/components/icons/StarDataLogoWordmark.svelte";
  import {
    createAdminServiceCreateOrganization,
    getAdminServiceListOrganizationsQueryKey,
    type RpcStatus,
  } from "@rilldata/web-admin/client";
  import type { AxiosError } from "axios";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient.ts";

  const createOrgMutation = createAdminServiceCreateOrganization();
  $: ({ isPending, error } = $createOrgMutation);
  $: errorMessage = error
    ? ((error as unknown as AxiosError<RpcStatus>)?.response?.data?.message ??
      error.message)
    : undefined;

  async function createOrg(name: string, displayName: string) {
    await $createOrgMutation.mutateAsync({
      data: {
        name,
        displayName,
      },
    });

    await queryClient.invalidateQueries({
      queryKey: getAdminServiceListOrganizationsQueryKey(),
    });

    // This navigation gets cancelled if we do not have `setTimeout` here.
    setTimeout(() => void goto(`/${name}/-/create-project`));
  }
</script>

<div class="flex flex-col items-center gap-4 mx-auto w-fit">
  <StarDataLogoWordmark size="lg" />
  <div class="auth-title text-center">
    Create an organization
  </div>

  <div class="flex flex-col gap-6 text-left auth-card">
    <div>
      <div class="auth-card__title">Name your organization</div>
      <div class="auth-card__subtitle">
        You can change the name in organization settings.
      </div>
    </div>
    <CreateNewOrgForm {createOrg} size="xl" showUrlField={false} />
    {#if errorMessage}
      <div class="text-sm text-destructive">{errorMessage}</div>
    {/if}
    <div class="w-full flex justify-end">
      <Button
        type="primary"
        submitForm
        form={CreateNewOrgFormId}
        loading={isPending}
      >
        Continue
      </Button>
    </div>
  </div>
</div>
