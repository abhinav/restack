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

	Restacker Restacker

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Run runs the "restack edit" command.
func (e *Edit) Run(ctx context.Context) (err error) {
	tempDir, err := ioutil.TempDir("", "restack.")
	if err != nil {
		return fmt.Errorf("create temporary directory: %v", err)
	}
	defer func() {
		err = multierr.Append(err, os.RemoveAll(tempDir))
	}()

	outFilePath, err := e.restack(ctx, tempDir)
	if err != nil {
		return err
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

	if err := renameFile(outFilePath, e.Path); err != nil {
		return fmt.Errorf("overwrite %q: %v", e.Path, err)
	}

	return nil
}

func (e *Edit) restack(ctx context.Context, tempDir string) (outfile string, err error) {
	inFile, err := os.Open(e.Path)
	if err != nil {
		return "", fmt.Errorf("open %q: %v", e.Path, err)
	}
	defer func() {
		err = multierr.Append(err, inFile.Close())
	}()

	// Need to call the file git-rebase-todo to make sure file-type detection
	// in different editors works correctly.
	outFilePath := filepath.Join(tempDir, "git-rebase-todo")
	outFile, err := os.Create(outFilePath)
	if err != nil {
		return "", fmt.Errorf("create file %q: %v", outFilePath, err)
	}
	defer func() {
		err = multierr.Append(err, outFile.Close())
	}()

	// TODO: Guess remote name
	req := Request{RemoteName: "origin", From: inFile, To: outFile}
	err = e.Restacker.Restack(ctx, &req)
	return outFilePath, err
}
