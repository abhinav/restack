package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/abhinav/restack"
	"go.uber.org/multierr"
)

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatalf("%+v", err)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("USAGE: %v COMMAND", os.Args[0])
	}

	switch os.Args[1] {
	case "install":
		// TODO: use current binary location instead of assuming it's on $PATH
		// TODO: store current core.editor value in command as an argument

		git := restack.DefaultGit
		if err := git.SetGlobalConfig("sequence.editor", "restack edit"); err != nil {
			return fmt.Errorf("failed to set sequence editor: %v", err)
		}

		log.Print("Successfully installed restack.")
		return nil
	case "edit":
		return edit(os.Args[2:])
	default:
		return fmt.Errorf("unknown command %q", os.Args[1])
	}
}

func edit(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("USAGE: %v edit [OPTIONS] FILE", os.Args[0])
	}

	fs := restack.DefaultFilesystem

	inFilePath := args[0]
	inFile, err := fs.ReadFile(inFilePath)
	if err != nil {
		return fmt.Errorf("failed to open %q for reading: %v", args[0], err)
	}

	tempDir, err := fs.TempDir("restack.")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer fs.RemoveAll(tempDir)

	// Need to call the file git-rebase-todo to make sure file-type detection
	// in different editors works correctly.
	outFilePath := filepath.Join(tempDir, "git-rebase-todo")
	outFile, err := fs.WriteFille(outFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %v", outFilePath, err)
	}

	// TODO: Guess remote name
	r := restack.Restacker{RemoteName: "origin", FS: fs}
	if err := r.Run(outFile, inFile); err != nil {
		outFile.Close()
		inFile.Close()
		return err
	}

	if err := multierr.Append(outFile.Close(), inFile.Close()); err != nil {
		return fmt.Errorf("failed to close files: %v", err)
	}

	cmd := exec.Command("nvim", outFilePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to edit %q: %v", outFilePath, err)
	}

	if err := fs.Rename(outFilePath, inFilePath); err != nil {
		return fmt.Errorf("failed to overwrite %q: %v", inFilePath, err)
	}

	return nil
}
