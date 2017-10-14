package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/abhinav/restack"
)

const _editScript = `#!/bin/sh -e

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
`

type setupCmd struct {
	// TODO: dry-run mode?
	git restack.Git
	fs  restack.FS
}

func newSetupCmd() *setupCmd {
	fs := restack.DefaultFilesystem
	return &setupCmd{
		git: restack.NewSystemGit(fs),
		fs:  fs,
	}
}

func (i *setupCmd) Execute([]string) error {
	restackDir := filepath.Join(os.Getenv("HOME"), ".restack")
	if err := i.fs.MkdirAll(restackDir); err != nil {
		return fmt.Errorf("failed to create directory %q: %v", restackDir, err)
	}

	editCmd := filepath.Join(restackDir, "edit.sh")
	w, err := i.fs.WriteExecutableFile(editCmd)
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %v", editCmd, err)
	}
	defer w.Close()

	if _, err := io.WriteString(w, _editScript); err != nil {
		return fmt.Errorf("failed to write to file %q: %v", editCmd, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := i.git.SetGlobalConfig(ctx, "sequence.editor", editCmd); err != nil {
		return fmt.Errorf("failed to set sequence editor: %v", err)
	}

	log.Print("restack has been set up successfully.")
	return nil
}
