#!/usr/bin/env bash
set -euo pipefail

# The generator, built below. docslock runs it once this script has
# succeeded (DOCS_LOCK_THEN: see lib/underlock.sh), and passes it the lock's
# descriptor, so it holds the lock itself for its write.
case "$(uname -s 2>/dev/null)" in
MINGW* | MSYS* | CYGWIN*) DOCS_LOCK_THEN="target/docs-gen/generate_docs.exe" ;;
*) DOCS_LOCK_THEN="target/docs-gen/generate_docs" ;;
esac

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
# it runs under the lock before reading anything, and again just before
# writing.

if [ $# -gt 0 ]; then
    echo "usage: scripts/generate.sh (no arguments; settings come from the environment)" >&2
    exit 2
fi

# Every path here is relative to the repository root, wherever this is run
# from.
cd "$(dirname "$0")/.."

bash scripts/docs.sh

mkdir -p target/docs-gen
go build -o "$DOCS_LOCK_THEN" scripts/generate_docs.go

# Started by docslock --then, which runs the generator next; started inside
# another docs run, which holds no descriptor to pass it, it runs here and
# checks the lock by its token.
if [ "${DOCS_LOCK_THEN_SET:-}" != "${DOCS_LOCK_TOKEN:-}" ] || [ -z "${DOCS_LOCK_TOKEN:-}" ]; then
    "$DOCS_LOCK_THEN"
fi
