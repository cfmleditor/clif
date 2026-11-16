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

# One docs run at a time: they share fixed paths (DIR.new, DIR.old,
# docs/data.building), and one run's cleanup would delete another's
# half-built copy or move its DIR.old back mid-swap. mkdir is atomic; the
# lock holds the owner's PID, so a lock left by a run that died (kill -9) is
# taken over rather than blocking every later run.
DOCS_LOCK="docs/.lock"

# docs_lock: take the lock, or exit 1 if a live run holds it.
docs_lock() {
    mkdir -p docs
    if ! mkdir "$DOCS_LOCK" 2>/dev/null; then
        local owner
        owner=$(cat "$DOCS_LOCK/pid" 2>/dev/null || true)
        if [ -n "$owner" ] && kill -0 "$owner" 2>/dev/null; then
            echo "error: another docs run (PID $owner) is in progress; wait for it" >&2
            exit 1
        fi
        rm -rf "$DOCS_LOCK"
        mkdir "$DOCS_LOCK"
    fi
    echo $$ > "$DOCS_LOCK/pid"
}

# docs_unlock: release it, if this run holds it.
docs_unlock() {
    if [ "$(cat "$DOCS_LOCK/pid" 2>/dev/null || true)" = "$$" ]; then
        rm -rf "$DOCS_LOCK"
    fi
}
