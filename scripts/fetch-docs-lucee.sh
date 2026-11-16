#!/usr/bin/env bash
set -euo pipefail

# Lucee's docs are vendored in docs/lucee (committed): Lucee publishes only
# its latest lucee-docs.zip, so a version cannot be pinned and fetched again
# later, only kept. A plain run uses the committed copy and touches no
# network, so a blocked docs.lucee.org cannot stall `make generate`.
# DOCS_BUMP=1 (make docs-bump) downloads the latest, replaces docs/lucee's
# JSON with it, and records the zip, and a hash of the JSON vendored from it,
# in docs/lucee/SOURCE, so the change arrives as a diff to review and a hand
# edit to the vendored files fails TestVendoredLuceeMatchesItsSource.

# Every path here is relative to the repository root, wherever this is run
# from (clif may be a subdirectory of another repository).
cd "$(dirname "$0")/.."

DIR="docs/lucee"
SOURCE="$DIR/SOURCE"
URL="https://docs.lucee.org/lucee-docs.zip"
NEW="$DIR.new"

# shellcheck source=lib/sha256.sh
. scripts/lib/sha256.sh
# shellcheck source=lib/swapdir.sh
. scripts/lib/swapdir.sh
# shellcheck source=lib/retry.sh
. scripts/lib/retry.sh

# The trap before the lock: an exit the moment the lock is taken still
# releases it. Until then this run touches nothing shared.
TMP=""
cleanup() {
    if [ -n "$TMP" ]; then
        rm -rf "$TMP"
    fi
    if docs_have_lock; then
        swap_recover "$DIR"
        rm -rf "$NEW"
    fi
    docs_unlock
}
trap cleanup EXIT
docs_lock

# On every run, not just a bump: an interrupted bump may have left the
# committed copy aside in docs/lucee.old.
swap_recover "$DIR"

# A plain run only reports; scripts/assemble-docs.sh is the one place that
# requires the vendored copy to be there.
if [ "${DOCS_BUMP:-}" != "1" ]; then
    echo "Lucee docs vendored in $DIR ($(awk '$1 == "zip-sha256" { print "zip sha256 " substr($2, 1, 12) }' "$SOURCE" 2>/dev/null)); \`make docs-bump\` takes Lucee's latest"
    exit 0
fi

TMP=$(mktemp -d)
rm -rf "$NEW"

echo "Downloading lucee-docs.zip..."
if ! retry curl -fsSL --connect-timeout 10 --max-time 600 -D "$TMP/headers" "$URL" -o "$TMP/lucee-docs.zip"; then
    echo "error: could not download $URL; $DIR left as it was" >&2
    exit 1
fi

echo "Extracting..."
unzip -qo "$TMP/lucee-docs.zip" "lucee-docs-json-zipped.zip" -d "$TMP"
unzip -qo "$TMP/lucee-docs-json-zipped.zip" "lucee-docs-json.zip" -d "$TMP"
unzip -qo "$TMP/lucee-docs-json.zip" "*.json" -d "$TMP/staged"

count=$(find "$TMP/staged" -maxdepth 1 -name '*.json' | wc -l | tr -d ' ')
if [ "$count" -eq 0 ]; then
    echo "error: $URL held no JSON docs; $DIR left as it was" >&2
    exit 1
fi


# The ETag of the response that delivered this zip, not of a separate request
# that may have seen another version. curl -L dumps every response's headers
# in turn; each status line starts a new one, so only the final response's
# ETag survives, and none (rather than a redirect's) when it sends none.
# The header name is matched on its own, so `ETag:"x"` (no space after the
# colon, as valid HTTP allows) is read as well as `ETag: "x"`.
etag=$(tr -d '\r' < "$TMP/headers" | awk '
    /^HTTP\// { e = "" }
    tolower(substr($0, 1, 5)) == "etag:" { v = substr($0, 6); gsub(/^[ \t]+|[ \t]+$/, "", v); e = v }
    END { print e }')

# Build the new copy beside the old one, then swap: a failure before the swap
# leaves docs/lucee as it was, and one during it is undone by cleanup.
mkdir -p "$NEW"
cp "$TMP/staged/"*.json "$NEW/"

# One pass over the copy: a file that does not parse, or names no valid
# entry, is refused here, before it can be committed (vendored, it would sit
# in docs/lucee for good; make generate only skips it, as upstream's to fix),
# and otherwise the hash of the JSON is what SOURCE records.
if ! json_hash=$(go run scripts/check_vendored.go checkhash "$NEW"); then
    echo "error: Lucee's zip holds the broken files above; $DIR left as it was. Report them upstream and bump later." >&2
    exit 1
fi
{
    if [ -f "$SOURCE" ]; then
        sed -n '/^#/p' "$SOURCE"
    else
        echo "# Lucee's docs, vendored: the JSON files from $URL"
        echo "# (lucee-docs-json-zipped.zip > lucee-docs-json.zip). Lucee publishes only its"
        echo "# latest zip, so a pinned one could not be downloaded again; committing the"
        echo "# extracted files is what makes a regeneration reproducible on any machine."
        echo "# \`make docs-bump\` replaces them with the latest and rewrites this file."
    fi
    echo "zip-sha256 $(sha256 "$TMP/lucee-docs.zip")"
    echo "json-sha256 $json_hash"
    if [ -n "$etag" ]; then
        echo "etag $etag"
    fi
    echo "fetched $(date -u +%Y-%m-%d)"
} > "$NEW/SOURCE"

# Anything else committed beside the JSON (notes, a README, a subdirectory)
# comes along: only the top-level JSON and SOURCE are Lucee's to replace.
if [ -d "$DIR" ]; then
    find "$DIR" -mindepth 1 -maxdepth 1 ! -name '*.json' ! -name SOURCE -exec cp -Rp {} "$NEW/" \;
fi

# docs/lucee may be missing (deleted, or never checked out): swap_in then
# just puts the new copy in place.
swap_in "$NEW" "$DIR"

echo "Vendored $count Lucee JSON files in $DIR; review and commit the diff"
