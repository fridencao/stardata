// Self-hosted auth token storage (localStorage-backed).
//
// StarData issues stateless JWTs from its own /auth/login endpoint. The SPA
// stores the raw token here and the runtime clients (local-service + runtime
// v2) attach it as `Authorization: Bearer <token>` on every request.

const TOKEN_KEY = "stardata_token";

export function getStardataToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function setStardataToken(token: string): void {
  try {
    localStorage.setItem(TOKEN_KEY, token);
  } catch {
    /* storage unavailable (private mode / SSR) — ignore */
  }
}

export function clearStardataToken(): void {
  try {
    localStorage.removeItem(TOKEN_KEY);
  } catch {
    /* ignore */
  }
}

export interface StarDataClaims {
  id?: string;
  name?: string;
  email?: string;
  admin?: boolean;
  /** Page "spaces" the user may see: "business" and/or "tech". */
  spaces?: string[];
}

function base64UrlDecode(input: string): string {
  const b64 = input.replace(/-/g, "+").replace(/_/g, "/");
  const pad =
    b64.length % 4 === 0 ? "" : "=".repeat(4 - (b64.length % 4));
  const bin = atob(b64 + pad);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return new TextDecoder().decode(bytes);
}

// decodeStardataToken extracts the (unverified) JWT payload for display only.
// Signature is NOT checked — for showing the user's name/email in the UI.
//
// NOTE: StarData issues custom attributes inside the JWT `attr` claim
// (a nested object), NOT at the payload top level. So we read id/name/email/
// admin/spaces from `payload.attr`, falling back to the standard `sub` claim.
export function decodeStardataToken(
  token?: string | null,
): StarDataClaims | null {
  const t = token ?? getStardataToken();
  if (!t) return null;
  const parts = t.split(".");
  if (parts.length < 2) return null;
  try {
    const payload = JSON.parse(base64UrlDecode(parts[1])) as Record<string, any>;
    const attr = (payload.attr ?? {}) as Record<string, any>;
    const spaces = normalizeSpaces(attr.spaces);
    return {
      id: (attr.id as string) ?? (payload.sub as string) ?? undefined,
      name: attr.name as string | undefined,
      email: attr.email as string | undefined,
      admin: attr.admin === true,
      spaces,
    };
  } catch {
    return null;
  }
}

// normalizeSpaces coerces the JWT `spaces` claim (which may arrive as a
// string, string[], or undefined) into a clean string[].
function normalizeSpaces(v: unknown): string[] {
  if (v == null) return [];
  if (Array.isArray(v)) {
    return v.filter((x): x is string => typeof x === "string");
  }
  if (typeof v === "string") {
    return v
      .split(",")
      .map((s) => s.trim())
      .filter((s) => s.length > 0);
  }
  return [];
}
