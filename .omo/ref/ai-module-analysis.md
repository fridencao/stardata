# StarData AI 模块分析（Phase 2.1）

> 调研对象：`/Users/xinjian/Work/Project/RD/StarData`（fork 自 rilldata/rill，较新基线）
> 目的：为 Phase 2「智能问数 / NL→Metrics→Chart」提供改造地图
> 方法：Explore 子代理通读 `runtime/ai/` + Chat UI，主代理复核 `openai.go` 以确认 Provider 插入点

---

## ⚠️ 关键认知修正（相对于原实施计划）

原 `.omo/plans/star-data-platform.md` Phase 2 假设存在一个简单的 `generate.go` 单提示词模板做 text-to-SQL，并列出 `runtime/ai/` 旧结构（client.go / generate.go / metrics.go / chat.go / tools.go）。

**实际当前 fork 的 AI 模块是一个 agent 工具调用循环系统**（RouterAgent → AnalystAgent / DeveloperAgent / FeedbackAgent，配 MCP 风格工具，最终产出 metrics view **图表 spec** 或语义层查询），**不是简单的"NL 直接生成 SQL"**。计划里的 `generate.go` / `metrics.go` 等文件已不存在。后续改造必须基于真实代码，不能照搬计划里的文件路径。

---

## 一、`runtime/ai/` 模块结构

核心文件（均在 `runtime/ai/`）：

| 文件 | 职责 |
|---|---|
| `ai.go` | `Runner`（工具注册中心，`NewRunner` L46）、`Session`/`BaseSession`、`Complete()` 完成循环（L1142，工具调用循环 L1282–1399）、消息序列化 |
| `router_agent.go` | **NL 入口**，把用户 prompt 路由到子 agent |
| `analyst_agent.go` | **核心分析 agent**，组装 system/user prompt 并跑循环（`Handler` L102） |
| `developer_agent.go` / `feedback_agent.go` | 代码生成 / 反馈 agent |
| `create_chart.go` | **图表生成工具**（产出 metrics view 图表 spec） |
| `metrics_view_get.go` / `metrics_view_summary.go` / `metrics_view_query.go` / `query_sql.go` / `list_metrics_views.go` | 语义层查询工具 |
| `instructions/` | 提示词模板（embed 在 `data/` 下的 markdown） |
| `evals/` | agent 行为回归测试 fixture（`.yaml`） |

---

## 二、LLM Provider 抽象与配置

- 抽象接口在 **`runtime/drivers/ai.go:10`**：
  ```go
  type AIService interface {
      Complete(ctx, *CompleteOptions) (*CompleteResult, error)
  }
  ```
- 配置来源：`rill.yaml` 的 **`ai_connector`** 字段（`runtime/parser/parse_rillyaml.go:64`），运行时经 `runtime/connections.go:102` `Runtime.AI()` → `ResolveAIConnector()` 取得 handle。**无专用 env var**；通过 connector 配置（`api_key` / `base_url` / `model`）注入。

---

## 三、已有 Provider 与 DeepSeek 插入点

已有 3 个 AI driver（均 `ImplementsAI: true` + `drivers.Register` + `RegisterAsConnector`）：

- `runtime/drivers/openai/openai.go`
- `runtime/drivers/claude/claude.go`（Anthropic）
- `runtime/drivers/gemini/gemini.go`（Google）

**新增 DeepSeek 的精确插入点（已实施 ✅）**：复制 `runtime/drivers/openai/` → `runtime/drivers/deepseek/deepseek.go`，仅改：
- `init()` 注册名 `"openai"` → `"deepseek"`（`drivers.Register` + `RegisterAsConnector`）
- `Open()` 默认分支 `base_url` 默认填 `https://api.deepseek.com/v1`
- `getModel()` 默认 `"deepseek-chat"`（用户可配 `deepseek-reasoner`）
- `Driver()` 返回 `"deepseek"`、`Complete()` 的 `Provider` 返回 `"deepseek"`

DeepSeek 兼容 OpenAI API 格式，直接复用 `openai-go/v3` SDK，无需改接口层。

**生产引入点（blank import 触发 `init` 注册）**：`cli/cmd/runtime/start.go:49`、`cli/cmd/admin/start.go:46`（本次已加 deepseek 引入）。

**启用方式**：在 `rill.yaml` 配
```yaml
ai_connector: deepseek
connectors:
  deepseek:
    api_key: <YOUR_DEEPSEEK_KEY>
    # base_url 默认 https://api.deepseek.com/v1；model 默认 deepseek-chat
```

---

## 四、NL→查询（图表）路径

1. 前端问题 → HTTP SSE `POST /v1/instances/{id}/ai/complete/stream?stream=messages`
2. 后端 `runtime/server/chat.go:206` `Complete()` → L293 `session.CallTool(... ai.RouterAgentName ...)` 调用 `RouterAgent`（`router_agent.go`，入口 agent）
3. RouterAgent 路由到 `AnalystAgent.Handler`（`analyst_agent.go:102`）
4. 加载 **system prompt**：`analyst_agent.go:207` `instructions.Load("analysis.md", ...)`（`runtime/ai/instructions/data/analysis.md`）
5. **user prompt 模板**：`analyst_agent.go:291–387`（Go `text/template`，**全英文硬编码**）
6. 跑 `s.Complete(...)` 工具循环（L194），工具集：`ListMetricsViews` / `GetMetricsView` / `QueryMetricsViewSummary` / `QueryMetricsView` / `CreateChart`（L164–171）。LLM 通过工具调用产出 **metrics view 查询**或 **图表 spec**（不是原始 SQL；SQL 路径在 `query_sql.go` 另作工具）
7. 入口函数即 `RouterAgent` / `AnalystAgent.Handler`

