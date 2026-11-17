# shellcheck shell=bash
# Sourced first by every docs script: runs the script under docs/.lock, held by
# scripts/docslock.go (the operating system's own file lock, so it is released
# however its holder ends, and no stale lock is ever left to judge).
#
# A script started by one that holds the lock carries on, inheriting it; any
# other re-runs itself through docslock, from the repository root. Holding it
# means docslock --held agrees: the descriptor passed as fd 3 is docs/.lock as
# it is on disk (on Windows, which passes a child no descriptor, that some
# process holds the lock). DOCS_LOCK_HELD alone proves nothing: it is exported
# to everything under a docs run, a shell opened from one included.
#
# docslock is built and run, not `go run`: that turns a failing exit, a stop
# by a signal included, into exit 1, losing the code the script exits with.
#
# Sourced as: . "$(dirname "$0")/lib/underlock.sh"  (before anything else)
docs_root=$(cd "$(dirname "$0")/.." && pwd) || exit 1
case "$(uname -s 2>/dev/null)" in
MINGW* | MSYS* | CYGWIN*) docslock="$docs_root/target/docs-gen/docslock.exe" ;;
*) docslock="$docs_root/target/docs-gen/docslock" ;;
esac

# Started under the lock: the DOCS_LOCK_HELD flag, and docslock --held finds
# fd 3 is docs/.lock as it is on disk (bash's own [ -ef ] cannot say: macOS
# reports /dev/fd/3 on the descriptor filesystem's device).
if [ "${DOCS_LOCK_HELD:-}" != 1 ] || [ ! -x "$docslock" ] ||
    ! (cd "$docs_root" && "$docslock" --held); then
    cd "$docs_root" || exit 1
    # Built only when missing or older than its source: never over a copy
    # another run is executing (which Windows refuses outright), and through
    # a temporary file renamed into place, so no run starts a half-written one.
    if [ ! -x "$docslock" ] || [ -n "$(find scripts/docslock.go internal/flock go.mod go.sum \
        \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -newer "$docslock" 2>/dev/null | head -n 1)" ]; then
        mkdir -p target/docs-gen
        go build -o "$docslock.$$.tmp" scripts/docslock.go || exit 1
        mv -f "$docslock.$$.tmp" "$docslock" || exit 1
    fi
    exec "$docslock" bash "scripts/${0##*/}" "$@"
fi
unset docs_root docslock

# docslock sends a signal that stops the run to the whole of it, this script
# and what it is running at once. bash acts on a trapped signal only once the
# command it is waiting on has finished, so these let that command stop (the
# generator finish writing, say) before the EXIT trap cleans up after it.
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

# docs_cleanup_begin: the first line of a script's EXIT trap. A second signal
# there would cut the cleanup short; none is acted on from then on.
docs_cleanup_begin() { trap '' INT TERM HUP; }

# go and git run without fd 3, docs/.lock's descriptor: both can leave a
# process behind (the go command's telemetry, git's detached gc or
# maintenance), which would hold the lock after the run ends. A docs script
# or the generator binary, which must hold it, is never run through these.
# git's background gc and maintenance are off too, and it never prompts.
go() { command go "$@" 3>&-; }
git() { GIT_TERMINAL_PROMPT=0 command git -c gc.auto=0 -c maintenance.auto=false "$@" 3>&-; }
