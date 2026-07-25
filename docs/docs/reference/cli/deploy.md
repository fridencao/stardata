---
note: GENERATED. DO NOT EDIT.
title: stardata deploy
---
## stardata deploy

Deploy project to StarData Cloud

```
stardata deploy [<path>] [flags]
```

### Flags

```
      --path string             Path to project repository (default: current directory) (default ".")
      --subpath string          Relative path to project in the repository (for monorepos)
      --remote string           Remote name (default: origin) (default "origin")
      --org string              Org to deploy project in
      --project string          Project name (default: Git repo name)
      --description string      Project description
      --public                  Make dashboards publicly accessible
      --provisioner string      Project provisioner
      --primary-branch string   Git branch to deploy from (default: the default Git branch)
      --push-env                Push local .env file to StarData Cloud (default true)
      --force-push              Force push local changes in case of StarData managed repos
      --managed                 Create project using stardata managed repo
      --github                  Use github repo to create the project
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

