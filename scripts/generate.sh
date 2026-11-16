#!/usr/bin/env bash
set -euo pipefail

# make generate: stages the pinned sources and regenerates
# internal/docs/generated_docs.go (and generated_sources.txt), holding
# docs/.lock across the whole run, so no `make docs-bump` can swap a source in
# between the fetch and the generator.
#
#   scripts/generate.sh                   fetch cfdocs, assemble, generate
#   scripts/generate.sh --generate-only   generate from what is staged (CI's
#                                         second comparison, with the PR's base)
#
# The lock is the scripts' alone (scripts/lib/swapdir.sh): the generator never
# takes one, only checks it was started under one (DOCS_LOCK_OWNER), so the
# rules live in one place.

# Every path here is relative to the repository root, wherever this is run
# from.
cd "$(dirname "$0")/.."

# shellcheck source=lib/swapdir.sh
. scripts/lib/swapdir.sh

trap docs_unlock EXIT
docs_lock

if [ "${1:-}" != "--generate-only" ]; then
    bash scripts/fetch-docs-cfdocs.sh
    bash scripts/fetch-docs-lucee.sh
    bash scripts/assemble-docs.sh
fi

go run scripts/generate_docs.go
