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
export function decodeStardataToken(
  token?: string | null,
): StarDataClaims | null {
  const t = token ?? getStardataToken();
  if (!t) return null;
  const parts = t.split(".");
  if (parts.length < 2) return null;
  try {
    return JSON.parse(base64UrlDecode(parts[1])) as StarDataClaims;
  } catch {
    return null;
  }
}
