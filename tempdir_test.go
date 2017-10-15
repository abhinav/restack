package restack

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func tempDir(t *testing.T) (done func()) {
	cwd, err := os.Getwd()
	require.NoError(t, err, "failed to determine current directory")

	tmpDir, err := ioutil.TempDir("", "restack.test")
	require.NoError(t, err, "failed to create temporary directory")

	require.NoError(t, os.Chdir(tmpDir),
		"failed to change directory")
	return func() {
		require.NoError(t, os.Chdir(cwd),
			"failed to restore old directory")
	}
}
