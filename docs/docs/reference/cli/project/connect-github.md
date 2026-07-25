---
note: GENERATED. DO NOT EDIT.
title: stardata project connect-github
---
## stardata project connect-github

Deploy project to StarData Cloud by pulling project files from a git repository

```
stardata project connect-github [<path>] [flags]
```

### Flags

```
      --path string             Path to project repository (default: current directory) (default ".")
      --subpath string          Relative path to project in the repository (for monorepos)
      --remote string           Remote name (default: origin) (default "origin")
      --org string              Org to deploy project in
      --name string             Project name (default: Git repo name)
      --description string      Project description
      --public                  Make dashboards publicly accessible
      --provisioner string      Project provisioner
      --primary-branch string   Git branch to deploy from (default: the default Git branch)
      --push-env                Push local .env file to StarData Cloud (default true)
```

### Global flags

```
      --api-token string   Token for authenticating with the cloud API
      --format string      Output format (options: "human", "json", "csv") (default "human")
  -h, --help               Print usage
      --interactive        Prompt for missing required parameters (default true)
```

### SEE ALSO

* [stardata project](project.md)	 - Manage projects

