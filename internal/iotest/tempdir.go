package iotest

import (
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
