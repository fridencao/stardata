import { MetricsEventScreenName } from "@rilldata/web-common/metrics/service/MetricsTypes";
import type { Page } from "@sveltejs/kit";
import type { V1ProjectPermissions } from "@rilldata/web-admin/client";

// TODO: update all methods to use partial Page based on what is needed, so that it can be called in loader functions.

export type Space = "portal" | "studio" | "admin";

/**
 * Single source of truth: which spaces a user may access, derived from their
 * permissions. Replaces the inline `manageProject`/`manageOrg` ternaries
 * scattered across layouts.
 */
export function spacesForUser(perm: {
  readOrg?: boolean;
  manageProject?: boolean;
  manageOrg?: boolean;
}): Space[] {
  const spaces: Space[] = ["portal"]; // everyone lands in the business portal
  if (perm.manageProject) spaces.push("studio");
  if (perm.manageOrg) spaces.push("admin");
  return spaces;
}

export function isOrganizationPage(page: Page): boolean {
  return (
    page.route.id === "/[organization]" ||
    !!page.route?.id?.startsWith("/[organization]/-/users") ||
    !!page.route?.id?.startsWith("/[organization]/-/settings")
  );
}

export function withinOrganization({ route }: Pick<Page, "route">): boolean {
  return !!route?.id?.startsWith("/[organization]");
}

export function isProjectCreatePage(page: Page): boolean {
  return page.route.id === "/[organization]/-/create-project";
}

export function isProjectPage(page: Page): boolean {
  const routeId = page.route?.id;
  if (!routeId) return false;
  return (
    routeId === "/[organization]/[project]" ||
    (routeId.startsWith("/[organization]/[project]/-/") &&
      !routeId.startsWith("/[organization]/[project]/-/invite") &&
      !routeId.startsWith("/[organization]/[project]/-/share") &&
      !routeId.startsWith("/[organization]/[project]/-/edit"))
  );
}

export function withinProject(page: Page): boolean {
  return !!page.route?.id?.startsWith("/[organization]/[project]");
}

/**
 * Business-portal pages (StarData): project home, chat, boards, and the
 * reports/alerts list + detail views. These render the portal chrome
 * (PortalNav + PortalTabs) instead of the technical ProjectHeader + ProjectTabs,
 * so navigating into 报表/告警 keeps the user inside the business portal.
 * (Email deep-link sub-routes open/unsubscribe/export are intentionally
 * excluded — they are standalone CTA pages, not portal views.)
 */
export function isPortalPage(page: Page): boolean {
  const routeId = page.route?.id;
  if (!routeId) return false;
  return (
    routeId === "/[organization]/[project]" ||
    routeId.startsWith("/[organization]/[project]/chat") ||
    routeId.startsWith("/[organization]/[project]/boards") ||
    routeId === "/[organization]/[project]/-/reports" ||
    routeId === "/[organization]/[project]/-/reports/[report]" ||
    routeId === "/[organization]/[project]/-/alerts" ||
    routeId === "/[organization]/[project]/-/alerts/[alert]"
  );
}

export function isMetricsExplorerPage(page: Page): boolean {
  return (
    page.route.id === "/[organization]/[project]/explore/[dashboard]" ||
    page.route.id ===
      "/[organization]/[project]/-/share/[token]/explore/[dashboard]" ||
    page.route.id === "/-/embed"
  );
}

export function isCanvasDashboardPage(page: Page): boolean {
  return (
    page.route.id === "/[organization]/[project]/canvas/[dashboard]" ||
    page.route.id ===
      "/[organization]/[project]/-/share/[token]/canvas/[dashboard]"
  );
}

export function isPersonalFilePage(page: Page): boolean {
  return page.route.id === "/[organization]/[project]/-/personal/[name]";
}

/**
 * Returns true if the page is any kind of dashboard page (either a Metrics Explorer or a Custom Dashboard).
 */
export function isAnyDashboardPage(page: Page): boolean {
  return isMetricsExplorerPage(page) || isCanvasDashboardPage(page);
}

export function isReportPage(page: Page): boolean {
  return page.route.id === "/[organization]/[project]/-/reports/[report]";
}

