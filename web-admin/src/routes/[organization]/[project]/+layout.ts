import { extractBranchFromPath, removeBranchFromPath } from "@rilldata/web-admin/features/branches/branch-utils.ts";
import { maybeRedirectToEditableDeployment } from "@rilldata/web-admin/features/branches/deployment-utils.ts";
import { isEditPage } from "@rilldata/web-admin/features/navigation/nav-utils.ts";
import { redirect } from "@sveltejs/kit";

export const load = async ({
  params: { organization, project },
  parent,
  route,
  url,
}) => {
  const { organizationPermissions } = await parent();

  if (!organizationPermissions.manageOrg) return;

  // Edit pages handle their own branch routing; everything below is non-edit only.
  if (isEditPage({ route })) return;

  // Branch deployments are only viewable from inside `/-/edit`. A stale link
  // or leftover in-session navigation that lands an `@branch` on a non-edit
  // page (settings, status, home …) drops the branch and shows production
  // instead of hard-failing with a 404.
  const branch = extractBranchFromPath(url.pathname);
  if (branch) {
    throw redirect(307, removeBranchFromPath(url.pathname) + url.search);
  }

  await maybeRedirectToEditableDeployment(organization, project, url);
};
