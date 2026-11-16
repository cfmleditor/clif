#!/usr/bin/env bash
set -euo pipefail

# Stages cfdocs JSON in docs/src/cfdocs, which scripts/generate_docs.go reads
# in place (docs/data/order points it there).
#
# The commit fetched is the one docs/sources.lock pins, so a regeneration is
# reproducible: the same lock gives the same docs. A missing pin is an error,
# never filled in from upstream: only `make docs-bump` moves the lock. With
# DOCS_BUMP=1 this stages master's current commit instead and leaves the lock
# alone; make docs-bump writes it once every source has staged.

# Every path here is relative to the repository root, wherever this is run
# from (clif may be a subdirectory of another repository).
cd "$(dirname "$0")/.."

SRC="docs/src/cfdocs"
REPO="https://github.com/cfmleditor/cfdocs.git"
SHA_FILE="docs/.sha-cfdocs"
NEW="$SRC.new"

# shellcheck source=lib/swapdir.sh
. scripts/lib/swapdir.sh
# shellcheck source=lib/retry.sh
. scripts/lib/retry.sh

# The trap before the lock: an exit the moment the lock is taken still
# releases it. Until then this run touches nothing shared.
TMP=$(mktemp -d)
cleanup() {
    rm -rf "$TMP"
    if docs_have_lock; then
        swap_recover "$SRC"
        rm -rf "$NEW"
    fi
    docs_unlock
}
trap cleanup EXIT
docs_lock

# An interrupted earlier fetch may have left the previous copy aside (and no
# marker, so it is never mistaken for the pin): put it back.
swap_recover "$SRC"

if [ "${DOCS_BUMP:-}" = "1" ]; then
    if ! remote=$(retry git ls-remote "$REPO" refs/heads/master); then
        echo "error: could not reach $REPO to read cfdocs' master commit" >&2
        exit 1
    fi
    target=$(printf '%s\n' "$remote" | cut -f1)
    if [ -z "$target" ]; then
        echo "error: $REPO has no master branch" >&2
        exit 1
    fi
else
    target=$(bash scripts/docs-lock.sh get cfdocs)
fi

if [ -f "$SHA_FILE" ] && [ -d "$SRC" ] && [ "$(cat "$SHA_FILE")" = "$target" ]; then
    echo "cfdocs at ${target:0:8}, already staged"
    exit 0
fi

echo "Downloading cfdocs data/en at ${target:0:8}..."
git init -q "$TMP/cfdocs"
git -C "$TMP/cfdocs" remote add origin "$REPO"
git -C "$TMP/cfdocs" sparse-checkout set data/en
if ! retry git -C "$TMP/cfdocs" fetch -q --depth 1 --filter=blob:none origin "$target"; then
    echo "error: could not fetch cfdocs commit $target; if it is gone upstream, \`make docs-bump\` pins master" >&2
    exit 1
fi
# The checkout fetches the blobs it needs (a partial clone), so it can meet
# the same blip.
if ! retry git -C "$TMP/cfdocs" checkout -q FETCH_HEAD; then
    echo "error: could not check out cfdocs commit $target" >&2
    exit 1
fi

if [ ! -d "$TMP/cfdocs/data/en" ]; then
    echo "error: cfdocs commit $target has no data/en; $SRC left as it was" >&2
    exit 1
fi
count=$(find "$TMP/cfdocs/data/en" -maxdepth 1 -name '*.json' | wc -l | tr -d ' ')
if [ "$count" -eq 0 ]; then
    echo "error: cfdocs commit $target has no data/en/*.json; $SRC left as it was" >&2
    exit 1
fi

# Built beside the staged copy and swapped in, with the commit marker removed
# first and written only once the new copy is in place: a copy that fails or
# is interrupted part way can never leave a partial directory that the marker
# still calls the pinned commit.
rm -rf "$NEW"
mkdir -p "$NEW"
cp "$TMP/cfdocs/data/en/"*.json "$NEW/"

rm -f "$SHA_FILE"
swap_in "$NEW" "$SRC"
echo "$target" > "$SHA_FILE"

echo "Copied $count JSON files to $SRC/"
