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

# docslock reports its own path as the OS gives it, which on Windows is
# C:\...: as an MSYS or Cygwin path, for [ -x ] and to run it.
case "$(uname -s 2>/dev/null)" in
MINGW* | MSYS* | CYGWIN*)
    if [ -n "${DOCS_LOCK_BIN:-}" ] && command -v cygpath >/dev/null 2>&1; then
        DOCS_LOCK_BIN=$(cygpath -u "$DOCS_LOCK_BIN")
    fi
    ;;
esac

docs_held=""
if [ -n "${DOCS_LOCK_TOKEN:-}" ]; then
    # Inside a run, or a token leaked from one. Inside a run whose helper has
    # gone (target/ deleted meanwhile), taking the lock again would only find
    # this run's own docslock holding it: say what happened instead.
    if [ -n "${DOCS_LOCK_BIN:-}" ] && [ -x "$DOCS_LOCK_BIN" ]; then
        if (cd "$docs_root" && "$DOCS_LOCK_BIN" --held); then
            docs_held=1
        fi
    elif docs_record=$(awk '{ print $1, $2 }' "$docs_root/docs/.flock" 2>/dev/null) &&
        [ "${docs_record#* }" = "$DOCS_LOCK_TOKEN" ] && ps -p "${docs_record%% *}" >/dev/null 2>&1; then
        # The record names this token, and its holder lives, so this is
        # inside a live run whose helper has gone. A token that matches
        # nothing, or only the record of a holder that has died (killed
        # before it could clear it), was leaked from a run that has ended,
        # and the lock is taken as usual below.
        echo "error: the docs lock's helper (${DOCS_LOCK_BIN:-not named}) is gone during this docs run:" >&2
        echo "       was target/ deleted meanwhile? Let the run finish, then run this again." >&2
        exit 1
    fi
fi

if [ -z "$docs_held" ]; then
    cd "$docs_root" || exit 1
    # Hashed: what docslock is built from, as `go list -deps` finds it, so
    # nothing else forces a rebuild and nothing it comes to import is missed.
    # This module's own packages by their sources (not their tests), a
    # dependency's by its directory in the module cache, which names its
    # version; go.mod for the toolchain.
    # shellcheck source=sha256.sh
    . scripts/lib/sha256.sh
    docs_deps=$(go list -deps -f '{{if not .Standard}}{{.Dir}}{{end}}' scripts/docslock.go) || exit 1
    docs_hash=$({
        cat scripts/docslock.go go.mod
        printf '%s\n' "$docs_deps" | while IFS= read -r d; do
            case "$d" in
            "$docs_root"/scripts) ;; # docslock.go itself, above; not its neighbours
            "$docs_root"/*)
                for f in "$d"/*.go; do
                    case "$f" in *_test.go) ;; *) cat "$f" ;; esac
                done
                ;;
            *) echo "$d" ;;
            esac
        done
    } | sha256 /dev/stdin | cut -c1-16) || exit 1
    case "$(uname -s 2>/dev/null)" in
    MINGW* | MSYS* | CYGWIN*) docslock="target/docs-gen/docslock-$docs_hash.exe" ;;
    *) docslock="target/docs-gen/docslock-$docs_hash" ;;
    esac
    if [ ! -x "$docslock" ]; then
        mkdir -p target/docs-gen
        go build -o "$docslock.$$.tmp" scripts/docslock.go || exit 1
        # Another run starting at once may have built the same one first, and
        # be executing it: then this build is not needed, and is not moved
        # over it (Windows would refuse).
        if [ -e "$docslock" ]; then
            rm -f "$docslock.$$.tmp"
        else
            mv "$docslock.$$.tmp" "$docslock" || exit 1
        fi
        # Builds of older sources are left: one may be what a run is
        # executing. They are small; `make clean` removes them.
    fi
    # DOCS_LOCK_THEN (set by a script before sourcing this, not exported):
    # a program docslock runs once the script has succeeded, passing it the
    # lock's descriptor (scripts/generate.sh's generator).
    exec "$docslock" ${DOCS_LOCK_THEN:+--then "$DOCS_LOCK_THEN"} bash "scripts/${0##*/}" "$@"
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

# A terminal's Ctrl-C or hang-up reaches this script and what it is running
# at once (they share docslock's process group); a SIGTERM, or a signal sent
# to docslock alone, docslock passes to the script it started. bash acts on a
# trapped signal only once the command it is waiting on has finished, so
# these let that command stop (or, for a signal that reached this script
# alone, finish) before the EXIT trap cleans up after it. Each then ends this
# script by the same signal, not just its exit code, so what ran it sees an
# interrupted run; docslock ends by it in turn (flock.Raise).
trap 'trap - INT; kill -INT $$' INT
trap 'trap - TERM; kill -TERM $$' TERM
trap 'trap - HUP; kill -HUP $$' HUP

# docs_cleanup_begin: the first line of a script's EXIT trap. A second signal
# there would cut the cleanup short; none is acted on from then on.
docs_cleanup_begin() { trap '' INT TERM HUP; }
