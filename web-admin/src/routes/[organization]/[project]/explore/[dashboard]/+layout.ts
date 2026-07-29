import { redirect } from "@sveltejs/kit";

// StarData: the metrics explorer is a technical surface that exposes
// measures/dimensions. Business viewers must use the portal Boards page
// (`/boards`) instead. Fail closed.
export const load = async ({ parent, params: { organization, project } }) => {
  const { projectPermissions } = await parent();
  if (!projectPermissions?.manageProject) {
    throw redirect(307, `/${organization}/${project}/boards`);
  }
};
