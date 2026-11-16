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
# The lock is a symlink whose target names its owner: creating a symlink is
# atomic, and it is born holding the owner, so there is no moment when a lock
# exists without one (a directory, then a PID file, had that gap, and a run
# arriving in it took the lock as stale). The owner is "<pid>@<start time>":
# a lock left by a run that died (kill -9) names a process that is gone, or a
# PID since reused by another process, whose start time then differs, so it is
# taken over rather than blocking every later run.
DOCS_LOCK="docs/.lock"

# docs_owner PID: the owner string for a process, or nothing if it is gone.
docs_owner() {
    local start
    start=$(ps -p "$1" -o lstart= 2>/dev/null | tr -s ' ' '_' | sed 's/^_//; s/_$//')
    [ -n "$start" ] && echo "$1@$start"
}

# docs_lock: take the lock, or exit 1 if a live run holds it.
docs_lock() {
    mkdir -p docs
    DOCS_LOCK_OWNER=$(docs_owner $$)
    if ln -s "$DOCS_LOCK_OWNER" "$DOCS_LOCK" 2>/dev/null; then
        return 0
    fi

    local held pid
    held=$(readlink "$DOCS_LOCK" 2>/dev/null || true)
    pid=${held%%@*}
    if [ -n "$pid" ] && [ "$(docs_owner "$pid")" = "$held" ]; then
        echo "error: another docs run (PID $pid) is in progress; wait for it" >&2
        exit 1
    fi

    # Stale: its owner is gone. Two runs may both get here; only one ln wins.
    rm -f "$DOCS_LOCK"
    if ! ln -s "$DOCS_LOCK_OWNER" "$DOCS_LOCK" 2>/dev/null; then
        echo "error: another docs run took the lock first; wait for it" >&2
        exit 1
    fi
}

# docs_unlock: release it, if this run holds it.
docs_unlock() {
    if [ -n "${DOCS_LOCK_OWNER:-}" ] && [ "$(readlink "$DOCS_LOCK" 2>/dev/null || true)" = "$DOCS_LOCK_OWNER" ]; then
        rm -f "$DOCS_LOCK"
    fi
}
