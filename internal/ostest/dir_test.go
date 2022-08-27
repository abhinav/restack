package ostest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChdir(t *testing.T) {
	old, err := os.Getwd()
	require.NoError(t, err, "get cwd")

	tempdir := t.TempDir()

	// On macOS, tempdir may be a symlink that reports a different working
	// directory once Chdir-ed into.
	d, err := filepath.EvalSymlinks(tempdir)
	require.NoError(t, err, "unable to evaluate symlink %q", tempdir)
	tempdir = d

	ft := fakeT{T: t}
	Chdir(&ft, tempdir)

	cwd, err := os.Getwd()
	require.NoError(t, err, "get cwd")
	assert.Equal(t, tempdir, cwd, "unexpected cwd after Chdir")

	ft.runCleanups()

	cwd, err = os.Getwd()
	require.NoError(t, err, "get cwd")
	assert.Equal(t, old, cwd, "unexpected cwd after Chdir cleanup")
}
