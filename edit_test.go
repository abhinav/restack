package restack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/abhinav/restack/internal/editorfake"
	"github.com/abhinav/restack/internal/test"
	"github.com/abhinav/restack/internal/testwriter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _noop = "noop\n"

func TestEdit(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "git-rebase-todo")

	require.NoError(t,
		os.WriteFile(file, []byte(_noop), 0o600),
		"write temporary file")

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
	require.NoError(t, err, "edit failed")

	gotOutput, err := os.ReadFile(file)
	require.NoError(t, err, "read output")

	assert.Equal(t, editorOutput, string(gotOutput),
		"output mismatch")
}

type fakeRestacker struct {
	T test.T

	ran        bool
	WantInput  string
	GiveOutput string
	FailWith   error
}

func (r *fakeRestacker) VerifyRan() {
	assert.True(r.T, r.ran, "restack never executed")
}

func (r *fakeRestacker) Restack(ctx context.Context, req *Request) error {
	t := r.T
	r.ran = true

	gotInput, err := io.ReadAll(req.From)
	if !assert.NoError(t, err, "read input") {
		return err
	}

	assert.Equal(t, r.WantInput, string(gotInput), "input mismatch")

	_, err = req.To.Write([]byte(r.GiveOutput))
	if !assert.NoError(t, err, "write output") {
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
	assert.Error(t, err, "edit must fail")
	assert.Contains(t, err.Error(), "no such file")
}

// Handle failures in restacking the instructions.
func TestEdit_RestackFailed(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "git-rebase-todo")

	require.NoError(t,
		os.WriteFile(file, []byte(_noop), 0o600),
		"write temporary file")

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
	assert.Error(t, err, "edit must fail")
	assert.Contains(t, err.Error(), "great sadness")
}

// Handle non-zero codes from editors.
func TestEdit_EditorFailed(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "git-rebase-todo")

	require.NoError(t,
		os.WriteFile(file, []byte{}, 0o600),
		"write temporary file")

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
	assert.Error(t, err, "edit must fail")
	assert.Contains(t, err.Error(), "exit status 1")
}

// Handle failures in renaming if, for example, the file was deleted by the
// editor.
func TestEdit_RenameFailed(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "git-rebase-todo")

	require.NoError(t,
		os.WriteFile(file, []byte{}, 0o600),
		"write temporary file")

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
	assert.Error(t, err, "edit must fail")
	assert.Contains(t, err.Error(), fmt.Sprintf("overwrite %q", file))
	assert.Contains(t, err.Error(), "no such file or directory")
}
