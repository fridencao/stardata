# Phase 5：DB-Versioned Semantic Layer 设计文档

> 版本：v1.0（设计稿）
> 前置依赖：Phase 4 企业端产品设计已完成实施
> 决策基线：单 org + 多数据域；用 DB 版本化替代 file + git + branch + dev-deployment 模型
> 范围：语义层定义存储、版本管理、发布管线、编辑锁、回滚、可见性控制

---

## §1 背景与目标

### 1.1 现状

StarData 是 rilldata/rill 的深度 fork（Go backend + SvelteKit frontend），为中国金融客户私有化部署场景定制。当前语义层定义的存储与管理沿用 Rill 原生架构：

| 层级 | 当前机制 | 问题 |
|---|---|---|
| 存储 | YAML 文件 → tar.gz archive（`archive.CreateFromBlobs`） | 金融客户环境无 git 服务器，无法做真正的版本对比/回滚 |
| 草稿 | `syncEditable` (`runtime/drivers/admin/repo_archive.go`) 解压到本地文件系统作为可编辑工作区 | 单实例时文件系统状态脆弱，崩溃即丢失 |
| 分支 | `@branch` URL 段 + dev deployment（`environment='dev'`） | 对银行用户无意义，增加部署复杂度 |
| 发布门控 | `publish.yaml` + `runtime/publishgate.go` 读取文件声明 | 无法做到资源级粒度的可见性控制 |
| 版本 | `project_publishes` 表记录发布历史（archive hash） | 无法做到资源级差异对比，回滚需整体覆盖 |

### 1.2 目标

**用纯 DB 版本化方案彻底替换 file-based + git + branch + dev-deployment 模型**：

1. 所有语义层定义（source、model、metrics_view、explore、canvas 等）以 JSONB 存储在 PostgreSQL
2. 版本号自增，支持原子发布、预览（dry-run）、双人审批回滚
3. 资源级可见性控制：发布后治理者可逐条开放给业务侧
4. 编辑锁保证同一时刻单人编辑、自动保存草稿
5. 不再依赖 git、不再有 branch 概念、不再有 dev deployment

### 1.3 前置术语

| 术语 | 含义 |
|---|---|
| 数据域（project） | 一个独立的语义层 + 数据 + 权限边界 |
| 治理者（governor） | project admin，使用 Studio 工作台编辑语义层 |
| 业务用户 | project viewer，使用门户看板/对话/探索 |
| 发布版本（published version） | 经过 dry-run 验证并通过的原子快照 |
| 草稿（draft） | 治理者编辑中但尚未发布的资源状态 |

---

## §2 目标架构

### 2.1 数据模型（DB Schema）

新增 6 张表，全部位于 admin database（PostgreSQL）。迁移文件从 `admin/database/postgres/migrations/0100.sql` 起顺序编号。

#### 2.1.1 `semantic_resources` — 语义资源核心表

存储所有资源定义，取代原先散落在 archive 中的 YAML 文件。

```sql
CREATE TYPE semantic_resource_kind AS ENUM (
    'source', 'model', 'metrics_view', 'explore', 'canvas',
    'report', 'alert', 'theme', 'api', 'config'
);

CREATE TYPE semantic_resource_status AS ENUM (
    'draft', 'published', 'validating', 'rejected'
);

CREATE TABLE semantic_resources (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id          UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    resource_kind       semantic_resource_kind NOT NULL,
    resource_name       TEXT NOT NULL,
    -- definition 存储解析后的资源定义，等价于原 YAML 内容的结构化形态
    definition          JSONB NOT NULL,
    -- version 在 project 内自增；同一 (project, kind, name) 的多个 version 形成历史链
    version             INTEGER NOT NULL,
    status              semantic_resource_status NOT NULL DEFAULT 'draft',
    created_by_user_id  UUID REFERENCES users (id) ON DELETE SET NULL,
    created_on          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_on          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 同一 project 内，同一 kind + name 的同一 version 唯一
CREATE UNIQUE INDEX semantic_resources_unique_version
    ON semantic_resources (project_id, resource_kind, lower(resource_name), version);

-- Parser 按 project 全量拉取时的主索引
CREATE INDEX semantic_resources_project_status
    ON semantic_resources (project_id, status);

-- 取某资源最新 draft 的加速索引
CREATE INDEX semantic_resources_latest_draft
    ON semantic_resources (project_id, resource_kind, lower(resource_name), version DESC)
    WHERE status = 'draft';
```

**关键设计说明**：

- `definition` 用 JSONB 而非 TEXT：便于做资源引用关系查询（如「哪些 metrics_view 依赖 model X」可以用 JSONB 路径查询，无需全量解析）。
- 保留 `version` 列而非直接覆盖：任何一次 save 都产生新行，旧行不删除，天然形成完整审计链。
- `resource_name` 用 `lower()` 建唯一索引：与 Rill runtime 的资源名大小写不敏感语义一致。
- Model 的 SQL 文本作为 `definition->>'sql'` 存储；Source 的 connector 配置作为 `definition->'connector'` 存储。

#### 2.1.2 `resource_visibility` — 资源级业务可见性

