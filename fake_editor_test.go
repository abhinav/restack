package restack

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

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
	//
	// Check omitted if empty.
	WantContents string `json:"wantContents"`

	// Either the contents to write to the file, or the prefix to prepend
	// to its contents.
	GiveContents string `json:"giveContents"`
	AddPrefix    string `json:"addPrefix"`

	// Exit code to use.
	ExitCode int `json:"exitCode"`

	// If set to true, the editor will delete the file before exiting.
	DeleteFile bool `json:"deleteFile"`
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
	file := args[0]

	got, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %q: %v", file, err)
	}

	if len(cfg.WantContents) > 0 {
		if diff := cmp.Diff(cfg.WantContents, string(got)); len(diff) > 0 {
			return fmt.Errorf("contents mismatch: (-want, +got)\n%s", diff)
		}
	}

	var give []byte
	if len(cfg.AddPrefix) > 0 {
		give = append([]byte(cfg.AddPrefix), got...)
	} else {
		give = []byte(cfg.GiveContents)
	}

	if err := ioutil.WriteFile(file, give, 0644); err != nil {
		return fmt.Errorf("write output: %v", err)
	}

	if cfg.DeleteFile {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("delete file: %v", err)
		}
	}

	os.Exit(cfg.ExitCode)
	return nil
}
