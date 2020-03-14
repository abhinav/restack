package restack

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

// Setup is the "restack setup" command.
type Setup struct {
	PrintScript bool

	Stdout io.Writer
	Stderr io.Writer
}

// Run runs the setup command.
func (s *Setup) Run(ctx context.Context) error {
	if s.PrintScript {
		fmt.Fprint(s.Stdout, _editScript)
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %v", err)
	}

	// TODO: Use config dir in the future
	restackDir := filepath.Join(home, ".restack")
	if err := os.MkdirAll(restackDir, 0755); err != nil {
		return fmt.Errorf("create directory %q: %v", restackDir, err)
	}

	editCmd := filepath.Join(restackDir, "edit.sh")
	if err := ioutil.WriteFile(editCmd, []byte(_editScript), 0755); err != nil {
		return fmt.Errorf("write file %q: %v", editCmd, err)
	}

	cmd := exec.CommandContext(ctx, "git", "config", "--global", "sequence.editor", editCmd)
	cmd.Stdout = s.Stdout
	cmd.Stderr = s.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("set sequence editor: %v", err)
	}

	fmt.Fprintln(s.Stderr, "restack has been set up successfully.")
	return nil
}

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
