---
title: Build StarData Projects with Your Favorite IDE
description: Use VS Code, IntelliJ, or any IDE to create and edit StarData projects with real-time feedback
sidebar_label: External IDE Integration
sidebar_position: 00
---


## Use Any IDE for StarData Development

StarData projects are just files and folders that you can edit with any code editor or IDE. Whether you prefer VS Code, IntelliJ, Vim, or any other editor, you can create and modify StarData projects directly from your favorite development environment.

### How It Works

StarData projects consist of:
- **SQL files** (`.sql`) for models and metrics views
- **YAML files** (`.yml`) for project configuration
- **Data files** in various formats

You can edit these files in any IDE, and StarData will automatically detect changes and provide real-time feedback.

![](https://cdn.rilldata.com/docs/release-notes/36_hot_reload.gif)

## Using AI Agents to Build StarData Projects

StarData ships built-in instructions that teach AI coding agents like **Claude Code** and **Cursor** how to build StarData projects. A single `stardata init` command scaffolds everything your agent needs — resource schemas, best practices, and development conventions.

```bash
# Add Claude Code instructions to your project
stardata init --template claude

# Or add Cursor rules
stardata init --template cursor
```

