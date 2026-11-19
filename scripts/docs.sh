#!/usr/bin/env bash
set -euo pipefail
command=docs
if [ "${1:-}" = --cfdocs ]; then
    command=docs-cfdocs
    shift
fi
exec bash "$(dirname "$0")/docs-driver.sh" "$command" "$@"
