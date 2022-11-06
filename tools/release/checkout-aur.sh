#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

err() {
	echo >&2 "$@"
}

if [[ -z "$AUR_KEY" || -z "$AUR_REPO" || -z "$AUR_DIR" ]]; then
	err "Please set AUR_{KEY,REPO,DIR}"
	exit 1
fi

KEY_PATH=$(mktemp --tmpdir id_XXXXX)
trap 'rm $KEY_PATH' EXIT
echo "$AUR_KEY" > "$KEY_PATH"

export GIT_SSH_COMMAND="ssh -i $KEY_PATH -o StrictHostKeyChecking=accept-new -F /dev/null"
git clone "$AUR_REPO" "$AUR_DIR"
