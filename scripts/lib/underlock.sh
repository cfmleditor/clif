# shellcheck shell=bash
# Sourced first by every docs script: runs the script under docs/.lock, held by
# scripts/docslock.go (the operating system's own file lock, so it is released
# however its holder ends, and no stale lock is ever left to judge). A script
# started directly re-runs itself through docslock.go, from the repository
# root; one started by a script that already holds the lock (DOCS_LOCK_HELD)
# just carries on, inheriting it.
#
# Sourced as: . "$(dirname "$0")/lib/underlock.sh"  (before anything else)
if [ -z "${DOCS_LOCK_HELD:-}" ]; then
    cd "$(dirname "$0")/.." || exit 1
    exec go run scripts/docslock.go bash "scripts/${0##*/}" "$@"
fi
