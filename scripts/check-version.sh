#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

version="${1:-}"
if [[ -z "$version" ]]; then
	echo >&2 "usage: $0 VERSION"
	exit 1
fi

got=$(grep '^const Version = ".*"'  version.go  | cut -d'"' -f2)

if [[ "$got" == "$version" ]]; then
	exit 0
fi

echo >&2 "version.go mismatch:"
echo >&2 "  want: $version"
echo >&2 "   got: $got"
exit 1
