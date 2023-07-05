#!/usr/bin/env bash

# This fixture contains a simple stack of commits.
#
# o [main] initial commit
# |
# o [foo] foo
# |
# o [bar] bar
# |
# o [baz, wip] baz
#
# It also uses ';' as the comment character.

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

git commit --allow-empty -m "empty commit"
git config core.commentChar ';'

git checkout -b foo
add_and_commit foo

git checkout -b bar
add_and_commit bar

git checkout -b baz
add_and_commit baz

git checkout -b wip
