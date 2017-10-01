package restack

import (
	"fmt"
	"os/exec"
)

// Git provides access to git commands.
type Git interface {
	// Changes the global configuration for the given key.
	//
	// Equivalent to,
	//
	//   git config --global $name $value
	SetGlobalConfig(name, value string) error
}

// DefaultGit is an instance of Git that operates on the git repository in the
// current directory.
var DefaultGit Git = defaultGit{}

type defaultGit struct{}

func (defaultGit) SetGlobalConfig(name, value string) error {
	return gitRun("config", "--global", name, value)
}

func gitRun(args ...string) error {
	if err := exec.Command("git", args...).Run(); err != nil {
		return fmt.Errorf("failed to run git with %q: %v", args, err)
	}
	return nil
}
