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
    mkdir -p target/docs-gen
    go build -o "$docslock" scripts/docslock.go || exit 1
    exec "$docslock" bash "scripts/${0##*/}" "$@"
fi
unset docs_root docslock
