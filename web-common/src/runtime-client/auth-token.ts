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
