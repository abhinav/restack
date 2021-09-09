package ostest

import (
	"os"

	"github.com/abhinav/restack/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Chdir changes the working directory to the given directory for the duration
// of the current test.
//
// The old working directory is restored when the test exits.
func Chdir(t test.T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	require.NoError(t, err, "get cwd")

	require.NoError(t,
		os.Chdir(dir), "chdir %q", dir)

	t.Cleanup(func() {
		assert.NoError(t,
			os.Chdir(oldDir),
			"chdir to old %q", oldDir)
	})
}
