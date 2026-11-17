#!/usr/bin/env bash
set -euo pipefail

# The generator, built below, is run by docslock once this script has
# succeeded (docslock --then: see lib/underlock.sh), which passes it the
# lock's descriptor, so it holds the lock itself for its write. A plain shell
# variable, not exported: underlock.sh ignores an exported one.
docs_want_then=1

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

# docslock --then names a file for this script, the command it starts, to
# write the generator's path to (DOCS_LOCK_THEN_FILE): read and unset at once,
# so nothing this runs (a nested generate.sh, say) takes it for its own.
then_file=${DOCS_LOCK_THEN_FILE:-}
unset DOCS_LOCK_THEN_FILE

bash scripts/docs.sh

# Built under the lock, checked just before, to a path named for a hash of
# its sources as they are now (lib/hashbuild.sh), unless that version is built
# already; never over a copy that may be executing.
# shellcheck source=lib/hashbuild.sh
. scripts/lib/hashbuild.sh
docs_still_locked
gen=$(docs_built_path scripts/generate_docs.go generate_docs)
docs_build scripts/generate_docs.go "$gen"

# Started by docslock --then: the generator is named to it, to run next with
# the lock's descriptor. The file is this run's (its name holds the run's
# token); one leaked from elsewhere names nothing, and the generator runs
# here. Started inside another docs run, which holds no descriptor to pass
# it, it runs here too, and checks the lock by its token.
case "$then_file" in
?*"-${DOCS_LOCK_TOKEN:-none}")
    printf '%s\n' "$gen" > "$then_file"
    ;;
*)
    "$gen"
    ;;
esac
