#!/usr/bin/env bash
set -euo pipefail
# Compatibility entry point; the Go driver owns the lock and all docs writes.
exec bash "$(dirname "$0")/docs-driver.sh" pin "$@"
