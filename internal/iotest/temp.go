package iotest

import (
	"io/ioutil"
	"os"

	"github.com/abhinav/restack/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TempDir creates a new temporary directory inside the current test context.
//
// It deletes the directory when the test finishes.
func TempDir(t test.T, prefix string) string {
	t.Helper()

	dir, err := ioutil.TempDir("", prefix)
	require.NoError(t, err, "make tempdir")

	t.Cleanup(func() {
		assert.NoError(t,
			os.RemoveAll(dir),
			"delete tempdir %q", dir)
	})

	return dir
}

// TempFile creates a new temporary file inside the current test context.
//
// It deletes the file when the test finishes.
func TempFile(t test.T, prefix string) *os.File {
	t.Helper()

	f, err := ioutil.TempFile("", prefix)
	require.NoError(t, err, "make tempfile")
	name := f.Name()

	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			assert.ErrorIs(t, err, os.ErrClosed,
				"close tempfile %q", name)
		}

		assert.NoError(t, os.Remove(name),
			"delete tempfile %q", name)
	})

	return f
}
