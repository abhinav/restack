package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/abhinav/restack/internal/testutil"
)

func TestRun_Version(t *testing.T) {
	var stdout bytes.Buffer
	opts := options{
		Stdout: &stdout,
		Stderr: testutil.NewWriter(t),
	}
	err := run(&opts, []string{"-version"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if stdout.Len() == 0 {
		t.Errorf("stdout should contain version information")
	}
}

func TestRun_CommandErrors(t *testing.T) {
	tests := []struct {
		name string
		give []string
		want error
	}{
		{
			name: "no arguments",
			want: errCommandUnspecified,
		},
		{
			name: "unknown command",
			give: []string{"foo"},
			want: errUnknownCommand("foo"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			opts := &options{
				Stdout: testutil.NewWriter(t),
				Stderr: &stderr,
			}

			if err := run(opts, tt.give); !errors.Is(err, tt.want) {
				t.Errorf("unexpected error: got %v, want %v", err, tt.want)
			}

			if stderr.Len() == 0 {
				t.Errorf("stderr should contain usage")
			}
		})
	}
}

func TestNewSetup_Errors(t *testing.T) {
	tests := []struct {
		name string
		give []string
		want string
	}{
		{
			name: "too many arguments",
			give: []string{"foo", "bar"},
			want: "too many arguments",
		},
		{
			name: "unknown flag",
			give: []string{"-foo"},
			want: "flag provided but not defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newSetup(&options{
				Stdout: new(bytes.Buffer),
				Stderr: new(bytes.Buffer),
			}, tt.give)

			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("unexpected error: got %v, should contain %q", err, tt.want)
			}
		})
	}
}

func TestNewSetup(t *testing.T) {
	var stdout, stderr bytes.Buffer

	_, err := newSetup(&options{
		Stdout: &stdout,
		Stderr: &stderr,
	}, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if stdout.Len() > 0 {
		t.Errorf("stdout should be empty, got:\n%s", stdout.String())
	}

	if stderr.Len() > 0 {
		t.Errorf("stderr should be empty, got:\n%s", stderr.String())
	}
}

func TestNewEdit_EditorParsing(t *testing.T) {
	tests := []struct {
		name   string
		EDITOR string // environment variable
		e      string // -e argument
		editor string // --editor argument

		want string
	}{
		{
			name: "no arguments or environment",
			want: "vim",
		},
		{
			name:   "set by environment",
			EDITOR: "nvim",
			want:   "nvim",
		},
		{
			name: "set by -e",
			e:    "macvim",
			want: "macvim",
		},
		{
			name:   "set by --editor",
			editor: "gvim",
			want:   "gvim",
		},
		{
			name:   "environment override with -e",
			EDITOR: "nano",
			e:      "vim",
			want:   "vim",
		},
		{
			name:   "environment override with --editor",
			EDITOR: "vim",
			editor: "nvim",
			want:   "nvim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := make(map[string]string)

			if e := tt.EDITOR; len(e) > 0 {
				env["EDITOR"] = e
			}

			var args []string
			if len(tt.e) > 0 {
				args = append(args, "-e", tt.e)
			}
			if len(tt.editor) > 0 {
				args = append(args, "--editor", tt.editor)
			}
			args = append(args, "file")

			var stdout, stderr bytes.Buffer
			got, err := newEdit(&options{
				Stdout: &stdout,
				Stderr: &stderr,
				Getenv: func(k string) string { return env[k] },
			}, args)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if got.Editor != tt.want {
				t.Errorf("unexpected editor %q, want %q", got.Editor, tt.want)
			}

			if stdout.Len() > 0 {
				t.Errorf("stdout should be empty, got:\n%s", stdout.String())
			}

			if stderr.Len() > 0 {
				t.Errorf("stderr should be empty, got:\n%s", stderr.String())
			}
		})
	}
}

func TestNewEdit_FileParsing(t *testing.T) {
	tests := []struct {
		name string
		args []string

		want    string
		wantErr error
	}{
		{
			name: "path specified",
			args: []string{"foo"},
			want: "foo",
		},
		{
			name:    "no path specified",
			args:    []string{"-e", "foo"},
			wantErr: errNoFileSpecified,
		},
		{
			name:    "too many paths",
			args:    []string{"foo", "bar"},
			wantErr: errTooManyArguments{Got: 2, Want: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newEdit(&options{
				Stdout: new(bytes.Buffer),
				Stderr: new(bytes.Buffer),
				Getenv: func(string) string { return "" },
			}, tt.args)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("unexpected error: %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.want != got.Path {
				t.Errorf("unexpected path %q, want %q", got.Path, tt.want)
			}
		})
	}
}
