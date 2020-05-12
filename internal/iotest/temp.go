package iotest

import (
	"errors"
	"io/ioutil"
	"os"
)

// T is a subset of the testing.T interface.
type T interface {
	Helper()
	Cleanup(func())
	Fatalf(string, ...interface{})
	Errorf(string, ...interface{})
}

// TempDir creates a new temporary directory inside the current test context.
//
// It deletes the directory when the test finishes.
func TempDir(t T, prefix string) string {
	t.Helper()

	dir, err := ioutil.TempDir("", prefix)
	if err != nil {
		t.Fatalf("make tempdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("delete tempdir %q: %v", dir, err)
		}
	})

	return dir
}

// TempFile creates a new temporary file inside the current test context.
//
// It deletes the file when the test finishes.
func TempFile(t T, prefix string) *os.File {
	t.Helper()

	f, err := ioutil.TempFile("", prefix)
	if err != nil {
		t.Fatalf("make tempfile: %v", err)
	}
	name := f.Name()

	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			if !errors.Is(err, os.ErrClosed) {
				t.Errorf("close tempfile %q: %v", name, err)
			}
		}

		if err := os.Remove(name); err != nil {
			t.Errorf("delete tempfile %q: %v", name, err)
		}
	})

	return f
}
