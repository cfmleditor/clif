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
# It is built to a path named for a hash of its sources (lib/hashbuild.sh),
# and a run's scripts ask the copy it runs as (DOCS_LOCK_BIN), whatever is
# checked out or built meanwhile.
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

# shellcheck source=hashbuild.sh
. "$docs_root/scripts/lib/hashbuild.sh"

docs_held=""
if [ -n "${DOCS_LOCK_TOKEN:-}" ]; then
    # Inside a run, or a token leaked from one: asked of the copy of docslock
    # the run runs as. Should that be gone (target/ deleted meanwhile), it is
    # built again from the sources, so a run that lost it carries on, and a
    # token leaked from a run that has ended still matches nothing.
    if [ -z "${DOCS_LOCK_BIN:-}" ] || [ ! -x "$DOCS_LOCK_BIN" ]; then
        DOCS_LOCK_BIN=$(cd "$docs_root" && docs_built_path scripts/docslock.go docslock) &&
            (cd "$docs_root" && docs_build scripts/docslock.go "$DOCS_LOCK_BIN") || exit 1
        DOCS_LOCK_BIN="$docs_root/$DOCS_LOCK_BIN"
        export DOCS_LOCK_BIN
    fi
    if (cd "$docs_root" && "$DOCS_LOCK_BIN" --held); then
        docs_held=1
    fi
fi

if [ -z "$docs_held" ]; then
    cd "$docs_root" || exit 1
    docslock=$(docs_built_path scripts/docslock.go docslock) || exit 1
    docs_build scripts/docslock.go "$docslock" || exit 1
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
