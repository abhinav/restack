package restack

import (
	"context"
	_ "embed" // for setup script
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

// Setup is the "restack setup" command.
type Setup struct {
	PrintScript bool

	Stdout io.Writer
	Stderr io.Writer
}

//go:embed edit.sh
var _editScript []byte

// Run runs the setup command.
func (s *Setup) Run(ctx context.Context) error {
	if s.PrintScript {
		fmt.Fprint(s.Stdout, string(_editScript))
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %v", err)
	}

	// TODO: Use config dir in the future
	restackDir := filepath.Join(home, ".restack")
	if err := os.MkdirAll(restackDir, 0755); err != nil {
		return fmt.Errorf("create directory %q: %v", restackDir, err)
	}

	editCmd := filepath.Join(restackDir, "edit.sh")
	if err := ioutil.WriteFile(editCmd, _editScript, 0755); err != nil {
		return fmt.Errorf("write file %q: %v", editCmd, err)
	}

	cmd := exec.CommandContext(ctx, "git", "config", "--global", "sequence.editor", editCmd)
	cmd.Stdout = s.Stdout
	cmd.Stderr = s.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("set sequence editor: %v", err)
	}

	fmt.Fprintln(s.Stderr, "restack has been set up successfully.")
	return nil
}
