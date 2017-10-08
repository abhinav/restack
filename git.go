package restack

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"go.uber.org/multierr"
)

// Git provides access to git commands.
type Git interface {
	// Changes the global configuration for the given key.
	//
	// Equivalent to,
	//
	//   git config --global $name $value
	SetGlobalConfig(ctx context.Context, name, value string) error

	// Returns a mapping from abbreviated hash to list of refs at that hash.
	ListHeads(ctx context.Context) (map[string][]string, error)
}

// DefaultGit is an instance of Git that operates on the git repository in the
// current directory.
var DefaultGit Git = defaultGit{}

type defaultGit struct{}

func (defaultGit) SetGlobalConfig(ctx context.Context, name, value string) (err error) {
	return gitRun(ctx, "config", "--global", name, value)
}

func (defaultGit) ListHeads(ctx context.Context) (map[string][]string, error) {
	cmd := exec.CommandContext(ctx, "git", "show-ref", "--heads", "--abbrev")
	out, err := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to run git show-ref: %v", err)
	}
	defer func() {
		err = multierr.Append(err, cmd.Wait())
	}()

	return parseGitShowRef(out)
}

func gitRun(ctx context.Context, args ...string) error {
	if err := exec.CommandContext(ctx, "git", args...).Run(); err != nil {
		return fmt.Errorf("failed to run git with %q: %v", args, err)
	}
	return nil
}

func parseGitShowRef(r io.Reader) (map[string][]string, error) {
	refs := make(map[string][]string)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		toks := strings.Split(line, " ")
		if len(toks) != 2 {
			continue
		}
		hash := toks[0]
		ref := toks[1]
		refs[hash] = append(refs[hash], ref)
	}
	return refs, scanner.Err()
}