---

## 五、语义层集成

模型/指标视图**不是一次性 catalog dump**，而是通过工具按需注入 prompt：

- `GetMetricsView`（`metrics_view_get.go`）→ 把某 metrics view 的 spec 序列化为 JSON，作为工具结果回流给 LLM
- `QueryMetricsViewSummary`（`metrics_view_summary.go`）提供数据时间跨度等
- 数据源来自 `t.Runtime.Controller(ctx, instanceID)` 读取 catalog（如 `analyst_agent.go:393` `ctrl.Get` 取 Explore/MetricsView 资源）
- 图表 schema 来自 `metricsview.ChartsJSONSchema`（`create_chart.go:35`）
- 用户可附带过滤条件：`where` / `where_per_metrics_view` 经 `metricsview.ExpressionToSQL` 转 SQL 注入模板（`analyst_agent.go:272–287`）

---

## 六、Chat / Ask UI

- 前端目录 `web-common/src/features/chat/`：`DashboardChat.svelte`、`ProjectChat.svelte`、`DeveloperChat.svelte`、`layouts/`（fullpage/inline/sidebar）、`core/conversation.ts`（核心）
- **前端→后端调用点**：`web-common/src/features/chat/core/conversation.ts:354`
  ```ts
  baseUrl = `${host}/v1/instances/${instanceId}/ai/complete/stream?stream=messages`  // SSE 流式
  ```
- **后端 handler**：`runtime/server/chat.go`
  - `CompleteStreaming`（L325，gRPC streaming）
  - `CompleteStreamingHandler`（L464，HTTP SSE 适配器）
  - 二者最终都进入 `ai.Session` → `RouterAgent`
- 返回：流式 `Message` proto（`runtimev1.Message`），图表以 `create_chart` 工具结果的 JSON spec 回传，前端 `messages/chart/ChartBlock.svelte` 渲染
- 本地路由：`web-local/src/features/chat/LocalFullPageChat.svelte`

---

## 七、硬编码英文 / 需中文化位置

- `runtime/ai/instructions/data/analysis.md` —— 整个 system prompt（必改）
- `analyst_agent.go:291–387` userPrompt 模板（全英文）
- `create_chart.go:418–1270` `createChartDescription`（巨大英文工具说明）
- 各 agent 的 Meta 串：`"Routing prompt..."` / `"Routed prompt"`；`analyst_agent.go:81–82` `"Analyzing..."` / `"Analysis completed"`；`create_chart.go:51–52` `"Creating chart..."` / `"Created chart"`；`create_chart.go:193` `"Chart created successfully: %s"`
- `ai.go` 内错误/截断提示（如 L1628 `"... [%d messages omitted for brevity] ..."`）
- 前端：`web-common/src/features/chat/core/messages/tools/tool-display-names.ts`、`feedback/feedback-categories.ts`、`ChatInput.svelte` / `Messages.svelte` 等用户可见文案

---

## 八、已有参考文档（无需另写架构 README）

- 指令即文档：`runtime/ai/instructions/data/` 含 `analysis.md`（分析 agent 角色/流程）、`development.md`（开发 agent）、`AGENTS.md`、`resources/{model,metrics_view,explore,canvas,connector,rillyaml,theme}.md`（各资源 schema 与 agent 用法）
- `runtime/ai/evals/*.yaml`（如 `AnalystBasic`、`AnalystCharts_*`、`RouterAgent`、`DeveloperShopify`）—— 描述期望 agent 行为的回归测试集，是理解「正确答案长什么样」的最佳参考
- 前端 `web-common/src/features/chat/core/messages/README.md` 描述消息块渲染

---

## 九、Phase 2 推荐执行顺序（基于现实）

1. **2.2.1 新增 DeepSeek Provider（✅ 已完成）**：复制 openai→deepseek，默认 base_url/model，`rill.yaml` 配 `ai_connector: deepseek` 即用。这是让中文 LLM 接入、后续中文提示词生效的前置条件。
2. **2.2.2 多 Provider 路由（配置化）**：沿用 `rill.yaml` `ai_connector` 机制，新增 qwen/glm 等 driver（或同一 deepseek driver 指向不同 `base_url`）；runtime 已按 connector 名解析，基本免费。
3. **2.2.3 中文提示词工程**：改写 `analysis.md` + `analyst_agent.go:291–387` 模板 + `createChartDescription` 为中文（度量/维度中文名映射、日期表达「上个月/同比/环比」、聚合语义）。配合 2.2.4。
4. **2.2.4 语义层中文映射**：model/metrics 定义增加 `label_cn`，`resources/*.md` 与 `GetMetricsView` 注入中文别名。
5. **2.2.5 LLM 输出结构化**：`CompleteOptions.OutputSchema`（已有 JSON schema 机制）用于把回答结构化为时间/维度/度量。
6. **2.3 Chat UI**：可先 HTML 原型确认交互，再集成 web-common；流式/SSE 已通，主要补中文文案与建议问题（`SuggestionChips`）。
7. **2.4 / 2.5 增强与可追溯**：多轮下钻、模糊纠正、异常检测、Chain-of-Thought 记录、查看/编辑生成查询。

---

## 十、风险与缓解

- 提示词中文化是精度关键，需多轮迭代（见计划风险表：中文 NL→SQL 精度不达预期）。
- agent 系统比「单提示词」复杂，改动面集中在 `analyst_agent.go` + `instructions/data`，需对照 `evals/*.yaml` 回归。
- 全量 `go build ./...` 受 `testcontainers` 代理问题限制（仅 test/helper 包），用 scoped build 验证（本次 deepseek driver 已绿：`go build ./runtime/drivers/deepseek/... ./cli/cmd/runtime/... ./cli/cmd/admin/...`）。
