import { redirect } from "@sveltejs/kit";

export const load = async ({ parent, params: { organization, project }, url }) => {
  // StarData: the project home is role-adaptive. Business viewers land on the
  // business portal home (this page). Technical governors (manageProject) have
  // their own entry — the Studio workbench — so bounce them there instead of
  // showing the business-facing home. They can still reach chat/boards via the
  // portal nav (those are separate routes, unaffected by this redirect).
  //
  // `?preview` opts out of the bounce so a governor can inspect exactly what a
  // business user sees (the "Preview business view" entry in Studio links here).
  const { projectPermissions } = await parent();
  if (projectPermissions?.manageProject && !url.searchParams.has("preview")) {
    throw redirect(307, `/${organization}/${project}/-/edit/studio`);
  }
};
