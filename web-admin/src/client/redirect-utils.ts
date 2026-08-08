import { ADMIN_URL } from "@rilldata/web-admin/client/http-client";
import { redirect } from "@sveltejs/kit";

/**
 * Redirects to the branded web-admin login page (styled like the welcome flow)
 * by throwing a SvelteKit redirect. Use this in SvelteKit load functions
 * (+page.ts, +layout.ts, etc.).
 *
 * @param redirectTo Optional path the user originally requested; carried as a
 *   `?redirect=` query param so the login page can send them back after auth.
 */
export function redirectToLogin(redirectTo?: string) {
  throw redirect(307, buildWebLoginUrl(redirectTo));
}

/**
 * Redirects straight to the identity provider (Keycloak) login using
 * window.location.href. Use this in Svelte component event handlers (onClick).
 */
export function redirectToLoginFromComponent() {
  window.location.href = buildKeycloakLoginUrl();
}

export function redirectToLogout() {
  window.location.href = buildLogoutUrl();
}

/**
 * Branded web-admin login page URL (e.g. /-/welcome/login?redirect=...).
 * This is what unauthenticated users are sent to; the page itself then has a
 * "Sign in" button that calls redirectToLoginFromComponent() -> Keycloak.
 */
function buildWebLoginUrl(redirectTo?: string) {
  const path = "/-/welcome/login";
  const target = new URL(window.location.origin + path);
  if (redirectTo && redirectTo !== path) {
    target.searchParams.set("redirect", redirectTo);
  }
  return target.pathname + target.search;
}

/**
 * Actual OIDC authorize URL on the admin/Keycloak side.
 */
function buildKeycloakLoginUrl() {
  const u = new URL(ADMIN_URL);
  u.pathname = appendPath(u.pathname, "auth/login");
  u.searchParams.set("redirect", window.location.href);
  return u.toString();
}

function buildLogoutUrl() {
  const u = new URL(ADMIN_URL);
  u.pathname = appendPath(u.pathname, "auth/logout");
  u.searchParams.set("redirect", buildWebLoginUrl());
  return u.toString();
}

function appendPath(path: string, suffix: string) {
  return `${path.replace(/\/$/, "")}/${suffix}`;
}
