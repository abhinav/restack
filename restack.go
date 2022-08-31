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

	gr.Finish()
	return nil
}

type gitRestack struct {
	RemoteName     string
	RebaseHeadName string
	KnownBranches  map[string][]Branch
	To             io.Writer

	lastLineBranches []Branch
	updatedBranches  []string
	wrotePushSection bool
}

func (r *gitRestack) Process(line string) error {
	if len(line) == 0 {
		// Empty lines delineate sections.
		// Push out any pending "git branch -x" statements
		// before printing the next section.
		var err error
		if !r.updatePreviousBranches() {
			// updatePreviousBranches adds a newline
			// after the git branch statements
			// if there were any new entries.
			// So we don't need to add another.
			_, err = fmt.Fprintln(r.To, line)
		}
		return err
	}

	// If we see comments, write the push section first.
	if len(line) > 0 && line[0] == '#' {
		r.updatePreviousBranches()
		if err := r.writePushSection(false, true); err != nil {
			return err
		}
	}

	// (p[ick]|f[ixup]|s[quash]) hash ...
	parts := strings.SplitN(line, " ", 3)
	if len(parts) > 1 {
		switch parts[0] {
		case "f", "fixup", "s", "squash":
			// Do nothing.
		default:
			r.updatePreviousBranches()
		}
	}

	// Most lines go in as-is.
	if _, err := fmt.Fprintln(r.To, line); err != nil {
		return err
	}

	if len(parts) < 2 {
		return nil
	}

	switch parts[0] {
	case "p", "pick", "r", "reword", "e", "edit":
		r.lastLineBranches = r.KnownBranches[parts[1]]
	}

	return nil
}

func (r *gitRestack) Finish() {
	r.updatePreviousBranches()
	r.writePushSection(true, false)
}

// Adds "git branch -f" directives for recorded branches, if any.
// Reports whether entries were added.
func (r *gitRestack) updatePreviousBranches() (updated bool) {
	branches := r.lastLineBranches
	r.lastLineBranches = nil

	for _, b := range branches {
		if b.Name == r.RebaseHeadName {
			continue
		}

		fmt.Fprintf(r.To, "exec git branch -f %v\n", b.Name)
		r.updatedBranches = append(r.updatedBranches, b.Name)
		updated = true
	}

	// Add an empty line between branch sections.
	if updated {
		fmt.Fprintln(r.To)
	}
	return updated
}

const _pushSectionPrefix = "# Uncomment this section to push the changes.\n"

func (r *gitRestack) writePushSection(padBefore, padAfter bool) error {
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
