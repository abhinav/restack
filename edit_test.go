package restack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhinav/restack/internal/editorfake"
	"github.com/abhinav/restack/internal/iotest"
	"github.com/abhinav/restack/internal/testwriter"
	"github.com/google/go-cmp/cmp"
)

var _noop = "noop\n"

func TestEdit(t *testing.T) {
	dir := iotest.TempDir(t, "edit")
	file := filepath.Join(dir, "git-rebase-todo")

	if err := ioutil.WriteFile(file, []byte(_noop), 0600); err != nil {
		t.Fatalf("write temporary file: %v", err)
	}

	restackerOutput := "x echo hello world"
	restacker := fakeRestacker{
		T:          t,
		WantInput:  _noop,
		GiveOutput: restackerOutput,
	}
	defer restacker.VerifyRan()

	editorOutput := "x echo hello to you too"
	editor := editorfake.New(t,
		editorfake.WantContents(restackerOutput),
		editorfake.GiveContents(editorOutput),
	)

	ctx := context.Background()
	err := (&Edit{
		Editor:    editor,
		Path:      file,
		Restacker: &restacker,
		Stdin:     new(bytes.Buffer),
		Stdout:    testwriter.New(t),
		Stderr:    testwriter.New(t),
	}).Run(ctx)

	if err != nil {
		t.Errorf("edit failed: %v", err)
	}

	gotOutput, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if diff := cmp.Diff(editorOutput, string(gotOutput)); len(diff) > 0 {
		t.Errorf("output mismatch: (-want, +got):\n%s", diff)
	}
}

type fakeRestacker struct {
	T *testing.T

	ran        bool
	WantInput  string
	GiveOutput string
	FailWith   error
}

func (r *fakeRestacker) VerifyRan() {
	if !r.ran {
		r.T.Errorf("restack never executed")
	}
}

func (r *fakeRestacker) Restack(ctx context.Context, req *Request) error {
	t := r.T
	r.ran = true

	gotInput, err := ioutil.ReadAll(req.From)
	if err != nil {
		t.Errorf("read input: %v", err)
		return err
	}

	if diff := cmp.Diff(r.WantInput, string(gotInput)); len(diff) > 0 {
		t.Errorf("input mismatch: (-want, +got):\n%s", diff)
	}

	if _, err := req.To.Write([]byte(r.GiveOutput)); err != nil {
		t.Errorf("write output: %v", err)
		return err
	}

	return r.FailWith
}

// Handle missing files.
func TestEdit_MissingFile(t *testing.T) {
	ctx := context.Background()
	err := (&Edit{
		Editor:    "false",
		Path:      "does not exist",
		Restacker: &fakeRestacker{T: t},
		Stdin:     new(bytes.Buffer),
		Stdout:    testwriter.New(t),
		Stderr:    testwriter.New(t),
	}).Run(ctx)
	if err == nil {
		t.Errorf("edit must fail")
	}
	errorMustContain(t, err, "no such file")
}

// Handle failures in restacking the instructions.
func TestEdit_RestackFailed(t *testing.T) {
	dir := iotest.TempDir(t, "edit-restack-fail")
	file := filepath.Join(dir, "git-rebase-todo")

	if err := ioutil.WriteFile(file, []byte(_noop), 0600); err != nil {
		t.Fatalf("write temporary file: %v", err)
	}

	ctx := context.Background()
	err := (&Edit{
		Editor: "false",
		Path:   file,
		Restacker: &fakeRestacker{
			T:         t,
			WantInput: _noop,
			FailWith:  errors.New("great sadness"),
		},
		Stdin:  new(bytes.Buffer),
		Stdout: testwriter.New(t),
		Stderr: testwriter.New(t),
	}).Run(ctx)
	if err == nil {
		t.Errorf("edit must fail")
	}
	errorMustContain(t, err, "great sadness")
}

// Handle non-zero codes from editors.
func TestEdit_EditorFailed(t *testing.T) {
	dir := iotest.TempDir(t, "edit-editor-fail")
	file := filepath.Join(dir, "git-rebase-todo")

	if err := ioutil.WriteFile(file, []byte{}, 0600); err != nil {
		t.Fatalf("write temporary file: %v", err)
	}

	restacker := fakeRestacker{T: t}
	defer restacker.VerifyRan()

	editor := editorfake.New(t, editorfake.ExitCode(1))

	ctx := context.Background()
	err := (&Edit{
		Editor:    editor,
		Path:      file,
		Restacker: &restacker,
		Stdin:     new(bytes.Buffer),
		Stdout:    testwriter.New(t),
		Stderr:    testwriter.New(t),
	}).Run(ctx)
	if err == nil {
		t.Fatalf("edit must fail")
	}

	errorMustContain(t, err, "exit status 1")
}

// Handle failures in renaming if, for example, the file was deleted by the
// editor.
func TestEdit_RenameFailed(t *testing.T) {
	dir := iotest.TempDir(t, "edit-rename-fail")
	file := filepath.Join(dir, "git-rebase-todo")

	if err := ioutil.WriteFile(file, []byte{}, 0600); err != nil {
		t.Fatalf("write temporary file: %v", err)
	}

	restacker := fakeRestacker{T: t}
	defer restacker.VerifyRan()

	editor := editorfake.New(t, editorfake.DeleteFile())

	ctx := context.Background()
	err := (&Edit{
		Editor:    editor,
		Path:      file,
		Restacker: &restacker,
		Stdin:     new(bytes.Buffer),
		Stdout:    testwriter.New(t),
		Stderr:    testwriter.New(t),
	}).Run(ctx)
	if err == nil {
		t.Fatalf("edit must fail")
	}

	errorMustContain(t, err, fmt.Sprintf("overwrite %q", file))
	errorMustContain(t, err, "no such file or directory")
}

func errorMustContain(t *testing.T, err error, needle string) {
	t.Helper()

	if !strings.Contains(err.Error(), needle) {
		t.Errorf("error %v must contain %q", err, needle)
	}
}
