#!/usr/bin/env bash
# Check tracked and non-ignored Go source, including new implementation files.
# Local toolchains, module caches and scratch probes under target/ are not source.
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"

files=()
while IFS= read -r -d '' file; do
	# An unstaged deletion is absent from the working tree, not unformatted.
	if [ -f "$file" ]; then
		files+=("$file")
	fi
done < <(git ls-files --cached --others --exclude-standard -z -- '*.go')

if [ "${#files[@]}" -eq 0 ]; then
	exit 0
fi

unformatted=$(gofmt -l "${files[@]}")
if [ -n "$unformatted" ]; then
	printf 'gofmt needed on:\n%s\n' "$unformatted"
	exit 1
fi
