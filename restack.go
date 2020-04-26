package restack

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// Request is a request to process a rebase instruction list.
type Request struct {
	// Name of the git remote. If set, an opt-in section that pushes restacked
	// branches to this remote will also be generated.
	//
	// This field is optional.
	RemoteName string

	// Input and output instruction lists.
	From io.Reader
	To   io.Writer
}

// Restacker processes the rebase instruction list.
type Restacker interface {
	Restack(context.Context, *Request) error
}

// GitRestacker restacks instruction lists using the provided Git instance.
type GitRestacker struct {
	// Controls access to Git commands.
	//
	// This field is required.
	Git Git
}

var _ Restacker = (*GitRestacker)(nil)

// Restack process the provided instruction list.
func (r *GitRestacker) Restack(ctx context.Context, req *Request) error {
	src := req.From
	dst := req.To

	rebasingBranch, err := r.Git.RebaseHeadName(ctx)
	if err != nil {
		return err
	}

	knownBranches, err := r.Git.ListHeads(ctx)
	if err != nil {
		return err
	}

	var (
		branches         []string
		wrotePushSection bool
	)

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		line := scanner.Text()

		// If we found an empty line, the instructions section is over. We
		// will add our push instructions here.
		if len(line) == 0 {
			if err := r.writePushSection(req.RemoteName, dst, branches); err != nil {
				return err
			}
			wrotePushSection = true
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
		if len(parts) < 2 {
			continue
		}

		refs, ok := knownBranches[parts[1]]
		if !ok {
			continue
		}

		addedBranchUpdates := false
		for _, ref := range refs {
			ref = strings.TrimPrefix(ref, "refs/heads/")
			if ref == rebasingBranch {
				continue
			}

			if _, err := fmt.Fprintf(dst, "exec git branch -f %v\n", ref); err != nil {
				return err
			}
			branches = append(branches, ref)
			addedBranchUpdates = true
		}

		// Add an empty line between branch sections.
		if addedBranchUpdates {
			fmt.Fprintln(dst)
		}
	}

	if !wrotePushSection {
		if err := r.writePushSection(req.RemoteName, dst, branches); err != nil {
			return err
		}
	}

	return scanner.Err()
}

const _pushSectionPrefix = "\n# Uncomment this section to push the changes.\n"

func (r *GitRestacker) writePushSection(remoteName string, dst io.Writer, branches []string) error {
	if len(branches) == 0 || len(remoteName) == 0 {
		return nil
	}

	if _, err := io.WriteString(dst, _pushSectionPrefix); err != nil {
		return err
	}

	for _, b := range branches {
		if _, err := fmt.Fprintf(dst, "# exec git push -f %s %s\n", remoteName, b); err != nil {
			return err
		}
	}

	return nil
}
