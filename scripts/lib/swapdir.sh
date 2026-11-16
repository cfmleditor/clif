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
# and it is born holding the owner, so two runs can never both take it. A
# lock is never taken over: deciding that its owner is gone needs ps (absent
# in minimal containers) and a start time it formats by locale and time zone,
# and two runs deciding at once could both win. A lock left by a run that was
# killed outright is removed by hand, as the error says; every normal exit,
# Ctrl-C included, releases it.
DOCS_LOCK="docs/.lock"

# docs_lock: take the lock, or exit 1 if a run holds it.
docs_lock() {
    mkdir -p docs
    if ! ln -s "$$" "$DOCS_LOCK" 2>/dev/null; then
        local held
        held=$(readlink "$DOCS_LOCK" 2>/dev/null || echo "?")
        echo "error: docs/.lock is held by PID $held: another docs run is in progress." >&2
        echo "       If none is (a run killed with kill -9 leaves it), delete it: rm docs/.lock" >&2
        exit 1
    fi
    DOCS_LOCKED=1
}

# docs_unlock: release it, if this run holds it.
docs_unlock() {
    if [ -n "${DOCS_LOCKED:-}" ] && [ "$(readlink "$DOCS_LOCK" 2>/dev/null || true)" = "$$" ]; then
        rm -f "$DOCS_LOCK"
    fi
}
