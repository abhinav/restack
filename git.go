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

//go:generate mockgen -destination=mock_git_test.go -package=restack -self_package github.com/abhinav/restack github.com/abhinav/restack Git

// Git provides access to git commands.
type Git interface {
	// Returns a mapping from abbreviated hash to list of refs at that hash.
	ListHeads(ctx context.Context) (map[string][]string, error)

	// RebaseHeadName returns the name of the branch being rebased or an empty
	// string if we're not in the middle of a rebase.
	RebaseHeadName(ctx context.Context) (string, error)
}

// SystemGit uses the global `git` command to perform git operations.
type SystemGit struct {
	Getenv func(string) string
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

const _gitDir = "GIT_DIR"

// RebaseHeadName implements Git.RebaseHeadName.
func (g *SystemGit) RebaseHeadName(ctx context.Context) (string, error) {
	// git stores information about the rebase under either .git/rebase-apply
	// or .git/rebase-merge. Either way, the branch name is stored in a file
	// called head-name in that directory.
	//
	// See https://github.com/git/git/blob/2f0e14e649d69f9535ad6a086c1b1b2d04436ef5/wt-status.c#L1473

	gitDir := g.Getenv(_gitDir)
	if gitDir == "" {
		gitDir = ".git"
	}

	for _, stateDir := range _rebaseStateDirs {
		headFile := filepath.Join(gitDir, stateDir, "head-name")
		if _, err := os.Stat(headFile); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}

		nameBytes, err := ioutil.ReadFile(headFile)
		if err != nil {
			return "", fmt.Errorf("read %q: %v", headFile, err)
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
