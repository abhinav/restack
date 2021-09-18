package restack

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shortcut to ioutil.WriteFile.
func writeFile(t testing.TB, path, body string, perm os.FileMode) {
	t.Helper()

	require.NoError(t,
		ioutil.WriteFile(path, []byte(body), perm),
		"write %q", path)
}

// Shortcut to ioutil.ReadFile + os.Stat.
func readFileAndMode(t testing.TB, path string) (string, os.FileMode) {
	t.Helper()

	body, err := ioutil.ReadFile(path)
	require.NoError(t, err, "read %q", path)

	info, err := os.Stat(path)
	require.NoError(t, err)

	return string(body), info.Mode()
}

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

		writeFile(t, src, "body", 0644)
		require.NoError(t, renameFile(src, dst),
			"rename failed")

		got, perm := readFileAndMode(t, dst)
		assert.Equal(t, "body", got, "body mismatch")
		assert.Equal(t, os.FileMode(0644), perm,
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

		writeFile(t, src, "body", 0644)
		require.NoError(t, renameFile(src, dst),
			"rename failed")

		got, perm := readFileAndMode(t, dst)
		assert.Equal(t, "body", got, "body mismatch")
		assert.Equal(t, os.FileMode(0644), perm,
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

		writeFile(t, src, "body", 0644)
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

		writeFile(t, src, "body", 0644)

		require.NoError(t, unsafeRenameFile(src, dst),
			"unsafe rename failed")

		got, perm := readFileAndMode(t, dst)
		assert.Equal(t, "body", got, "body mismatch")
		assert.Equal(t, os.FileMode(0644), perm,
			"permissions mismatch")

		_, err := os.Stat(src)
		require.Error(t, err, "%q must not exist", src)
	})

	t.Run("dst exists", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		src := filepath.Join(tempDir, "foo")
		dst := filepath.Join(tempDir, "bar")

		writeFile(t, src, "body1", 0755)
		writeFile(t, dst, "body2", 0600)

		require.NoError(t, unsafeRenameFile(src, dst),
			"unsafe rename failed")

		got, perm := readFileAndMode(t, dst)
		assert.Equal(t, "body1", got, "body mismatch")
		assert.Equal(t, os.FileMode(0755), perm,
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

		writeFile(t, src, "body", 0644)

		err := unsafeRenameFile(src, dst)
		require.Error(t, err, "unsafe rename should fail")

		// src should rename unchanged.
		got, perm := readFileAndMode(t, src)
		assert.Equal(t, "body", got)
		assert.Equal(t, os.FileMode(0644), perm,
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
