package editorfake

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/abhinav/restack/internal/iotest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWantContents(t *testing.T) {
	t.Run("match", func(t *testing.T) {
		s := state{Contents: "foo"}
		assert.NoError(t, WantContents("foo").run(&s),
			`"foo" should match itself`)
	})

	t.Run("mismatch", func(t *testing.T) {
		s := state{Contents: "bar"}
		assert.Error(t,
			WantContents("foo").run(&s),
			`expected error on matching "foo" and "bar"`)
	})
}

func TestGiveContents(t *testing.T) {
	file := filepath.Join(iotest.TempDir(t, "give-contents"), "file")

	s := state{Path: file}
	require.NoError(t,
		GiveContents("foo").run(&s))

	got, err := ioutil.ReadFile(file)
	require.NoError(t, err, "read %q", file)

	want := "foo"
	assert.Equal(t, want, string(got), "file contents mismatch")
	assert.Equal(t, want, s.Contents, "state.Contents mismatch")
}

func TestAddPrefix(t *testing.T) {
	file := filepath.Join(iotest.TempDir(t, "give-contents"), "file")

	s := state{Path: file, Contents: "foo"}
	require.NoError(t, AddPrefix("bar").run(&s))

	got, err := ioutil.ReadFile(file)
	require.NoError(t, err, "read %q", file)

	want := "barfoo"
	assert.Equal(t, want, string(got), "file contents mismatch")
	assert.Equal(t, want, s.Contents, "state.Contents mismatch")
}

func TestDeleteFile(t *testing.T) {
	dir := iotest.TempDir(t, "give-contents")

	t.Run("file does not exist", func(t *testing.T) {
		file := filepath.Join(dir, "does-not-exist")

		s := state{Path: file}
		assert.Error(t, DeleteFile().run(&s),
			"expected error in deleting %q", file)
	})

	t.Run("file exists", func(t *testing.T) {
		file := filepath.Join(dir, "file")

		require.NoError(t,
			ioutil.WriteFile(file, []byte("foo"), 0644),
			"write %q", file)

		s := state{Path: file}
		require.NoError(t, DeleteFile().run(&s))

		info, err := os.Stat(file)
		assert.Error(t, err,
			"file %q should not exist, got %v", file, info)
	})
}

func TestExitCode(t *testing.T) {
	var s state
	assert.NoError(t, ExitCode(2).run(&s))
	assert.Equal(t, 2, s.ExitCode, "unexpected exit code")
}
