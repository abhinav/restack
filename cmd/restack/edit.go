package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/abhinav/restack"
	"go.uber.org/multierr"
)

type editCmd struct {
	Editor string `short:"e" long:"editor" env:"EDITOR" default:"vim" description:"Editor used to edit the file."`
	Args   struct {
		File string `positional-arg-name:"FILE" description:"Path to file being edited."`
	} `positional-args:"yes" required:"yes"`

	fs  restack.FS
	git restack.Git
}

func newEditCmd() *editCmd {
	fs := restack.DefaultFilesystem
	return &editCmd{
		fs:  fs,
		git: restack.NewSystemGit(fs),
	}
}

func (editCmd) Name() string      { return "edit" }
func (editCmd) ShortDesc() string { return "Edits the instruction list for an interactive rebase." }
func (editCmd) LongDesc() string {
	return "Edits a git-rebase-todo with branch restacking. " +
		"This command is meant to be called by `git` directly. " +
		"See https://github.com/abhinav/restack#setup."

}

func (e *editCmd) Execute([]string) error {
	inFilePath := e.Args.File
	inFile, err := e.fs.ReadFile(inFilePath)
	if err != nil {
		return fmt.Errorf("failed to open %q for reading: %v", inFilePath, err)
	}

	tempDir, err := e.fs.TempDir("restack.")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer e.fs.RemoveAll(tempDir)

	// Need to call the file git-rebase-todo to make sure file-type detection
	// in different editors works correctly.
	outFilePath := filepath.Join(tempDir, "git-rebase-todo")
	outFile, err := e.fs.WriteFile(outFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %v", outFilePath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// TODO: Guess remote name
	r := restack.Restacker{RemoteName: "origin", Git: e.git}
	if err := r.Run(ctx, outFile, inFile); err != nil {
		err = multierr.Append(err, outFile.Close())
		err = multierr.Append(err, inFile.Close())
		return err
	}

	if err := multierr.Append(outFile.Close(), inFile.Close()); err != nil {
		return fmt.Errorf("failed to close files: %v", err)
	}

	cmd := exec.Command(e.Editor, outFilePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to edit %q: %v", outFilePath, err)
	}

	if err := e.fs.Rename(outFilePath, inFilePath); err != nil {
		return fmt.Errorf("failed to overwrite %q: %v", inFilePath, err)
	}

	return nil
}
