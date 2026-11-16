# shellcheck shell=bash
# Shared by the docs scripts (fetch-docs-cfdocs.sh, fetch-docs-lucee.sh,
# assemble-docs.sh), so they replace directories and exclude each other one
# way.
#
# Replacing a directory without ever leaving it missing or half-written: a
# caller builds the new copy beside DIR (in DIR.new, say), then swap_in moves
# DIR aside to DIR.old, the new copy into DIR, and deletes DIR.old. A failure
# or interrupt between the two moves leaves DIR missing and DIR.old whole:
# swap_recover puts it back. swap_in runs it first itself, and a caller runs
# it on every exit (trap), so the previous copy always returns.

# swap_recover DIR: put DIR's previous copy back if a swap was interrupted.
swap_recover() {
    if [ ! -d "$1" ] && [ -d "$1.old" ]; then
        mv "$1.old" "$1"
    fi
}

# swap_in NEW DIR: replace DIR (which may not exist) with NEW.
swap_in() {
    local new=$1 dir=$2
    # Before anything is deleted: an interrupted earlier swap's DIR.old may
    # be the only copy left.
    swap_recover "$dir"
    rm -rf "$dir.old"
    if [ -d "$dir" ]; then
        mv "$dir" "$dir.old"
    fi
    mv "$new" "$dir"
    rm -rf "$dir.old"
}

# One docs run at a time: they share fixed paths (DIR.new, DIR.old), and one
# run's cleanup would delete another's half-built copy or move its DIR.old
# back mid-swap.
#
# The lock is a symlink naming its owner's PID: creating a symlink is atomic,
# and it is born holding the owner, so two runs can never both take it. A lock
# whose PID no longer exists (a run killed with kill -9, or a closed terminal)
# is taken over: `kill -0` says so, with no ps and nothing locale- or
# time-zone-dependent, and the takeover happens under DOCS_LOCK.takeover, an
# atomic mkdir, so two runs cannot both take it over. A PID since reused by
# another process still looks alive: that run stops and says to delete the
# lock, which is the safe way to be wrong. scripts/generate_docs.go follows the
# same rules.
DOCS_LOCK="docs/.lock"

# docs_owner_gone PID: true only if no process has that PID. kill -0 fails
# for "no such process" and for "not permitted" (a live process of another
# user) alike, so its message tells them apart; any other, or a message in
# another language, counts as alive.
docs_owner_gone() {
    local out
    case "$1" in ''|*[!0-9]*) return 1 ;; esac
    out=$(LC_ALL=C kill -0 "$1" 2>&1) && return 1
    case "$out" in *"No such process"*|*"no such process"*) return 0 ;; esac
    return 1
}

# docs_lock: take the lock, or exit 1 if a live run holds it.
docs_lock() {
    mkdir -p docs
    if ln -s "$$" "$DOCS_LOCK" 2>/dev/null; then
        DOCS_LOCKED=1
        return 0
    fi

    local held
    held=$(readlink "$DOCS_LOCK" 2>/dev/null || echo "?")
    if docs_owner_gone "$held" && mkdir "$DOCS_LOCK.takeover" 2>/dev/null; then
        # One taker at a time; and only the lock that was found stale goes.
        if [ "$(readlink "$DOCS_LOCK" 2>/dev/null || true)" = "$held" ]; then
            rm -f "$DOCS_LOCK"
        fi
        if ln -s "$$" "$DOCS_LOCK" 2>/dev/null; then
            rmdir "$DOCS_LOCK.takeover"
            echo "note: took over docs/.lock from PID $held, which is no longer running" >&2
            DOCS_LOCKED=1
            return 0
        fi
        rmdir "$DOCS_LOCK.takeover"
        held=$(readlink "$DOCS_LOCK" 2>/dev/null || echo "?")
    fi

    echo "error: docs/.lock is held by PID $held: another docs run is in progress." >&2
    echo "       If none is (that PID now belongs to another process), delete it: rm docs/.lock" >&2
    exit 1
}

# docs_unlock: release it, if this run holds it.
docs_unlock() {
    if [ -n "${DOCS_LOCKED:-}" ] && [ "$(readlink "$DOCS_LOCK" 2>/dev/null || true)" = "$$" ]; then
        rm -f "$DOCS_LOCK"
    fi
}
