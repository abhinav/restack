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

//go:generate $MOCKGEN -destination=mock_git_test.go -package=restack github.com/abhinav/restack Git

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

	// RebaseHeadName returns the name of the branch being rebased or an empty
	// string if we're not in the middle of a rebase.
	RebaseHeadName(ctx context.Context) (string, error)
}

// SystemGit uses the global `git` command to perform git operations.
type SystemGit struct{ fs FS }

// NewSystemGit builds a new SystemGit.
//
// The provides FS is used for file-system operations.
func NewSystemGit(fs FS) *SystemGit {
	return &SystemGit{fs: fs}
}

// SetGlobalConfig implements Git.SetGlobalConfig.
func (*SystemGit) SetGlobalConfig(ctx context.Context, name, value string) error {
	cmd := exec.CommandContext(ctx, "git", "config", "--global", name, value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to change git config: %v", err)
	}
	return nil
}

// ListHeads implements Git.ListHeads.
func (*SystemGit) ListHeads(ctx context.Context) (map[string][]string, error) {
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

var _rebaseStateDirs = []string{"rebase-apply", "rebase-merge"}

// RebaseHeadName implements Git.RebaseHeadName.
func (g *SystemGit) RebaseHeadName(ctx context.Context) (string, error) {
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
		if !g.fs.FileExists(headFile) {
			continue
		}

		r, err := g.fs.ReadFile(headFile)
		if err != nil {
			return "", fmt.Errorf("failed to read %q: %v", headFile, err)
		}

		// It's okay to defer from inside the loop here because if we got here,
		// this is the last iteration of the loop.
		defer r.Close()

		nameBytes, err := ioutil.ReadAll(r)
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
