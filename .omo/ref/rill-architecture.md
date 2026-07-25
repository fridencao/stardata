# Rill Architecture Reference (StarData Fork)

> Core architecture patterns discovered from reading the codebase.
> Updated: 2026-07-25

---

## 1. High-Level Architecture

```
┌──────────────────────────────────────────────────────────┐
│                          CLI (main.go)                     │
│                     github.com/fridencao/stardata/cli      │
└──────────────────────┬───────────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────────┐
│                     Runtime (runtime/)                     │
│                                                           │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │  Controller   │  │ AI Module    │  │   MetricsView  │   │
│  │ (resources    │  │ (agents,     │  │   (semantic    │   │
│  │  DAG, queue)  │  │  tools, MCP) │  │    layer, SQL) │   │
│  └──────┬───────┘  └──────┬───────┘  └───────┬────────┘   │
│         │                 │                  │            │
│         └─────────────────┼──────────────────┘            │
│                           │                               │
│                    ┌──────▼──────┐                        │
│                    │ Reconcilers │                        │
│                    │ (per type)  │                        │
│                    └──────┬──────┘                        │
│                           │                               │
│                    ┌──────▼──────┐                        │
│                    │  Drivers    │                        │
│                    │ (OLAP, etc) │                        │
│                    └─────────────┘                        │
│                                                           │
│  ┌───────────────────────────────────────────────────┐   │
│  │              Catalog (PostgreSQL DB)               │   │
│  │     Resources, DAG edges, state, errors            │   │
│  └───────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────┘
```

---

## 2. Controller — Resource Lifecycle

**File**: `runtime/controller.go` (1622 lines)

### Core Loop (eventLoop goroutine)

```
enqueue(resource) → processQueue() → processCompletedInvocation()
                      ↓
              markPending(resource + descendents)
                      ↓
              trySchedule(resource) → invoke() → goroutine
                      ↓
              reconciler.Reconcile(ctx, n, changes)
                      ↓
              processCompletedInvocation → enqueue children / waitlist
```

### Key Concepts

| Concept | Description |
|---------|-------------|
| **Resources** | Typed objects (source, model, metrics_view, explore, alert, etc.) stored in catalog |
| **DAG** | Directed acyclic graph tracking resource parent-child dependencies |
| **Queue** | Resources awaiting reconciliation; processed in batches on each loop iteration |
| **Invocation** | Tracks a running reconcile goroutine (name, cancel, waitlist, start time) |
| **Waitlist** | Resources waiting for a running invoc to finish (e.g. when a parent is cancelled and needs re-scheduling) |
| **SpecVersion** | Bumped on each spec change; used to detect stale results |

### Scheduling Priority (in trySchedule)
1. **Deletes** run first (highest priority)
2. **Renames** run second
3. **Regular reconciles** run last (only if no deletes/renames pending)

### State Machine per Resource
```
PENDING → RUNNING → IDLE (success/error)
  ↑                     |
  └───── (on change) ───┘
```

### Key Methods

| Method | Line | Purpose |
|--------|------|---------|
| `enqueue()` | 1052 | Adds resource to queue; notifies event loop |
| `processQueue()` | 1071 | Batch schedules all queued resources (Phase 1: mark, Phase 2: schedule) |
| `markPending()` | 1105 | Marks resource + all descendents as PENDING; cancels running invocations |
| `trySchedule()` | 1220 | Checks parent status, priority, then invokes |
| `invoke()` | 1284 | Starts goroutine for reconciler; tracks in `c.invocations` |
| `processCompletedInvocation()` | ~1400 | Handles result (error/warning/success); processes waitlist; enqueues children; migrates renamed |

### Waitlist Pattern
When a running child needs to be re-scheduled because its parent changed:
1. `addToWaitlist(child, specVersion)` marks child to be revisited
2. When invoc completes, `processCompletedInvocation` re-enqueues waitlisted resources

### Rename Handling
- Renames are tracked separately (`c.catalog.renamed`)
- During rename reconcile, old resource is marked deleted and new name is created
- If a resource is renamed while reconciling, the old invocation's waitlist re-enqueues the new name
- `safeRename` turns ambiguous renames into creates to avoid conflicts

### Flush & Reconcile
- `Catalog` state is flushed to DB every `c.flushInterval` (default 10s)
- `Reconcile` is called explicitly by `rill run` (long-running process) watching for catalog changes
- Reconcile is NOT a Kubernetes-style controller — it watches a local catalog DB, not a k8s API

---

## 3. Resource Types & Reconcilers

