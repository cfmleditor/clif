# shellcheck shell=bash
# retry CMD...: run a network command up to three times, waiting 5s and then
# 10s between tries, so a passing blip (a CI runner's, say) does not fail the
# run. Shared by fetch-docs-cfdocs.sh and fetch-docs-lucee.sh, so both
# fetches meet a flaky network alike.
retry() {
    local attempt
    for attempt in 1 2 3; do
        if "$@"; then
            return 0
        fi
        if [ "$attempt" -lt 3 ]; then
            echo "  (failed; retrying in $((attempt * 5))s)" >&2
            sleep $((attempt * 5))
        fi
    done
    return 1
}
