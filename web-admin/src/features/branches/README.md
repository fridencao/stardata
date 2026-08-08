# Branches feature — legacy (archive-mode only)

**Status:** legacy. Kept as compatibility shims for projects still using the
archive-based semantic layer (`projects.semantic_layer_mode = 'archive'`).

## What lives here

`BranchDeploymentStopped.svelte`, `BranchesSection.svelte`,
`DeleteBranchConfirmDialog.svelte`, `branch-actions.ts`, `branch-utils.ts`,
`deployment-utils.ts` — the whole branch/dev-deployment machinery from the
Rill fork.

## Why it is still here

Phase 5 shipped a DB-versioned semantic layer that does not use branches
(one draft per project, versions are DB snapshots — see
`design/phase5-db-versioned-semantic-layer.md`). But the migration plan is
non-destructive:

- Existing projects created with `semantic_layer_mode = 'archive'` continue
  to use file-based semantics, and they still need branch and dev-deployment
  UI.
- Only new projects (Phase 5+) opt into `db` mode, which bypasses this whole
  directory via runtime checks in `studio/[domain]/+layout.svelte` and admin
  server refusing dev deployments for DB-mode projects.

## When to delete

Delete only after **all archive-mode projects have been migrated or
retired**. Until then, removing this code will break editing for legacy
projects — even though DB-mode projects never touch it.

Grep marker for future cleanup: `Phase 5.4 legacy-branch-shim`.
