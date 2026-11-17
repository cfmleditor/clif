# shellcheck shell=bash
# The docs pipeline's Go programs (scripts/docslock.go, scripts/generate_docs.go),
# each kept at a path named for a hash of the binary itself: a new build that
# differs lands beside the old, so a copy some run is executing is never
# replaced (Windows refuses that outright). Sourced by lib/underlock.sh and
# scripts/generate.sh; every path is relative to the repository root.

# shellcheck source=sha256.sh
. "$(dirname "${BASH_SOURCE[0]}")/sha256.sh"

# docs_build MAIN.go NAME: build MAIN.go and print the path it is kept at,
# target/docs-gen/NAME-<hash of the binary>. Built every time: go's build
# cache makes that a fraction of a second when nothing changed, and it is
# complete by construction, as whatever the binary depends on (sources at any
# depth, embedded files, cgo and compiler settings, the toolchain) is in the
# binary. -buildvcs=false keeps the repository's state out of it, so the same
# sources give the same binary. Built to a temporary file of this process's
# own; kept where it is if that binary is there already (its time stamp
# refreshed, as docslock prunes builds by how long they go unused), moved
# into place otherwise, never over one (mv -n: another run starting at once
# may have put the same binary there first, and be executing it).
docs_build() {
    local main=$1 name=$2 ext="" tmp hash path
    case "$(uname -s 2>/dev/null)" in
    MINGW* | MSYS* | CYGWIN*) ext=.exe ;;
    esac
    mkdir -p target/docs-gen || return 1
    tmp="target/docs-gen/$name.$$.tmp"
    go build -buildvcs=false -o "$tmp" "$main" || { rm -f "$tmp"; return 1; }
    hash=$(sha256 "$tmp" | cut -c1-16) || { rm -f "$tmp"; return 1; }
    path="target/docs-gen/$name-$hash$ext"
    if [ -x "$path" ]; then
        rm -f "$tmp"
        touch -c "$path" 2>/dev/null || true
    else
        # Not executable: a stub left by an interrupted move. Never a binary
        # a run is executing, which would be executable.
        [ -e "$path" ] && rm -f "$path"
        mv -n "$tmp" "$path" 2>/dev/null || true
        rm -f "$tmp"
    fi
    [ -x "$path" ] || return 1
    printf '%s\n' "$path"
}
