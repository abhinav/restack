package main

import (
	"fmt"
	"log"
	"os"

	"github.com/abhinav/restack"
	flags "github.com/jessevdk/go-flags"
)

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatalf("%+v", err)
	}
}

func run() error {
	var opts struct {
		Version bool `long:"version" description:"Prints the current version of restack."`
	}

	parser := flags.NewParser(&opts, flags.HelpFlag)

	// If --version was specified, a command probably was not. So we need to
	// make subcommands optional and validate manually.
	parser.SubcommandsOptional = true
	parser.CommandHandler = func(c flags.Commander, args []string) error {
		switch {
		case opts.Version:
			fmt.Printf("restack v%s\n", restack.Version)
		case c == nil:
			parser.WriteHelp(os.Stderr)
		default:
			return c.Execute(args)
		}
		return nil
	}

	if _, err := parser.AddCommand(
		"setup", "Sets up restack",
		"Alters your git configuration to use restack during rebases.",
		newSetupCmd(),
	); err != nil {
		return err
	}

	if _, err := parser.AddCommand(
		"edit", "Edits the instruction list for an interactive rebase.",
		"Edits a git-rebase-todo with branch restacking. "+
			"This is typically called directly by git after a `restack setup`.",
		newEditCmd(),
	); err != nil {
		return err
	}

	_, err := parser.Parse()
	return err
}
