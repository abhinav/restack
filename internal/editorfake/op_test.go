package editorfake

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/abhinav/restack/internal/iotest"
)

func TestWantContents(t *testing.T) {
	t.Run("match", func(t *testing.T) {
		s := state{Contents: "foo"}
		if err := WantContents("foo").run(&s); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		s := state{Contents: "bar"}
		if err := WantContents("foo").run(&s); err == nil {
			t.Error(`expected error on matching "foo" and "bar"`)
		}
	})
}

func TestGiveContents(t *testing.T) {
	file := filepath.Join(iotest.TempDir(t, "give-contents"), "file")

	s := state{Path: file}
	if err := GiveContents("foo").run(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatalf("read %q: %v", file, err)
	}

	want := "foo"
	if string(got) != want {
		t.Errorf("file contents mismatch: want %q, ot %q", want, got)
	}

	if s.Contents != want {
		t.Errorf("state.Contents mismatch: want %q, ot %q", want, got)
	}
}

func TestAddPrefix(t *testing.T) {
	file := filepath.Join(iotest.TempDir(t, "give-contents"), "file")

	s := state{Path: file, Contents: "foo"}
	if err := AddPrefix("bar").run(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatalf("read %q: %v", file, err)
	}

	want := "barfoo"
	if string(got) != want {
		t.Errorf("file contents mismatch: want %q, ot %q", want, got)
	}

	if s.Contents != want {
		t.Errorf("state.Contents mismatch: want %q, ot %q", want, got)
	}
}

func TestDeleteFile(t *testing.T) {
	dir := iotest.TempDir(t, "give-contents")

	t.Run("file does not exist", func(t *testing.T) {
		file := filepath.Join(dir, "does-not-exist")

		s := state{Path: file}
		if err := DeleteFile().run(&s); err == nil {
			t.Errorf("expected error in deleting %q", file)
		}
	})

	t.Run("file exists", func(t *testing.T) {
		file := filepath.Join(dir, "file")

		if err := ioutil.WriteFile(file, []byte("foo"), 0644); err != nil {
			t.Fatalf("write %q: %v", file, err)
		}

		s := state{Path: file}
		if err := DeleteFile().run(&s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if info, err := os.Stat(file); err == nil {
			t.Errorf("file %q should not exist, got %v", file, info.Mode())
		}
	})
}

func TestExitCode(t *testing.T) {
	var s state
	if err := ExitCode(2).run(&s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if got := s.ExitCode; got != 2 {
		t.Errorf("unexpected exit code %v, want 2", got)
	}
}