治理者对每个资源单独控制是否对业务侧开放（Q13-B 方案）。取代 `publish.yaml` 文件。

```sql
CREATE TABLE resource_visibility (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id          UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    resource_kind       semantic_resource_kind NOT NULL,
    resource_name       TEXT NOT NULL,
    -- 默认 false：新资源发布后仍需治理者显式开启才对业务可见
    visible             BOOLEAN NOT NULL DEFAULT false,
    updated_by_user_id  UUID REFERENCES users (id) ON DELETE SET NULL,
    updated_on          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX resource_visibility_unique
    ON resource_visibility (project_id, resource_kind, lower(resource_name));
```

**默认不可见（fail-closed）**：新建资源在没有对应 `resource_visibility` 行、或 `visible=false` 时，业务侧一律不可见。这是安全默认值——避免治理者误发布内部中间表给业务用户。

#### 2.1.3 `project_versions` — 项目级原子版本快照

```sql
CREATE TYPE project_version_status AS ENUM (
    'draft', 'published', 'validating', 'rejected'
);

CREATE TABLE project_versions (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id            UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    version               INTEGER NOT NULL,
    status                project_version_status NOT NULL DEFAULT 'validating',
    published_by_user_id  UUID REFERENCES users (id) ON DELETE SET NULL,
    published_on          TIMESTAMPTZ,
    note                  TEXT NOT NULL DEFAULT '',
    -- dry-run 失败时的错误报告（结构化 JSON，供 Studio 展示）
    validation_report     JSONB,
    created_on            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_on            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX project_versions_unique
    ON project_versions (project_id, version);

-- 查询「当前已发布版本」的加速索引
CREATE INDEX project_versions_published
    ON project_versions (project_id, version DESC)
    WHERE status = 'published';
```

`projects` 表新增一列指向当前生效版本：

```sql
ALTER TABLE projects
    ADD COLUMN current_published_version_id UUID
        REFERENCES project_versions (id) ON DELETE SET NULL;

-- 标记老架构项目（详见 §5 迁移策略）
ALTER TABLE projects
    ADD COLUMN semantic_layer_mode TEXT NOT NULL DEFAULT 'db_versioned';
    -- 取值：'db_versioned' | 'legacy_archive'
```

#### 2.1.4 `project_version_resources` — 版本与资源的关联表

一次发布 = 创建一个 `project_versions` 行 + 把当时的资源行集合快照到本表。

```sql
CREATE TABLE project_version_resources (
    project_version_id   UUID NOT NULL REFERENCES project_versions (id) ON DELETE CASCADE,
    semantic_resource_id UUID NOT NULL REFERENCES semantic_resources (id) ON DELETE RESTRICT,
    PRIMARY KEY (project_version_id, semantic_resource_id)
);

-- 按版本拉取全量资源（Parser 加载路径）
CREATE INDEX project_version_resources_by_version
    ON project_version_resources (project_version_id);

-- 反查：某资源行被哪些版本引用（用于判断能否物理清理）
CREATE INDEX project_version_resources_by_resource
    ON project_version_resources (semantic_resource_id);
```

`ON DELETE RESTRICT` 保证被任何版本引用的资源行不会被误删——这是回滚能力的前提。

#### 2.1.5 `editing_locks` — 项目级编辑锁

```sql
CREATE TABLE editing_locks (
    project_id          UUID PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    locked_by_user_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    locked_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- 默认 locked_at + TTL，TTL 可配置，默认 2 小时
    expires_at          TIMESTAMPTZ NOT NULL,
    last_heartbeat      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 后台清理任务扫描过期锁
CREATE INDEX editing_locks_expires_at ON editing_locks (expires_at);
```

`project_id` 作为主键即天然保证「一个数据域同时只有一把锁」。

#### 2.1.6 `rollback_requests` — 双人审批回滚

```sql
CREATE TYPE rollback_request_status AS ENUM (
    'pending', 'approved', 'rejected', 'executed'
);

CREATE TABLE rollback_requests (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id            UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    target_version        INTEGER NOT NULL,
    requested_by_user_id  UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    approved_by_user_id   UUID REFERENCES users (id) ON DELETE RESTRICT,
    status                rollback_request_status NOT NULL DEFAULT 'pending',
    reason                TEXT NOT NULL DEFAULT '',
    requested_on          TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_on           TIMESTAMPTZ
);

-- 同一 project 同时只允许一个 pending 回滚请求
CREATE UNIQUE INDEX rollback_requests_single_pending
    ON rollback_requests (project_id)
    WHERE status = 'pending';

CREATE INDEX rollback_requests_project_history
    ON rollback_requests (project_id, requested_on DESC);
```

**审批人必须与发起人不同**：由 API 层校验 `approved_by_user_id != requested_by_user_id`，DB 层用 CHECK 约束兜底：

```sql
ALTER TABLE rollback_requests
    ADD CONSTRAINT rollback_dual_approval
    CHECK (approved_by_user_id IS NULL OR approved_by_user_id <> requested_by_user_id);
```

### 2.2 Runtime 架构变化

