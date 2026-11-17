#!/usr/bin/env bash
set -euo pipefail

# Under the docs lock (scripts/docslock.go), like every docs script: see
# lib/underlock.sh.
# shellcheck source=lib/underlock.sh
. "$(dirname "$0")/lib/underlock.sh"

# make generate: stages the pinned sources and regenerates
# internal/docs/generated_docs.go (and generated_sources.txt), all under
# the docs lock, so no `make docs-bump` can swap a source in between the fetch
# and the generator. It takes no arguments; the generator's settings come from
# the environment (DOCS_ALLOW_REMOVALS, DOCS_BASELINE_REF,
# DOCS_ALLOWED_REMOVALS_FILE: see scripts/generate_docs.go).
#
# The generator is built and run as a binary, not with `go run`, so its exit
# code (128 + a signal's number, when one stops it) comes through. It checks
# it runs under the lock (DOCS_LOCK_TOKEN) before reading anything, and again
# just before writing.

if [ $# -gt 0 ]; then
    echo "usage: scripts/generate.sh (no arguments; settings come from the environment)" >&2
    exit 2
fi

# Every path here is relative to the repository root, wherever this is run
# from.
cd "$(dirname "$0")/.."

bash scripts/docs.sh

mkdir -p target/docs-gen
gen="target/docs-gen/generate_docs$(go env GOEXE)"
go build -o "$gen" scripts/generate_docs.go
"$gen"
