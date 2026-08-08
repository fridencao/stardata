import { projectWelcomeStatus } from "@rilldata/web-admin/features/welcome/project/welcome-status.ts";
import { isProjectWelcomePage } from "@rilldata/web-admin/features/navigation/nav-utils.ts";
import { setRuntimeEditEnvironment } from "@rilldata/web-common/features/entity-management/edit-environment.ts";
import { redirect } from "@sveltejs/kit";
import { CreateProjectBranchName } from "@rilldata/web-admin/features/projects/publish-project.ts";

// Setting the environment here ensures the readonly check sees "cloud" at construction.
setRuntimeEditEnvironment("cloud");

export const load = async ({ parent, params: { organization, project }, route }) => {
  // StarData: the raw edit IDE (file tree / connectors / SQL editor) is a
  // technical surface. Business viewers must never reach it — fail closed.
  const { projectPermissions } = await parent();
  if (!projectPermissions?.manageProject) {
    throw redirect(307, `/${organization}/${project}/chat`);
  }

  if (
    projectWelcomeStatus.isProjectWelcomeStep(project) &&
    !isProjectWelcomePage({ route })
  ) {
    throw redirect(
      307,
      `/${organization}/${project}/@${CreateProjectBranchName}/-/edit/welcome`,
    );
  }
};
