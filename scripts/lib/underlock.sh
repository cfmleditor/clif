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
# It is kept at a path named for a hash of the binary (lib/hashbuild.sh),
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

# docs_lock_bin: make sure DOCS_LOCK_BIN, the copy of docslock the run runs
# as, is there to ask. Should it be gone (target/ deleted meanwhile, by a
# `make clean` elsewhere), it is built again from the sources: a missing
# helper is not a lost lock, and a check must not report it as one.
docs_lock_bin() {
    if [ -n "${DOCS_LOCK_BIN:-}" ] && [ -x "$DOCS_LOCK_BIN" ]; then
        return 0
    fi
    DOCS_LOCK_BIN=$(cd "$docs_root" && docs_build scripts/docslock.go docslock) || return 1
    DOCS_LOCK_BIN="$docs_root/$DOCS_LOCK_BIN"
    export DOCS_LOCK_BIN
}

docs_held=""
if [ -n "${DOCS_LOCK_TOKEN:-}" ]; then
    # Inside a run, or a token leaked from one: asked of the copy of docslock
    # the run runs as (rebuilt if gone, so a run that lost it carries on; a
    # token leaked from a run that has ended still matches nothing).
    #
    # A run whose docslock has ended (killed, its record left naming this
    # token) stops here: taking the lock afresh would let this script, and
    # each step after it, run under short locks of their own, with other runs
    # free to slip in between them.
    docs_lock_bin || exit 1
    docs_status=0
    (cd "$docs_root" && "$DOCS_LOCK_BIN" --held) || docs_status=$?
    case "$docs_status" in
    0) docs_held=1 ;;
    3)
        echo "error: this docs run's docslock has ended (killed?), so nothing holds the lock for it any more:" >&2
        echo "       stopping. Start the run again." >&2
        exit 1
        ;;
    esac
    unset docs_status
fi

if [ -z "$docs_held" ]; then
    cd "$docs_root" || exit 1
    docslock=$(docs_build scripts/docslock.go docslock) || exit 1
    # generate.sh, alone, is run with --then: docslock runs the program it
    # names once it has succeeded, the generator, passing it the lock's
    # descriptor. Chosen by the script's name, not by anything in the
    # environment, which a value left there could switch on or off.
    docs_then=""
    case "${0##*/}" in
    generate.sh) docs_then=--then ;;
    esac
    exec "$docslock" $docs_then bash "scripts/${0##*/}" "$@"
fi
unset docs_held

# Scratch: each run builds what it then swaps or renames into place under
# names of its own, BASE.<its token> (docs_scratch), never a name another run
# could be writing. So a run whose docslock was killed can only ever touch
# its own scratch, and only the final swap, rename or marker write, the step
# that changes what other runs read, needs docs_still_locked just before it.
# docs_clear_scratch removes what killed runs left (BASE, as earlier versions
# named it, and BASE.*): call it holding the lock, before writing this run's.
docs_scratch() { printf '%s.%s\n' "$1" "${DOCS_LOCK_TOKEN:0:16}"; }
docs_clear_scratch() {
    local s
    for s in "$1" "$1".*; do
        rm -rf "$s"
    done
}

# docs_holds_lock: whether this run still holds the lock. Should its docslock
# have been killed, which frees the lock at once, another run may have taken
# it, and every path the two share (DIR.new, DIR.old, the order's temporary
# file, the cfdocs marker) is then that run's to change, not this one's.
docs_holds_lock() { docs_lock_bin && (cd "$docs_root" && "$DOCS_LOCK_BIN" --held); }

# docs_still_locked: exit 1 unless this run still holds the lock. Called just
# before every change a docs script makes to what other runs read (a swap or
# a recovery, the order and pin renames, the cfdocs marker, clearing others'
# scratch); an EXIT trap tests docs_holds_lock instead, and leaves those
# alone. A run's own scratch (docs_scratch) needs no check.
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
#
# The first of them ignores every one from then on: a second copy (a signal
# sent to docslock's whole process group reaches this script directly, and
# again through docslock, which cannot tell it from one sent to it alone)
# then does nothing, rather than cut the cleanup short.
#
# The script's EXIT trap is run here, before the re-raise, and then cleared:
# a bash that a signal kills (its default restored, to end by it) runs no
# EXIT trap of its own, so the cleanup would never happen.
docs_die() {
    trap '' INT TERM HUP
    # The cleanup runs whatever fails in it: under the script's set -e, a
    # failing line would end the script there, by its code, not the signal.
    set +e
    # trap -p prints the EXIT trap as a command, `trap -- '<it>' EXIT`,
    # quoted as bash reads it: bash itself parses it back, quotes and all.
    local -a on_exit=()
    eval "on_exit=($(trap -p EXIT))"
    trap - EXIT
    if [ -n "${on_exit[2]:-}" ]; then
        eval "${on_exit[2]}"
    fi
    trap - "$1"
    kill -"$1" $$
}
trap 'docs_die INT' INT
trap 'docs_die TERM' TERM
trap 'docs_die HUP' HUP

# docs_cleanup_begin: the first line of a script's EXIT trap. A second signal
# there would cut the cleanup short; none is acted on from then on.
docs_cleanup_begin() { trap '' INT TERM HUP; }
