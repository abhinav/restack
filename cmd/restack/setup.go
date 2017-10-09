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
	return &setupCmd{git: restack.DefaultGit}
}

func (i *setupCmd) Execute([]string) error {
	// TODO: store current core.editor value in command as an argument

	restackPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find path to restack executable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := i.git.SetGlobalConfig(ctx, "sequence.editor", restackPath+" edit"); err != nil {
		return fmt.Errorf("failed to set sequence editor: %v", err)
	}

	log.Print("restack has been set up successfully.")
	return nil
}