**单 runtime 实例、无 branch、无 dev deployment**：所有 project 只有一个「当前发布版本」+ 一份 draft overlay，全部由 admin 通过 gRPC 通知 runtime 加载。

#### 2.2.1 `DBRepoDriver`：从 DB 读语义资源的 RepoStore

新增 `runtime/drivers/dbrepo/`，实现 `drivers.RepoStore` 接口。Parser 仍然工作在「虚拟文件」抽象上，但底层不再来自 tar.gz / 本地文件系统，而是来自 `semantic_resources` 表。

```go
// runtime/drivers/dbrepo/repo.go（示意）
type dbRepo struct {
    projectID string
    version   int         // 0 = 当前 published 版本；>0 = 指定版本
    overlay   bool        // true = 在 published 之上叠加最新 draft
    db        *pgxpool.Pool
}

// ListRecursive 返回该 project 下所有 (kind, name) 组合作为 "virtual paths"
func (r *dbRepo) ListRecursive(ctx context.Context, glob string, skipDirs bool) ([]drivers.DirEntry, error)

// Get 按虚拟路径（形如 "metrics_view/orders.yaml"）从 DB 读回 definition，
// 序列化为 Rill parser 期望的 YAML 字节流
func (r *dbRepo) Get(ctx context.Context, path string) (string, error)
```

**Draft overlay 语义**：当 `overlay=true` 时，同一 (kind, name) 存在 draft 与 published 两条记录，返回 draft；否则返回 published。这是治理者在 Studio 看到最新草稿、业务侧看到已发布版本的底层机制。

#### 2.2.2 Reconciler 触发：内部 gRPC 通知（Q18=A）

替换掉原来「文件系统 fsnotify → controller 重新解析」的链路。发布/回滚完成后，admin 通过内部 gRPC 通知 runtime：

```protobuf
// runtime/proto/rill/runtime/v1/internal.proto
service RuntimeInternalService {
    rpc NotifyVersionChange(NotifyVersionChangeRequest)
        returns (NotifyVersionChangeResponse);
}

message NotifyVersionChangeRequest {
    string project_id = 1;
    int32  version    = 2;   // 目标版本号
    string reason     = 3;   // "publish" | "rollback" | "draft_save"
}
```

Runtime 收到通知后：

1. 关闭旧 instance（graceful）
2. 用 `DBRepoDriver{version: N}` 打开新 instance
3. 触发一次全量 reconcile

#### 2.2.3 Resolver 权限分流

在 `runtime/server/queries.go` 请求入口按 caller identity 区分：

| 调用者 | 数据源 | 可见性过滤 |
|---|---|---|
| 业务用户（viewer） | `DBRepoDriver{version: current_published, overlay: false}` | 应用 `resource_visibility.visible=true` 过滤 |
| 治理者（manageProject）Studio 视图 | `DBRepoDriver{version: current_published, overlay: true}` | 全部资源可见（含 draft） |
| 治理者门户查看模式（预览已发布效果） | 同业务用户 | 同业务用户 |

治理者的两种视图切换由 Studio UI 传递明确的 header（`X-Studio-View: draft|published`）控制，避免依赖角色隐式推断。

#### 2.2.4 Publish gate 迁移

现有 `runtime/publishgate.go` 读取 `publish.yaml` 文件的逻辑改为查询 `resource_visibility` 表：

```go
// runtime/publishgate.go（重写后示意）
func (g *PublishGate) CheckResourceVisible(
    ctx context.Context,
    projectID string,
    kind, name string,
) (bool, error) {
    row, err := g.db.QueryRow(ctx, `
        SELECT visible FROM resource_visibility
        WHERE project_id = $1 AND resource_kind = $2 AND lower(resource_name) = lower($3)
    `, projectID, kind, name).Scan(&visible)
    // 无行 → fail-closed，返回 false
    if errors.Is(err, pgx.ErrNoRows) {
        return false, nil
    }
    return visible, err
}
```

配套的 AI 侧 publish gate（`runtime/ai/publish_gate_ai_test.go`）保持接口不变，只替换数据源。

### 2.3 发布流程（Publish Pipeline）

治理者从 Studio 点击「发布」触发以下时序：

```
Governor            Admin API                Temp Runtime         Prod Runtime
   │                    │                          │                    │
   │─── click publish ─►│                          │                    │
   │                    │─ INSERT project_versions │                    │
   │                    │     (status=validating)  │                    │
   │                    │─ SELECT draft resources ─┐                    │
   │                    │─ INSERT project_version_resources             │
   │                    │─── spin up temp instance ►                    │
   │                    │                          │─ full reconcile ─┐ │
   │                    │                          │  (with sampling) │ │
   │                    │                          │◄─ result ────────┘ │
   │                    │◄── reconcile OK / fail ──│                    │
   │                    │─ UPDATE version status                        │
   │                    │  = published / rejected                       │
   │                    │─ UPDATE projects.current_published_version_id │
   │                    │─── NotifyVersionChange ──────────────────────►│
   │                    │                          │                    │─ hot reload
   │◄── success + report│                          │                    │
   │                    │                          │─── shutdown ───────┘
```