export function isAlertPage(page: Page): boolean {
  return page.route.id === "/[organization]/[project]/-/alerts/[alert]";
}

export function isReportExportPage(page: Page): boolean {
  return (
    page.route.id ===
    "/[organization]/[project]/[dashboard]/-/reports/[report]/export"
  );
}

export function isPublicURLPage(page: Page): boolean {
  if (!page.route.id) return false;

  return (
    page.route.id.startsWith("/[organization]/[project]/-/share/[token]") ||
    isPublicReportPage(page) ||
    isPublicAlertPage(page)
  );
}

export function isPublicReportPage(page: Page): boolean {
  return (
    !!page.route.id?.startsWith(
      "/[organization]/[project]/-/reports/[report]",
    ) && page.url.searchParams.has("token")
  );
}

export function isPublicAlertPage(page: Page): boolean {
  return (
    !!page.route.id?.startsWith("/[organization]/[project]/-/alerts/[alert]") &&
    page.url.searchParams.has("token")
  );
}

export function isEditPage({ route }: Pick<Page, "route">): boolean {
  return (
    !!route?.id?.startsWith("/[organization]/[project]/-/edit") ||
    !!route?.id?.startsWith("/studio/[domain]")
  );
}

/**
 * Studio pages (StarData): the guided workbench (overview/sources/semantics/
 * publish) nested inside the edit route group. They render the portal-style
 * chrome (PortalNav + StudioTabs) instead of the technical ProjectHeader.
 */
export function isStudioPage({ route }: Pick<Page, "route">): boolean {
  return (
    !!route?.id?.startsWith("/[organization]/[project]/-/edit/studio") ||
    !!route?.id?.startsWith("/studio/[domain]")
  );
}

/**
 * True when the page is the explore or canvas preview inside Cloud Rill
 * Developer (`/-/edit/(viz)/{explore,canvas}/[name]`). `isMetricsExplorerPage`
 * and `isCanvasDashboardPage` only match production routes, so this is the
 * editor-side equivalent for surfaces that need to swap chat affordances.
 */
export function isEditDashboardPreviewPage({
  route,
}: Pick<Page, "route">): boolean {
  return !!route?.id?.startsWith("/[organization]/[project]/-/edit/(viz)/");
}

export function isProjectRequestAccessPage(page: Page): boolean {
  return !!page.route.id?.startsWith(
    "/[organization]/[project]/-/request-access",
  );
}

export function isProjectInvitePage(page: Page): boolean {
  return page.route.id === "/[organization]/[project]/-/invite";
}

export function isWelcomePage({ route }: Pick<Page, "route">): boolean {
  return !!route.id?.startsWith("/-/welcome");
}

export function isProjectWelcomePage({ route }: Pick<Page, "route">): boolean {
  return !!route.id?.startsWith("/[organization]/[project]/-/edit/welcome");
}

export function isAuthPage({ route }: Pick<Page, "route">): boolean {
  return !!route.id?.startsWith("/-/auth");
}

/**
 * Returns true if the page is a page that is part of the onboarding flow.
 * Project invite page, org/project welcome page, and project create page are all onboarding pages as of now.
 * @param page
 */
export function isOnboardingPage(page: Page): boolean {
  return (
    isProjectInvitePage(page) ||
    isWelcomePage(page) ||
    isProjectWelcomePage(page) ||
    isProjectCreatePage(page)
  );
}

export function getScreenNameFromPage(page: Page): MetricsEventScreenName {
  switch (true) {
    case isOrganizationPage(page):
      return MetricsEventScreenName.Organization;
    case isProjectPage(page):
      return MetricsEventScreenName.Project;
    case isMetricsExplorerPage(page):
      return MetricsEventScreenName.Dashboard;
    case isCanvasDashboardPage(page):
      return MetricsEventScreenName.Canvas;
    case isReportPage(page):
      return MetricsEventScreenName.Report;
    case isAlertPage(page):
      return MetricsEventScreenName.Alert;
    case isReportExportPage(page):
      return MetricsEventScreenName.ReportExport;
  }
  return MetricsEventScreenName.Unknown;
}
