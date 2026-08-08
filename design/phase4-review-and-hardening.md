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
| D5 | Feature Access 后端双保险 | ~~延后至 Phase 4 之后~~ → **已在 Phase 4 内实现并完成真实栈验证**（见 §4c） | 用户「现在就做」 |
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
- Studio 路由从 `-/edit/studio` 迁到 `/studio/[domain]`，解耦 dev deployment 生命周期

### 实施过程中新增并已完成的项

- **P2-7 RouterAgent 对 DeepSeek 的 tool-calling 兼容性**（已修）：router agent 用「无 tools 的结构化输出」选 agent，DeepSeek 却把选择当 tool call 返回（`name="Agent choice"`），框架去执行不存在的 tool 而报错——**生产 ChatBI 走 router agent 入口时会失败**。修复在 deepseek 驱动：请求未声明 tools 时把返回的 tool call 规范化为携带 JSON 的 text block。真实 DeepSeek 端到端验证通过。
- **parser 冷启动 bug**（已修）：`ignorePathPrefixes` 只在 `Reparse` / `IsSkippable` 生效，初次全量 `parsePaths` 未过滤，导致每个启用门控的项目启动时都带一条 `/publish.yaml: resource type not specified` 解析错误。
- **R1-X 草稿区目录权限 bug**（已修）：见 `phase4-review-and-hardening-r1-findings.md`。

## 4. 已接受的技术债档案

以下项目**主动延后**，在客户方安全 / 交付 review 时可援引本文档说明是审慎权衡的结果，非遗漏：

| 债务 | 现状 | 风险 | 缓解 |
|---|---|---|---|
| Studio 挂 `-/edit/studio` | 生命周期绑 dev deployment，URL 暴露 `-/edit` | 治理者遇到 dev deployment 问题时体验割裂 | P2-5 的 error boundary 消化 90% 场景 |
| `-/ai`、`-/dashboards` 等 legacy 路由保留 | 未删，仍可访问 | 内部路径不一致，可能被 bookmark 分发 | 门户 tab 已经指向新路由，legacy 路径视为内部实现 |
| `svelte-check --tsconfig` 模式约 585 个类型错误 | 深层类型债；**注意 CI 并不跑这个模式** | 无法启用更严格的类型门禁 | CI 实际跑 `--no-tsconfig`，已在 V-5 中清零；严格模式清理属独立课题 |

## 4b. 待验证清单（Verification Backlog）

已实现但**尚未在真实环境确认**的部分。每一项都对应一个具体的复现步骤，不是"再看一眼"级别的检查。

**全部已在真实 docker-compose 栈上闭合**（环境：postgres:15 + Keycloak OIDC + admin + runtime + web-admin + nginx，Playwright 驱动真实 Chromium）。

| # | 待验证项 | 状态 | 结论 |
|---|---|---|---|
| V-1 | Studio error boundary 的错误文案匹配 | ✅ 已验证 | 停 runtime 制造故障 → 实际走的是上层「分支已休眠 + 恢复分支」专属 UX，boundary 不触发。**结论：不需改**，真实体验比兜底文案更好 |
| V-2 | `?preview=1` 不被上游 layout 截胡 | ✅ 已验证 | 浏览器实测：无 preview → 重定向 Studio；`?preview=1` → 停在门户首页，未被 `+layout.ts` 截胡 |
| V-3 | R1 全链路 docker-compose 人工走查 | ✅ 已验证 | edit session 创建 / publish v1 / publish（修复后）/ rollback / 多次 archive 切换后 Studio 存活 / **内容 diff（改 displayName → publish → prod 生效 → rollback → 回退）**；容器内 draft 目录 0755。过程发现并修复 2 个 P0（见下） |
| V-4 | 发布门控 HTTP 层端到端 | ✅ 已验证 | prod JWT 直连 runtime：published→200 / draft→403 / ListResources 隐藏 draft / 数据查询 draft→403；**外加 fail-closed**：破坏 publish.yaml → 三条路径全 4xx/5xx 拒绝，恢复后正常 |
| V-5 | svelte-check 处理策略 | ✅ 已闭合 | CI 跑 `--no-tsconfig`，3 个既有错误已修，web-admin / web-common 均 0 error |
| V-6 | `build:i18n` 锁定 node ≥20 | ✅ 已闭合 | 加 prebuild 守卫 + engines.node，node 18/22 双向实测 |

