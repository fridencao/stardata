# Phase 1: 私有化部署底座 — 实施计划

> **For agentic workers:** REQUIRED: Use subagent-driven-development to implement. Steps use checkbox syntax.

**Goal:** Complete all Phase 1 items to reach "私有化部署就绪" milestone

**Current Status:** Phase 1.2 (部署配置) done, Phase 1.3 (品牌) done, Phase 1.1 (认证) not started in this plan, Phase 1.4 (多租户) deferred

---

## M1 (Dual Portal 双门户骨架) — Completed ✅

> **Plan file**: `docs/superpowers/plans/2026-07-26-dual-portal-m1-skeleton.md`
> **Branch**: `feature/dual-portal-m1` | **Status**: All 11 tasks done, build green, E2E fixture adapted

### Chunk 1: 路由基座与业务门户
| Task | Step | Status | Commit |
|------|------|--------|--------|
| Task 1: 建分支 | git checkout -b feature/dual-portal-m1 | ✅ | — |
| Task 2: IDE 首页移至 /files | 创建 files/+page.svelte，删除旧 +page.svelte | ✅ | 6e7831049 |
| Task 3: 路由常量重定义 | 重写 route-constants.ts，更新 +layout.ts | ✅ | e4b6e9d82 |
| Task 4: 角色 store + PortalNav | 创建 portal-role-store.ts + PortalNav.svelte | ✅ | e5c54a1c0 |
| Task 5: 业务门户壳+首页 | 创建 (portal)/+layout.svelte + +page.svelte | ✅ | 9970fb9c7 |
| Task 6: /ai → /chat 迁移+301 | 移动目录，更新路径引用，创建重定向桩 | ✅ | befac2531 + c04069d50 |
| Task 7: /boards 页+ /dashboards 301 | 创建 boards/+page.svelte，更新所有路由链接 | ✅ | c3d60c02f |

