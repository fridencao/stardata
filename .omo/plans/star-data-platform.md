# StarData 智能 BI 平台 · 项目实施计划

> **基底项目**: [rilldata/rill](https://github.com/rilldata/rill) (Apache-2.0) — 22.7k stars, Go + SvelteKit
> **策略**: Fork + 深度定制（保留引擎层，增强 AI 模块，定制前端）
> **前端**: 保留 Svelte，品牌 UI 定制
> **部署**: 私有化部署（Docker Compose / 单二进制）
> **AI**: 增强现有 AI 模块，接入中文 LLM

---

## 一、项目结构

```
StarData/
├── .omo/                    # 工作计划与状态
├── cli/                     # Fork: CLI 入口（改品牌名）
├── runtime/                 # Fork: Go 核心引擎（改少量代码）
│   ├── ai/                  # ★ 重点改造：增强 AI 模块
│   ├── security/            # ★ 改造：替换认证系统
│   └── drivers/             # 保留：数据源驱动
├── web-common/              # ★ 改造：组件库定制主题
├── web-local/               # ★ 改造：主 UI 应用
├── web-admin/               # 精简：管理后台（私有化场景可砍）
├── web-integration/         # 可选：嵌入 SDK
├── deploy/                  # ★ 新增：私有化部署配置
│   ├── docker-compose.yml
│   ├── config.example.yaml
│   └── nginx/
└── proto/                   # 保留：API 定义（按需扩展）
```

---

## 二、Phase 0：基础环境搭建（预期 1-2 天）

### 0.1 Fork 并初始化仓库

```
目标: 将 rilldata/rill fork 到组织仓库，clone 到本地可开发
```

| # | 操作 | 验收标准 |
|---|---|---|
| 0.1.1 | 在 GitHub 上 fork rilldata/rill 到组织账户 | 组织仓库就绪 |
| 0.1.2 | Clone fork 到 `StarData/` 目录 | 本地代码完整 |
| 0.1.3 | 重命名模块为 `github.com/<org>/stardata`（go mod） | Go 编译通过 |
| 0.1.4 | 更新 CLI 入口名 `rill` → `stardata` | CLI 命令生效 |
| 0.1.5 | 清理上游 CI/CD 配置，替换为自己的 | 不再依赖 Rill Cloud |
| 0.1.6 | 本地执行 `stardata start` 验证全链路 | 能在 localhost:9009 打开 UI |

### 0.2 开发环境就绪

```
目标: 团队成员能拉代码即开发
```

| # | 操作 | 验收标准 |
|---|---|---|
| 0.2.1 | 梳理 Go 依赖（go.sum）和 Node 依赖（package.json） | `go build` + `npm install` 通过 |
| 0.2.2 | 编写 `.env.development` 开发环境变量 | 本地热重载正常 |
| 0.2.3 | 更新 README / CONTRIBUTING | 新成员可快速上手指南 |
| 0.2.4 | （可选）配置 VSCode DevContainer | 一键拉起开发环境 |

### 0.3 理解 Rill 内部运行机制（调研输出）

```
目标: 团队对关键模块有共识，后续改造不盲动
```

- 阅读 `runtime/controller.go` — 资源生命周期管理
- 阅读 `runtime/ai/*.go` — AI 查询流程
- 阅读 `runtime/metricsview/` — 语义层引擎
- 阅读 `web-common/src/` — 组件体系
- **输出**: Rill 内部架构文档（团队 Wiki）

### Markers for Phase 0

✅ 本地能跑通 `start`，数据能导入、仪表盘能渲染
✅ 项目名、CLI 名已改
✅ Go 和 Node 依赖均能正常构建

---

## 三、Phase 1：私有化部署底座（预期 5-7 天）

### 1.1 认证系统改造

```
现状: 依赖 Rill Cloud 的 GitHub OAuth 认证
目标: 替换为支持 JWT/OIDC 的自认证系统，适配私有化部署
```

**方案**: 在 `runtime/` 中新增认证中间件，接入通用 OIDC 协议

| # | 文件 | 操作 |
|---|---|---|
| 1.1.1 | `runtime/security/` | 新增包，实现 `Authenticator` 接口 |
| 1.1.2 | `runtime/security/auth.go` | JWT 签发 + 验证（支持 RS256/ES256） |
| 1.1.3 | `runtime/security/oidc.go` | OIDC 集成（支持 Keycloak / Authing / 自建） |
| 1.1.4 | `runtime/security/middleware.go` | gRPC 拦截器 + HTTP 中间件 |
| 1.1.5 | `runtime/server/` | 注册登录/登出/token 刷新 API |
| 1.1.6 | web-local | 添加登录页、token 管理 UI |
| 1.1.7 | `runtime/security.go` | 将原有的 `SecurityClaims` 迁移到新系统 |

**支持的功能**:
- 用户名/密码登录（本地账户）
- OIDC SSO（对接企业 IdP）
- API Token 机制（供程序化访问）
- 角色权限（admin / viewer / editor）

### 1.2 部署配置体系

```
目标: 让运维人员通过 YAML 配置文件即可部署，不需要了解代码
```

| # | 操作 | 内容 |
|---|---|---|
| 1.2.1 | 新增 `deploy/` 目录 | 放置全部部署配置 |
| 1.2.2 | `deploy/docker-compose.yml` | stardata + 可选 Keycloak + 可选 MinIO |
| 1.2.3 | `deploy/config.example.yaml` | 配置模板，注释完整 |
| 1.2.4 | `deploy/Dockerfile` | 优化：多阶段构建，减小镜像体积 |
| 1.2.5 | `deploy/nginx.conf` | 反向代理 + SSL termination |
| 1.2.6 | 移除 `rill deploy` 对 Cloud 的依赖逻辑 | 删掉相关 CLI 命令 |
| 1.2.7 | 新增 `stardata init` 命令 | 初始化配置向导 |

**config.yaml 核心结构**:
```yaml
server:
  http_port: 9080
  grpc_port: 9090
  external_url: https://star-data.company.com

auth:
  provider: oidc       # local / oidc / jwt
  jwt_secret: ...
  oidc:
    issuer_url: ...
    client_id: ...
    client_secret: ...

database:
  connector: duckdb    # 或 clickhouse
  path: /data/stardata

storage:
  type: local          # local / s3 / minio
  path: /data/storage
```

### 1.3 前端品牌定制

```
目标: 客户看不出这是 Rill fork，UI 有自己的品牌感
```

| # | 操作 | 文件范围 |
|---|---|---|
| 1.3.1 | 修改主题色、字体、间距 tokens | `web-common/src/theme/` |
| 1.3.2 | 替换 Logo、Favicon | 各 UI 入口 |
| 1.3.3 | 修改产品名称、文档链接 | 全局搜索 "Rill" → "StarData" |
| 1.3.4 | 定制登录页面 | `web-local/src/routes/login/` |
| 1.3.5 | 定制仪表盘空状态/新手引导 | `web-common/src/components/` |
| 1.3.6 | 移除 Rill Cloud 相关 UI 元素 | 管理后台链接等 |

### 1.4 多租户隔离（私有化 MVP 可选）

```
优先级: 低 — 第一阶段先单租户跑通
```

- 数据目录按 `tenant_id` 隔离
- 用户与租户关联
- 数据源配置按租户隔离
- 行级安全（RLS）策略分层

### Markers for Phase 1

✅ 用户能注册/登录/登出
✅ OIDC 可接入企业 IdP
✅ `docker compose up` 一键拉起
✅ 品牌替换完成，前端外观统一
✅ 管理页面链接指向自有文档

---

## 四、Phase 2：智能问数（预期 2-3 周）

这是 StarData 的核心差异点，整个 Phase 2 围绕 **NL→Metrics→Chart** 链条展开。

### 2.1 深入理解 Rill AI 模块

```
文件: runtime/ai/
```

先彻底理解上游 AI 模块的设计：

```go
runtime/ai/
├── client.go          # AI provider 客户端抽象
├── completion.go      # LLM 调用封装
├── generate.go        # 提示词构建
├── metrics.go         # 指标相关 AI 能力
├── chat.go            # 对话管理
├── tools.go           # function calling tools
└── version.go         # 版本
```

**需要调研的关键问题**:
- 它如何把语义层定义翻译成 LLM 能理解的上下文？
- 提示词模板是否硬编码了英文？
- 是否支持多轮对话 / 追问 / 澄清？
- 错误处理和降级策略是什么？

**输出**: AI 模块源码分析文档，标注需要改造的位置

### 2.2 中文 NL→SQL 能力

```
目标: 用户用中文自然语言提问，系统返回数据图表
```

| # | 操作 | 详情 |
|---|---|---|
| 2.2.1 | 集成中文 LLM Provider | 新增 `runtime/ai/provider/deepseek.go` — 兼容 OpenAI API 格式 |
| 2.2.2 | 可选多 Provider 路由 | 配置化选择 DeepSeek / Qwen / GLM / 私有模型 |
| 2.2.3 | 中文提示词工程 | 重写 `generate.go` 模板：度量/维度中文名映射、日期表达（"上个月""同比""环比"）、聚合语义 |
| 2.2.4 | 语义层中文映射 | 模型定义中增加 `label_cn` 字段，AI 上下文自动注入中文别名 |
| 2.2.5 | LLM 输出解析 | 将 NL 回答结构化输出为 JSON schema 再转为查询 |

**示例对话流程**:
```
用户: "上个月各渠道的销售额是多少？"
  → AI 解析: 时间=上个月, 维度=渠道, 度量=销售额
  → 查询语义层: metrics_sql
  → 返回: 柱状图 + 自然语言总结

用户: "哪个渠道增长最快？"
  → AI 识别是追问: 复用上下文，增加增长率计算
  → 返回: 排序列 + 高亮标识
```

### 2.3 AI 对话 UI 组件

```
目标: 在 web-common 中建设 Chat 交互组件
```

| # | 组件 | 说明 |
|---|---|---|
| 2.3.1 | `ChatInput.svelte` | 多行输入框 + 快捷提问入口 |
| 2.3.2 | `ChatMessage.svelte` | 消息气泡：用户文本 / AI 回答 + 图表 |
| 2.3.3 | `ChatHistory.svelte` | 对话历史列表 |
| 2.3.4 | `ChartCard.svelte` | 图表展示卡片（嵌入已有 Chart 组件） |
| 2.3.5 | `SuggestionChips.svelte` | 自动推荐问题（"查看本月趋势""Top 10 客户"） |
| 2.3.6 | 新增对话路由 | `web-local/src/routes/chat/` |

**交互设计要点**:
- 流式输出：AI 边生成边展示文字 + 图表逐步渲染
- 追问时自动携带上下文（当前提问的维度/度量/时间范围）
- 每次 AI 回答后提供"修改查询"按钮，可手动编辑生成了的 SQL
- 快捷入口：从仪表盘直接"问这个图表"进入对话模式

### 2.4 智能查询增强

```
目标: 让 AI 不只是"查数"，而是能理解业务语义
```

| # | 能力 | 实现方式 |
|---|---|---|
| 2.4.1 | 不完整提问引导 | 缺时间范围 → 追问"您想看哪个时间段？" |
| 2.4.2 | 模糊查询纠正 | "销售额" → 可能匹配 total_sales / net_sales → AI 自动选择 |
| 2.4.3 | 多轮聚合叠加 | "按渠道看" → "再按地区细分" → 识别维度下钻 |
| 2.4.4 | 数据解读 | 自动添加环比/同比对比，并生成一句话总结 |
| 2.4.5 | 异常检测 | 自动识别数据中的异常点并标注 |

### 2.5 查询可追溯性

```
目标: 用户信任 AI 给出的数据，能审查和修正
```

| # | 操作 |
|---|---|
| 2.5.1 | 每次查询记录完整的 AI Chain-of-Thought |
| 2.5.2 | 用户可展开查看"AI 生成的 SQL/查询条件" |
| 2.5.3 | 支持一键"编辑查询" → 切换到高级模式（SQL 编辑器） |
| 2.5.4 | 查询历史可回溯、可复用 |

### Markers for Phase 2

✅ 用户能用中文提问并得到图表回答
✅ 多轮对话、追问、上下文保持正常
✅ 追问时维度下钻、时间范围叠加正确
✅ 查询结果可溯源（查看 AI 推理过程和生成的查询）
✅ 支持切换 LLM 供应商（配置化）

---

## 五、Phase 3：产品化打磨（持续）

### 3.1 数据源接入

| # | 功能 |
|---|---|
| 3.1.1 | UI 配置数据源连接（DuckDB / ClickHouse / MySQL / PostgreSQL） |
| 3.1.2 | 网络隔离环境的数据源代理连接 |
| 3.1.3 | 数据源连接测试与健康检查 |
| 3.1.4 | 数据源连接池管理 |

### 3.2 分享与协作

| # | 功能 |
|---|---|
| 3.2.1 | 仪表盘分享（链接 + 密码） |
| 3.2.2 | 仪表盘导出（PDF / PNG / CSV） |
| 3.2.3 | 定时报告（邮件 / 企微 / 钉钉推送） |
| 3.2.4 | AI 问答结果分享 |

### 3.3 运维能力

| # | 功能 |
|---|---|
| 3.3.1 | 操作审计日志 |
| 3.3.2 | 查询性能监控（慢查询日志） |
| 3.3.3 | LLM 调用统计与成本追踪 |
| 3.3.4 | 系统健康检查 API |
| 3.3.5 | Prometheus + Grafana 监控集成 |

### 3.4 企业特性

| # | 功能 |
|---|---|
| 3.4.1 | LDAP/AD 用户同步 |
| 3.4.2 | 资源级别的权限控制（RBAC） |
| 3.4.3 | 审计日志导出到 SIEM |
| 3.4.4 | 高可用部署方案（多副本） |
| 3.4.5 | 数据备份与恢复 |

---

## 六、架构决策记录 (ADR)

| # | 决策 | 依据 |
|---|---|---|
| ADR-001 | 保留 Svelte 前端，不重写 | 最小化 diff，复用 Rill 上游更新；团队接受 Svelte |
| ADR-002 | 认证系统在 runtime 层实现，不依赖外部服务 | 私有化部署必需能力；OIDC 协议标准兼容 |
| ADR-003 | AI 模块复用 Rill 架构，新增 Provider 插件 | Rill 的 `runtime/ai/` 结构清晰，扩展点预留 |
| ADR-004 | 语义层作为 NL→SQL 的数据基础 | Rill metricsview 已是最佳实践，无需再造轮子 |
| ADR-005 | Docker Compose 为首选部署方式 | 单机场景覆盖 80% 私有化需求；复杂场景可升 K8s |
| ADR-006 | LLM 路由设为配置化，不硬编码 | 客户可能使用国产模型或私有部署的 LLM |
| ADR-007 | 对话状态由前端维护，API 无状态 | 保持后端无状态便于水平扩展 |

---

## 七、风险与缓解措施

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| Rill 上游版本 API 变更导致 merge 冲突 | 中 | 高 | Fork 后锁定基线版本，只 cherry-pick 安全修复 |
| 中文 NL→SQL 精度不达预期 | 中 | 高 | Phase 2 安排充分的提示词迭代周期；做好人机协作（用户可编辑 SQL） |
| DuckDB 承载多租户并发性能不足 | 中 | 中 | 驱动层抽象，可切换到 ClickHouse / StarRocks |
| 团队 Svelte 学习曲线 | 低 | 中 | Phase 0 安排 Svelte 快速上手；web-common 组件库模式复用 |
| 私有化客户对部署复杂度的抱怨 | 中 | 中 | Docker Compose 必须做到一条命令拉起来；提供快速开始脚本 |

---

## 八、依赖与外部资源

| 依赖 | 用途 | 替代方案 |
|---|---|---|
| DuckDB | 默认 OLAP 引擎 | ClickHouse / StarRocks / Apache Doris |
| DeepSeek API / Qwen API | 中文 LLM | 任意兼容 OpenAI API 的模型 |
| OIDC Provider (Keycloak) | 企业 SSO | Authing / 自建 / LDAP |
| Docker + Docker Compose | 部署 | K8s Helm Chart |

---

## 九、工作量估算

| Phase | 预估人天 | 核心产出 |
|---|---|---|
| Phase 0 — 基础环境 | 1-2 人天 | fork 就绪，本地可运行 |
| Phase 1 — 私有化底座 | 5-7 人天 | 认证/部署/品牌就绪 |
| Phase 2 — 智能问数 | 10-15 人天 | AI 对话功能可用 |
| Phase 3 — 产品化 | 持续迭代 | 企业特性按需推进 |

**总计 MVP（Phase 0 + 1 + 2）**: 约 16-24 人天（一个工程师 3-4 周）

---

## 十、执行建议

1. **先跑通再改造**: Phase 0 必须完整验证本地全链路后再进入 Phase 1，避免"建在流沙上"
2. **小步提交**: 每个子任务一个 commit，标注 `Rill upstream #xxxx` 便于 tracking
3. **AI 模块先调研再改**: Phase 2 需要花至少 1 天完整通读 `runtime/ai/` 源码，写分析文档，再动手
4. **原型驱动**: Phase 2.3 的对话 UI 可以先做一个独立 HTML 原型确认交互，再集成到 web-common
5. **持续 sync**: 定期拉取 Rill 上游更新，评估关键修复（非必需），保持可 merge 状态
6. **每次启动会话前**: 执行 `stardata start` 确认基线未被破坏

---

*本计划为 v1.0，随项目深入可调整。关键变更需更新此文档。*