**详细步骤**：

1. **锁定草稿快照**：admin 在事务内 `SELECT ... FOR UPDATE` 最新 draft 资源行集合，在 `project_version_resources` 插入关联行。这一步之后即使治理者继续编辑，也不会影响本次发布的输入。
2. **创建临时 runtime 实例**：物理上启动一个独立的 runtime process（或 in-process instance），OLAP 后端指向独立的临时 schema（如 `duckdb_dryrun_<version>`），避免污染生产数据。
3. **执行 dry-run reconcile**：controller 走完整个 resource state machine。**大 model 使用抽样物化**（Q25=B）：`materialize=true` 的 model 在 dry-run 模式下仅物化前 100K 行，SQL 层面通过 `LIMIT` 或 `TABLESAMPLE` 注入。
4. **结果判定**：
   - 成功 → `UPDATE project_versions SET status='published'`；`UPDATE projects SET current_published_version_id=...`；通过 `NotifyVersionChange` 通知生产 runtime。
   - 失败 → `UPDATE project_versions SET status='rejected', validation_report=...`；不改 `current_published_version_id`；返回结构化错误报告给 Studio 展示。
5. **临时实例清理**：不论成功失败都清理临时 schema、shutdown 临时 instance。
6. **发布后可见性**：新资源的 `resource_visibility.visible` 默认为 `false`。治理者需在 Studio 逐条开关（或批量勾选）「对业务开放」。

### 2.4 预览（Preview / Dry-Run）

预览与发布**共享同一条 dry-run 管线**，只是不 commit：

- 治理者点「预览」→ admin 创建 `project_versions` 行（status=validating）→ 走完全 dry-run → 结果返回后**不**更新 `current_published_version_id`。
- 预览生成的临时 schema 保留 5 分钟供治理者在 Studio 交互式查看结果（试查询、试渲染看板）。
- 大 model 的抽样数据在 UI 上通过统一 badge 明确标注「抽样预览 · 100K 行」。
- 治理者确认无误后可点「转为发布」，直接把此次预览的 `project_versions` 行状态升级为 `published`——无需再跑一次 reconcile。

### 2.5 回滚

**双人审批 + 90 天窗口 + 异步重物化**：

1. **发起**：治理者 A 在 Studio「发布历史」页选中目标版本 N，填写回滚原因，创建 `rollback_requests` 行（status=pending）。校验：
   - N 必须在最近 90 天内（`published_on > now() - '90 days'::interval`）
   - N 的状态必须是 `published`
   - 该 project 无其他 pending 回滚请求
2. **审批**：另一名治理者 B（`approved_by != requested_by`）在 Studio 上看到待审列表，点「同意」→ status=approved。
3. **执行**：admin 后台 worker 检测 approved 状态，执行：
   - `UPDATE projects SET current_published_version_id = <N 对应的 project_versions.id>`
   - `NotifyVersionChange(project_id, N, reason='rollback')`
   - 生产 runtime 重新加载版本 N 的资源
   - 触发异步全量物化（不 sample）
4. **物化期间的一致性**：
   - Runtime 状态机进入 `remateralizing` 阶段，此期间旧 model 表继续服务查询（返回 stale 数据）
   - 前端所有看板/探索页显示统一 banner：**「数据正在更新中，当前展示为回滚前的缓存数据」**
   - 物化完成后自动清除 banner
5. **审计事件**：`rollback_requested` / `rollback_approved` / `rollback_executed` / `rollback_completed` 四条独立审计日志。
6. **超 90 天的回滚**：需走 archive recovery 流程（Phase 5 之外的运维工具），不在 Studio UI 内提供入口。

### 2.6 编辑锁（Editing Lock）

**项目级、TTL 2 小时、心跳自动保存**（Q20 方案）：

- **获锁**：治理者进入 Studio 编辑视图 → `AcquireLock(project_id)` API：
  - 无锁 → 创建行 `(locked_by=me, locked_at=now, expires_at=now+2h, last_heartbeat=now)`，返回成功
  - 有锁且 `expires_at > now` 且 `locked_by != me` → 返回 409，提示「A 正在编辑，剩余 X 分钟」，并让 B 进入只读模式
  - 有锁但 `expires_at <= now` → 视为过期，覆盖为 me
- **心跳 + 自动保存**：前端每 60 秒 `Heartbeat(project_id, dirty_resources[])`：
  - 更新 `last_heartbeat = now, expires_at = now + 2h`
  - `dirty_resources` 中的编辑器状态写入 `semantic_resources`（新 draft 行 or 更新最近 draft 行）
- **主动释放**：治理者点「保存并退出」或关闭编辑器 → `ReleaseLock(project_id)`，删除行
- **强制解锁（踹锁）**：其他治理者可点「强制接管」→ 记录审计事件 → 删除原锁 → 立即获锁
  - 原编辑者未提交的 draft 保留（不删 `semantic_resources` 中的 draft 行），新编辑者可选择「继续 A 的草稿」或「丢弃从当前发布版本重新开始」
