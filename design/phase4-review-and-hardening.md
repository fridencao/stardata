# StarData Phase 4 评审与加固 Plan

> 版本：v1.0（评审共识稿）
> 关联文档：`design/phase4-enterprise-app.md`（原设计）、`design/feature-access-control.md`（功能权限矩阵）
> 定位：Phase 4 改造完成度评审 + 收尾加固行动清单
> 决策基线：面向企业交付（如银行 OCBC）的双重目标——「shippable-to-customer」与「架构可持续」并重

---

## 1. 评审结论概要

Phase 4「双层角色化智能 BI」改造，整体完成度约 **80%**。核心链路已通，安全边界与收尾质量未闭合。

- ✅ **契约层**（语义层 + 发布 + 需求回流 + 功能矩阵）代码全栈就位。
- ⚠️ **关键假设未验证**：archive-editable 草稿区 → publish → rollback 的端到端可靠性从未在部署态验证过（R1 spike 代码合入，但未做闭环走查）。
- ⚠️ **发布门控不是安全边界**：目前仅 ChatBI AI 工具走了 server-side gate 且 fail-open，`/boards`、`/explore/[name]`、`/canvas/[name]`、非 AI 查询 API 完全未受门控。
- ❌ **交付质量基线缺失**：审计日志零基建、Phase 4 新功能零 E2E 覆盖、审计/系统/独立需求页缺失。
- ⚠️ **架构债**：Studio 挂在 `-/edit/studio` 复用 dev deployment 生命周期，路由与错误信息暴露技术概念；治理者被强制 redirect 到 Studio 导致发布效果无法自验证。

## 2. 已确认的决策记录

| # | 议题 | 决策 | 备注 |
|---|---|---|---|
| D1 | 评审基线 | **同时**评估 shippable 与架构可持续，按严重度排序 | 用户确认 |
| D2 | R1 全链路验证 | **先做端到端验证脚本**，暴露并发/覆盖/回滚风险 | 用户确认 |
| D3 | 发布门控语义 | 需要作为**安全边界**看待 | 用户确认 |
| D4 | 发布门控修复方案 | 采纳**runtime 层统一拦截 + fail-closed**（3-5 天）方案 | 用户确认 |
| D5 | Feature Access 后端双保险 | **主动延后至 Phase 4 之后**（作为已接受的技术债存档） | 用户确认 |
| D6 | 审计基建 | Phase 4 收尾（4c）内做**最小落库**，UI 展示后置 | 用户确认 |
| D7 | E2E 测试 | Phase 4 收尾内补 **3 个 spec**（portal-home / studio-publish / feature-access） | 用户确认 |
| D8 | 治理者落地页 | **保留现状**（进 Studio），Studio 内加「预览业务视图」入口作为自验证手段 | 用户确认 |
| D9 | 术语泄漏 | i18n 层替换（业务上下文 `project` → 「数据域」），代码路由不动 | 用户确认 |
| D10 | Studio 路由解耦（`-/edit/studio` → `/studio/[domain]`） | **接受为技术债**，Phase 4 内不重构，仅补 Studio 层 error boundary | 用户确认 |

## 3. 行动清单（按优先级）

### P0（Ship-blocker）

- **P0-1 R1 端到端验证脚本**（1 天）
  - 基于 `deploy/docker-compose.*` 起完整栈，脚本走完：创建项目 → dev session 编辑 YAML → publish → 拉 prod 页面验证变化 → 再次编辑 → publish v2 → rollback 到 v1 → 验证 prod 与 dev 状态。
  - **强制暴露 3 个风险的真实程度**：
    1. 并发写入：两 shell 同时改同一文件后 publish，是否 half-written？
    2. 新 archive 覆盖：治理者 A publish 后，治理者 B 的 dev 未提交编辑是否被 `syncEditable` 全量覆盖？
    3. rollback 影响 dev：rollback 后 dev instance 的草稿目录是否被强制同步到旧版本？
  - **验收**：脚本能可重复执行；输出结论文档（`design/phase4-review-and-hardening-r1-findings.md`）；若任一风险坐实，转成新的 P0 修复项。
  - **阻断**：P0-2 依赖此项结论调整 fail-closed 判定源。

- **P0-2 发布门控升级为 runtime 层统一拦截（fail-closed）**（3-5 天）
  - 把 `runtime/ai/publish_gate.go` 的 YAML 直读升级为 runtime 资源级属性（参考现有 security policy 的读取路径）。
  - **拦截点**：
    - AI 工具（`metrics_view_list` / `metrics_view_get`）已有，改为 fail-closed。
    - Explore / Canvas 资源 list：`resource_list` resolver 层过滤未发布。
    - Explore / Canvas 直连查询：resolver 入口检查资源是否 published。
    - Embed / public URL：单独 review 是否豁免（一般外链应受同样的 gate）。
  - **豁免**：治理者（`ManageProject`）可访问未发布资源。
  - **验收**：手动用非治理账号访问未发布 explore/canvas/embed URL → 403 或 404；`publish.yaml` 缺失或损坏 → 全部资源被拒绝（fail-closed），治理者仍可访问；P1-4 spec 覆盖。

