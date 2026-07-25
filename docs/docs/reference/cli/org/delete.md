---
note: GENERATED. DO NOT EDIT.
title: stardata org delete
---
## stardata org delete

Delete organization

### Synopsis

Delete an organization and all its associated projects.
This operation cannot be undone. Use --force to skip confirmation.

```
stardata org delete [<org-name>] [flags]
```

### Examples

```
  stardata org delete myorg
  stardata org delete myorg --force
```

### Global flags

```
      --api-token string   Token for authenticating with the cloud API
      --format string      Output format (options: "human", "json", "csv") (default "human")
  -h, --help               Print usage
      --interactive        Prompt for missing required parameters (default true)
```

### SEE ALSO

* [stardata org](org.md)	 - Manage organizations

