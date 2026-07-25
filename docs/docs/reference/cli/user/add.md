---
note: GENERATED. DO NOT EDIT.
title: stardata user add
---
## stardata user add

Add user to a project, organization or group

```
stardata user add [flags]
```

### Flags

```
      --canvas stringArray    Canvas resource to restrict to (repeat for multiple)
      --email string          Email of the user
      --explore stringArray   Explore resource to restrict to (repeat for multiple)
      --group string          User group
      --org string            Organization
      --project string        Project
      --restrict-resources    Restrict the user to the provided resources (defaults to true when resources are provided)
      --role string           Role of the user (options: admin, editor, viewer, guest)
```

### Global flags

```
      --api-token string   Token for authenticating with the cloud API
      --format string      Output format (options: "human", "json", "csv") (default "human")
  -h, --help               Print usage
      --interactive        Prompt for missing required parameters (default true)
```

### SEE ALSO

* [stardata user](user.md)	 - Manage users

