#!/usr/bin/env bash
set -euo pipefail
# Build before taking the docs lock. Every docs mutation is in the driver,
# which exec keeps as the process the caller signals and waits for.
cd "$(dirname "$0")/.."
# shellcheck source=lib/hashbuild.sh
. scripts/lib/hashbuild.sh
driver=$(docs_build ./cmd/docpipe docpipe)
exec "$driver" "$@"
