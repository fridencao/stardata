---
title: Deploy Dashboards 
sidebar_label: Deploy Dashboards 
sidebar_position: 00
---

<!-- WARNING: There are links to this page in source code. If you move it, find and replace the links and consider adding a redirect in docusaurus.config.js. -->

## Overview


Deploying dashboards from StarData Developer allows you to share dashboards with other users, leverage [StarData Cloud capabilities](/guide/dashboards/explore), [embed StarData](/developers/embed/iframe) into other applications, and more!

The flow diagram below shows two options for deploying an existing project. 

**Deploy via the UI or CLI using `stardata project deploy`**: 
```mermaid
graph LR;
    A(Local code files);
    B(StarData Cloud);
    A--deploy (once)/update via UI -->B;

```
---
**Deploy via the CLI via `stardata project connect-github`**:
```mermaid
graph LR;
    A(Local code files);
    B(StarData Cloud);
    C(GitHub);


    A--1. deploy via CLI (once) -->B;
    C-- Continuous Deployment-->B;
    A--2. Push changes to GitHub-->C;
```
    
## Deploying a project from StarData Developer
Starting from **v0.48**, we have introduced the possibility to push dashboards _directly from StarData Developer to StarData Cloud_. On the dashboard page, you can select the `Deploy` button and follow the steps to deploy to StarData Cloud.

![Deploy UI](/img/deploy/existing-project/deploy-ui.gif)

Now that your project has been deployed to StarData Cloud, you will need to ensure that your users have access! Please refer to the [user management](/guide/administration/users-and-access/user-management) section.

If you make changes locally on StarData Developer, you will need to push the contents to StarData Cloud by selecting the `Update` button.

![Redeploy](/img/deploy/existing-project/redeploy.gif)

:::tip On an older version of StarData?

You can easily check the version of StarData that you are using in StarData Developer by running the following command:

```bash
stardata --version
```

