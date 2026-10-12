# shellcheck shell=bash
# sha256 prints the SHA-256 of the file named by $1: sha256sum where there is
# one (Linux), shasum -a 256 otherwise (macOS). Sourced by the packaging
# scripts, so they compute it one way; scripts/fetch-cflint.sh keeps its own
# copy, since it is run on its own.
sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	else
		shasum -a 256 "$1" | cut -d' ' -f1
	fi
}
