package restack

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// Restacker reads the todo list of an interactive rebase and writes a new
// version of it with the provided configuration.
type Restacker struct {
	// Name of a git remote. If non-empty, an opt-in section that pushes
	// restacked branches to this remote is also generated.
	RemoteName string

	// FS controls how Restacker accesses the filesystem. Defaults to
	// DefaultFilesystem.
	FS FS

	// Controls access to Git commands.
	Git Git
}

const _pushSectionPrefix = "\n# Uncomment this section to push the changes.\n"

// Run reads rebase instructions from src and writes them to dst based on the
// Restacker configuration.
func (r Restacker) Run(ctx context.Context, dst io.Writer, src io.Reader) error {
	if r.FS == nil {
		r.FS = DefaultFilesystem
	}

	if r.Git == nil {
		r.Git = DefaultGit
	}

	knownBranches, err := r.Git.ListHeads(ctx)
	if err != nil {
		return err
	}

	var branches []string

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		line := scanner.Text()

		// If we found an empty line, the instructions section is over. We
		// will add our push instructions here.
		if len(line) == 0 && len(branches) > 0 && len(r.RemoteName) > 0 {
			if _, err := io.WriteString(dst, _pushSectionPrefix); err != nil {
				return err
			}

			for _, b := range branches {
				if _, err := fmt.Fprintf(dst, "# exec git push -f %s %s\n", r.RemoteName, b); err != nil {
					return err
				}
			}
		}

		// Most lines go in as-is.
		if _, err := fmt.Fprintln(dst, line); err != nil {
			return err
		}

		if !strings.HasPrefix(line, "pick ") {
			continue
		}

		// pick [hash] [msg]
		parts := strings.SplitN(line, " ", 3)
		if len(parts) != 3 {
			continue
		}

		refs, ok := knownBranches[parts[1]]
		if !ok {
			continue
		}

		for _, ref := range refs {
			ref = strings.TrimPrefix(ref, "refs/heads/")
			if _, err := fmt.Fprintf(dst, "exec git branch -f %v\n", ref); err != nil {
				return err
			}
			branches = append(branches, ref)
		}
	}

	return scanner.Err()
}