- **后台清理**：runtime worker 每分钟扫 `expires_at < now()` 的行并删除，避免僵尸锁挤占入口

**Governor B 的只读模式**：即使拿不到锁，B 仍可看到 A 正在编辑的 draft 内容（`DBRepoDriver.overlay=true` 视图），但所有编辑器 UI 显示为 disabled，写操作 API 返回 403。业务用户任何时候都只看 published，不受编辑锁影响。

### 2.7 编辑器：文件树 IDE 彻底移除

**核心原则**：治理者不再面对「YAML 文件」概念，只面对「资源」概念。

| 资源类型 | 编辑形态 |
|---|---|
| Model | Studio「数据模型」页嵌入 Monaco editor，编辑 SQL。JOIN/CTE 等复杂逻辑通过 SQL 自由表达 |
| Metrics view / dimensions / measures | 表单化引导编辑器（已在 `web-common/src/features/studio/StudioSemanticEditorPage.svelte` 部分实现） |
| Source / connector | 表单化（已在 add-data flow 存在） |
| Explore / Canvas | 现有可视化搭建器 |
| Theme / API / Config | 表单化（未来可加 JSON schema 驱动的动态表单） |

**保存语义**：任何编辑器保存都只做**语法校验 + 引用校验**（例如 metrics view 引用的 model 是否存在），**不做物化**。语法错误 inline 展示。物化仅在 dry-run / 发布 / 回滚时发生。

---

## §3 去掉的模块（Modules to Remove）

### 3.1 Backend：archive-based 草稿机制

| 路径 | 处理 |
|---|---|
| `runtime/drivers/admin/repo_archive.go` | 删除 `syncEditable`（`repo_archive.go:69`）及整个 `archiveRepo` 的可编辑分支 |
| `runtime/drivers/admin/repo_archive_test.go` | 删除（archive-based reconcile 测试整体失效） |
| `runtime/pkg/archive/archive.go` — `CreateFromBlobs` / `Download` | **保留但缩范围**：仅用于导出/备份场景（见 §8），不再用于 dev draft 加载 |
| `archive` 相关 reconcile 集成测试 | 删除或改写为 `DBRepoDriver` 版本 |

### 3.2 Branch 概念全链路移除

**前端**：

- `web-admin/src/hooks.ts` — 移除 `@branch` 相关的 reroute 逻辑
- `web-admin/src/features/branches/` 整个目录：
  - `BranchDeploymentStopped.svelte`
  - `BranchesSection.svelte`
  - `DeleteBranchConfirmDialog.svelte`
  - `branch-utils.ts` + `branch-utils.spec.ts`（含 `extractBranchFromPath`、`branchPathPrefix`）
  - `branch-actions.ts`
  - `deployment-utils.ts` 中的 branch 分支逻辑
- `web-admin/src/features/edit-session/EditBranchDialog.svelte`
- `web-admin/src/routes/[organization]/[project]/-/status/branches/` 路由
- `web-common/src/lib/i18n/gen/messages/` 下 ~40 个 `branch_*` / `edit_*branch*` 文案条目（含 `branch_hibernate`、`branch_resume`、`project_branch_hibernated` 等）

**Backend**：

- `deployments` 表中 `environment='dev'` 的行创建路径
- `CreateProjectBranchName` 及相关命名工具
- dev deployment TTL / hibernate 唤醒逻辑
- Admin API 中的 branch CRUD RPC

### 3.3 发布门控文件化机制

- `publish.yaml` 的解析与写入（替换为 `resource_visibility` 表查询）
- `runtime/publishgate.go` 中的文件读取路径（保留接口，重写实现）
- `runtime/publishgate_test.go` 改写为 DB fixture 驱动

### 3.4 IDE 文件树与工作区路由

- `(workspace)` 路由组整体删除
- `WorkspaceDispatcher` 组件
- `FileAndResourceWatcher`（替换为 §2.2.2 的 DB 变更通知）
- 文件树侧栏、新建文件/重命名/删除文件的所有 UI 与 API
- `editorRoutePrefix` store 及所有消费方（当前有 ~19 处引用，包括 `web-admin/src/routes/[organization]/[project]/-/edit/studio/**` 整个旧 Studio 路由树，Phase 4 已迁移到 `/studio/[domain]`，此处做最终清理）
- `web-admin/src/features/projects/status/overview/DeploymentSection.svelte`、`ProjectHeader.svelte`、`WelcomeRedirector.svelte` 中的 `editorRoutePrefix` 拼接逻辑

### 3.5 清理顺序约束

上述删除**必须放在 Phase 5.4**（见 §6），原因：5.1–5.3 期间新旧两套并存，老架构项目（`semantic_layer_mode='legacy_archive'`）仍需 archive 读取路径提供只读能力。5.4 时确认所有存量项目已冻结为只读且不再需要重新 reconcile 后，方可物理删除代码。

---

## §4 保留和复用的模块

