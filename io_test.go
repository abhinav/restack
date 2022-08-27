package restack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/abhinav/restack/internal/iotest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hijackRename(t testing.TB, newFn func(string, string) error) {
	t.Helper()

	oldFn := _osRename
	t.Cleanup(func() { _osRename = oldFn })

	_osRename = newFn
}

func TestRenameFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "foo")
		dst := filepath.Join(tempDir, "bar")

		iotest.WriteFile(t, src, "body", 0o644)
		require.NoError(t, renameFile(src, dst),
			"rename failed")

		assert.Equal(t, "body", iotest.ReadFile(t, dst), "body mismatch")
		assert.Equal(t, os.FileMode(0o644), iotest.Stat(t, dst).Mode(),
			"permissions mismatch")

		_, err := os.Stat(src)
		require.Error(t, err, "%q must not exist", src)
	})

	t.Run("cross-device link error", func(t *testing.T) {
		hijackRename(t, func(string, string) error {
			return fmt.Errorf("great sadness: %w", syscall.EXDEV)
		})

		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "foo")
		dst := filepath.Join(tempDir, "bar")

		iotest.WriteFile(t, src, "body", 0o644)
		require.NoError(t, renameFile(src, dst),
			"rename failed")

		assert.Equal(t, "body", iotest.ReadFile(t, dst), "body mismatch")
		assert.Equal(t, os.FileMode(0o644), iotest.Stat(t, dst).Mode(),
			"permissions mismatch")

		_, err := os.Stat(src)
		require.Error(t, err, "%q must not exist", src)
	})

	t.Run("other error", func(t *testing.T) {
		hijackRename(t, func(string, string) error {
			return errors.New("great sadness")
		})

		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "foo")
		dst := filepath.Join(tempDir, "bar")

		iotest.WriteFile(t, src, "body", 0o644)
		require.Error(t, renameFile(src, dst),
			"rename should fail")
	})
}

func TestUnsafeRenameFile(t *testing.T) {
	t.Parallel()

	t.Run("dst does not exist", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "foo")
		dst := filepath.Join(tempDir, "bar")

		iotest.WriteFile(t, src, "body", 0o644)

		require.NoError(t, unsafeRenameFile(src, dst),
			"unsafe rename failed")

		assert.Equal(t, "body", iotest.ReadFile(t, dst), "body mismatch")
		assert.Equal(t, os.FileMode(0o644), iotest.Stat(t, dst).Mode(),
			"permissions mismatch")

		_, err := os.Stat(src)
		require.Error(t, err, "%q must not exist", src)
	})

	t.Run("dst exists", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "foo")
		dst := filepath.Join(tempDir, "bar")

		iotest.WriteFile(t, src, "body1", 0o755)
		iotest.WriteFile(t, dst, "body2", 0o600)

		require.NoError(t, unsafeRenameFile(src, dst),
			"unsafe rename failed")

		assert.Equal(t, "body1", iotest.ReadFile(t, dst), "body mismatch")
		assert.Equal(t, os.FileMode(0o755), iotest.Stat(t, dst).Mode(),
			"permissions mismatch")

		_, err := os.Stat(src)
		require.Error(t, err, "%q must not exist", src)
	})

	t.Run("dst failed", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "foo")
		dst := filepath.Join(tempDir, "bar", "baz", "qux")
		// Parent directories don't exist.

		iotest.WriteFile(t, src, "body", 0o644)

		err := unsafeRenameFile(src, dst)
		require.Error(t, err, "unsafe rename should fail")

		// src should rename unchanged.
		assert.Equal(t, "body", iotest.ReadFile(t, src))
		assert.Equal(t, os.FileMode(0o644), iotest.Stat(t, src).Mode(),
			"permissions mismatch")
	})

	t.Run("src does not exist", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "foo")
		dst := filepath.Join(tempDir, "bar")

		require.Error(t, unsafeRenameFile(src, dst))
	})
}
