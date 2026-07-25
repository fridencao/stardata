<h3 align="center">StarData — 智能问数 · 智能 BI</h3>

<p align="center">
  Based on <a href="https://github.com/rilldata/rill">rilldata/rill</a> (Apache-2.0)
</p>

<p align="center">
  <a href="LICENSE.md" target="_blank">
    <img src="https://img.shields.io/github/license/fridencao/stardata.svg" alt="GitHub license">
  </a>
  <a href="https://github.com/fridencao/stardata/releases" target="_blank">
    <img src="https://img.shields.io/github/tag/fridencao/stardata.svg" alt="GitHub tag">
  </a>
  <a href="https://github.com/fridencao/stardata/commits" target="_blank">
    <img src="https://img.shields.io/github/commit-activity/y/fridencao/stardata.svg" alt="GitHub commit activity">
  </a>
</p>

---

**StarData** 是一个面向中文用户的智能 BI 平台，基于 Rill 开源项目定制开发。

核心能力:
- **智能问数** — 用中文自然语言提问，AI 自动完成数据查询与分析
- **语义层** — 统一的维度/度量定义体系
- **高性能 OLAP** — 基于 DuckDB / ClickHouse，亚秒级查询
- **私有化部署** — Docker Compose 一键拉起，支持 OIDC 企业认证

## Get Started

```bash
# From source
go build -o stardata cli/main.go
./stardata start my-project        # create a project and open the UI
```

### Scaffold a project

Use `stardata init` to scaffold a project interactively:

```
➜ stardata init
? Project name my-project
? OLAP engine duckdb
? Agent instructions claude

Created a new StarData project at ~/my-project
Added Claude instructions in .claude and .mcp.json

Success! Run the following command to start the project:

  stardata start my-project
```

## Why StarData?

- **Build with agents** — BI-as-code (YAML + SQL) means coding agents can author projects, dashboards, and security policies end-to-end
- **Semantic layer** — Single source of truth for dimensions, measures, and time grains — defined in YAML, generating SQL at query time against your OLAP engine
- **AI-powered analytics** — Conversational BI in Chinese; AI agents connect via MCP server
- **Real-time performance** — Sub-second queries at any scale; ClickHouse for billions of rows, DuckDB for smaller datasets and fast iteration
- **Private deployment** — Docker Compose or single binary, fully on-premise

## Capabilities

### Local Development

- [**Connectors**](https://docs.rilldata.com/build/connectors/) — S3, GCS, databases, and 20+ sources
- [**OLAP Engines**](https://docs.rilldata.com/developers/build/connectors/olap) — Managed ClickHouse or DuckDB included, or connect an external engine (ClickHouse Cloud, Druid, Pinot, MotherDuck)
- [**SQL Models**](https://docs.rilldata.com/build/models/) — Transform raw data with SQL, join models together
- [**Data Profiling**](https://docs.rilldata.com/build/models) — Instant column stats and distributions
- [**Incremental Ingestion**](https://docs.rilldata.com/build/models/incremental-models) — Load only new data on each run to keep large datasets current without full refreshes
- [**Semantic Layer**](https://docs.rilldata.com/build/metrics-view/) — Dimensions, measures, and time grains in YAML
- [**Row Access Policies**](https://docs.rilldata.com/build/metrics-view/security) — Per-user, per-group data access control
- [**Local Dashboards**](https://docs.rilldata.com/build/dashboards) — Preview and explore dashboards locally

### Deployment

- **Docker Compose** — One-click private deployment
- **OIDC Authentication** — Enterprise SSO (OIDC/LDAP)
- **Conversational BI** — Ask questions in natural language (Chinese supported)
- **MCP Server** — Connect AI agents to your semantic layer
- **Custom APIs & Embedding** — Expose metrics via REST or embed dashboards
- **Alerts & Reports** — Threshold alerting, code-defined or UI-defined

## How It Works

Define everything in code — models, metrics, dashboards — and Rill handles the rest.

**1. Connect data** — `models/events.yaml`

```yaml
type: model
connector: duckdb
materialize: true

sql: |
  select * from read_parquet('gs://rilldata-public/auction_data.parquet')
```

**2. Define metrics** — `metrics/events_metrics.yaml`

```yaml
version: 1
type: metrics_view
model: events
timeseries: timestamp

dimensions:
  - name: country
    column: country
  - name: device
    column: device_type

measures:
  - name: total_events
    expression: count(*)
  - name: revenue
    expression: sum(price * quantity)
    description: Total revenue
```

**3. Create a dashboard** — `dashboards/events_explore.yaml`

```yaml
type: explore

display_name: "Events Dashboard"
metrics_view: events_metrics

dimensions: "*"
measures: "*"
```

**4. Deploy**

```bash
stardata start my-project        # run locally
```

Your metrics view is immediately queryable — add YAML files to configure dashboards, alerts, and custom APIs.

## Learn More

- [Rill Documentation](https://docs.rilldata.com/) — upstream project docs

## Contributing

See our [Contributing Guide](CONTRIBUTING.md) to get started.
