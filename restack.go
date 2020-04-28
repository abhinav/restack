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

	rebasingBranch, err := r.Git.RebaseHeadName(ctx)
	if err != nil {
		return err
	}

	branches, err := r.Git.ListBranches(ctx)
	if err != nil {
		return err
	}

	knownBranches := make(map[string][]Branch)
	for _, b := range branches {
		knownBranches[b.Hash] = append(knownBranches[b.Hash], b)
	}

	gr := gitRestack{
		RemoteName:     req.RemoteName,
		RebaseHeadName: rebasingBranch,
		KnownBranches:  knownBranches,
		To:             req.To,
	}

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		gr.Process(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return gr.WritePushSection(true, false)
}

type gitRestack struct {
	RemoteName     string
	RebaseHeadName string
	KnownBranches  map[string][]Branch
	To             io.Writer

	updatedBranches  []string
	wrotePushSection bool
}

func (r *gitRestack) Process(line string) error {
	// If we see comments, write the push section first.
	if len(line) > 0 && line[0] == '#' {
		if err := r.WritePushSection(false, true); err != nil {
			return err
		}
	}

	// Most lines go in as-is.
	if _, err := fmt.Fprintln(r.To, line); err != nil {
		return err
	}

	if !strings.HasPrefix(line, "pick ") {
		return nil
	}

	// pick [hash] [msg]
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		return nil
	}

	branches, ok := r.KnownBranches[parts[1]]
	if !ok {
		return nil
	}

	addedBranchUpdates := false
	for _, b := range branches {
		if b.Name == r.RebaseHeadName {
			continue
		}

		if _, err := fmt.Fprintf(r.To, "exec git branch -f %v\n", b.Name); err != nil {
			return err
		}
		r.updatedBranches = append(r.updatedBranches, b.Name)
		addedBranchUpdates = true
	}

	// Add an empty line between branch sections.
	if addedBranchUpdates {
		fmt.Fprintln(r.To)
	}

	return nil
}

const _pushSectionPrefix = "# Uncomment this section to push the changes.\n"

func (r *gitRestack) WritePushSection(padBefore, padAfter bool) error {
	if r.wrotePushSection {
		return nil
	}
	r.wrotePushSection = true

	if len(r.updatedBranches) == 0 || len(r.RemoteName) == 0 {
		return nil
	}

	if padBefore {
		io.WriteString(r.To, "\n")
	}
	if _, err := io.WriteString(r.To, _pushSectionPrefix); err != nil {
		return err
	}

	for _, b := range r.updatedBranches {
		if _, err := fmt.Fprintf(r.To, "# exec git push -f %s %s\n", r.RemoteName, b); err != nil {
			return err
		}
	}

	if padAfter {
		io.WriteString(r.To, "\n")
	}

	return nil
}
