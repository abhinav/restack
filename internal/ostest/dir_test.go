package ostest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/abhinav/restack/internal/iotest"
)

func TestChdir(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}

	tempdir := iotest.TempDir(t, "chdir")

	// On macOS, tempdir may be a symlink that reports a different working
	// directory once Chdir-ed into.
	if d, err := filepath.EvalSymlinks(tempdir); err != nil {
		t.Fatalf("unable to evaluate symlink %q: %v", tempdir, err)
	} else {
		tempdir = d
	}

	ft := fakeT{T: t}
	Chdir(&ft, tempdir)

	if cwd, err := os.Getwd(); err != nil {
		t.Fatalf("get cwd: %v", err)
	} else if cwd != tempdir {
		t.Errorf("unexpected cwd after Chdir %v, want %v", cwd, tempdir)
	}

	ft.runCleanups()

	if cwd, err := os.Getwd(); err != nil {
		t.Fatalf("get cwd: %v", err)
	} else if cwd != old {
		t.Errorf("unexpected cwd after Chdir %v, want %v", cwd, old)
	}

}
