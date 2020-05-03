package iotest

import (
	"errors"
	"os"
	"testing"
)

type fakeT struct {
	*testing.T

	cleanups []func()
}

func (t *fakeT) Cleanup(f func()) {
	t.cleanups = append(t.cleanups, f)
}

func (t *fakeT) runCleanup() {
	for _, f := range t.cleanups {
		defer f()
	}
}

func TestTempDir(t *testing.T) {
	ft := fakeT{T: t}

	dir := TempDir(&ft, "foo")
	if len(dir) == 0 {
		t.Fatal("expected a directory")
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !info.IsDir() {
		t.Errorf("expected directory, got %v", info.Mode())
	}

	ft.runCleanup()

	if info, err = os.Stat(dir); err == nil {
		t.Errorf("expected error, got %v", info.Mode())
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("unexpected error %v, expected %v", err, os.ErrNotExist)
	}
}
