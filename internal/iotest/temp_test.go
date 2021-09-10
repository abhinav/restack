package iotest

import (
	"os"
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

func TestTempDir(t *testing.T) {
	ft := fakeT{T: t}

	dir := TempDir(&ft, "foo")
	assert.NotEmpty(t, dir, "expected a directory")

	info, err := os.Stat(dir)
	require.NoError(t, err)

	assert.True(t, info.IsDir(),
		"expected directory, got %v", info.Mode())

	ft.runCleanup()

	info, err = os.Stat(dir)
	assert.ErrorIs(t, err, os.ErrNotExist,
		"directory should not exist after cleanup, got %v", info)
}

func TestTempFile(t *testing.T) {
	t.Run("automatically close", func(t *testing.T) {
		ft := fakeT{T: t}

		file := TempFile(&ft, "foo")
		require.NotNil(t, file, "expected a file")

		info, err := os.Stat(file.Name())
		require.NoError(t, err)

		mode := info.Mode()
		assert.True(t, mode.IsRegular(),
			"expected file, got %v", mode)

		ft.runCleanup()

		info, err = os.Stat(file.Name())
		assert.ErrorIs(t, err, os.ErrNotExist,
			"file should not exist after cleanup, got %v", info)
	})

	t.Run("already closed", func(t *testing.T) {
		f := TempFile(t, "foo")
		require.NoError(t, f.Close(), "close %q", f.Name())
	})
}
