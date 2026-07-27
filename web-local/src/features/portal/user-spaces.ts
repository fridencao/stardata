import {
  decodeStardataToken,
  getStardataToken,
} from "@rilldata/web-common/runtime-client/auth-token";

export const ALL_SPACES = ["business", "tech"] as const;
export type Space = (typeof ALL_SPACES)[number];

/**
 * Returns the page "spaces" a user is allowed to see, derived from the
 * role carried in the JWT (`attr.spaces`). When there is no token (auth
 * disabled / Rill Developer local mode) we default to full visibility so the
 * app keeps working for developers.
 */
export function userSpaces(): string[] {
  const token = getStardataToken();
  if (!token) return [...ALL_SPACES];
  const claims = decodeStardataToken(token);
  if (!claims || !claims.spaces || claims.spaces.length === 0) {
    return [...ALL_SPACES];
  }
  return claims.spaces;
}

export function canViewBusiness(spaces?: string[]): boolean {
  return (spaces ?? userSpaces()).includes("business");
}

export function canViewTech(spaces?: string[]): boolean {
  return (spaces ?? userSpaces()).includes("tech");
}

/**
 * Default landing page for a user, based on the spaces they may see.
 * Business space wins (most users land on the portal home); otherwise the
 * technical workbench; falls back to "/" when nothing matches.
 */
export function defaultHome(spaces?: string[]): string {
  const sp = spaces ?? userSpaces();
  if (sp.includes("business")) return "/";
  if (sp.includes("tech")) return "/studio";
  return "/";
}
