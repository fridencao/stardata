#!/usr/bin/env python3
"""One-off migration: update references from rill.*.v1 proto packages to stardata.*.v1"""
import os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

INCLUDE_DIRS = [
    "runtime",
    "admin",
    "cli",
    "web-common/src",
    "web-common/tests",
    "web-admin/src",
    "web-admin/tests",
    "web-local/src",
    "web-local/tests",
    "web-integration",
    "scripts",
]
INCLUDE_FILES = [
    "web-admin/orval.config.ts",
    "proto/README.md",
]
EXCLUDE_PARTS = {
    ".git", "node_modules", ".svelte-kit", "build", "dist",
    ".claude", "proto/gen", "src/proto/gen", "src/client/gen",
}
EXTS = {".go", ".ts", ".tsx", ".svelte", ".js", ".mjs", ".cjs", ".yaml", ".yml", ".json", ".md", ".sh", ".py", ".sql"}

REPLACEMENTS = [
    ("proto/gen/rill/", "proto/gen/stardata/"),
    ("rill.runtime.v1", "stardata.runtime.v1"),
    ("rill.admin.v1", "stardata.admin.v1"),
    ("rill.local.v1", "stardata.local.v1"),
    ("rill.ai.v1", "stardata.ai.v1"),
    ("rill.ui.v1", "stardata.ui.v1"),
]


def excluded(path):
    rel = os.path.relpath(path, ROOT)
    if "proto/gen" in rel.replace(os.sep, "/"):
        return True
    return any(part in EXCLUDE_PARTS for part in rel.split(os.sep))


def process(path):
    _, ext = os.path.splitext(path)
    if ext not in EXTS:
        return
    if os.path.abspath(path) == os.path.abspath(__file__):
        return
    try:
        with open(path, "r", encoding="utf-8") as fh:
            content = fh.read()
    except (UnicodeDecodeError, OSError):
        return
    updated = content
    for old, new in REPLACEMENTS:
        updated = updated.replace(old, new)
    if updated != content:
        with open(path, "w", encoding="utf-8") as fh:
            fh.write(updated)
        print(f"updated: {os.path.relpath(path, ROOT)}")


for d in INCLUDE_DIRS:
    base = os.path.join(ROOT, d)
    for dirpath, dirnames, filenames in os.walk(base):
        dirnames[:] = [x for x in dirnames if x not in EXCLUDE_PARTS]
        if excluded(dirpath):
            continue
        for name in filenames:
            process(os.path.join(dirpath, name))

for f in INCLUDE_FILES:
    p = os.path.join(ROOT, f)
    if os.path.exists(p):
        process(p)

print("done")
