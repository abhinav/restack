package restack

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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

	// Retrives the value of a logical git variable.
	//
	// See `man git-var`
	Var(ctx context.Context, name string) (string, error)

	// Returns a mapping from abbreviated hash to list of refs at that hash.
	ListHeads(ctx context.Context) (map[string][]string, error)

	// RebaseHeadName returns the name of the branch being rebased or an empty
	// string if we're not in the middle of a rebase.
	RebaseHeadName(ctx context.Context) (string, error)
}

// DefaultGit is an instance of Git that operates on the git repository in the
// current directory.
var DefaultGit Git = defaultGit{}

type defaultGit struct{}

func (defaultGit) SetGlobalConfig(ctx context.Context, name, value string) error {
	cmd := exec.CommandContext(ctx, "git", "config", "--global", name, value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to change git config: %v", err)
	}
	return nil
}

func (defaultGit) ListHeads(ctx context.Context) (map[string][]string, error) {
	cmd := exec.CommandContext(ctx, "git", "show-ref", "--heads", "--abbrev")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to run git show-ref: %v", err)
	}
	defer func() {
		err = multierr.Append(err, cmd.Wait())
	}()

	return parseGitShowRef(out)
}

func (defaultGit) Var(ctx context.Context, name string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "var", name)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not get git var value: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

var _rebaseStateDirs = []string{"rebase-apply", "rebase-merge"}

func (defaultGit) RebaseHeadName(ctx context.Context) (string, error) {
	// git stores information about the rebase under either .git/rebase-apply
	// or .git/rebase-merge. Either way, the branch name is stored in a file
	// called head-name in that directory.
	//
	// See https://github.com/git/git/blob/2f0e14e649d69f9535ad6a086c1b1b2d04436ef5/wt-status.c#L1473

	gitDir := os.Getenv("GIT_DIR")
	if gitDir == "" {
		gitDir = ".git"
	}

	for _, stateDir := range _rebaseStateDirs {
		headFile := filepath.Join(gitDir, stateDir, "head-name")
		if info, err := os.Stat(headFile); os.IsNotExist(err) || info.IsDir() {
			continue
		}

		nameBytes, err := ioutil.ReadFile(headFile)
		if err != nil {
			return "", fmt.Errorf("failed to read %q: %v", headFile, err)
		}

		// TODO: Use separate type to represent branch name vs ref name.
		name := strings.TrimSpace(string(nameBytes))
		return strings.TrimPrefix(name, "refs/heads/"), nil
	}

	return "", nil
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