### V-3 / V-4 真实栈走查额外发现并修复的 3 个 P0（单元测试均无法发现）

| # | 问题 | commit |
|---|---|---|
| R1-Y | `UpdateProject` 无条件跑 primary-branch 守卫 → 只要有 dev session，publish 恒 500 死锁。**发布模型对 archive 项目 100% 不可用** | `ca45ae390` |
| R1-X 第二层 | `untar` 用 file mode(0644) 建嵌套目录 → 第二次 publish 触发 re-sync 时 `dashboards/*.yaml` EACCES | `e92ddaf2c` |
| （构建） | `Dockerfile.admin` 多余 apt 依赖在受限网络 404 → 镜像构建 exit 100 | `755fa4bdf` |

> 原则：这些项在 Phase 4 收尾判定（§6）之前必须闭合 V-1 ~ V-4 —— **已全部闭合**。

## 4c. D5 Feature Access 后端双保险（已实现并验证）

原计划延后（见 §3 P3+），后按用户要求在 Phase 4 内完成。

**实现**（commit `4d2b27a4b`）

- `runtime/security.go`：新增 `ReadDashboards(0x1C) / ReadReports(0x1D) / ReadAlerts(0x1E)` 三个 permission，并入 `AllPermissions`。
- `runtime/featureaccess.go`：`CheckFeatureAccess` 按 resource kind → permission 映射（`Explore`/`Canvas` → ReadDashboards，`Report` → ReadReports，`Alert` → ReadAlerts）。`MetricsView` / `Model` / `Source` / `API` **故意不门控**——它们是共享底座，一旦门控会连带打断 Chat 与语义层。
- `runtime/resources.go`：`ApplySecurityPolicy` 里紧跟发布门控之后调用，复用同一个 metadata 收口点，`ListResources` / `GetResource` / resolver 全覆盖。
- `admin/server/runtime_jwt.go`：按 `projectPermissions.Access*` 授予对应 permission；`AccessChat=false` 时移除 `UseAI`。**没有 project permissions 的 token（magic link / embed / service attribute override）保持旧行为**——全功能开放，避免静默破坏既有集成。

**真实栈验证**（docker-compose，org `OCBC` / project `retail` / instance `45c5…deee`）

管理员身份无法验证：`admin/permissions.go` 对 `ManageProject` 用户提前返回并把所有 `Access*` 置 true（设计如此，防止管理员把自己锁在外面）。故新建非管理员身份 `viewer@stardata.local`（org viewer + project viewer）。

以 `org_feature_defaults` 设 `dashboards=false`，同一 viewer 身份两次取 JWT 对照：

- `dashboards=false` → JWT `ins` = `[25,23,21,27,29,30]`（**28=ReadDashboards 缺失**，29/30 在）；`GetResource(Explore/published_explore)` → **403 action not allowed**；`ListResources` 只返回 `MetricsView/published_mv`、`Model/orders`、`ProjectParser/parser`，**Explore 被过滤**；`MetricsView/published_mv` → **200**（未被门控，符合设计）。
- `dashboards=true` → JWT `ins` = `[25,23,21,27,28,29,30]`；同一个 Explore → **200**，`ListResources` 出现 `Explore/published_explore`。

单向翻转再翻回，确认因果关系是 feature flag 本身而非其他变量。单元测试 `runtime/featureaccess_test.go` 8 个 subtest 全绿。

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