### P1（Phase 4 收尾，4c 内做）

- **P1-3 审计最小埋点（先落库）**（1 天）
  - 新表 `admin_audit_events(id, org_id, project_id, actor_user_id, event_type, target_id, payload_json, created_at)`。
  - `admin/audit/RecordAudit(ctx, event)` helper，写库失败仅记日志不阻断业务。
  - **必埋入口**：`publishProject` / `rollbackForOrgAndProject` / `SetFeatureAccess` / `SetOrgFeatureDefaults` / member add/remove / role change。
  - **验收**：mutating RPC 调用一次 → 表里出现一条事件；DB 查询能按 `org_id + created_at desc` 高效拉审计流水。

- **P1-4 三个 E2E spec**（2-3 天）
  - `web-admin/tests/portal-home.spec.ts`：业务账号登录 → 落门户首页 → 推荐问题存在 → 点击进入 chat 且带 `?q=` 预填。
  - `web-admin/tests/studio-publish.spec.ts`：治理者登录 → 修改 YAML → publish → 业务账号看到变化 → rollback → 业务账号看到旧版。
  - `web-admin/tests/feature-access.spec.ts`：管理员在矩阵关闭某用户 `accessChat` → 该用户 Chat tab 消失。
  - **验收**：CI 中三个 spec 全绿；每个 spec 覆盖 happy path + 1 个 fail path。

### P2（Phase 4 收尾，含在 4c）

- **P2-5 治理者「预览业务视图」入口 + Studio error boundary**（1 天）
  - Studio 顶部或用户菜单加「以业务视图预览」按钮 → 跳转 `/{org}/{project}`。
  - Studio 层 `+error.svelte` 捕获 edit-session 类报错，翻译为治理者可懂的文案（"工作台正在准备数据环境" / "环境启动失败，请联系管理员"），保留原始错误在开发者面板。
  - **验收**：治理者能一键切换到业务视图并返回；kill dev deployment 后 Studio 显示中文人话错误而非 500 堆栈。

- **P2-6 术语替换 + Requests 独立列表页**（1-2 天）
  - i18n 消息层：所有业务上下文（门户导航、Chat 空态、Boards 空态、错误提示）里的 `project` / `项目` / `-/edit` 等替换为「数据域」/「工作台」等业务语言（术语依据 `design/phase4-enterprise-app.md` §8）。
  - `web-admin/src/routes/[organization]/[project]/-/edit/studio/requests/+page.svelte`：独立需求列表，支持筛选（open/done）、批量标记完成、跳转对应对话上下文。Overview / Publish 的 `RequestsTodo` 加「查看全部 →」链接跳转此页。
  - **验收**：全站扫描无业务上下文残留裸英文技术词；需求条目 >5 时 Overview 页面布局不再挤爆。

### P3+（Phase 4 之后）

- `/settings/system` 审计日志 UI（读取 P1-3 埋的表）
- `/settings/ai`：LLM 供应商 / 密钥 / 连通性测试可写
- `/settings/domains` 独立化（业务概念完全脱离 project 术语）
- **Feature Access 后端双保险**（延后项，见 D5）：runtime API 层校验 `AccessChat / AccessDashboards / AccessReports / AccessAlerts` 拒绝越权
- Studio 路由从 `-/edit/studio` 迁到 `/studio/[domain]`，解耦 dev deployment 生命周期

### 实施过程中新增并已完成的项

- **P2-7 RouterAgent 对 DeepSeek 的 tool-calling 兼容性**（已修）：router agent 用「无 tools 的结构化输出」选 agent，DeepSeek 却把选择当 tool call 返回（`name="Agent choice"`），框架去执行不存在的 tool 而报错——**生产 ChatBI 走 router agent 入口时会失败**。修复在 deepseek 驱动：请求未声明 tools 时把返回的 tool call 规范化为携带 JSON 的 text block。真实 DeepSeek 端到端验证通过。
- **parser 冷启动 bug**（已修）：`ignorePathPrefixes` 只在 `Reparse` / `IsSkippable` 生效，初次全量 `parsePaths` 未过滤，导致每个启用门控的项目启动时都带一条 `/publish.yaml: resource type not specified` 解析错误。
- **R1-X 草稿区目录权限 bug**（已修）：见 `phase4-review-and-hardening-r1-findings.md`。

