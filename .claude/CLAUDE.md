# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is StarData

StarData is an intelligent BI platform for Chinese-speaking users, forked from [rilldata/rill](https://github.com/rilldata/rill) (Apache-2.0). It adds AI-powered conversational analytics, semantic layer definitions, and private deployment capabilities.

Core principles inherited from Rill:

- **Code-first**: Projects are configured with versioned YAML and SQL files — connectors, models, metrics views, dashboards, alerts, reports.
- **Parser → Resolver → Reconciler pipeline**: Project files are parsed into resources, resolved against dependencies, then reconciled to desired state (materialized into DuckDB/ClickHouse, etc.).
- **Declarative dashboarding**: Metrics views define dimensions and measures that power two dashboard types — **explore** (drill-down, slice-and-dice) and **canvas** (free-form charts and tables).

Two deployment modes share the same codebase:

- **Rill Developer** (local) — single Go binary with embedded CLI, runtime, and `web-local` frontend. Code-first workflow for data engineers.
- **Rill Cloud** (hosted) — separate `admin` service, runtime(s), and `web-admin` frontend with auth, billing, multi-tenancy, and collaboration.

### Key Directories

| Directory | Purpose |
|---|---|
| `runtime/` | Data plane: parser, resolvers, reconcilers, connector drivers, query engine, AI integration |
| `admin/` | Cloud control plane: auth, billing, provisioning, project management, jobs |
| `cli/` | CLI commands and local application server (embeds runtime + frontend) |
| `proto/` | gRPC/protobuf API definitions — source of truth for all APIs |
| `web-common/` | Shared frontend library (components, utilities, runtime client) consumed by both web apps |
| `web-local/` | Local frontend (Rill Developer) — SvelteKit app on port 3001 |
| `web-admin/` | Cloud frontend (Rill Cloud) — admin console |
| `docs/` | Documentation site |

### StarData-Specific Modifications

- **Portal feature**: Role-based portal experience (`web-local/src/features/portal/`) with publish gate support (`publish.yaml`, `requests.yaml`)
- **AI metrics**: AI-powered query generation and recommended questions from metrics views
- **DuckDB extension**: Custom DuckDB build with StarData extensions (via `scripts/embed_duckdb_ext/`)
- **Examples**: Cloned from `fridencao/stardata-examples` (subset: rill-openrtb-prog-ads, rill-github-analytics, rill-cost-monitoring)

## Development

### Build & Run

```bash
# Full build (Go + embed frontend examples + npm build + link dist)
make cli

# Go binary only (skip frontend build)
make cli-only

# Build just CLI (assumes frontend already built in cli/pkg/web/embed/dist/)
go build -o stardata cli/main.go

# Run the built binary against a project
./stardata start my-project

# Local development (runtime + web dev server in parallel)
npm run dev
# or
npm run dev-runtime &    # Go backend on default port
npm run dev-web -- --port 3001   # Frontend on 3001
```

### Tests

```bash
# All Go tests
go test ./...

# Go tests with coverage
make coverage.go

# Frontend unit tests (web-common, fast)
npm run test -w web-common

# Frontend unit tests (web-admin)
cd web-admin && npx vitest run

# Frontend E2E tests (Playwright, slow)
npm run test -w web-local
npm run test -w web-admin

# Single Playwright test
cd web-local && npx playwright test --headed --project=e2e-chrome -g "test name"
```

### Code Generation

```bash
# Regenerate API bindings from .proto files (OpenAPI, gRPC, frontend clients)
make proto.generate

# Regenerate CLI/docs from code
make docs.generate

# Generate i18n messages (Paraglide)
npm run build:i18n

# Generate Orval API client for web-common
npm run generate:runtime-client -w web-common

# Generate Orval client for web-admin
npm run generate:client -w web-admin
```

### Quality

```bash
# Frontend lint + format + type check
npm run quality          # local
npm run quality:ci       # CI mode (fails fast)

# Go linting
golangci-lint run ./path/to/package/

# Auto-format
npm run format           # frontend
# Go: golangci-lint run --fix
```

### Clean & Setup

```bash
# Remove built dev-project
npm run clean

# Reinstall everything
npm install && npm run build

# Sync SvelteKit typegen
npm run generate:sveltekit -w web-common
```

## Architecture Deep Dive

### Backend Pipeline (Parse → Resolve → Reconcile)

1. **Parser** (`runtime/parser/parser.go`): Reads YAML/SQL project files, produces `Resource` objects with specs matching `proto/rill/runtime/v1/*.proto` definitions. Each resource has a `ResourceKind` (source, model, metrics_view, explore, connector, canvas, alert, report, theme, component, api).
2. **Resolver** (`runtime/resolver/`): Resolves cross-resource references (e.g., a model references a connector). Builds a DAG.
3. **Reconciler** (`runtime/reconciler/`): Drives each resource to its desired state — e.g., materializing a SQL model into the OLAP engine, validating a metrics view's schema, creating a connector connection.
4. **Drivers** (`runtime/drivers/`): Abstract OLAP engines (DuckDB, ClickHouse), connectors (file, S3, GCS, databases), and AI providers (Claude, DeepSeek). Each driver implements a standard interface.

### Runtime Server

The runtime exposes a gRPC server (`runtime/server/`) mapped to REST via Connect RPC. Key services include query execution, resource management, metrics view operations, and AI chat.

### Frontend Architecture

- **SvelteKit** apps with path aliases `@rilldata/web-*` pointing to each workspace.
- **TanStack Query** (`@tanstack/svelte-query`) for server state fetching via auto-generated Orval clients (`web-common/src/runtime-client/`).
- **Svelte 5** migration in progress (Svelte 4 current baseline). Svelte stores for client-side global state.
- **Shared library** (`web-common/src/lib/`) contains utilities, formatters, i18n (Paraglide), error handling, and domain-agnostic components.
- **Vega/Vega-Lite** for chart rendering (overridden to v6.x+ in package.json overrides).
- **CodeMirror 6** for SQL/YAML editors.

### API Definition Flow

`.proto` files → `buf generate` → Connect RPC Go server + OpenAPI JSON + Orval TypeScript client → Svelte Query hooks in frontend. Never hand-edit generated files.

## Code Conventions

### Go (as defined in `.claude/rules/code-review.md` and upstream)

- Uber Go style guide. Use `golangci-lint` after changes.
- Functions sorted in call order; grouped by receiver; utilities at end.
- Prefer `errors` from stdlib (not `pkg/errors`). Use `require.NoError` in tests.
- Import paths use `github.com/fridencao/stardata` (fork module path).

### Frontend (as defined in `.cursor/rules/`)

- **Svelte**: Callback props over `createDispatchEvent`. Component events handled via prop callbacks.
- **Naming**: PascalCase Svelte components, kebab-case TS/JS files and directories.
- **Tailwind v4**: Inline classes only — no `<style>` blocks in `.svelte` files.
- **State**: Svelte stores for client state, TanStack Query for server state.
- **i18n**: Paraglide-JS for internationalization. Build with `npm run build:i18n`.

### Cursor Rules (auto-applied)

Rules in `.cursor/rules/` are automatically loaded:
- `codegraph.mdc` — CodeGraph MCP tool selection (always applied)
- `svelte.mdc` — Svelte best practices for `.svelte` files
- `naming-conventions.mdc` — File/component naming standards
- `tailwind-v4.mdc` — Tailwind CSS v4 inline class rules
- `frontend-development.mdc` — Component architecture, state management, API patterns

## Tool Usage

- **WebFetch/WebSearch lose information** on dense pages. For reference docs, download raw and grep directly: `curl -sL 'URL' | sed 's/<[^>]*>//g' | grep -i 'pattern'`.
- **CodeGraph MCP** (`codegraph_*` tools) provides AST-level symbol queries — prefer over grep for structural questions. See `.cursor/rules/codegraph.mdc` for full usage guide.

## Tips

- **Monorepo**: npm workspaces (frontend) + Go modules (backend). Root `package.json` coordinates workspaces: `docs`, `web-admin`, `web-common`, `web-integration`, `web-local`.
- **Path aliases**: `@rilldata/web-local`, `@rilldata/web-common`, `@rilldata/web-admin` configured in tsconfig/vite.
- **Dev server ports**: web-local runs on 3001, backend runtime on its own port. Use `npm run dev` to start both.
- **Embedded dashboards**: Explore and Canvas dashboards can be embedded via iframe. Changes to dashboard components may affect both surfaces.
- **DuckDB extension**: Custom builds live in `scripts/embed_duckdb_ext/`. Modifying DuckDB requires rebuilding the extension before `make cli`.
