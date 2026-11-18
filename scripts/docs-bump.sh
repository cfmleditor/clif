#!/usr/bin/env bash
set -euo pipefail

# Under the docs lock (scripts/docslock.go), like every docs script: see
# lib/underlock.sh.
# shellcheck source=lib/underlock.sh
. "$(dirname "$0")/lib/underlock.sh"

# make docs-bump: moves the cfdocs pin in docs/sources.lock and the vendored
# docs/lucee to upstream's latest, and stages them. Follow with
# `make generate`, and review its diff with the lock's and docs/lucee's.
#
# It holds the docs lock across every step (each runs under it), so a `make docs`
# or `make generate` started meanwhile cannot slip in between, restage the old
# pin, and have it written as the new one.
#
# cfdocs is staged and its commit checked before anything committed changes;
# only then is docs/lucee replaced, and the lock written last, by a step that
# can no longer fail on its input. A failure before the Lucee swap leaves
# every committed file as it was (the staged cfdocs is then simply not the
# pinned one, which the generator refuses), so the two are bumped together.

# Every path here is relative to the repository root, wherever this is run
# from.
cd "$(dirname "$0")/.."

DOCS_BUMP=1 bash scripts/fetch-docs-cfdocs.sh
commit=$(cat docs/.sha-cfdocs)
bash scripts/docs-lock.sh check "$commit"
DOCS_BUMP=1 bash scripts/fetch-docs-lucee.sh
bash scripts/docs-lock.sh set cfdocs "$commit"
bash scripts/assemble-docs.sh
