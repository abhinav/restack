// See https://github.com/abhinav/restack#readme.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/abhinav/restack"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	parser, opts := newParser()
	if _, err := parser.ParseArgs(args); err != nil {
		return err
	}

	if opts.Version {
		fmt.Fprintf(stdout, "restack v%s\n", restack.Version)
		return nil
	}

	getenv := os.Getenv
	git := &restack.SystemGit{Getenv: getenv}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	switch {
	case opts.Edit != nil:
		e := opts.Edit
		return (&restack.Edit{
			Editor:    e.Editor,
			Path:      e.Args.File,
			Restacker: &restack.GitRestacker{Git: git},
			Stdin:     stdin,
			Stdout:    stdout,
			Stderr:    stderr,
		}).Run(ctx)

	case opts.Setup != nil:
		return (&restack.Setup{
			PrintScript: opts.Setup.EditScript,
			Stdout:      stdout,
			Stderr:      stderr,
		}).Run(ctx)
	default:
		parser.WriteHelp(stderr)
	}

	return nil
}
