package restack

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/abhinav/restack/internal/iotest"
	"github.com/abhinav/restack/internal/ostest"
	"github.com/abhinav/restack/internal/testwriter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetup(t *testing.T) {
	home := iotest.TempDir(t, "setup")
	ostest.Setenv(t, "HOME", home)

	twriter := testwriter.New(t)
	setup := &Setup{
		Stdout: twriter,
		Stderr: twriter,
	}

	ctx := context.Background()
	require.NoError(t, setup.Run(ctx), "setup must not fail")

	scriptPath := filepath.Join(home, ".restack/edit.sh")
	scriptInfo, err := os.Stat(scriptPath)
	require.NoError(t, err, "want edit script: %v", scriptPath)

	mode := scriptInfo.Mode()
	assert.NotZero(t, mode&0100, "edit.sh: want executable, got %v", mode)

	cmd := exec.Command("git", "config", "--global", "sequence.editor")
	cmd.Stderr = twriter
	out, err := cmd.Output()
	require.NoError(t, err, "git config check")
	out = bytes.TrimSpace(out)

	assert.Equal(t, string(out), scriptPath,
		"git sequence.editor should match")
}

func TestSetup_PrintScript(t *testing.T) {
	var stdout bytes.Buffer
	setup := &Setup{
		PrintScript: true,
		Stdout:      &stdout,
		Stderr:      testwriter.New(t),
	}

	ctx := context.Background()
	require.NoError(t, setup.Run(ctx), "setup failed")

	assert.NotEmpty(t, stdout.String(),
		"setup --print-edit-script should not be empty")
}

func TestSetup_NoHome(t *testing.T) {
	ostest.Unsetenv(t, "HOME")

	twriter := testwriter.New(t)
	setup := &Setup{
		Stdout: twriter,
		Stderr: twriter,
	}

	ctx := context.Background()
	require.Error(t, setup.Run(ctx), "setup must fail")
}
