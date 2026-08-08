import { redirect } from "@sveltejs/kit";
import type { LayoutLoad } from "./$types";

export const load: LayoutLoad = async ({ parent, params }) => {
  const { organizationPermissions } = await parent();

  if (!organizationPermissions?.manageOrg) {
    throw redirect(307, `/${params.organization}`);
  }

  return {};
};