### Chunk 2: Studio 壳、根布局收口与验证
| Task | Step | Status | Commit |
|------|------|--------|--------|
| Task 8: Studio 壳 | 创建 StudioSidebar + studio/* 占位页 | ✅ | c06ec3b64 |
| Task 9: 根布局收口 | 隐藏旧 ApplicationHeader，删除 PreviewModeNav | ✅ | 5092781ea |
| Task 10: 回归验证 | npm run check ✅, npm run build ✅, E2E fixture 适配 (`/files` 入口, `../stardata` 二进制) | ✅ | 51303bd5b |
| Task 11: 收尾 | 分支待审阅合入 | ⏳ | — |

### Markers
- ✅ `/` 显示业务门户首页 + 导航
- ✅ `/files` 显示原 IDE
- ✅ `/chat` 可用且旧 `/ai` 301
- ✅ `/boards` 看板页正常且旧 `/dashboards` 301
- ✅ `/studio` 可访问含侧边栏和占位页
- ✅ 角色切换器 localStorage 持久化
- ✅ 旧路由 (`/ai`, `/dashboards`) 均 301 到位
- ✅ `npm run check -w web-local` → 0 errors
- ✅ `npm run build -w web-local` → 成功
- ✅ E2E fixture 已适配 `await page.goto(…/files)` 和 `spawn("../stardata")`

**Tech Stack:** Go + Svelte + TypeScript + Docker Compose

---

## Scope Check

Phase 1 is decomposed into 3 independent subsystems — each can be worked in parallel:
1. **1.1 Auth system** — entirely new Go package + web UI
2. **1.2 Deployment cleanup** — fix existing deploy code + rebrand CLI init
3. **1.3 Branding** — replace remaining "Rill" text + login page + empty states

Phase 1.4 (multi-tenant) is intentionally deferred — out of scope.

---

## File Structure

### Files to Create
```
runtime/security/
├── auth.go              # JWT sign/verify (RS256, ES256, HS256)
├── oidc.go              # OIDC authorization code flow
├── middleware.go         # gRPC interceptor + HTTP middleware
├── store.go             # Local user store (bcrypt passwords)
├── token.go             # API token management
└── server.go            # Login/logout/token refresh API handlers

web-local/src/routes/login/
├── +page.svelte         # Login page
├── +page.ts             # Login page load logic
└── login.css            # Login page styles

web-local/src/routes/auth/
├── callback/
│   └── +page.svelte     # OIDC callback handler
└── token/
    └── +page.svelte     # Token management UI

web-common/src/theme/
├── tokens.css           # Design tokens (colors, fonts, spacing)
└── stardata-theme.ts    # Theme constants
```

### Files to Modify
```
cli/cmd/initialize/init.go         # Rebrand "Rill" → "StarData" in user strings
cli/cmd/deploy/deploy.go           # Remove "rill cloud" references (local deploy only)
cli/cmd/deploy/deploy_test.go      # Fix test references to "rill"
cli/cmd/auth/                      # Auth commands → local auth
runtime/security.go                # Add Authenticator interface, integrate SecurityClaims
web-local/src/routes/+layout.svelte # Add login guard, check auth state
web-local/src/lib/time-ranges-test.ts # Rename RillGrain type
web-common/src/features/dashboards/time-controls/... # Empty states
```

---

## Implementation Tasks

### Task 1.1: Auth System — Go Backend (JWT + OIDC + Local)

**Files:**
- Create: `runtime/security/auth.go`
- Create: `runtime/security/oidc.go`
- Create: `runtime/security/middleware.go`
- Create: `runtime/security/store.go`
- Create: `runtime/security/token.go`
- Create: `runtime/security/server.go`
- Modify: `runtime/security.go` (add Authenticator interface)

**Steps:**

- [ ] **Step 1.1.1: Define the Authenticator interface in runtime/security.go**

```go
// Authenticator handles authentication and token verification.
type Authenticator interface {
    // Authenticate validates credentials and returns claims.
    Authenticate(ctx context.Context, provider string, creds map[string]string) (*SecurityClaims, error)
    // ValidateToken validates a JWT token and returns claims.
    ValidateToken(ctx context.Context, tokenString string) (*SecurityClaims, error)
    // GenerateToken generates a JWT token for the given claims.
    GenerateToken(ctx context.Context, claims *SecurityClaims) (string, error)
    // SupportedProviders returns the list of supported auth providers.
    SupportedProviders() []string
}
```

- [ ] **Step 1.1.2: Implement JWT auth (`runtime/security/auth.go`)**
  - JWT signing/verification supporting HS256, RS256, ES256
  - Configurable via `jwt_secret` (HMAC) or `jwt_private_key`/`jwt_public_key` (RSA/ECDSA)
  - Token expiry (default 24h), refresh token support
  - Claims extraction (user_id, email, name, roles)

- [ ] **Step 1.1.3: Implement OIDC auth (`runtime/security/oidc.go`)**
  - Authorization code flow (redirect → callback → token exchange)
  - Discovery URL parsing (`.well-known/openid-configuration`)
  - ID token validation (issuer, audience, expiry, nonce)
  - User info endpoint mapping (sub → user_id, email, name)

- [ ] **Step 1.1.4: Implement local user store (`runtime/security/store.go`)**
  - In-memory user store with bcrypt password hashing
  - Support loading from config file (`local_users`)
  - CRUD for local users (admin API)

- [ ] **Step 1.1.5: Implement API token management (`runtime/security/token.go`)**
  - Long-lived API token generation (SHA-256 hashed, stored in catalog)
  - Token scopes/restrictions
  - Token CRUD via admin API

- [ ] **Step 1.1.6: Implement gRPC + HTTP middleware (`runtime/security/middleware.go`)**
  - gRPC unary interceptor: extract token from metadata, validate, inject into context
  - HTTP middleware: extract token from `Authorization: Bearer` header or cookie
  - Skip auth for public endpoints (`/login`, `/auth/callback`, health check)

- [ ] **Step 1.1.7: Implement login/logout/token API handlers (`runtime/security/server.go`)**
  - `POST /auth/login` — local username/password → JWT
  - `GET /auth/oidc/login` — redirect to OIDC provider
  - `GET /auth/oidc/callback` — OIDC callback → JWT
  - `POST /auth/refresh` — refresh token
  - `POST /auth/logout` — invalidate token
  - `GET /auth/me` — current user info

- [ ] **Step 1.1.8: Wire auth into runtime server startup**
  - Read auth config from config.yaml
  - Initialize Authenticator based on `auth.provider`
  - Register gRPC interceptor
  - Register HTTP handlers

- [ ] **Step 1.1.9: Create `runtime/security/security_test.go` with unit tests**
  - JWT token roundtrip
  - OIDC callback validation
  - Local user authentication
  - Middleware extraction

---

### Task 1.2: Deployment Cleanup

**Files:**
- Modify: `cli/cmd/initialize/init.go`

**Steps:**

- [ ] **Step 1.2.1: Rebrand CLI init command (`cli/cmd/initialize/init.go`)**
  - Change all `"Rill project"` → `"StarData project"`
  - Change `rill init` examples → `stardata init`
  - Change `rill start` examples → `stardata start`
  - Change `my-rill-project` → `my-stardata-project`
  - Change `HasRillProject` → check for `rill.yaml` or `stardata.yaml`

---

### Task 1.3: Frontend Branding

**Files:**
- Create: `web-common/src/theme/tokens.css`
- Modify: `web-local/src/routes/+layout.svelte` (add auth guard)
- Review & fix remaining "Rill" text references

**Steps:**

- [ ] **Step 1.3.1: Review and fix remaining "Rill" brand references**
  - `web-local/src/lib/time-ranges-test.ts` — rename `RillGrain` → `StarDataGrain`
  - `web-local/src/routes/+layout.svelte` — fix "Rill 'intake' service" comment
  - `web-common/src/features/workspaces/ParquetWorkspace.svelte` — fix "Rill" comment
  - `web-common/src/components/diff/Diff2HtmlView.svelte` — fix "Rill's diff" comment

- [ ] **Step 1.3.2: Verify StarData favicon exists**
  - Check `web-local/static/` for favicon.ico, apple-touch-icon.png
  - Create if missing

- [ ] **Step 1.3.3: Brand-text sweep — grep for "Rill" in user-facing strings**
  - Search for `"Rill"` (not in import paths, not in proto, not in enum values)
  - Replace with "StarData"
  - Focus on: `.svelte` files, `.ts` user-facing strings

- [ ] **Step 1.3.4: Empty states customization**
  - Review dashboard empty states in `web-common/src/features/dashboards/`
  - Replace any "Rill" or generic empty states with StarData branding
  - Add first-run hints (import data, create dashboard)

---

### Markers for Phase 1 Completion

- ✅ Local auth: login/logout/token refresh works end-to-end
- ✅ OIDC auth can be enabled via config
- ✅ API token management (create/revoke)
- ✅ CLI init command fully rebranded to StarData
- ✅ No "Rill" text remaining in user-facing code
- ✅ Auth-protected routes redirect to login
- ✅ `deploy/` configs are complete (done)

---

## Execution Plan

Recommended execution order (sequential dependencies):

```
Task 1.2 (Deploy cleanup) ──────┐
                                  ├── (independent, can be parallel)
Task 1.3 (Branding) ─────────────┤
                                  │
Task 1.1 (Auth system) ──────────┘  ← largest effort, most complex
```

- **Task 1.2** and **Task 1.3** can run in parallel (independent frontend vs CLI)
- **Task 1.1** is the biggest item — it's entirely new Go backend code + web UI
