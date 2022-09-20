#!/usr/bin/env bash

die() {
	echo >&2 "$@"
	exit 1
}

add_and_commit() {
	[[ $# -eq 1 ]] || die "add_and_commit expects one argument"

	echo "$1" > "$1"
	git add "$1"
	git commit -m "add $1"
}

add_and_commit foo
git checkout -b feature1
add_and_commit bar
git checkout -b feature2
add_and_commit baz
add_and_commit qux

GIT_SEQUENCE_EDITOR=add_break.sh \
	git rebase -i feature1
