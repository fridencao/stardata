---
title: "Clone a Project - Quick Start"
sidebar_label: "Clone an existing Project"
sidebar_position: 3
hide_table_of_contents: false

tags:
  - Getting Started
  - Quickstart
  - Tutorial
---

# Clone a Project - Quick Start

This guide will help you get started with an existing StarData project by cloning it from a repository and setting it up locally.

## Prerequisites

Before you begin, make sure you have:

- **StarData CLI** installed ([Installation Guide](/developers/get-started/install))
```bash
curl https://rill.sh | sh
```
- **Access to the [StarData Project](https://ui.rilldata.com/)** 


## Step 1: Clone the Repository
Depending on whether your project is synced to GitHub or not, select the correct clone method. If you are unsure, please see the Settings page in the project.

### From GitHub
![Github Pushed Changes](/img/tutorials/rill-advanced/github-pushed-changes.png)

```bash
# Clone the repository
git clone https://github.com/username/rill-project.git # Replace 'username' and 'stardata-project' with your actual URL
cd <project-name>
```

### Using StarData CLI

![Status](/img/tutorials/rill-advanced/status.png)
```bash
# Clone from StarData
stardata project clone <project-name>
```

## Step 2: Explore the Project Structure

A typical StarData project contains:

```
<project-name>/
├── rill.yaml              # Project configuration
├── sources/               # Data source definitions
│   ├── database.yaml      # Database connections
│   ├── api.yaml          # API endpoints
│   └── files.yaml        # File-based sources
├── models/                # SQL transformations
│   ├── staging/          # Staging models
│   ├── marts/            # Business logic models
│   └── metrics/          # Metric definitions
├── dashboards/           # Dashboard configurations
│   └── main_dashboard.yaml
├── alerts/               # Alert definitions
├── .env                  # Environment variables (not in git, need to run stardata env pull)
└── .gitignore           # Git ignore rules
```

## Step 3: Set Up Environment Variables

If you cloned the project via GitHub, you will need to run the following command to bring down the environment variables to your local machine.

```bash
stardata env pull
```

If you cloned the project via the StarData CLI, you should see the following in the Terminal:
```bash
Updated .env file with cloud credentials from project "your-project-here".
```

:::tip Admin of your project?

As an admin, when running `stardata start`, we'll automatically retrieve your credentials for you. No need for extra steps.

:::
## Step 4: Check your Source YAML before starting StarData
We want to check to see if any `{{if dev}} ... {{end}}` parameters have been set in your source ingestion. If not, when you start StarData, this will initiate a full ingestion of your data, which might take some time and, depending on the source location, could incur costs (e.g., Snowflake, BigQuery). However, if your data is not that large, it may be safe to start StarData without these guardrails. 

## Step 5: Start StarData Developer

### Start the Development Server
![Clone Project](/img/tutorials/quickstart/clone-project.png)

```bash
# Start StarData Developer
stardata start
```

This will:
- Start the web UI at `http://localhost:9009`
    - Initiate ingestion of data sources 
    - Start building your models and dashboards
    - Show any errors or warnings



## Step 6: Explore the Project and Make Changes

Once your sources and models have built and you are able to explore your dashboards, make the needed changes to the files and get ready to update your StarData project.

:::warning Changes to sources and models

Changes to sources and models will initiate a full refresh of the source, unless otherwise indicated via `patch_mode`. We highly recommend reviewing the changes to ensure that you do not push unwanted changes to your production environment. 
:::

### via git
For projects that were cloned via git, you'll need to run the required git commands to add, commit, and push changes. Keep in mind the basic git practices about merging files to main without having an approval process.


### via StarData Update Button

For projects cloned via the CLI, the underlying connection to the deployment will also be brought locally so that when the button to "Deploy" is now "Update" the existing deployment. Keep in mind the warning above about changes to sources and models. 