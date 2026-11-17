#!/usr/bin/env bash
set -euo pipefail

# Under docs/.lock (scripts/docslock.go), like every docs script: see
# lib/underlock.sh.
# shellcheck source=lib/underlock.sh
. "$(dirname "$0")/lib/underlock.sh"

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

ORDER="docs/data/order"
SOURCES="docs/src/cfdocs docs/retained docs/lucee"

# shellcheck source=lib/swapdir.sh
. scripts/lib/swapdir.sh

trap 'docs_cleanup_begin; rm -f "$ORDER.tmp"' EXIT

# An interrupted fetch or bump may have left a copy aside: put each back
# before checking for it.
swap_recover docs/src/cfdocs
swap_recover docs/lucee

# cfdocs and Lucee must be here: a missing one is an error, never left out of
# the order. cfdocs is the one fetched (staged under docs/src/); docs/lucee is
# committed (Lucee's docs are vendored: it publishes only its latest zip, so a
# version cannot be fetched again). docs/retained may hold no JSON at all: the
# generator says a retained file can be deleted once Lucee documents its name
# again, and deleting the last one must not break anything.
if [ ! -d docs/src/cfdocs ] || [ ! -f docs/.sha-cfdocs ]; then
    echo "error: cfdocs is not staged; run \`make docs\` to fetch it" >&2
    exit 1
fi
if ! ls docs/lucee/*.json >/dev/null 2>&1; then
    echo "error: no vendored Lucee docs in docs/lucee; restore them (git checkout -- docs/lucee)" >&2
    exit 1
fi
mkdir -p docs/retained

# One small file, written beside itself and renamed into place: a rename is
# atomic, so a failure or interrupt leaves the previous order whole.
mkdir -p "$(dirname "$ORDER")"
: > "$ORDER.tmp"
for src in $SOURCES; do
    echo "${src##*/} $src" >> "$ORDER.tmp"
    echo "  $src → $(find "$src" -maxdepth 1 -name '*.json' | wc -l | tr -d ' ') JSON files, read in place"
done
mv "$ORDER.tmp" "$ORDER"

echo "Assembled $ORDER"
