package testutil

import (
	"io/ioutil"
	"os"
)

// TempDir creates a new temporary directory inside the current test context.
//
// It deletes the directory when the test finishes.
func TempDir(t TestingT) string {
	t.Helper()

	dir, err := ioutil.TempDir("", t.Name())
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
