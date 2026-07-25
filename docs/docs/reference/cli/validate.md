---
note: GENERATED. DO NOT EDIT.
title: stardata validate
---
## stardata validate

Validate project resources

```
stardata validate [<path>] [flags]
```

### Flags

```
  -e, --env strings                    Set environment variables
      --reset                          Clear and re-ingest source data
      --pull-env                       Pull environment variables from StarData Cloud before starting the project (default true)
      --environment string             Environment name (default "dev")
      --verbose                        Sets the log level to debug
      --silent                         Suppress all log output by setting log level to panic, overrides verbose flag
      --debug                          Collect additional debug info
      --log-format string              Log format (options: "console", "json") (default "console")
      --model-timeout-seconds uint32   Timeout for reconciliation of models, set 0 for no timeout (default 60)
  -o, --output-file string             Output file for validation results (JSON format)
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