| 模块 | 复用方式 |
|---|---|
| Runtime reconciler core（`runtime/controller.go`、resource state machine） | **完全保留**，仅把输入源从 files 换成 DB。这是本方案能低成本落地的关键——reconciler 本身对「资源从哪来」是无感的 |
| DuckDB / ClickHouse connector + 物化引擎 | 完全保留，无改动 |
| 安全层 | `ApplySecurityPolicy` 保留；`CheckPublishGate` 重写实现（读 DB）但接口不变；`CheckFeatureAccess` 保留 |
| Admin API 框架 | gRPC-gateway、auth middleware、audit interceptor 全部复用，只新增 RPC |
| 前端业务门户 | `PortalNav`、`StudioTabs`、chat、boards、explore viewer 全部不动 |
| 发布历史 UI | 保留页面结构，数据源从 `project_publishes` 换成 `project_versions`（多出「资源级 diff」能力） |
| Audit trail | 保留，扩展新事件类型（见下） |
| Rill parser | 保留。`DBRepoDriver` 把 JSONB 序列化回 YAML 字节流喂给 parser，parser 代码零改动 |

**新增审计事件类型**：

```
semantic_resource_saved       // 草稿保存
semantic_resource_deleted
project_version_preview       // 预览触发
project_version_published
project_version_rejected
resource_visibility_changed
editing_lock_acquired
editing_lock_force_released   // 踹锁（高危，必审）
rollback_requested
rollback_approved
rollback_rejected
rollback_executed
rollback_completed
```

---

## §5 迁移策略（Migration Path，Q28=C）

**策略 C：新老并存，老项目冻结只读，不做数据迁移。**

| 维度 | 处理 |
|---|---|
| 升级后新建项目 | `semantic_layer_mode='db_versioned'`，直接走新架构 |
| 存量项目 | `semantic_layer_mode='legacy_archive'`，冻结为**只读**：已发布看板/对话/探索全部正常可用，但所有编辑入口 disabled |
| 存量发布历史 | `project_publishes` 表**原样保留**，不迁移，Studio 发布历史页对老项目继续读该表（只查询、不新增） |
| 老项目 UI 提示 | Admin/Studio 在老项目上展示 banner：**「此数据域使用旧架构，仅支持查看。如需修改语义层，请新建数据域。」** |
| 迁移工具 | Phase 5 **不做**。未来可选：手工触发的 `archive → semantic_resources` 导入工具（解析 tar.gz 中 YAML → JSONB 行 → 创建初始 published version） |

**选择 C 的理由**：金融客户环境下，存量数据域承载着生产报表，任何自动迁移都有静默改变数字的风险。冻结只读让存量业务零风险，新需求走新架构，风险面被限制在「新建项目」这一可控范围。

**代价与接受**：客户如需修改老数据域的指标，必须在新数据域重建。这个成本对早期客户（数据域数量少）可接受；数量变多后再补迁移工具。

---

## §6 分阶段实施路径

总计约 11 周，4 个子阶段。每个子阶段结束都是可交付、可回退的状态。

### Phase 5.1 — Foundation（约 3 周）

**状态：已完成。** 实施中的两处偏离设计稿的决策记录如下。

| # | 任务 | 交付物 | 状态 |
|---|---|---|---|
| 1 | DB schema | `0100.sql`–`0102.sql`：`semantic_resources` + `editing_locks` + `projects.semantic_layer_mode` | ✅ |
| 2 | DB→文件渲染 | `admin/semantic_render.go` + `PullVirtualRepo` 的 DB 分支 | ✅ |
| 3 | 编辑锁 API | `AcquireEditLock` / `Heartbeat` / `Release` / `Get` / `ForceRelease` + 过期清理 worker | ✅ |
| 4 | 语义资源 CRUD RPC | `Save` / `List` / `Get` / `Delete` + 语法与引用校验 | ✅ |
| 5 | Studio UI（PoC） | `StudioDBResourceEditor.svelte` + `/studio/[domain]/db-editor/[kind]/[name]` | ✅ |
| 6 | 停止创建 dev deployment | `CreateDeployment` 对 DB 模式项目拒绝 `environment=dev` | ✅ |

**实施决策 1（偏离设计稿）**：原计划新建独立的 `runtime/drivers/dbrepo` 包实现 `RepoStore`。核实后放弃——`RepoStore` 有 17 个方法，且 runtime **不直连 admin 的 Postgres**（既有架构是 runtime 经 gRPC 向 admin 要文件）。改为把 DB→文件的渲染边界画在 **admin 一侧**，复用既有的 `PullVirtualRepo` 传输通道：runtime / parser / reconciler / watcher **一行未改**。

**实施决策 2**：`definition` JSONB 用 `{"raw": "<原始编辑器文本>"}` 存储并逐字回写，而非结构化字段再序列化。这样 parser 保真度是 100%，彻底避开 YAML 双向序列化漂移。结构化字段（供依赖查询）作为附加信息，永不作为渲染依据。

**5.1 已知局限（转入 5.2）**：
- draft 资源目前发给所有环境；published/draft 分流随发布管线落地
- 删除资源不回收 runtime 上已物化的文件
- 语义视图列表页仍链向文件版编辑器；DB 编辑器需直达 `/studio/[domain]/db-editor/[kind]/[name]`（完整编辑器移植在 5.3）

