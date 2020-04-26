package restack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhinav/restack/internal/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestMain(m *testing.M) {
	flag.Parse()

	if cfgFile := os.Getenv("TEST_FAKE_EDITOR_CONFIG"); len(cfgFile) > 0 {
		if err := fakeEditorMain(cfgFile); err != nil {
			log.Fatalf("editor failed: %+v", err)
		}
		os.Exit(0)
	}

	os.Exit(m.Run())
}

// Specifies the behavior of a fake editor to use in tests.
type fakeEditorConfig struct {
	// Expected contents of the file.
	WantContents string `json:"wantContents"`

	// Contents to write to the edited file.
	GiveContents string `json:"giveContents"`
}

// Build returns the string to use as the editor inside a test.
//
// The returned editor will behave as configured, exiting with a non-zero exit
// code if expectations were not met.
func (cfg *fakeEditorConfig) Build(t *testing.T) string {
	testExe, err := os.Executable()
	if err != nil {
		t.Fatalf("determine test executable: %v", err)
	}

	f, err := ioutil.TempFile("", "fake-editor-config")
	if err != nil {
		t.Fatalf("create fake editor config: %v", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(cfg); err != nil {
		t.Fatalf("encode fake editor config: %v", err)
	}

	return fmt.Sprintf("TEST_FAKE_EDITOR_CONFIG=%v %v", f.Name(), testExe)
}

func fakeEditorMain(cfgFile string) error {
	f, err := os.Open(cfgFile)
	if err != nil {
		return fmt.Errorf("open config: %v", err)
	}
	defer f.Close()

	var cfg fakeEditorConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return fmt.Errorf("decode config: %v", err)
	}

	args := flag.Args()
	if len(args) == 0 {
		return errors.New("usage: editor file")
	}

	got, err := ioutil.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("read %q: %v", args[0], err)
	}

	if diff := cmp.Diff(cfg.WantContents, string(got)); len(diff) > 0 {
		return fmt.Errorf("contents mismatch: (-want, +got)\n%s", diff)
	}

	if err := ioutil.WriteFile(args[0], []byte(cfg.GiveContents), 0644); err != nil {
		return fmt.Errorf("write output: %v", err)
	}

	return nil
}

var _noop = "noop\n"

func TestEdit(t *testing.T) {
	dir := testutil.TempDir(t)
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
	editor := fakeEditorConfig{
		WantContents: restackerOutput,
		GiveContents: editorOutput,
	}

	ctx := context.Background()
	err := (&Edit{
		Editor:    editor.Build(t),
		Path:      file,
		Restacker: &restacker,
		Stdin:     new(bytes.Buffer),
		Stdout:    testutil.NewWriter(t),
		Stderr:    testutil.NewWriter(t),
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

func TestEdit_MissingFile(t *testing.T) {
	ctx := context.Background()
	err := (&Edit{
		Editor:    "false",
		Path:      "does not exist",
		Restacker: &fakeRestacker{T: t},
		Stdin:     new(bytes.Buffer),
		Stdout:    testutil.NewWriter(t),
		Stderr:    testutil.NewWriter(t),
	}).Run(ctx)
	if err == nil {
		t.Errorf("edit must fail")
	}
	errorMustContain(t, err, "no such file")
}

func TestEdit_RestackFailed(t *testing.T) {
	dir := testutil.TempDir(t)
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
		Stdout: testutil.NewWriter(t),
		Stderr: testutil.NewWriter(t),
	}).Run(ctx)
	if err == nil {
		t.Errorf("edit must fail")
	}
	errorMustContain(t, err, "great sadness")
}

func errorMustContain(t *testing.T, err error, needle string) {
	t.Helper()

	if !strings.Contains(err.Error(), needle) {
		t.Errorf("error %v must contain %q", err, needle)
	}
}
