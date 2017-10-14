package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/abhinav/restack"
)

type setupCmd struct {
	// TODO: dry-run mode?
	git restack.Git
}

func newSetupCmd() *setupCmd {
	return &setupCmd{git: restack.NewSystemGit(restack.DefaultFilesystem)}
}

func (i *setupCmd) Execute([]string) error {
	restackPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find path to restack executable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	editor, err := i.git.Var(ctx, "GIT_EDITOR")
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("%v edit -e %v", restackPath, editor)
	if err := i.git.SetGlobalConfig(ctx, "sequence.editor", cmd); err != nil {
		return fmt.Errorf("failed to set sequence editor: %v", err)
	}

	log.Print("restack has been set up successfully.")
	return nil
}
