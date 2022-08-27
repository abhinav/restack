package restack

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/abhinav/restack/internal/editorfake"
	"github.com/abhinav/restack/internal/ostest"
	"github.com/abhinav/restack/internal/testwriter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemGit_RebaseHeadName(t *testing.T) {
	ostest.Setenv(t, "HOME", t.TempDir())
	ostest.Chdir(t, t.TempDir())

	gitInit(t)

	// Make this git repository use our fake editor to edit rebase
	// instructions. The fake editor will leave the contents as-is, adding
	// a "break" instruction to the top.
	editor := editorfake.New(t, editorfake.AddPrefix("break\n"))
	git(t, "config", "sequence.editor", editor)

	touch(t, "foo")
	git(t, "add", "foo")
	git(t, "commit", "-m", "add foo")

	git(t, "checkout", "-b", "feature")

	touch(t, "bar", "baz")
	git(t, "add", "bar", "baz")
	git(t, "commit", "-m", "add bar and baz")

	touch(t, "qux")
	git(t, "add", "qux")
	git(t, "commit", "-am", "add qux")

	git(t, "rebase", "-i", "HEAD~2")

	ctx := context.Background()

	sg := SystemGit{Getenv: os.Getenv}
	branch, err := sg.RebaseHeadName(ctx)
	require.NoError(t, err, "determine rebase head name")

	assert.Equal(t, "feature", branch, "unexpected head")
}

func TestSystemGit_ListBranches(t *testing.T) {
	ostest.Setenv(t, "HOME", t.TempDir())
	ostest.Chdir(t, t.TempDir())

	gitInit(t)

	// master
	touch(t, "foo")
	git(t, "add", "foo")
	git(t, "commit", "-m", "add foo")

	// feature1
	git(t, "checkout", "-b", "feature1")
	touch(t, "bar")
	git(t, "add", "bar")
	git(t, "commit", "-m", "add bar")

	// feature2
	git(t, "checkout", "-b", "feature2")
	touch(t, "baz")
	git(t, "add", "baz")
	git(t, "commit", "-m", "add baz")

	// feature3
	git(t, "checkout", "-b", "feature3", "master")
	touch(t, "qux")
	git(t, "add", "qux")
	git(t, "commit", "-m", "add qux")

	ctx := context.Background()

	sg := SystemGit{Getenv: os.Getenv}
	bs, err := sg.ListBranches(ctx)
	require.NoError(t, err, "list branches")

	wantBranches := map[string]struct{}{
		"master":   {},
		"feature1": {},
		"feature2": {},
		"feature3": {},
	}

	for _, b := range bs {
		_, ok := wantBranches[b.Name]
		if assert.True(t, ok, "unexpected branch: %v (%v)", b.Name, b.Hash) {
			delete(wantBranches, b.Name)
		}
	}

	assert.Empty(t, wantBranches, "missing branches")
}

func gitInit(t *testing.T) {
	t.Helper()

	git(t, "init")
	git(t, "config", "user.name", "potato")
	git(t, "config", "user.email", "noreply@example.com")
}

func git(t *testing.T, args ...string) {
	t.Helper()

	require.NoError(t, gitCmd(t, args...).Run(),
		"git %q", args)
}

func gitCmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Stdout = testwriter.New(t)
	cmd.Stderr = testwriter.New(t)
	return cmd
}

func touch(t *testing.T, paths ...string) {
	t.Helper()

	for _, path := range paths {
		require.NoError(t,
			os.WriteFile(path, []byte{}, 0o644),
			"touch %q", path)
	}
}
