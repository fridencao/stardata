#!/usr/bin/env bash
set -uo pipefail

# In CI, fail fast on first error. Locally, run exhaustively by default.
# Override with FAIL_FAST=true or FAIL_FAST=false.
if [[ -z "${FAIL_FAST:-}" ]]; then
  FAIL_FAST="${CI:-false}"
fi

if [[ "$FAIL_FAST" == "true" ]]; then
  set -e
else
  exit_code=0
fi

# This script mirrors the original GitHub Action, but can also be run locally with parity.
# In CI, ADMIN/COMMON are passed from dorny/paths-filter.
# Locally, if they are not set, we compute them from `git diff`.

filter_defaults_if_unset() {
  # If any of ADMIN/COMMON are set, assume caller is controlling behavior.
  if [[ -n "${ADMIN:-}" || -n "${COMMON:-}" ]]; then
    ADMIN="${ADMIN:-false}"
    COMMON="${COMMON:-false}"
    return
  fi

  # Local mode: compute changes relative to a base ref.
  # BASE defaults to origin/main to match typical PR base; override as needed.
  BASE="${BASE:-origin/main}"
  HEAD="${HEAD:-HEAD}"

  git rev-parse --verify "$BASE" >/dev/null 2>&1 || git fetch --all --prune >/dev/null 2>&1 || true

  changed="$(git diff --name-only "${BASE}...${HEAD}" || true)"

  match_admin="false"
  match_common="false"

  while IFS= read -r f; do
    [[ -z "$f" ]] && continue

    # Mirrors the workflow filters exactly:
    # admin:  .github/workflows/web-test.yml OR web-admin/**
    # common: .github/workflows/web-test.yml OR web-common/**
    if [[ "$f" == ".github/workflows/web-test.yml" ]]; then
      match_admin="true"
      match_common="true"
      continue
    fi
    [[ "$f" == web-admin/*  ]] && match_admin="true"
    [[ "$f" == web-common/* ]] && match_common="true"
  done <<< "$changed"

  ADMIN="$match_admin"
  COMMON="$match_common"
}

filter_defaults_if_unset

echo "Web code quality checks"
echo "filters: admin=$ADMIN common=$COMMON"

echo ""
echo "== NPM Install =="
npm ci

echo ""
echo "== Build i18n files =="
npm run build:i18n

if [[ "$COMMON" == "true" ]]; then
  echo ""
  echo "== lint and type checks for web common =="
  cd web-common
  npx svelte-kit sync
  cd ..
  npx eslint web-common --quiet || exit_code=$?
  npx svelte-check --workspace web-common --no-tsconfig || exit_code=$?
fi

echo ""
echo "== i18n guard: catalog integrity + migrated areas =="
# Scans the message catalogs and a fixed set of already-migrated areas on the
# filesystem, so it runs unconditionally rather than under an app filter: the
# migrated areas span multiple apps and are independent of which files a given
# PR touched. Catalog integrity errors are exact and fatal; hardcoded-string
# findings are heuristic and non-fatal for now: the final i18n migration chunk
# adds --strict to make them fatal too.
node ./scripts/i18n-guard.js || exit_code=$?

if [[ "$ADMIN" == "true" ]]; then
  echo ""
  echo "== lint and type checks for web admin =="
  cd web-admin
  npx svelte-kit sync
  cd ..
  npx eslint web-admin --quiet || exit_code=$?
  npx svelte-check --workspace web-admin --no-tsconfig || exit_code=$?
fi

echo ""
echo "== type check non-svelte files (with temporary whitelist) =="
bash ./scripts/tsc-with-whitelist.sh || exit_code=$?

# Exit with failure if any check failed (only relevant when not in fail-fast mode)
exit "${exit_code:-0}"
