package main

import (
	"log"

	flags "github.com/jessevdk/go-flags"
)

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatalf("%+v", err)
	}
}

func run() error {
	// No global options at this time.
	var opts struct{}

	parser := flags.NewParser(&opts, flags.HelpFlag)

	if _, err := parser.AddCommand(
		"setup", "Sets up restack",
		"Alters your git configuration to use restack during rebases.",
		newSetupCmd(),
	); err != nil {
		return err
	}

	if _, err := parser.AddCommand(
		"edit", "Edits a git-rebase-todo",
		"Edits a git-rebase-todo with branch restacking. "+
			"This is typically called directly by git after a `restack setup`.",
		newEditCmd(),
	); err != nil {
		return err
	}

	_, err := parser.Parse()
	return err
}
