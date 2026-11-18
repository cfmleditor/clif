#!/usr/bin/env bash
set -euo pipefail

# set runs under the docs lock (scripts/docslock.go), like every docs script
# that changes anything: see lib/underlock.sh. get and check only read, so
# they take no lock and print nothing but their answer.
if [ "${1:-}" = set ]; then
    # shellcheck source=lib/underlock.sh
    . "$(dirname "$0")/lib/underlock.sh"
fi

# Reads and writes docs/sources.lock, which pins each fetched doc source:
#
#   cfdocs <commit>    the cfdocs commit fetch-docs-cfdocs.sh checks out
#
# (Lucee's docs are vendored in docs/lucee instead: see fetch-docs-lucee.sh.)
#
#   docs-lock.sh get <source>          print the pin; exit 1 unless pinned once
#   docs-lock.sh set <source> <commit> write it, keeping the other lines
#   docs-lock.sh check <commit>        exit 1 unless it is a commit set would take
#
# scripts/generate_docs.go is the strict reader of the whole file (every line,
# every rule). This only refuses what would make its own answer wrong: a
# source pinned twice, or not at all, and a value that is not a commit.
LOCK="docs/sources.lock"

# Relative to the repository root, wherever this is run from.
cd "$(dirname "$0")/.."

usage() {
    echo "usage: docs-lock.sh get <source> | set <source> <commit> | check <commit>" >&2
    exit 2
}

valid_commit() { printf '%s' "$1" | grep -Eq '^[0-9a-f]{40}$'; }

case "${1:-}" in
get)
    [ $# -eq 2 ] || usage
    pins=$([ -f "$LOCK" ] && awk -v s="$2" '$1 == s { print $2 }' "$LOCK" || true)
    count=$(printf '%s' "$pins" | grep -c . || true)
    if [ "$count" -ne 1 ]; then
        echo "error: $LOCK pins $2 $count times, not once; \`make docs-bump\` pins the latest" >&2
        exit 1
    fi
    if ! valid_commit "$pins"; then
        echo "error: $LOCK pins $2 at \"$pins\", which is not a commit" >&2
        exit 1
    fi
    echo "$pins"
    ;;
check)
    [ $# -eq 2 ] || usage
    if ! valid_commit "$2"; then
        echo "error: \"$2\" is not a 40-character commit" >&2
        exit 1
    fi
    ;;
set)
    [ $# -eq 3 ] || usage
    src=$2
    value=$3
    if ! valid_commit "$value"; then
        echo "error: will not pin $src at \"$value\": not a 40-character commit" >&2
        exit 1
    fi
    # This run's own (docs_scratch), renamed into place under the lock, and
    # removed if this stops first.
    docs_still_locked
    docs_clear_scratch "$LOCK.tmp"
    tmp=$(docs_scratch "$LOCK.tmp")
    trap 'docs_cleanup_begin; rm -f "$tmp"' EXIT
    {
        if [ -f "$LOCK" ]; then
            awk -v s="$src" '$1 != s' "$LOCK"
        else
            echo "# Pinned doc sources: scripts/fetch-docs-cfdocs.sh checks out exactly this"
            echo "# commit, so the same lock regenerates the same docs (Lucee's docs are vendored"
            echo "# in docs/lucee instead). \`make docs-bump\` moves it to upstream's latest; review"
            echo "# the generated_docs.go diff that \`make generate\` then makes."
        fi
        echo "$src $value"
    } > "$tmp"
    docs_still_locked
    mv "$tmp" "$LOCK"
    echo "Pinned $src at $value in $LOCK"
    ;;
*)
    usage
    ;;
esac
