# shellcheck shell=bash
# Sourced first by every docs script that changes anything: runs the script
# under the docs lock, held by scripts/docslock.go (the operating system's own
# file lock on docs/.flock, so it is released however its holder ends, and no
# stale lock is ever left to judge).
#
# A script started under the lock carries on; any other re-runs itself through
# docslock, from the repository root. Under the lock means `docslock --held`
# agrees: the DOCS_LOCK_TOKEN docslock gave the run is the one recorded in
# docs/.flock, and the lock is held. docslock clears its record as it exits,
# so a token leaked from a run that has ended (a shell opened from one)
# matches nothing, and such a script takes the lock itself.
#
# docslock is built and run, not `go run`: that turns a failing exit, a stop
# by a signal included, into exit 1, losing the code the script exits with.
# Its binary is named for a hash of its sources, so a changed source builds a
# new one beside it and a copy a run is executing is never replaced (Windows
# refuses that outright); a run's scripts ask the copy it runs as
# (DOCS_LOCK_BIN), whatever is checked out meanwhile.
#
# Sourced as: . "$(dirname "$0")/lib/underlock.sh"  (before anything else)
docs_root=$(cd "$(dirname "$0")/.." && pwd) || exit 1

docs_held=""
if [ -n "${DOCS_LOCK_TOKEN:-}" ]; then
    # Inside a run, or a token leaked from one. Inside a run whose helper has
    # gone (target/ deleted meanwhile), taking the lock again would only find
    # this run's own docslock holding it: say what happened instead.
    if [ -z "${DOCS_LOCK_BIN:-}" ] || [ ! -x "$DOCS_LOCK_BIN" ]; then
        echo "error: DOCS_LOCK_TOKEN is set, but the docs lock's helper (${DOCS_LOCK_BIN:-not named}) is missing:" >&2
        echo "       was target/ deleted during a docs run? Outside one, unset DOCS_LOCK_TOKEN and run this again." >&2
        exit 1
    fi
    if (cd "$docs_root" && "$DOCS_LOCK_BIN" --held); then
        docs_held=1
    fi
fi

if [ -z "$docs_held" ]; then
    cd "$docs_root" || exit 1
    docs_hash=$(cat scripts/docslock.go internal/flock/*.go go.mod go.sum |
        { if command -v sha256sum >/dev/null 2>&1; then sha256sum; else shasum -a 256; fi; } | cut -c1-16) || exit 1
    case "$(uname -s 2>/dev/null)" in
    MINGW* | MSYS* | CYGWIN*) docslock="target/docs-gen/docslock-$docs_hash.exe" ;;
    *) docslock="target/docs-gen/docslock-$docs_hash" ;;
    esac
    if [ ! -x "$docslock" ]; then
        mkdir -p target/docs-gen
        go build -o "$docslock.$$.tmp" scripts/docslock.go || exit 1
        mv -f "$docslock.$$.tmp" "$docslock" || exit 1
        # Builds of older sources are left: one may be what a run is
        # executing. They are small; `make clean` removes them.
    fi
    exec "$docslock" bash "scripts/${0##*/}" "$@"
fi
unset docs_held

# docs_holds_lock: whether this run still holds the lock. Should its docslock
# have been killed, which frees the lock at once, another run may have taken
# it, and every path the two share (DIR.new, DIR.old, the order's temporary
# file, the cfdocs marker) is then that run's to change, not this one's.
docs_holds_lock() { (cd "$docs_root" && "$DOCS_LOCK_BIN" --held); }

# docs_still_locked: exit 1 unless this run still holds the lock. Called just
# before every change a docs script makes to a shared path (lib/swapdir.sh's
# swap_in and swap_recover, each rm of one, the order and pin renames); an
# EXIT trap tests docs_holds_lock instead, and leaves those paths alone.
docs_still_locked() {
    if ! docs_holds_lock; then
        echo "error: this run no longer holds the docs lock (its docslock was stopped?): stopping before changing anything" >&2
        exit 1
    fi
}

# docslock passes a signal that stops the run to the whole of it, this script
# and what it is running at once. bash acts on a trapped signal only once the
# command it is waiting on has finished, so these let that command stop (the
# generator finish writing, say) before the EXIT trap cleans up after it.
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

# docs_cleanup_begin: the first line of a script's EXIT trap. A second signal
# there would cut the cleanup short; none is acted on from then on.
docs_cleanup_begin() { trap '' INT TERM HUP; }