### Phase 5.2 — Publish Pipeline（约 3 周）

**状态：5 / 6 子任务完成 + 前端发布页落地。**

| # | 任务 | 交付物 | 状态 |
|---|---|---|---|
| 1 | 版本快照逻辑 | `0103.sql` + `0104.sql` + `postgres/project_versions.go` + 事务化 `SnapshotDraftResources` | ✅ |
| 2 | Dry-run 门控 | **parser-only 实现**：`admin/dryrun.go` 把快照渲染到 tempdir → 用 `file` driver 开 repo → 跑 `runtime/parser` → 收集 ParseError | ✅ |
| 3 | 「预览」按钮 | 待 UI 加发布前预览触发（可复用 T2 的机制）| ⏳ |
| 4 | 「发布」按钮 | `admin.PublishProject` 六步管线 + `PublishSemanticProject` RPC + `ListSemanticVersions` + 前端发布页 | ✅ |
| 5 | 资源可见性开关 | `0105.sql` + `resource_visibility` DB + `Set`/`ListResourceVisibility` RPC + Studio 逐条开关 + 合成 `publish.yaml` | ✅ |
| 6 | 版本变更通知 | **意外收获**：既有 `TriggerParser` 就是通知原语，`PullVirtualRepo` 在下一次 pull 里重新渲染当前 DB 状态。无需新 gRPC | ✅ |

**T2 实施选择（偏离设计稿）**：设计稿要求「临时 runtime 实例拉起 / reconcile」。核实后选了**更轻的 parser-only 实现**——渲染到 tempdir，用 `file` driver 开 repo，跑 `runtime/parser`，收集 `ParseError`。**不启动任何 runtime instance、不涉及 provisioner、不涉及 OLAP**。

理由：
1. 治理者实际会犯的错——YAML 语法、kind 拼错、引用不存在——parser 就能全部抓住
2. 临时 instance 方案的最大坑是 **OLAP 隔离**：临时实例和生产共享 DuckDB/ClickHouse database 会污染真实数据；单独起一份 OLAP 存储是一整块 provisioner 工作
3. 显式承认边界：model SQL 对真实数据的正确性（例如列名不存在）**dry-run 抓不住**，需要完整 reconcile + OLAP 存储，属 5.3 范围

**T6 的意外简化**：设计稿写的 `NotifyVersionChange` 内部 gRPC 在乙方案下是多余的。既有 `TriggerParser` 让 runtime 做一次 `pull`，`PullVirtualRepo` 的 DB 分支自然渲染最新版本。发布/可见性变更调 `TriggerParser`（带 15s 超时、best-effort）即可。

**发布安全网现状**：发布已经**通过 dry-run 门控**——语法错、kind 错、悬空引用被拒绝在业务侧之前；仅剩 model SQL 运行时正确性到 5.3 才能守住。

### Phase 5.3 — Full Coverage（约 3 周）

| # | 任务 | 交付物 |
|---|---|---|
| 1 | 全资源类型 DB 化 | model（Monaco SQL 编辑器）、source/connector、explore、canvas、theme、api、config |
| 2 | 回滚双人审批流 | `rollback_requests` CRUD + 审批 UI + 90 天窗口校验 + 双人校验 |
| 3 | 回滚重物化管线 | 异步全量物化 worker + `remateralizing` 状态 + 前端「数据正在更新中」banner |
| 4 | 自动保存与锁体验 | 60 秒心跳 flush draft、TTL 配置项、强制解锁 UI + 孤儿草稿接管选择 |
| 5 | 审计事件补齐 | §4 列出的 13 类新事件全部落地 + 系统审计日志页可查 |

**退出条件**：§9 所有验收标准通过。

### Phase 5.4 — Cleanup & Polish（约 2 周）

| # | 任务 |
|---|---|
| 1 | 删除死代码：`repo_archive.go` 可编辑路径、branch UI 整目录、dev deployment 逻辑、`(workspace)` 路由组、`publish.yaml` 解析、~40 个 branch i18n 条目 |
| 2 | E2E 测试：publish / preview / rollback / 编辑锁竞争 / 可见性切换 五条主流程 |
| 3 | 性能调优：§2.1 各索引的实际 EXPLAIN 验证、dry-run 临时实例的连接池上限与排队策略 |
| 4 | 文档更新：运维手册（回滚流程、锁排障）、治理者操作手册 |

---

## §7 风险与缓解

### 7.1 与 Rill 上游永久分叉（高，已接受）

`runtime/parser`、`runtime/resources`、`runtime/controller` 三个核心包在本方案后**不可能再与上游 merge**——输入源从文件系统改为 DB 是根本性改动。

**缓解**：
- 严格约束改动边界：只改「资源如何进入 parser」，不改 parser 内部的 YAML schema 解析逻辑，因此上游的 schema 演进仍可人工对齐
- **connector / driver 层保持完全兼容**：上游对 DuckDB、ClickHouse、Snowflake 等连接器的改进仍可 cherry-pick，这是上游价值最大的部分
- 建立分叉点记录文档，标注每个已分叉文件的 upstream commit，便于后续人工对比

