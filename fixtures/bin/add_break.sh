#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# USAGE:
#   add_break.sh PATH
# Adds a line "break" to the top of the file at PATH.

out=$(mktemp)
echo break > "$out"
while read -r line; do
	echo "$line" >> "$out"
done < "$1"
mv "$out" "$1"
