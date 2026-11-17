#!/usr/bin/env bash
set -euo pipefail

# Under docs/.lock (scripts/docslock.go), like every docs script: see
# lib/underlock.sh.
# shellcheck source=lib/underlock.sh
. "$(dirname "$0")/lib/underlock.sh"

# make generate: stages the pinned sources and regenerates
# internal/docs/generated_docs.go (and generated_sources.txt), all under
# docs/.lock, so no `make docs-bump` can swap a source in between the fetch
# and the generator. It takes no arguments; the generator's settings come from
# the environment (DOCS_ALLOW_REMOVALS, DOCS_BASELINE_REF,
# DOCS_ALLOWED_REMOVALS_FILE: see scripts/generate_docs.go).
#
# The generator is built and run as a binary, not with `go run` (whose own
# child would not inherit fd 3), so it holds the lock's descriptor itself and
# checks it before reading or writing anything.

if [ $# -gt 0 ]; then
    echo "usage: scripts/generate.sh (no arguments; settings come from the environment)" >&2
    exit 2
fi

# Every path here is relative to the repository root, wherever this is run
# from.
cd "$(dirname "$0")/.."

bash scripts/fetch-docs-cfdocs.sh
bash scripts/fetch-docs-lucee.sh
bash scripts/assemble-docs.sh

mkdir -p target/docs-gen
go build -o target/docs-gen/generate_docs scripts/generate_docs.go
target/docs-gen/generate_docs
