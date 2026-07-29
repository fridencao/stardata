<script module lang="ts">
  export const CreateProjectFormId = "create-project-form";
</script>

<script lang="ts">
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import { page } from "$app/state";
  import {
    createAdminServiceCreateProject,
    getAdminServiceListProjectsForOrganizationQueryKey,
    type RpcStatus,
  } from "@rilldata/web-admin/client";
  import { Button } from "@rilldata/web-common/components/button";
  import Input from "@rilldata/web-common/components/forms/Input.svelte";
  import { defaults, superForm } from "sveltekit-superforms";
  import { yup } from "sveltekit-superforms/adapters";
  import { object, string } from "yup";
  import type { AxiosError } from "axios";
  import {
    type DeployError,
    getPrettyDeployError,
  } from "@rilldata/web-common/features/project/deploy/deploy-errors.ts";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient.ts";

  const {
    organization,
    defaultName = "new_project",
    onCreate,
    onDeployError,
  }: {
    organization: string;
    defaultName?: string;
    onCreate: (
      projectName: string,
      frontendUrl: string,
    ) => Promise<void> | void;
    onDeployError?: (deployError: DeployError) => void;
  } = $props();

  const schema = yup(
    object({
      name: string()
        .required(m.project_name_required())
        .matches(/^[a-zA-Z0-9][a-zA-Z0-9_-]*$/, m.project_name_format_error())
        .min(1, m.project_name_min_length())
        .max(40, m.project_name_max_length()),
    }),
  );

  const createProjectMutation = createAdminServiceCreateProject();

  // No need to be reactive to default name. It is derived from list of projects that wont change during the form creation.
  // svelte-ignore state_referenced_locally
  const { form, errors, enhance, submit, submitting } = superForm(
    defaults({ name: defaultName }, schema),
    {
      SPA: true,
      validators: schema,
      async onUpdate({ form }) {
        if (!form.valid) return;
        const project = form.data.name;

        // Create the project without a source (private deployments have no
        // GitHub integration; project files can be uploaded/connected later).
        const resp = await $createProjectMutation.mutateAsync({
          org: organization,
          data: {
            project,
            skipDeploy: true,
          },
        });
        void queryClient.invalidateQueries({
          queryKey:
            getAdminServiceListProjectsForOrganizationQueryKey(organization),
        });

        return onCreate(project, resp.project?.frontendUrl ?? "/");
      },
      onError({ result }) {
        const error =
          (result.error as AxiosError<RpcStatus>)?.response?.data?.message ??
          result.error.message;
        if (!error) return;
        // Mapping for backend error to a more user friendly UI error message.
        if (error.includes("a project with that name already exists")) {
          $errors["name"] = [
            `Project name '${$form.name}' is already taken. Please try a different name.`,
          ];
        } else {
          const deployError = getPrettyDeployError(new Error(error));
          if (deployError) onDeployError?.(deployError);
          $errors["name"] = [error];
        }
      },
      invalidateAll: false,
    },
  );
</script>

<form
  id={CreateProjectFormId}
  onsubmit={(e) => {
    e.preventDefault();
    submit(e);
  }}
  use:enhance
  class="flex flex-col gap-y-4"
>
  <Input
    bind:value={$form.name}
    errors={$errors?.name}
    id="name"
    label={m.common_name()}
    textClass="text-sm"
    alwaysShowError
    width="500px"
    size="xl"
    textInputPrefix="{page.url.origin}/{organization}/"
  />
  <div class="w-full flex justify-end">
    <Button
      type="primary"
      submitForm
      loading={$submitting}
      disabled={$submitting}
      onClick={submit}
    >
      {m.project_create_project()}
    </Button>
  </div>
</form>
