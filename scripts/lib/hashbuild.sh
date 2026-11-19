# shellcheck shell=bash
# Build the Go docs driver before it takes the lock. A binary hash keeps an
# executing copy intact, including on Windows; concurrent builds use their own
# scratch names and never overwrite a complete binary. Only make clean removes
# these binaries, since a caller may be about to execute one.

# shellcheck source=sha256.sh
. "$(dirname "${BASH_SOURCE[0]}")/sha256.sh"

# docs_build PACKAGE NAME: build PACKAGE and print the path it is kept at,
# target/docs-gen/NAME-<hash of the binary>. Built every time: go's build
# cache makes that a fraction of a second when nothing changed, and it is
# complete by construction, as whatever the binary depends on (sources at any
# depth, embedded files, cgo and compiler settings, the toolchain) is in the
# binary. -buildvcs=false keeps the repository's state out of it and -trimpath
# its absolute paths, so the same sources give the same binary in any
# checkout. Built to a temporary file of this process's
# own; kept where it is if that binary is there already, moved into place
# otherwise, never over one (mv -n: another run starting at once may have put
# the same binary there first, and be executing it). Builds are never
# removed but by `make clean`: a run may be about to execute any of them.
docs_build() {
    local main=$1 name=$2 ext="" tmp hash path
    case "$(uname -s 2>/dev/null)" in
    MINGW* | MSYS* | CYGWIN*) ext=.exe ;;
    esac
    mkdir -p target/docs-gen || return 1
    tmp="target/docs-gen/$name.$$.tmp"
    go build -trimpath -buildvcs=false -o "$tmp" "$main" || { rm -f "$tmp"; return 1; }
    hash=$(sha256 "$tmp" | cut -c1-16) || { rm -f "$tmp"; return 1; }
    path="target/docs-gen/$name-$hash$ext"
    if [ -x "$path" ]; then
        rm -f "$tmp"
    else
        mv -n "$tmp" "$path" 2>/dev/null || true
        rm -f "$tmp"
    fi
    [ -x "$path" ] || return 1
    printf '%s\n' "$path"
}