**File**: `runtime/reconcilers/` directory

| Resource Kind | Reconciler File | Purpose |
|---|---|---|
| `source` | `source.go` | Data source/connector ingestion |
| `model` | `model.go` | SQL model transformation |
| `metrics_view` | `metrics_view.go` | Metrics view definition (dims/measures/time) |
| `explore` | `explore.go` | Dashboard/explore configuration |
| `canvas` | `canvas.go` | Canvas dashboard (composable) |
| `alert` | `alert.go` | Alert definition & evaluation |
| `report` | `report.go` | Scheduled report |
| `connector` | `connector.go` | Data connector (DuckDB, ClickHouse, etc.) |
| `theme` | `theme.go` | UI theme/branding |
| `migration` | `migration.go` | Schema migration |
| `project_parser` | `project_parser.go` | Parses project YAML files into resources |
| `schedule` | `schedule.go` | Schedule management for alerts/reports |
| `resources` | `resources.go` | Resource management utilities |
| `validate` | `validate.go` | Validation utilities |

### Reconciler Interface
```go
type Reconciler interface {
    Reconcile(ctx context.Context, n *runtimev1.ResourceName) error
}
```

### Reconciler Contract
- **Must be idempotent** — can be called multiple times
- **Must respect ctx cancellation** — when cancelled, release resources (olap release, etc.)
- **Must handle NotFound** — resource may be deleted while reconciling
- Returns error for transient failures; returns nil for permanent success
- On return, the controller marks status IDLE and sets error if non-nil

---

## 4. AI Subsystem

**Files**: `runtime/ai/` directory

### Architecture

```
                ┌──────────────────┐
                │   MCP Server     │  (stdio-based, communicates with AI CLIs)
                │  runtime/ai/mcp/ │
                └────────┬─────────┘
                         │
                ┌────────▼─────────┐
                │     Runner       │  (manages tools, sessions, projectID)
                │  runtime/ai/     │
                └────────┬─────────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
  ┌───────▼──────┐ ┌────▼─────┐ ┌──────▼──────┐
  │ RouterAgent  │ │Analyst   │ │ Feedback    │
  │ (entry point)│ │ Agent    │ │ Agent       │
  └──────────────┘ └──────────┘ └─────────────┘
         │              │              │
         │              │              │
  ┌──────▼──────┐ ┌────▼─────┐        │
  │Developer    │ │Analyst   │        │
  │Agent        │ │Tools     │        │
  │(file ops,   │ │(SQL,     │        │
  │ YAML edit)  │ │ charts)  │        │
  └─────────────┘ └──────────┘        │
                        └─────────────┘
```

### Key Files

| File | Purpose |
|------|---------|
| `runner.go` | Main runner — manages projectID, tools, sessions, task execution |
| `router.go` | `RouterAgent` — routes requests to Analyst/Developer/Feedback agents |
| `router_agent.go` | Agent implementation: collects context, routes to sub-agents |
| `analyst.go` | `AnalystAgent` — data questioning via tools |
| `analyst_agent.go` | Implementation: exploration plan, tools, feedback |
| `developer.go` | `DeveloperAgent` — project file operations |
| `developer_agent.go` | Implementation: read/write files, helm, rill dev |
| `feedback.go` | `FeedbackAgent` — user feedback handling |
| `feedback_agent.go` | Implementation: accept/reject/refine |
| `install.go` | `InstallAgent` — CLI install & version management |
| `install_agent.go` | Implementation: check/install/version |
| `tools.go` | Tool interface: `RegisterTool()`, `Tool[Args, Result]` interface |
| `mcp/` | MCP server for AI CLI tool execution |

### Agent Communication Protocol
- Agents use a structured JSONL-based message format
- Shared `Context` struct passed through all agents with projectID, instanceID, env, etc.
- RouterAgent decides which sub-agent to use based on the user's task

### Tool Registration Pattern
```go
func RegisterTool(rt *runtime.Runtime, mgr *Manager)
```
Each tool implements:
- `Spec()` — tool metadata (name, description, parameters)
- `Handler()` — execution logic
- `CheckAccess()` — authorization check

### Tools (defined in `runtime/ai/tools/`)
- Data exploration: SQL queries, metrics view querying, chart generation
- Project management: read/write project files, parse YAML
- MCP-controlled: tools exposed through MCP server

