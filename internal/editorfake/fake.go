// Package editorfake provides a means of building a configurable fake editor
// executable.
//
// It works by hooking into the entry point of the current test executable
// with TryMain.
package editorfake

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/abhinav/restack/internal/iotest"
	"github.com/abhinav/restack/internal/test"
	"github.com/stretchr/testify/require"
)

// TryMain is the entry point for the fake editor. It runs the editor behavior
// if inside a fake editor environment.
//
// Use this in TestMain before calling m.Run().
//
//	func TestMain(m *testing.M) {
//	  editortest.TryMain()
//
//	  os.Exit(m.Run())
//	}
//
// This is a no-op if not inside a fake editor environment.
func TryMain() {
	if !flag.Parsed() {
		flag.Parse()
	}

	cfgFile := os.Getenv("TEST_FAKE_EDITOR_CONFIG")
	if len(cfgFile) == 0 {
		return
	}

	if err := main(cfgFile); err != nil {
		log.Fatalf("editor failed: %+v", err)
	}

	os.Exit(0)
}

type optionType int

const (
	optionTypeWantContents optionType = iota + 1
	optionTypeGiveContents
	optionTypeAddPrefix
	optionTypeDeleteFile
	optionTypeExitCode
)

type opConfig struct {
	Type  optionType
	Value Option
}

func (cfg *opConfig) UnmarshalJSON(b []byte) error {
	var raw struct {
		Type  *optionType
		Value json.RawMessage
	}
	raw.Type = &cfg.Type // deserialize into cfg.Type

	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	switch cfg.Type {
	case optionTypeWantContents:
		cfg.Value = new(wantContents)
	case optionTypeGiveContents:
		cfg.Value = new(giveContents)
	case optionTypeAddPrefix:
		cfg.Value = new(addPrefix)
	case optionTypeDeleteFile:
		cfg.Value = new(deleteFile)
	case optionTypeExitCode:
		cfg.Value = new(exitCode)
	default:
		return fmt.Errorf("unknown op type: %v", cfg.Type)
	}

	return json.Unmarshal(raw.Value, cfg.Value)
}

// Specifies the behavior of a fake editor to use in tests.
type config struct {
	Ops []opConfig
}

// New builds a new fake editor that runs the provided operations.
//
// It returns a shell command that, when invoked, acts like a valid editor.
func New(t test.T, ops ...Option) string {
	// Detect invocation of editorfake.New inside an editorfake. This
	// happens if we don't install this in TestMain.
	cfgFile := os.Getenv("TEST_FAKE_EDITOR_CONFIG")
	require.Empty(t, cfgFile,
		"already inside a test editor (TEST_FAKE_EDITOR_CONFIG=%v):\n"+
			"did you forget to call editorfake.TryMain?", cfgFile)

	var cfg config
	for _, op := range ops {
		cfg.Ops = append(cfg.Ops, opConfig{
			Type:  op.optionType(),
			Value: op,
		})
	}

	testExe, err := os.Executable()
	require.NoError(t, err, "determine test executable")

	f := iotest.TempFile(t, "fake-editor-config")
	defer f.Close()

	require.NoError(t, json.NewEncoder(f).Encode(cfg),
		"encode fake editor config")

	return fmt.Sprintf("TEST_FAKE_EDITOR_CONFIG=%v %v", f.Name(), testExe)
}

func main(cfgFile string) error {
	f, err := os.Open(cfgFile)
	if err != nil {
		return fmt.Errorf("open config: %v", err)
	}
	defer f.Close()

	var cfg config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return fmt.Errorf("decode config: %v", err)
	}

	args := flag.Args()
	if len(args) == 0 {
		return errors.New("usage: editor file")
	}
	file := args[0]

	contents, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %q: %v", file, err)
	}

	s := state{
		Path:     file,
		Contents: string(contents),
	}
	for _, op := range cfg.Ops {
		if err := op.Value.run(&s); err != nil {
			return err
		}
	}

	os.Exit(s.ExitCode)
	return nil
}
