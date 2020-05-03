package restack

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/abhinav/restack/internal/iotest"
	"github.com/abhinav/restack/internal/ostest"
	"github.com/abhinav/restack/internal/testwriter"
)

func TestSystemGit_RebaseHeadName(t *testing.T) {
	ostest.Chdir(t, iotest.TempDir(t, "git-rebase-head-name"))

	gitInit(t)

	// Make this git repository use our fake editor to edit rebase
	// instructions. The fake editor will leave the contents as-is, adding
	// a "break" instruction to the top.
	editor := fakeEditorConfig{AddPrefix: "break\n"}
	git(t, "config", "sequence.editor", editor.Build(t))

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
	if err != nil {
		t.Fatalf("determine rebase head name: %v", err)
	}

	if branch != "feature" {
		t.Errorf("unexpected head: got %q, want %q", branch, "feature")
	}
}

func TestSystemGit_ListBranches(t *testing.T) {
	ostest.Chdir(t, iotest.TempDir(t, "git-list-branches"))

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
	if err != nil {
		t.Fatalf("list branches: %v", err)
	}

	wantBranches := map[string]struct{}{
		"master":   {},
		"feature1": {},
		"feature2": {},
		"feature3": {},
	}

	for _, b := range bs {
		if _, ok := wantBranches[b.Name]; !ok {
			t.Errorf("unexpected branch: %v (%v)", b.Name, b.Hash)
		} else {
			delete(wantBranches, b.Name)
		}
	}

	for b := range wantBranches {
		t.Errorf("missing branch: %v", b)
	}
}

func gitInit(t *testing.T) {
	t.Helper()

	git(t, "init")
	git(t, "config", "user.name", "potato")
	git(t, "config", "user.email", "noreply@example.com")
}

func git(t *testing.T, args ...string) {
	t.Helper()

	if err := gitCmd(t, args...).Run(); err != nil {
		t.Fatalf("git %q: %v", args, err)
	}
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
		if err := ioutil.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("touch %q: %v", path, err)
		}
	}
}
