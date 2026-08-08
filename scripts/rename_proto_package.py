#!/usr/bin/env python3
"""Rename all occurrences of stardata.runtime.v1 to stardata.runtime.v1 across the codebase"""
import os
import re
import sys

# Paths to process
ROOT = os.path.dirname(os.path.abspath(__file__))

def find_files_with_pattern(root, patterns, exclude_dirs=None):
    """Find files matching any of the given patterns"""
    if exclude_dirs is None:
        exclude_dirs = ['.git', 'node_modules', 'dist', 'build']
    found = []
    for dirpath, dirnames, filenames in os.walk(root):
        # Exclude specified directories
        dirnames[:] = [d for d in dirnames if d not in exclude_dirs]
        for filename in filenames:
            for pattern in patterns:
                if filename.endswith(pattern):
                    found.append(os.path.join(dirpath, filename))
                    break
    return found

# Phase 1: Change proto package declarations
print("=== Phase 1: Update proto package declarations ===")
proto_files = find_files_with_pattern(ROOT, ['*.proto', '.*.proto'])
for f in proto_files:
    if 'rill/runtime/v1' in f or 'rill/ui/v1' in f:
        with open(f, 'r') as file:
            content = file.read()
        # Change package declaration
        content = re.sub(r'^package rill\.runtime\.v1;', 'package stardata.runtime.v1;', content, flags=re.MULTILINE)
        content = re.sub(r'^package rill\.ui\.v1;', 'package stardata.ui.v1;', content, flags=re.MULTILINE)
        # Change import paths inside proto files
        content = re.sub(r'import "rill/runtime/v1/([^"]+)"', r'import "stardata/runtime/v1/\1"', content)
        content = re.sub(r'import "rill/ui/v1/([^"]+)"', r'import "stardata/ui/v1/\1"', content)
        if content != open(f, 'r').read():
            with open(f, 'w') as file:
                file.write(content)
            print(f"  Updated: {f}")

# Phase 2: Update Go files that reference stardata.runtime.v1 in their package/import sections
print("\n=== Phase 2: Update Go source files ===")
go_files = find_files_with_pattern(ROOT, ['.go'])
for f in go_files:
    # Check if this is a proto-generated file or Go source
    with open(f, 'r') as file:
        content = file.read()
    changed = False
    # In proto-generated Go files (api.pb.go, etc.), package should stay rill
    # But in Go source that references rill runtime, update imports
    if 'proto/gen/stardata/runtime/v1' in content:
        content = content.replace('github.com/fridencao/stardata/proto/gen/stardata/runtime/v1',
                                 'github.com/fridencao/stardata/proto/gen/stardata/runtime/v1')
        changed = True
    if 'github.com/fridencao/stardata/proto/gen/stardata/admin/v1' in content:
        content = content.replace('github.com/fridencao/stardata/proto/gen/stardata/admin/v1',
                                 'github.com/fridencao/stardata/proto/gen/stardata/admin/v1')
        changed = True
    if changed:
        with open(f, 'w') as file:
            file.write(content)
        print(f"  Updated: {f}")

# Phase 3: Update TypeScript files
print("\n=== Phase 3: Update TypeScript source files ===")
ts_files = find_files_with_pattern(ROOT, ['.ts', '.tsx'])
for f in ts_files:
    with open(f, 'r') as file:
        content = file.read()
    changed = False
    # Update proto import paths
    if 'web-common/src/proto/gen/stardata/runtime/v1' in content:
        content = content.replace('web-common/src/proto/gen/stardata/runtime/v1',
                                 'web-common/src/proto/gen/stardata/runtime/v1')
        changed = True
    if 'web-local/src/proto/gen/stardata/runtime/v1' in content:
        content = content.replace('web-local/src/proto/gen/stardata/runtime/v1',
                                 'web-local/src/proto/gen/stardata/runtime/v1')
        changed = True
    if 'github.com/fridencao/stardata/proto/gen/stardata/runtime/v1' in content:
        content = content.replace('github.com/fridencao/stardata/proto/gen/stardata/runtime/v1',
                                 'github.com/fridencao/stardata/proto/gen/stardata/runtime/v1')
        changed = True
    if changed:
        with open(f, 'w') as file:
            file.write(content)
        print(f"  Updated: {f}")

print("\nDone!")