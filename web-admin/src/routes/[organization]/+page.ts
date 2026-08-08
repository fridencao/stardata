import { listProjectsForOrgQueryOptions } from "@rilldata/web-admin/features/projects/list-projects-query-options";
import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
import { error, redirect } from "@sveltejs/kit";
import type { V1Project } from "@rilldata/web-admin/client";

export const load = async ({ params: { organization }, parent }) => {
  const { organizationPermissions } = await parent();

  if (!organizationPermissions.readOrg) {
    throw error(403, m.route_error_no_org_permission());
  }

  // StarData: a single-project org lands directly on the project's business
  // portal, so business users never see the technical project listing.
  let projects: V1Project[] = [];
  try {
    const resp = await queryClient.fetchQuery(
      listProjectsForOrgQueryOptions(organization),
    );
    projects = resp.projects ?? [];
  } catch {
    // Fall through to the org overview if listing fails
  }
  if (projects.length === 1 && projects[0].name) {
    throw redirect(307, `/${organization}/${projects[0].name}`);
  }
};
