package iotest

import (
	"io/fs"
	"os"

	"github.com/abhinav/restack/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TempFile creates a new temporary file inside the current test context.
//
// It deletes the file when the test finishes.
func TempFile(t test.T, prefix string) *os.File {
	t.Helper()

	f, err := os.CreateTemp("", prefix)
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

// WriteFile is a shortcut to os.WriteFile for tests.
func WriteFile(t test.T, path, body string, perm os.FileMode) {
	t.Helper()

	require.NoError(t,
		os.WriteFile(path, []byte(body), perm),
		"write %q", path)
}

// ReadFile is a shortcut to os.ReadFile for tests.
func ReadFile(t test.T, path string) string {
	t.Helper()

	body, err := os.ReadFile(path)
	require.NoError(t, err, "read %q", path)
	return string(body)
}

// Stat is a shortcut to os.Stat for tests.
func Stat(t test.T, path string) fs.FileInfo {
	t.Helper()

	info, err := os.Stat(path)
	require.NoError(t, err)
	return info
}