### 7.2 大表物化超时（中）

500GB 级 model 的全量物化可能耗时数十分钟到数小时，无法在发布流程中同步等待。

**缓解**：
- Dry-run 阶段用抽样（Q25=B，前 100K 行）——验证的是 SQL 正确性与 schema 兼容性，不是数据完整性
- 全量物化异步执行，超时**不阻塞发布**：版本已标记 published，物化在后台推进
- 物化未完成期间业务侧看到上一版数据 + banner 提示
- 抽样带来的漏检风险（例如仅在完整数据中才出现的类型溢出）在发布后由物化失败告警捕获，治理者收到通知后回滚

### 7.3 编辑锁遗留（中）

进程崩溃、浏览器强杀、网络中断 → 心跳停止 → 锁滞留最长 2 小时。

**缓解**：
- TTL 到期自动释放（后台 worker 每分钟扫描）
- 任何治理者可**强制解锁**，无需等待，只需承担一条高危审计事件
- 孤儿 draft 完整保留，接管者可选择继承或丢弃——不会丢工作

### 7.4 回滚重物化期间业务看到旧数据（中）

回滚后 model 需要重新物化，期间查询返回的是回滚前的数据——与治理者预期的「回滚已生效」不一致。

**缓解**：
- 前端强制 banner **「数据正在更新中」**，不允许关闭，直到物化完成
- 审计日志记录完整回滚时间线（requested → approved → executed → completed），事后可精确追溯「某时刻业务看到的是哪个版本的数据」
- Studio 回滚确认弹窗预先告知预计物化时长

### 7.5 迁移无退路（低，Q28=C 已规避）

**风险被 C 方案结构性消除**：新项目用新架构，老项目冻结只读完全不受影响。如果新架构暴露 bug，受影响范围仅限升级后新建的数据域，存量生产报表零影响。

**残留风险**：客户在升级后大量新建数据域再遇到 bug，仍会有返工。缓解措施是 5.1/5.2 阶段先在单个试点数据域上验证，确认稳定后才向客户开放批量新建。

---

## §8 待决 / 后续

| 议题 | 当前判断 |
|---|---|
| 多 runtime 实例水平扩展 | 当前不做。单实例足够支撑目标客户规模；`NotifyVersionChange` 的接口设计已预留多实例广播的扩展空间 |
| 老项目自动迁移工具 | Phase 5 之后按需做。触发条件：客户存量数据域超过 5 个且明确要求改造 |
| AI / Chat 对 draft 资源的可见性 | **待定**。治理者在 Studio 内问 Chat 时，是否应该能看到尚未发布的 draft 指标？倾向于「可以，但答案带明确的『草稿指标』标注」，需产品确认 |
| 导出功能是否保留 | 倾向保留。「把项目导出为 tar.gz」对备份/审计/客户交付有价值，可复用现存的 `archive.CreateFromBlobs`，方向从「DB → tar.gz」（与原来相反）。需确认是否作为 Phase 5.4 的可选项 |
| 资源级 diff 展示 | `project_versions` 结构已支持精确到资源行的 diff，但 diff 可视化 UI 未排期。Phase 5 只提供「哪些资源变了」的列表，不提供 YAML 级 side-by-side |

---

## §9 验收标准

### 9.1 全新数据域端到端

新建数据域 → 配置数据源 → 建 model → 建 metrics view → 预览 → 发布 → 开放可见性 → 业务侧在门户看到指标并可探索。全程无 YAML 文件、无 git、无 branch 概念暴露。

### 9.2 已发布指标的修改隔离

修改一个已发布的 metrics view → 保存 draft → **此时业务侧数字不变** → 预览通过 → 发布 → 业务侧看到新数字。验证草稿态对业务侧完全不可见。

### 9.3 回滚双人审批

治理者 A 发起回滚到版本 N → A 自己无法审批（API 返回 403）→ 治理者 B 审批通过 → 执行 → 物化期间业务侧看到「数据正在更新中」banner → 物化完成 → 业务侧数字回到版本 N。审计日志含完整四段时间线。

### 9.4 编辑锁竞争

治理者 A 进入编辑 → 治理者 B 进入时提示「A 正在编辑」并进入只读模式（能看到 A 的 draft 内容但不能改）→ A 断开心跳 → TTL 到期 → B 可获锁 → B 看到 A 的孤儿草稿并可选择继承或丢弃。

补充验证：B 使用「强制接管」可立即获锁，并产生 `editing_lock_force_released` 审计事件。

### 9.5 资源级可见性

治理者关闭某个已发布指标的可见性 → 业务侧刷新后该指标立刻消失（含从对话/看板/探索三个入口）→ 其他指标完全不受影响 → 依赖该指标的看板给出明确的「指标已下线」提示而非报错。

### 9.6 老项目冻结（迁移验收）

升级后打开存量数据域 → 已发布看板/对话正常工作 → 所有编辑入口 disabled 且展示「此数据域使用旧架构，仅支持查看」banner → 发布历史页仍能读到 `project_publishes` 中的历史记录。