### Instructions System
**File**: `runtime/ai/instructions/instructions.go`
- Embedded markdown files in `runtime/ai/instructions/data/`
- Front-matter parsed with YAML description
- `Load(path)` / `LoadAll(opts)` — loads instruction files
- `External` option — controls whether output is for external (Claude Code, Cursor) or internal (Rill agents)
- Supports TODO tracking with `todo:` markers
- Instructions are about *when/how* to use tools, not just generic prompting

---

## 5. Metrics View & Semantic Layer

**Files**: `runtime/metricsview/` directory

### Concepts

| Concept | Description |
|---------|-------------|
| **Metrics View** | YAML-defined analytics layer (dimensions, measures, time grains) |
| **Dimension** | Categorical field (e.g., `customer_id`, `region`) |
| **Measure** | Aggregated numeric field (e.g., `SUM(revenue)`, `COUNT(*)`) |
| **Time Dimension** | Temporal field (day, week, month, etc.) |
| **Security** | Row-level access via `include`/`exclude` policies per dimension |
| **Comparison** | Period-over-period calculation (e.g., WoW, MoM, YoY) |

### SQL Generation Pipeline

```
MetricsViewSpec (YAML)
    │
    ▼
metricsview.AST (Expression tree)
    │
    ├── metricsview.Rewrite (connector-specific SQL rewrites)
    │     ├── duckdb_rewriter.go
    │     ├── clickhouse_rewriter.go
    │     ├── druid_rewriter.go
    │     ├── pinot_rewriter.go
    │     └── bigquery_rewriter.go
    │
    ├── metricsview.ExpressionToSQL() → string
    │
    ▼
Executor (runtime/metricsview/executor/)
    ├── Query()        ─ interactive queries (3 min timeout)
    ├── Export()       ─ bulk export (5 min timeout)
    ├── PivotQuery()   ─ pivoted crosstab queries (5 min timeout)
    ├── Summary()      ─ metrics summary
    ├── Rollup()       ─ time-series rollup
    ├── CompareQuery() ─ period-over-period comparison
    └── Search()       ─ dimension value search
```

### Expression Type System

```go
type Expression struct {
    Cond    *Condition    // WHERE-like condition
    Value   *Value        // column reference, literal, or sub-expression
}

type Condition struct {
    Op    Operator     // AND, OR, NOT, IN, LIKE, GT, GTE, LT, LTE, EQ, NEQ
    Exprs []*Expression
}

type Value struct {
    Name     string    // column reference
    Bool     *bool     // literal boolean
    String   string    // literal string
    Number   float64   // literal number
    Time     *TimeRange
    Array    []*Value  // for IN clauses
    SubExpr  *Expression // sub-expression
}
```

### Security Layer
- **`ResolvedSecurity`**: Runtime-resolved access policies
- **`ResolvedPolicy`**: Per-attribute include/exclude rules
- Applied as WHERE clause modifications during SQL generation
- Policies reference dimension names, wildcard-supported

---

## 6. Drivers (OLAP Backends)

**File**: `runtime/drivers/`

| Driver | File | Type |
|--------|------|------|
| DuckDB | `duckdb/` | Embedded OLAP (primary, local) |
| ClickHouse | `clickhouse/` | Cloud OLAP |
| Druid | `druid/` | Cloud OLAP |
| Pinot | `pinot/` | Cloud OLAP |
| BigQuery | `bigquery/` | Cloud OLAP |
| PostgreSQL | `postgres/` | System catalog |
| File | `file/` | File-based connector |
| SQLite | `sqlite/` | Embedded DB (future) |

### Driver Interface
```go
type Driver interface {
    Open(ctx context.Context, instanceID string, config map[string]any) (Store, error)
    Drop(ctx context.Context, instanceID string) error
}
```

### OLAPStore Interface
```go
type OLAPStore interface {
    Exec(ctx context.Context, stmt *drivers.Statement) error
    Query(ctx context.Context, stmt *drivers.Statement) (*drivers.Result, error)
    // ... meta methods
}
```

### Connector Config Pattern
Connectors are configured in `rill.yaml` or instance config:
```yaml
connectors:
  - name: my_duckdb
    type: duckdb
    config:
      path: data/warehouse.duckdb
```

---

## 7. Frontend Architecture

**Directory**: `web-common/src/`

### UI Framework
- **Svelte 5** (upgraded from Svelte 4; uses `$state`, `$derived`, `$effect` runes)
- **TypeScript** throughout
- **Tailwind CSS** for styling
- **Vega-Lite / Vega** for chart rendering (`components/vega/`)

