#!/usr/bin/env bash
set -euo pipefail

# Under the docs lock (scripts/docslock.go), like every docs script: see
# lib/underlock.sh.
# shellcheck source=lib/underlock.sh
. "$(dirname "$0")/lib/underlock.sh"

# make docs: stages cfdocs at the pinned commit, reports on the vendored
# Lucee docs, and assembles docs/data/order, all under one hold of the lock,
# so no `make docs-bump` can swap a source between the fetch and the
# assemble. --cfdocs (make docs-cfdocs) skips the Lucee report.

lucee=1
case "${1:-}" in
"") ;;
--cfdocs) lucee="" ;;
*)
    echo "usage: scripts/docs.sh [--cfdocs]" >&2
    exit 2
    ;;
esac

cd "$(dirname "$0")/.."

bash scripts/fetch-docs-cfdocs.sh
if [ -n "$lucee" ]; then
    bash scripts/fetch-docs-lucee.sh
fi
bash scripts/assemble-docs.sh
