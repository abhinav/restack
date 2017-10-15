package restack

import (
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultFilesystem(t *testing.T) {
	defer tempDir(t)()

	fs := DefaultFilesystem

	dname := "foo/bar"
	fname := filepath.Join(dname, "baz.txt")
	newFname := filepath.Join(dname, "qux.txt")

	t.Run("MkdirAll", func(t *testing.T) {
		_, err := fs.WriteFile(fname)
		require.Error(t, err,
			"must not be able to create file inside a directory that does not exist")

		require.NoErrorf(t, fs.MkdirAll(dname), "failed to MkdirAll(%q)", dname)
	})

	t.Run("FileExists/false", func(t *testing.T) {
		require.Falsef(t, fs.FileExists(dname),
			"%q is a directory, FileExists must return false", dname)
		require.Falsef(t, fs.FileExists(fname), "file %q must not exist", fname)
	})

	t.Run("WriteFile", func(t *testing.T) {
		f, err := fs.WriteFile(fname)
		require.NoError(t, err, "failed to create file: directories may not exist")

		_, err = io.WriteString(f, "hello world")
		require.NoError(t, err, "failed to write to file")

		require.NoError(t, f.Close(), "failed to close file")
	})

	t.Run("FileExists/true", func(t *testing.T) {
		require.Truef(t, fs.FileExists(fname), "file %q must exist", fname)
	})

	t.Run("ReadFile", func(t *testing.T) {
		f, err := fs.ReadFile(fname)
		require.NoError(t, err, "failed to read file")

		out, err := ioutil.ReadAll(f)
		require.NoError(t, err, "failed to read file")

		require.Equal(t, "hello world", string(out),
			"file contents did not match")
	})

	t.Run("Rename", func(t *testing.T) {
		require.Falsef(t, fs.FileExists(newFname), "file %q must not exist", newFname)
		require.NoError(t, fs.Rename(fname, newFname), "failed to rename file")
		require.Truef(t, fs.FileExists(newFname), "file %q must exist", newFname)
	})

	t.Run("TempDir", func(t *testing.T) {
		dir, err := fs.TempDir("zzz")
		require.NoError(t, err, "failed to make temporary directory")
		defer func() {
			require.NoError(t, fs.RemoveAll(dir),
				"failed to delete temporary directory")
		}()

		tfname := filepath.Join(dir, "foo.txt")

		f, err := fs.WriteFile(tfname)
		require.NoErrorf(t, err, "failed to write file %q", tfname)
		require.NoError(t, f.Close(), "failed to close file")
	})

	t.Run("RemoveAll", func(t *testing.T) {
		require.NoErrorf(t, fs.RemoveAll(dname), "failed to delete %q", dname)
		require.Falsef(t, fs.FileExists(fname), "file %q must not exist", fname)

		_, err := fs.WriteFile(fname)
		require.Error(t, err,
			"must not be able to create file inside a directory that does not exist")
	})
}