### Feature Organization
```
web-common/src/
├── features/
│   ├── chat/           ← AI chat interface (StarData's primary focus)
│   │   ├── core/          Conversation management, message types
│   │   ├── components/    Chat UI components
│   │   └── stores.svelte.js  State management
│   ├── explores/       ← Dashboard / explore views
│   ├── dashboards/     ← Legacy dashboards (replaced by explores?)
│   ├── alerts/         ← Alert configuration & management
│   ├── reports/        ← Report scheduling & management
│   ├── sources/        ← Data source management
│   ├── models/         ← SQL model management
│   ├── metrics-views/  ← Metrics view editor
│   ├── canvases/       ← Canvas dashboards
│   ├── themes/         ← Theme/brand customization
│   ├── connectors/     ← Connector management
│   └── ... (39 feature directories total)
├── components/         ← 62 reusable component directories
└── runtime-client/     ← Protobuf-generated TypeScript client
```

### Data Flow
1. **Protobuf → TypeScript**: `runtime-client/` generated from proto definitions
2. **Svelte Stores**: Reactive state management via `stores.svelte.js` (Svelte 5 runes)
3. **REST API**: Interacts with Runtime gRPC endpoint via HTTP/gRPC transcoding
4. **WebSocket**: Used for streaming query results

### Chat Feature (StarData Primary)
- `features/chat/core/conversation-manager.ts` — manages conversation state
- `features/chat/core/message-types.ts` — message schema
- Communicates with AI module via Runtime API
- Supports follow-up questions, data visualization in chat

---

## 8. Key Architectural Patterns

### Resource-DAG Pattern
- All user-configurable artifacts are **Resources** tracked in a catalog
- Resources form a dependency **DAG** with topological scheduling
- Any spec change re-reconciles the resource AND all dependents

### Connector Abstraction
- Multiple OLAP backends behind a unified interface
- Query rewrite layer per connector for dialect differences
- DuckDB is the default/local driver; ClickHouse is the primary cloud driver

### Agent Loop Pattern
- Router → sub-agents with dedicated tool sets
- Shared context (Context struct) passed between agents
- Each agent has a specific responsibility and tool access scope
- MCP integration for external AI CLI tool access

### State Management Pattern
- Svelte runes (`$state`, `$derived`, `$effect`) for frontend reactivity
- Protobuf-generated TypeScript client for API types
- Workspaces (VS Code extension pattern) not used — pure Svelte stores

---

## 9. Build & Configuration

### Environment File (`rill.yaml`)
```yaml
compiler:
  allow_unsafe: false
connectors:
  - name: my_connector
    type: duckdb
    config:
      path: data/warehouse.duckdb
```

### Important Build Details (StarData)
| Aspect | Value |
|--------|-------|
| Go module | `github.com/fridencao/stardata` |
| CLI binary name | `stardata` |
| Go proxy | `GOPROXY=https://goproxy.cn,direct` |
| npm registry | `--registry=https://registry.npmmirror.com` |
| Binary size | ~345MB (arm64, darwin) |
| Install prefix | `/usr/local/bin/stardata` |

### Build Commands
```bash
make stardata                 # Build Go binary
make frontend                 # Build web frontend
make lint                     # Run linters
make test                     # Run tests
make generate                 # Regenerate protos
```

---

## 10. Key Files Index

| File Path | Lines | Purpose |
|-----------|-------|---------|
| `cli/main.go` | — | CLI entry point |
| `runtime/controller.go` | 1622 | Resource lifecycle controller |
| `runtime/ai/runner.go` | — | AI runner (project, tools, sessions) |
| `runtime/ai/router.go` | — | RouterAgent (entry point) |
| `runtime/ai/analyst.go` | — | AnalystAgent spec |
| `runtime/ai/developer.go` | — | DeveloperAgent spec |
| `runtime/ai/tools.go` | — | Tool interface & registration |
| `runtime/ai/instructions/` | 221 | Instruction loading system |
| `runtime/metricsview/` | — | Semantic layer engine |
| `runtime/metricsview/executor/executor.go` | 1030 | Query execution |
| `runtime/drivers/` | — | Driver abstraction + OLAP connectors |
| `runtime/reconcilers/` | — | Per-resource reconcilers |
| `runtime/catalog.go` | — | Resource catalog (DB) |
| `runtime/runtime.go` | — | Runtime bootstrap |
| `web-common/src/features/chat/` | — | AI chat frontend |
| `web-common/src/features/explores/` | — | Dashboard UI |
| `web-common/src/runtime-client/` | — | Protobuf TS client |
| `proto/` | — | Proto definitions |
