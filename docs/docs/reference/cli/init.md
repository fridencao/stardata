---
note: GENERATED. DO NOT EDIT.
title: stardata init
---
## stardata init

Initialize a new StarData project

### Synopsis

Initialize a new StarData project. Use flags to customize the project or run interactively to be prompted for each option.

Available example projects:
  - stardata-cost-monitoring (duckdb)
  - stardata-github-analytics (duckdb)
  - stardata-openrtb-prog-ads (duckdb)


```
stardata init [<path>] [flags]
```

### Examples

```
  # Interactive initialization (prompts for all options)
  stardata init

  # Create an empty DuckDB project with Claude agent instructions
  stardata init my-project --olap duckdb --agent claude

  # Add Claude agent instructions to an existing StarData project
  stardata init ./existing-project --agent claude
```

### Flags

```
      --agent string     Agent instructions (options: claude, cursor, agentsmd, all, none) (default "claude")
      --example string   Example project name (default: empty project)
      --olap string      OLAP engine (options: duckdb, clickhouse) (default "duckdb")
```

### Global flags

```
      --api-token string   Token for authenticating with the cloud API
      --format string      Output format (options: "human", "json", "csv") (default "human")
  -h, --help               Print usage
      --interactive        Prompt for missing required parameters (default true)
```

### SEE ALSO

* [stardata](cli.md)	 - A CLI for StarData

