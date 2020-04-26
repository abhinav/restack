package main

import flags "github.com/jessevdk/go-flags"

type options struct {
	Version bool      `long:"version"`
	Edit    *editCmd  `command:"edit"`
	Setup   *setupCmd `command:"setup"`
}

type setupCmd struct {
	EditScript bool `long:"print-edit-script"`
}

type editCmd struct {
	Editor string `short:"e" long:"editor" env:"EDITOR" default:"vim"`
	Args   struct {
		File string `positional-arg-name:"FILE"`
	} `positional-args:"yes" required:"yes"`
}

func newParser() (*flags.Parser, *options) {
	var opts options
	parser := flags.NewParser(&opts, flags.HelpFlag)

	// If --version was specified, a command probably was not. So we need to
	// make subcommands optional and validate manually.
	parser.SubcommandsOptional = true

	parser.FindOptionByLongName("version").Description =
		"Prints the current version of restack."

	setup := parser.Find("setup")
	setup.ShortDescription =
		"Sets up restack"
	setup.LongDescription =
		"Alters your git configuration to use resack during interactive rebases. " +
			"If you would rather do this manually, " +
			"see https://github.com/abhinav/restack#manual-setup."

	setup.FindOptionByLongName("print-edit-script").Description =
		"Print the shell script that will be used as the editor for " +
			"interactive rebases."

	edit := parser.Find("edit")
	edit.ShortDescription =
		"Edits the instruction list for an interactive rebase."
	edit.LongDescription =
		"Edits a git-rebase-todo with branch restacking. " +
			"This command is meant to be called by `git` directly. " +
			"See https://github.com/abhinav/restack#setup."

	edit.FindOptionByLongName("editor").Description =
		"Editor to use to edit rebase instructions."
	edit.Args()[0].Description =
		"File to edit. This is the git rebase instruction list."

	return parser, &opts
}
