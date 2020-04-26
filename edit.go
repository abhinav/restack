package restack

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/multierr"
)

// Edit implements the "restack edit" command.
type Edit struct {
	// Editor to use for the file.
	Editor string

	// Path to file containing initial rebase instruction list.
	Path string

	Git    Git
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Run runs the "restack edit" command.
func (e *Edit) Run(ctx context.Context) error {
	inFile, err := os.Open(e.Path)
	if err != nil {
		return fmt.Errorf("open %q: %v", e.Path, err)
	}

	tempDir, err := ioutil.TempDir("", "restack.")
	if err != nil {
		return fmt.Errorf("create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Need to call the file git-rebase-todo to make sure file-type detection
	// in different editors works correctly.
	outFilePath := filepath.Join(tempDir, "git-rebase-todo")
	outFile, err := os.Create(outFilePath)
	if err != nil {
		return fmt.Errorf("create file %q: %v", outFilePath, err)
	}

	// TODO: Guess remote name
	r := Restacker{RemoteName: "origin", Git: e.Git}
	if err := r.Run(ctx, outFile, inFile); err != nil {
		err = multierr.Append(err, outFile.Close())
		err = multierr.Append(err, inFile.Close())
		return err
	}

	if err := multierr.Append(outFile.Close(), inFile.Close()); err != nil {
		return fmt.Errorf("close files: %v", err)
	}

	// Because GIT_EDITOR is meant to be interpreted by the shell, we need to
	// rely on sh to handle that. We run,
	//   sh -c "$GIT_EDITOR $1" "restack" $FILE
	// This has the effect of invoking GIT_EDITOR with the argument $FILE.
	cmd := exec.CommandContext(ctx, "sh", "-c", e.Editor+` "$1"`, "restack", outFilePath)
	cmd.Stdin = e.Stdin
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("edit %q: %v", outFilePath, err)
	}

	if err := os.Rename(outFilePath, e.Path); err != nil {
		return fmt.Errorf("overwrite %q: %v", e.Path, err)
	}

	return nil
}
