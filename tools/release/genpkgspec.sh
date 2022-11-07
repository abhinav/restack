#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# genpkgspec.sh VERSION TARBALLS
#
# Run this with paths to tarballs named restack-$os-$arch.tar.gz.
# This will generate target/formula/ and target/aur-bin.

DIR=$(dirname "$0")
BOTTLE_TMPL="$DIR/bottle.tmpl"
PKGBUILD_TMPL="$DIR/PKGBUILD.tmpl"

err() {
	echo >&2 "$@"
}

if [[ $# -lt 2 ]]; then
	err "USAGE: $0 VERSION FILES ..."
	exit 1
fi

export VERSION="$1"; shift
while [[ $# -gt 0 ]]; do
	FILE="$1"; shift
	SHA=$(sha256sum "$FILE" | awk '{print $1}')
	# This is absolutely horrendous,
	# but it'll work with all versions of Bash.
	eval "$(perl -se '
		$file =~ /(darwin|linux)[-_](amd64|arm64|armv7)/
			or die "Could not match: $file";
		print "export SHASUM_$1_$2=$shasum\n"
	' -- -file="$FILE" -shasum="$SHA")"
done

VARS=(
	VERSION
	SHASUM_darwin_amd64
	SHASUM_darwin_arm64
	SHASUM_linux_amd64
	SHASUM_linux_arm64
	SHASUM_linux_armv7
)
shellformat=""
for VAR in "${VARS[@]}"; do
	if [[ -z "${!VAR:-}" ]]; then
		err "Unset variable $VAR"
		exit 1
	fi
	shellformat="$shellformat \$$VAR"
done

mkdir -p target/formula
envsubst "$shellformat" < "$BOTTLE_TMPL" > target/formula/restack.rb

mkdir -p target/aur-bin
envsubst "$shellformat" < "$PKGBUILD_TMPL" > target/aur-bin/PKGBUILD
