import {
  adminServiceGetCurrentUser,
  adminServiceGetOrganizationNameForDomain,
  adminServiceListOrganizations,
  type V1Organization,
  type V1OrganizationPermissions,
} from "@rilldata/web-admin/client";
import {
  ADMIN_URL,
  CANONICAL_ADMIN_URL,
} from "@rilldata/web-admin/client/http-client";
import { getActiveOrgLocalStorageKey } from "@rilldata/web-admin/features/organizations/active-org/local-storage";
import { getFetchOrganizationQueryOptions } from "@rilldata/web-admin/features/organizations/selectors";
import { fetchProjectDeploymentDetails } from "@rilldata/web-admin/features/projects/selectors";
import { setRuntimeEditEnvironment } from "@rilldata/web-common/features/entity-management/edit-environment.ts";
import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient.js";
import { redirect } from "@sveltejs/kit";

// Setting the environment here ensures the readonly check sees "cloud" at construction.
setRuntimeEditEnvironment("cloud");

/**
 * Resolve the active organization for the current user. The `/studio/[domain]`
 * route intentionally omits the org segment ("org 段由默认值自动填充并对用户隐藏",
 * see design/phase4-enterprise-app.md §3.1), so we resolve it the same way the
 * home page's OrganizationRedirect does: custom domain → localStorage → first org.
 * SSR is disabled for web-admin, so `localStorage` is available in this loader.
 */
async function resolveActiveOrganization(): Promise<string | undefined> {
  // Scenario 1: running on a custom domain → resolve the org for that domain.
  if (ADMIN_URL !== CANONICAL_ADMIN_URL) {
    try {
      const res = await adminServiceGetOrganizationNameForDomain(
        window.location.hostname,
      );
      if (res.name) return res.name;
    } catch {
      // Fall through to the default behavior.
    }
  }

  // Scenario 2: user has an activeOrg in localStorage.
  try {
    const userId = (await adminServiceGetCurrentUser())?.user?.id;
    if (userId) {
      const activeOrg = localStorage.getItem(
        getActiveOrgLocalStorageKey(userId),
      );
      if (activeOrg) return activeOrg;
    }
  } catch {
    // Fall through to listing organizations.
  }

  // Scenario 3: fall back to the user's first organization.
  try {
    const orgs = (await adminServiceListOrganizations()).organizations;
    if (orgs && orgs.length > 0) return orgs[0].name;
  } catch {
    // No org resolvable.
  }

  return undefined;
}

export const load = async ({ params: { domain } }) => {
  const organization = await resolveActiveOrganization();
  // No resolvable org (not logged in / no membership): bounce to the home
  // redirector, which shows the welcome flow or picks an org.
  if (!organization) {
    throw redirect(307, "/");
  }

  const project = domain;

  // Fail closed: Studio is a technical governor surface. Business viewers must
  // never reach it, so verify `manageProject` before mounting the edit shell.
  const { projectPermissions } = await fetchProjectDeploymentDetails(
    organization,
    project,
    undefined,
  );
  if (!projectPermissions?.manageProject) {
    throw redirect(307, `/${organization}/${project}`);
  }

  // The root layout only resolves the org when the URL carries an
  // `[organization]` param, which this route intentionally does not. Fetch the
  // org here so the chrome (admin entry, plan name, logo) has what it needs.
  let organizationDetails: V1Organization | undefined;
  let organizationPermissions: V1OrganizationPermissions = {};
  let planDisplayName: string | undefined;
  try {
    const orgResp = await queryClient.fetchQuery(
      getFetchOrganizationQueryOptions(organization),
    );
    organizationDetails = orgResp?.organization;
    organizationPermissions = orgResp?.permissions ?? {};
    planDisplayName = orgResp?.organization?.billingPlanDisplayName;
  } catch {
    // Non-fatal: Studio itself only needs project-level permissions.
  }

  return {
    organization,
    project,
    projectPermissions,
    organizationDetails,
    organizationPermissions,
    planDisplayName,
  };
};
