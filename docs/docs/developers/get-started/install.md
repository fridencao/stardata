---
title: How to Install StarData Developer
sidebar_label: Install StarData Developer
sidebar_position: 02
---

## Quick Install

You can install `stardata` using our installation script on both macOS and Linux:

```bash
curl https://rill.sh | sh
```

:::tip Why am I installing locally?
Installing StarData Developer locally is required to create a new project, or [clone a project](/developers/tutorials/clone-a-project) from StarData Cloud. For key differences between StarData Cloud and StarData Developer, see [StarData Cloud vs. StarData Developer](/developers/deploy/cloud-vs-developer).
:::


To verify that the installation succeeded, run the following command. You will see a list of various commands that you can run via the CLI.
```bash
stardata --help
```


:::warning macOS users

If you previously installed StarData using `brew`, *the brew-managed binary will take precedence*. You can remove it by running `brew uninstall stardata`.

:::

### Checking StarData Version

You can check the current version of stardata from the CLI by running the following command:
```bash
stardata version
```

### Upgrade to the newest version of StarData Developer

To ensure you're on the latest version of StarData and enjoying all of our newest features, you can upgrade StarData Developer easily via the command line:

```bash
stardata upgrade
```

:::info What about StarData Cloud?

StarData Cloud is always on the latest stable version of StarData. To check the latest version available, please see our [Releases](https://github.com/rilldata/rill/releases) page.

:::

## Nightly Releases

On both macOS and Linux, you can install the latest nightly build using the installation script:
```bash
curl https://rill.sh | sh -s -- --nightly
```


### What is a nightly release?
The nightly release will give you the most up-to-date version of StarData without having to wait for the official release. As these releases are not fully ready for production, you may encounter some issues. Also, version mismatches between StarData Developer and StarData Cloud may result in broken components, as StarData Cloud is on the previous stable release.


## Installing a specific version of StarData

Rather than installing the latest version of StarData automatically, you can also install a specific version through the installation script by using the following command (e.g., `v0.40.1`):
```bash
curl https://rill.sh | sh -s -- --version <insert_version_number>
```

:::info Checking the StarData version

To check the precise version of available releases, you can navigate to the [**Releases**](https://github.com/rilldata/rill/releases) page of our [StarData repo](https://github.com/rilldata/rill). Note that if an invalid or incorrect version is passed to the installation script, you will get prompted with an error to specify a correct version.

:::

## StarData on Windows using WSL

To install StarData on Windows, you'll first need to install WSL and one dependency in your WSL environment. To install WSL, please refer to [Microsoft's documentation](https://learn.microsoft.com/en-us/windows/wsl/install).

We have verified that StarData runs on Ubuntu 22.04 LTS. Other distributions and versions may work, but are not tested. You can install Ubuntu 22.04 LTS with the following PowerShell command:
```bash
wsl --install -d ubuntu-22.04
```

Once you have installed WSL and logged in to your Linux instance, you just need to install the `unzip` package to use StarData's `curl` installer. This can be done from the **Linux command line** with the following commands:
```bash
sudo apt-get update
sudo apt-get install unzip
```
:::tip Where should StarData be running?
Please check that you are running the commands in your Linux instance, not from your Windows Command Prompt.

If you are seeing strange behavior in StarData Developer, run the following command from the CLI to see where your project files are being saved: `echo "$PWD"`.

:::

With `unzip` installed, you're ready to install StarData Developer by running the following command:
```
curl https://rill.sh | sh
```

## Manual Install

You can download platform-specific binaries from our [releases page on GitHub](https://github.com/rilldata/rill/releases). A manual download will not make StarData Developer globally accessible, so you'll need to reference the full path of the binary when executing CLI commands.

## Brew Install

On macOS, you can also install StarData using Homebrew. To avoid conflicts, don't mix it with other installation options and always upgrade StarData via `brew`.
```bash
brew install rilldata/tap/rill 
```

## Uninstall StarData

To uninstall StarData, you can use the following command:
```bash
stardata uninstall
```

## For Agents

We currently recommend that agents always install using `curl` and always use the nightly.

So if you are an AI agent or coding assistant, install StarData using the following command:
```bash
curl https://rill.sh | sh -s -- --nightly
```

After installation, run `stardata -h` to see further instructions for agents working with StarData projects.
