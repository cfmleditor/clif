# shellcheck shell=bash
# The docs pipeline's Go programs (scripts/docslock.go, scripts/generate_docs.go),
# built once per version of their sources, each to a path named for a hash of
# them: a changed source builds a new binary beside the old, so a copy some
# run is executing is never replaced (Windows refuses that outright), and an
# unchanged one is not built again. Sourced by lib/underlock.sh and
# scripts/generate.sh; every path is relative to the repository root.

# shellcheck source=sha256.sh
. "$(dirname "${BASH_SOURCE[0]}")/sha256.sh"

# docs_built_path MAIN.go NAME: print the path MAIN.go builds to, named NAME
# and the hash of what it is built from and how: go.mod and go.sum; the
# toolchain and the build settings (go version, GOFLAGS, GOTOOLCHAIN,
# CGO_ENABLED, GOOS, GOARCH, GOEXPERIMENT); and each package `go list -deps`
# names, a dependency in the module cache by its directory, which names its
# version, any other (this module's, a local `replace`) by its files (not
# its tests). Nothing else forces a rebuild, and nothing it comes to import
# is missed.
docs_built_path() {
    local main=$1 name=$2 root deps cache hash
    root=$(pwd) || return 1
    deps=$(go list -deps -f '{{if not .Standard}}{{.Dir}}{{end}}' "$main") || return 1
    cache=$(go env GOMODCACHE) || return 1
    # On Windows go prints C:\... paths and bash's pwd /c/...: the same form
    # for both, or nothing below would match.
    if command -v cygpath >/dev/null 2>&1; then
        deps=$(printf '%s\n' "$deps" | while IFS= read -r d; do cygpath -u "$d"; done) || return 1
        cache=$(cygpath -u "$cache") || return 1
    fi
    hash=$({
        cat "$main" go.mod
        cat go.sum 2>/dev/null
        go version
        go env GOFLAGS GOTOOLCHAIN CGO_ENABLED GOOS GOARCH GOEXPERIMENT
        printf '%s\n' "$deps" | while IFS= read -r d; do
            case "$d" in
            "$root"/scripts) ;; # MAIN.go itself, above; not its neighbours
            "$cache"/*) echo "$d" ;;
            *)
                echo "$d"
                for f in "$d"/*; do
                    case "$f" in *_test.go) ;; *) if [ -f "$f" ]; then cat "$f"; fi ;; esac
                done
                ;;
            esac
        done
    } | sha256 /dev/stdin | cut -c1-16) || return 1
    case "$(uname -s 2>/dev/null)" in
    MINGW* | MSYS* | CYGWIN*) echo "target/docs-gen/$name-$hash.exe" ;;
    *) echo "target/docs-gen/$name-$hash" ;;
    esac
}

# docs_build MAIN.go PATH: build MAIN.go to PATH (docs_built_path), unless it
# is there. Built to a temporary file and moved into place only if PATH is
# still free: another run starting at once may have built the same binary
# first, and be executing it, and it is not moved over (mv -n).
docs_build() {
    local main=$1 path=$2
    [ -x "$path" ] && return 0
    mkdir -p "$(dirname "$path")" || return 1
    go build -o "$path.$$.tmp" "$main" || return 1
    mv -n "$path.$$.tmp" "$path" 2>/dev/null || true
    rm -f "$path.$$.tmp"
    [ -x "$path" ] || return 1

    # Other versions of the same program, built from older sources, go: each
    # is several MB. Best-effort: Windows refuses to delete one a run is
    # executing, and on Unix that run keeps its copy until it ends (a docs
    # script that then finds its docslock gone builds the current one,
    # docs_lock_bin). A build in progress (*.tmp) is another run's, and left.
    local base=${path##*/} old
    for old in "$(dirname "$path")/${base%-*}" "$(dirname "$path")/${base%-*}"-*; do
        case "$old" in
        "$path" | *.tmp) ;;
        *) rm -f "$old" 2>/dev/null || true ;;
        esac
    done
}