## 4. 已接受的技术债档案

以下项目**主动延后**，在客户方安全 / 交付 review 时可援引本文档说明是审慎权衡的结果，非遗漏：

| 债务 | 现状 | 风险 | 缓解 |
|---|---|---|---|
| Feature Access 后端双保险缺失 | 前端隐藏 tab，后端 runtime API 未校验 access_* | 垂直越权：用户直接调 API 或猜路由访问被隐藏功能 | 结合发布门控 fail-closed（P0-2），关键数据查询已经受语义层 security policy 与发布门控双重拦截；剩余风险主要是 reports/alerts 列表接口的 metadata 泄漏 |
| Studio 挂 `-/edit/studio` | 生命周期绑 dev deployment，URL 暴露 `-/edit` | 治理者遇到 dev deployment 问题时体验割裂 | P2-5 的 error boundary 消化 90% 场景 |
| `-/ai`、`-/dashboards` 等 legacy 路由保留 | 未删，仍可访问 | 内部路径不一致，可能被 bookmark 分发 | 门户 tab 已经指向新路由，legacy 路径视为内部实现 |
| `svelte-check --tsconfig` 模式约 585 个类型错误 | 深层类型债；**注意 CI 并不跑这个模式** | 无法启用更严格的类型门禁 | CI 实际跑 `--no-tsconfig`，已在 V-5 中清零；严格模式清理属独立课题 |

## 4b. 待验证清单（Verification Backlog）

已实现但**尚未在真实环境确认**的部分。每一项都对应一个具体的复现步骤，不是"再看一眼"级别的检查。

| # | 待验证项 | 状态 | 为什么重要 | 关联改动 |
|---|---|---|---|---|
| V-1 | Studio error boundary 的错误文案匹配 | 待验证 | 正则（`runtime not reachable` / `not ready` / `failed\|error\|unavailable`）是读代码推断的；兜不住就会漏出原始技术错误，等于 P2-5 白做 | `studio/+error.svelte` |
| V-2 | `?preview=1` 不被上游 layout 截胡 | 待验证 | `?preview` 语义只在 `[project]/+page.ts` 实现；`+layout.ts` 的 `maybeRedirectToEditableDeployment` / branch / welcome 重定向都可能抢先跳走 | `[project]/+page.ts` |
| V-3 | R1 全链路 docker-compose 人工走查 | 待验证 | R1-X 修复只在 unit test 层验证过；容器内 `USER stardata`(uid 1001) + `runtime_data` volume 的实际行为未确认。另需确认 publish/rollback 可靠触发 prod 重部署 | `repo_archive.go` |
| V-4 | 发布门控 HTTP 层端到端 | 待验证 | P0-2 只有单元级验证；未用非治理账号实际访问未发布 explore/canvas/embed URL 确认 403/404 | `runtime/publishgate.go` 及三处接入点 |
| V-5 | svelte-check 处理策略 | ✅ 已闭合 | **更正先前记录**：CI 跑的是 `--no-tsconfig`，只有 3 个既有错误（不是 585；585 来自本地 `npm run check` 的 `--tsconfig` 模式）。3 个已修复，web-admin / web-common 均 0 error | `CreateProjectForm.svelte`、`OrgUsersTable.svelte` |
| V-6 | `build:i18n` 锁定 node ≥20 | ✅ 已闭合 | node 18 下 paraglide 报 `crypto is not defined` 静默失败 | `scripts/check-node-version.js`、`package.json` |

> 原则：这些项在 Phase 4 收尾判定（§6）之前必须闭合 V-1 ~ V-4。

## 5. 执行顺序与依赖

```
[P1-3 审计埋点] ─┐
                ├─ 独立可并行
[P2-6 术语替换] ─┤
                ├─ 独立可并行
[P2-5 预览入口] ─┘

[P0-1 R1 验证] ── 阻断 ──▶ [P0-2 发布门控加固] ── 阻断 ──▶ [P1-4 E2E spec]
```

建议下一个 sprint 唯一焦点：**P0-1 → P0-2**。P1/P2 项可穿插并行由不同人推进。

## 6. 交付验收（Phase 4 收尾判定）

- [ ] P0-1 R1 验证脚本可重复执行，findings 文档产出
- [ ] P0-2 未发布资源在所有 API 路径下对非治理者返回 403/404
- [ ] P1-3 mutating RPC 全部有审计写入
- [ ] P1-4 三个 spec 在 CI 中稳定通过
- [ ] P2-5 治理者可一键预览业务视图；Studio 错误信息业务语言化
- [ ] P2-6 全站 i18n 扫描无业务上下文的裸技术词；Requests 有独立列表页
