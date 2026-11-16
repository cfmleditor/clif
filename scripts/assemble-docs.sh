#!/usr/bin/env bash
set -euo pipefail

# Writes docs/data/order, which tells scripts/generate_docs.go which doc
# sources to read, from lowest precedence to highest (a later one wins, per
# entry), and the directory each is read from. Every source is read in place,
# never copied: the fetched cfdocs from docs/src/cfdocs (fetch-docs-cfdocs.sh
# swaps it in whole), docs/retained and the vendored docs/lucee from their
# committed directories. A copy could go stale when assembling failed; there
# is none.
#
# SOURCES is the only place the precedence is written down. cfdocs comes
# first; docs/retained (entries Lucee once published and then dropped,
# committed, see docs/retained/README.md) outranks it, so a retained Lucee
# signature beats cfdocs' Adobe one; Lucee comes last, and wins when it
# publishes a retained name again (the generator then says the retained copy
# can go).

# Every path here is relative to the repository root, wherever this is run
# from (clif may be a subdirectory of another repository).
cd "$(dirname "$0")/.."

DEST="docs/data"
SOURCES="docs/src/cfdocs docs/retained docs/lucee"

# shellcheck source=lib/swapdir.sh
. scripts/lib/swapdir.sh

docs_lock
BUILD="$DEST.building"
cleanup() {
    swap_recover "$DEST"
    rm -rf "$BUILD"
    docs_unlock
}
trap cleanup EXIT

# An interrupted fetch, bump or assemble may have left a copy aside: put each
# back before checking for it.
swap_recover docs/src/cfdocs
swap_recover docs/lucee
swap_recover "$DEST"

# Every source must be here: a missing one is an error, never left out of the
# order. cfdocs is the one fetched (staged under docs/src/); docs/retained and
# docs/lucee are committed (Lucee's docs are vendored: it publishes only its
# latest zip, so a version cannot be fetched again).
if [ ! -d docs/src/cfdocs ] || [ ! -f docs/.sha-cfdocs ]; then
    echo "error: cfdocs is not staged; run \`make docs\` to fetch it" >&2
    exit 1
fi
for src in docs/lucee docs/retained; do
    if ! ls "$src"/*.json >/dev/null 2>&1; then
        echo "error: no docs in $src; restore them (git checkout -- $src)" >&2
        exit 1
    fi
done

rm -rf "$BUILD"
mkdir -p "$BUILD"
: > "$BUILD/order"
for src in $SOURCES; do
    echo "${src##*/} $src" >> "$BUILD/order"
    echo "  $src → $(find "$src" -maxdepth 1 -name '*.json' | wc -l | tr -d ' ') JSON files, read in place"
done

# Swapped in with the old copy moved aside first: a failure or interrupt
# between the two moves leaves it aside, and cleanup (or the next run's start)
# puts it back, so never neither copy.
swap_in "$BUILD" "$DEST"

echo "Assembled $DEST/order"
