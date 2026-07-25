---
title: Deploy to StarData Cloud from GitLab
description: How to set up continuous deploys to StarData Cloud from GitLab
sidebar_label: Deploy from GitLab
sidebar_position: 10
---

While StarData Cloud natively integrates with [GitHub](https://github.com), you can also deploy your StarData project from [GitLab](https://about.gitlab.com/) using direct uploads from a [GitLab CI/CD pipeline](https://docs.gitlab.com/ee/ci/quick_start/).

Follow these steps to set up continuous deployment from GitLab to StarData Cloud:

1. Create a new GitLab repository and push your StarData project to it.

2. On your local, [authenticate with StarData Cloud](/guide/administration/users-and-access/user-management#logging-into-stardata-cloud) and create an organization (replace `my-org-name` with your desired name):
```bash
stardata login
stardata org create my-org-name
```

3. Create the project in StarData Cloud
```bash
stardata project deploy
```

:::note Multiple branches
If your repo contains multiple branches ensure the branch you want to deploy from via
```bash
stardata project edit --project my-project-name --prod-branch my-branch-name
```
:::

4. Provision a StarData Cloud [service account](/reference/cli/service/create) called `gitlab-ci` and copy its access token:
```
stardata service create gitlab-ci
```

5. Set the service token as a CI/CD variable called `RILL_SERVICE_TOKEN` in GitLab (from the repository page, it's under _Settings > CI/CD > Variables_).

6. Create a file named `.gitlab-ci.yml` at the root of the repository containing your StarData project. Paste the following contents into it (replace `my-org-name` and `my-project-name` with your desired names):
```yaml
deploy-stardata-cloud:
  stage: deploy
  script: 
    - curl -L -o $HOME/rill.zip https://cdn.rilldata.com/rill/latest/rill_linux_amd64.zip 
    - unzip -d $HOME $HOME/rill.zip 
    - git checkout -B "$CI_COMMIT_REF_NAME" "$CI_COMMIT_SHA"
    - $HOME/rill project deploy --org my-org-name --project my-project-name --interactive=false --api-token $RILL_SERVICE_TOKEN
```

Your StarData project should now automatically deploy to `ui.rilldata.com/my-org-name/my-project-name` each time changes are pushed to GitLab!

:::note File size limits
We enforce a file size limit of 100mb so ensure you do not unpack the stardata binary in the repo root or add it to your .gitignore
:::
