package iotest

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestTempFile(t *testing.T) {
	t.Run("automatically close", func(t *testing.T) {
		ft := fakeT{T: t}

		file := TempFile(&ft, "foo")
		require.NotNil(t, file, "expected a file")

		info := Stat(t, file.Name())

		mode := info.Mode()
		assert.True(t, mode.IsRegular(),
			"expected file, got %v", mode)

		ft.runCleanup()

		info, err := os.Stat(file.Name())
		assert.ErrorIs(t, err, os.ErrNotExist,
			"file should not exist after cleanup, got %v", info)
	})

	t.Run("already closed", func(t *testing.T) {
		f := TempFile(t, "foo")
		require.NoError(t, f.Close(), "close %q", f.Name())
	})
}

func TestReadWriteStatFile(t *testing.T) {
	t.Parallel()

	dst := filepath.Join(t.TempDir(), "foo")
	WriteFile(t, dst, "hello", 0o644)
	assert.Equal(t, "hello", ReadFile(t, dst))
	assert.Equal(t, fs.FileMode(0o644), Stat(t, dst).Mode())

	t.Run("already exists", func(t *testing.T) {
		WriteFile(t, dst, "world", 0o755)
		assert.Equal(t, "world", ReadFile(t, dst))
		assert.Equal(t, fs.FileMode(0o644), Stat(t, dst).Mode(),
			"WriteFile should not change permissions for existing files")
	})
}