If you are on an older version of StarData, it is **strongly recommended** to [upgrade](/developers/get-started/install#upgrade-to-the-newest-version-of-stardata-developer) to the latest version.

:::

### Syncing your GitHub Repository
:::note GitHub app permissions
This assumes that the installed GitHub app in your organization has write access. If unsure, please check with your GitHub admin.

The required permissions are:
 - Read access to metadata and pull requests
 - Read and write access to administration and code 
:::


At this point, you have the option to connect your StarData project to a GitHub Repository.

Navigating to the Settings page and selecting `Connect to GitHub` will prompt you to login and create a repository for your project. If you've already created a repository, check the box 'I've created a GitHub Repo' and add the permissions for StarData to access the repository.

:::info Check with your GitHub organization admin

If you're not the admin of your GitHub organization, they will likely need to first install the StarData Cloud app in your organization before you can proceed with deploying a project. After the StarData Cloud app is installed, it should have the following privileges:
:::



![Install StarData Cloud](/img/deploy/existing-project/install-stardata-cloud.png)


Once the permissions to the repository have been confirmed and set, you can continue to select the repository in the dropdown.
![Select Repo](/img/deploy/existing-project/select-repo.png)


Once completed, you'll see the newly updated repository on the UI of your settings page!

![Finished](/img/deploy/existing-project/finished.png)


:::warning Still unable to connect?
If you encounter issues, check that the app installation is not pending. Go to your organization's settings and click on Installed GitHub Apps. You will see a section of Pending GitHub Apps installation requests. If you're an Owner or App Manager, grant access to the StarData app if it is pending."
:::


## Deploying a project via the CLI

:::note
Starting from v0.49, we have deprecated `stardata deploy` in favor of `stardata project deploy` and `stardata project connect-github`. For more information on the `stardata deploy` command click [here](#deprecated-stardata-deploy).
:::

### Deploy project without GitHub Repository
You can add a GitHub Repository later.
```
stardata project deploy
Using org "Rill_Learn".

Starting upload.
All files uploaded successfully.

Created project "Rill_Learn/my-stardata-tutorial". Use `stardata project rename` to change name if required.

...

Your project can be accessed at: https://ui.rilldata.com/Rill_Learn/my-stardata-tutorial
Opening project in browser...
```

If you have not already [configured your connections' credentials](https://docs.rilldata.com/developers/build/connectors/credentials), you will be reminded here which connections are required.

**First deployment**

If this is your first deployment to StarData Cloud, you will get prompted to either sign up or log in (if you have an existing account on [StarData Cloud](https://ui.rilldata.com/)). Proceed with the sign up and email verification process for new users or authorization process for existing users. As a new user, you can expect to see the following page:

![StarData Cloud Sign In](/img/deploy/existing-project/rill-cloud-sign-in.png)



**Project Uploaded Successfully**

Once the project has been uploaded to StarData Cloud, you should be able to see the following page: 

![Status](/img/deploy/existing-project/status.png)






### Deploy Project with Repository
Follow the instructions in the Terminal to login to GitHub (if not already done so), and select your repository.
If you do not set any parameters, StarData will infer the project name based on the folder path and use this as both the repository and project name. If there are any overlaps, we will request for a new name.
```bash
stardata project connect-github
No git remote was found.
? Do you want to create a repo? Yes
? Select a GitHub account for the new repository royendo
Repository name "my-stardata-tutorial" is already taken
? Please provide alternate name my-stardata-tutorial-cli

Request submitted for creating repository. Checking completion status

Successfully created repository on "https://github.com/royendo/my-stardata-tutorial-cli"

Pushing local project to GitHub

Successfully pushed your local project to GitHub

Using org "Rill_Learn".

Created project "Rill_Learn/my-stardata-tutorial-cli". Use `stardata project rename` to change name if required.

StarData projects deploy continuously when you push changes to GitHub.

...

Your project can be accessed at: https://ui.rilldata.com/Rill_Learn/my-stardata-tutorial-cli
Opening project in browser...
```

Once completed, you will see the following in the settings page. Note that the GitHub repository is already set up!

![Cli Upload](/img/deploy/existing-project/cli-upload.png)



## Continuous Deployment 

Whether you decide to manage your StarData projects using GitHub or by re-running `stardata project deploy`, StarData should automatically detect changes that you have pushed locally and update your deployed project accordingly. Depending on the changes, this may result in a project reconciliation. If you are experiencing issues with the project after pushing changes with the CLI, please refer to the project's status page for more information, or run the following command:

```
stardata project status
```

Likewise, if using the UI by selecting the `Update` button, StarData will detect the changes in files and update your deployed project accordingly. Along with the above CLI command, you can view the status of the objects in the Status page.

:::tip Interested in using GitLab?

Check out our documentation on deploying a [StarData project using GitLab](/developers/deploy/deploy-dashboard/deploy-from-cli)!

:::


## Change your production branch

By default, StarData deploys from the [default branch](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-branches#about-the-default-branch) of your Git repository. You can change this to any branch you want.

To deploy your project from a different branch, run the following command:

```bash
stardata connect-github --prod-branch [PROD-BRANCH]
```



## Deploy from a monorepo

If your StarData project is in a sub-directory of a Git repository, use the `--subpath` option when creating your project:
```
stardata connect-github --subpath path/to/rill/project
```
:::warning
Note that you must run `stardata connect-github` from the <u>root</u> of your Git repository, **not** the root of your StarData project.
:::



## Deprecated StarData Deploy

When running `stardata deploy` you have two options: 
1. Enable automatic deploys to StarData Cloud via GitHub
2. Disable automatic deploys to StarData Cloud via GitHub

```
stardata deploy
? Enable automatic deploys to StarData Cloud from GitHub? 
```

### Enable Automatic deploys

Like running `stardata project connect-github`, you will be [prompted to create a github repository](#deploy-project-with-repository). Once created, StarData will deploy the project. You can confirm that the project has the correct repository linked from the UI on the settings page.


### Disable Automatic deploys

In this case, the project will be deployed to StarData Cloud without a GitHub repository connected. You can always [add a repository via the UI](#syncing-your-github-repository) at a later time.
