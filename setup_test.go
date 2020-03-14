package restack

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/abhinav/restack/internal/testutil"
)

func TestSetup(t *testing.T) {
	home := testutil.TempDir(t)
	testutil.Setenv(t, "HOME", home)

	twriter := testutil.NewWriter(t)
	setup := &Setup{
		Stdout: twriter,
		Stderr: twriter,
	}

	ctx := context.Background()
	if err := setup.Run(ctx); err != nil {
		t.Fatalf("setup must not fail")
	}

	scriptPath := filepath.Join(home, ".restack/edit.sh")
	scriptInfo, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("want edit script: %v", err)
	}

	if mode := scriptInfo.Mode(); mode&0100 == 0 {
		t.Errorf("edit.sh: want executable, got %v", mode)
	}

	cmd := exec.Command("git", "config", "--global", "sequence.editor")
	cmd.Stderr = twriter
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git config check: %v", err)
	}
	out = bytes.TrimSpace(out)

	if string(out) != scriptPath {
		t.Errorf("git sequence.editor = %q, want %q", out, scriptPath)
	}
}

func TestSetup_PrintScript(t *testing.T) {
	var stdout bytes.Buffer
	setup := &Setup{
		PrintScript: true,
		Stdout:      &stdout,
		Stderr:      testutil.NewWriter(t),
	}

	ctx := context.Background()
	if err := setup.Run(ctx); err != nil {
		t.Fatalf("Setup failed unexpectedly")
	}

	if stdout.Len() == 0 {
		t.Errorf("setup --print-edit-script got empty stdout")
	}
}

func TestSetup_NoHome(t *testing.T) {
	testutil.Unsetenv(t, "HOME")

	twriter := testutil.NewWriter(t)
	setup := &Setup{
		Stdout: twriter,
		Stderr: twriter,
	}

	ctx := context.Background()
	if err := setup.Run(ctx); err == nil {
		t.Errorf("setup must fail")
	}
}
