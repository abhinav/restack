#!/bin/sh -e

editor=$(git var GIT_EDITOR)
restack=$(command -v restack || echo "")

# $GOPATH/bin is not on $PATH but restack is installed.
if [ -z "$restack" ]; then
	if [ -n "$GOPATH" ] && [ -x "$GOPATH/bin/restack" ]; then
		restack="$GOPATH/bin/restack"
	fi
fi

if [ -n "$restack" ]; then
	"$restack" edit --editor="$editor" "$@"
else
	echo "WARNING:" >&2
	echo "  Could not find restack. Falling back to $editor." >&2
	echo "  To install restack, see https://github.com/abhinav/restack#installation" >&2
	echo "" >&2

	"$editor" "$@"
fi
