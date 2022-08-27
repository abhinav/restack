// See https://github.com/abhinav/restack#readme.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/abhinav/restack"
)

func main() {
	opts := options{
		Stdout: os.Stdout,
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
		Getenv: os.Getenv,
	}
	if err := run(&opts, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

type options struct {
	Stdin          io.Reader
	Stdout, Stderr io.Writer

	Getenv func(string) string

	Version bool
}

const _restackUsage = `usage: %v [options] command

The following commands are available:

  setup  Setups up restack
  edit   Edits the instruction list for an interactive rebase

The following options are available:
`

func run(opts *options, args []string) error {
	flag := flag.NewFlagSet("restack", flag.ContinueOnError)
	flag.SetOutput(opts.Stderr)
	flag.Usage = usage(flag, _restackUsage)
	flag.BoolVar(&opts.Version, "version", false,
		"Prints the current version of restack.")

	if err := flag.Parse(args); err != nil {
		return err
	}

	if opts.Version {
		fmt.Fprintf(opts.Stdout, "restack v%s\n", restack.Version)
		return nil
	}

	args = flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return errors.New("no command specified")
	}

	cmd, args := args[0], args[1:]
	var (
		c   command
		err error
	)
	switch cmd {
	case "setup":
		c, err = newSetup(opts, args)
	case "edit":
		c, err = newEdit(opts, args)
	default:
		flag.Usage()
		return fmt.Errorf("unrecognized command %q", cmd)
	}

	if err != nil {
		return err
	}

	return c.Run(context.Background())
}

type command interface {
	Run(context.Context) error
}

const _setupUsage = `usage: %v [options]

Configures Git to use restack during an interactive rebase. If you prefer to
configure Git manually, see https://github.com/abhinav/restack#manual-setup.

The following options are available:
`

func newSetup(opts *options, args []string) (*restack.Setup, error) {
	setup := restack.Setup{
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	}

	flag := flag.NewFlagSet("restack setup", flag.ContinueOnError)
	flag.SetOutput(opts.Stderr)
	flag.Usage = usage(flag, _setupUsage)
	flag.BoolVar(&setup.PrintScript, "print-edit-script", false,
		"Print the shell script that will be used as the editor for "+
			"interactive rebases.")

	if err := flag.Parse(args); err != nil {
		return nil, err
	}

	if flag.NArg() > 0 {
		flag.Usage()
		return nil, fmt.Errorf("too many arguments: %q", flag.Args())
	}

	return &setup, nil
}

const _editUsage = `usage: %v [options] file

Edits the provided interactive rebase instruction list, augmenting it as
needed. Configure Git to call this command as the rebase instruction list
editor. See https://github.com/abhinav/restack#setup for more.

The following arguments are expected:

  file  File to edit. This is the git rebase instruction list.

The following options are available:
`

func newEdit(opts *options, args []string) (*restack.Edit, error) {
	edit := restack.Edit{
		Stdin:  opts.Stdin,
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
		Restacker: &restack.GitRestacker{
			Git: &restack.SystemGit{
				Getenv: opts.Getenv,
			},
		},
	}

	flag := flag.NewFlagSet("restack edit", flag.ContinueOnError)
	flag.SetOutput(opts.Stderr)
	flag.Usage = usage(flag, _editUsage)
	flag.StringVar(&edit.Editor, "e", "",
		"Editor to use to edit rebase instructions.")
	flag.StringVar(&edit.Editor, "editor", "",
		"Same as -e.")

	if err := flag.Parse(args); err != nil {
		return nil, err
	}

	if len(edit.Editor) == 0 {
		edit.Editor = opts.Getenv("EDITOR")
	}

	if len(edit.Editor) == 0 {
		edit.Editor = "vim"
	}

	args = flag.Args()
	switch len(args) {
	case 0:
		flag.Usage()
		return nil, errors.New("no file specified: please provide a file name")
	case 1:
		edit.Path = args[0]
	default:
		flag.Usage()
		return nil, fmt.Errorf("too many arguments: %q, expected 1", args)
	}

	return &edit, nil
}

func usage(flag *flag.FlagSet, usage string) func() {
	return func() {
		w := flag.Output()
		fmt.Fprintf(w, usage, flag.Name())
		fmt.Fprintln(w)
		flag.PrintDefaults()
		fmt.Fprintln(w)
	}
}
