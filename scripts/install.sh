#!/bin/sh
# Install one clif release into a directory you own: no root, no package
# manager, the same binary on every machine that pins the same version.
#
#   scripts/install.sh [version] [dir]
#
# version   0.5.0, v0.5.0 or latest. Left out, the first of clif.version and
#           scripts/clif.version in the working directory names it, so a
#           project, its developers and its CI install one version from one
#           file; with neither, latest.
# dir       where clif goes: $CLIF_INSTALL_DIR, else ~/.local/bin.
#
# CLIF_DOWNLOAD_URL replaces https://github.com/cfmleditor/clif/releases, for
# a mirror laid out the same way (<url>/download/v<version>/<asset>, and
# <url>/latest/download/<asset>). It must be https.
#
# The archive is checked against the release's checksums.txt, which every
# release from 0.5.1 on publishes: for those, and for latest, a missing or
# unreadable checksums.txt stops the install. Older releases are checked when
# they have one. When gh is installed and signed in, an archive from a release
# that publishes build provenance (after 0.5.3) is also checked with
# `gh attestation verify`, which proves this repository's release workflow
# built it; without gh that check is skipped. Releases from before the rename
# to clif publish cfmleditor-lsp archives only; one of those is installed as
# clif.
set -eu

repo=cfmleditor/clif
first_checksums=0.5.1 # the first release with a checksums.txt
first_attested=0.5.4  # the first release built with provenance attestations

version=${1:-}
dir=${2:-${CLIF_INSTALL_DIR:-$HOME/.local/bin}}
releases=${CLIF_DOWNLOAD_URL:-https://github.com/cfmleditor/clif/releases}

if [ -z "$version" ]; then
	for f in clif.version scripts/clif.version; do
		if [ -f "$f" ]; then
			version=$(tr -d ' \t\r\n' <"$f")
			break
		fi
	done
fi
version=${version:-latest}

case "$releases" in
https://*) ;;
*)
	echo "clif install: CLIF_DOWNLOAD_URL must be an https:// URL, not $releases" >&2
	exit 1
	;;
esac

# The character test first: grep matches line by line, so a second line
# would otherwise pass.
case "$version" in
*[!0-9A-Za-z.-]*) bad_version=true ;;
*) bad_version=false ;;
esac
if [ "$version" != latest ] && { [ "$bad_version" = true ] || ! printf '%s\n' "$version" | grep -Eq '^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$'; }; then
	echo "clif install: '$version' is not a version (expected 0.5.0, v0.5.0 or latest)" >&2
	exit 1
fi

# at_least version min: whether version's X.Y.Z is min or later. A
# prerelease counts as its release, which is built by the same workflow.
at_least() {
	awk -v a="${1#v}" -v b="$2" 'BEGIN {
		sub(/[-+].*/, "", a); split(a, x, "."); split(b, y, ".")
		for (i = 1; i <= 3; i++) {
			if (x[i] + 0 > y[i] + 0) exit 0
			if (x[i] + 0 < y[i] + 0) exit 1
		}
		exit 0
	}'
}

# gh is used only to verify provenance, and only when it can: installed and
# signed in (or given GH_TOKEN). Asked here so latest can be resolved to the
# version it is, which decides whether there is an attestation to check.
attest=false
if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
	attest=true
	if [ "$version" = latest ] && tag=$(gh release view --repo "$repo" --json tagName -q .tagName 2>/dev/null) && [ -n "$tag" ]; then
		version=${tag#v}
	fi
fi
if [ "$attest" = true ] && { [ "$version" = latest ] || ! at_least "$version" "$first_attested"; }; then
	attest=false
fi

case "$(uname -s)" in
Darwin) os=darwin ;;
Linux) os=linux ;;
*)
	echo "clif install: $(uname -s) is not supported by this script; on Windows use scripts/install.ps1" >&2
	exit 1
	;;
esac

case "$(uname -m)" in
arm64 | aarch64) arch=arm64 ;;
x86_64 | amd64) arch=amd64 ;;
*)
	echo "clif install: no release for $(uname -m)" >&2
	exit 1
	;;
esac

if [ "$version" = latest ]; then
	base=$releases/latest/download
else
	base=$releases/download/v${version#v}
fi

fetch() { # url file
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL -o "$2" "$1"
	else
		wget -q -O "$2" "$1"
	fi
}

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	else
		shasum -a 256 "$1" | cut -d' ' -f1
	fi
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

name=clif
if ! fetch "$base/clif-$os-$arch.tar.gz" "$tmp/archive.tar.gz" 2>/dev/null; then
	name=cfmleditor-lsp
	fetch "$base/cfmleditor-lsp-$os-$arch.tar.gz" "$tmp/archive.tar.gz" || {
		echo "clif install: no $os/$arch archive for $version at $base" >&2
		exit 1
	}
fi

asset=$name-$os-$arch.tar.gz
if fetch "$base/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
	want=$(awk -v f="$asset" '$2 == f || $2 == "*" f { print $1 }' "$tmp/checksums.txt")
	got=$(sha256 "$tmp/archive.tar.gz")
	if [ -z "$want" ] || [ "$want" != "$got" ]; then
		echo "clif install: $asset does not match the release's checksums.txt (want ${want:-nothing}, got $got)" >&2
		exit 1
	fi
elif [ "$version" = latest ] || at_least "$version" "$first_checksums"; then
	# Every release since 0.5.1 has one, so its absence is a fault in the
	# download, not an old release: install nothing unverified.
	echo "clif install: could not fetch checksums.txt for $version from $base, so $asset cannot be verified" >&2
	exit 1
else
	echo "clif install: $version has no checksums.txt, so $asset was not verified" >&2
fi

if [ "$attest" = true ]; then
	if ! gh attestation verify "$tmp/archive.tar.gz" --repo "$repo" \
		--signer-workflow "$repo/.github/workflows/release.yml" >/dev/null; then
		echo "clif install: $asset has no valid build provenance from $repo" >&2
		exit 1
	fi
	echo "clif install: verified $asset's build provenance with gh attestation verify" >&2
fi

tar -xzf "$tmp/archive.tar.gz" -C "$tmp" "$name"
mkdir -p "$dir"
mv "$tmp/$name" "$dir/clif"
chmod 755 "$dir/clif"

echo "Installed $("$dir/clif" version) to $dir/clif"
case ":$PATH:" in
*":$dir:"*) ;;
*) echo "clif install: $dir is not on PATH; add it to use clif by name" >&2 ;;
esac
