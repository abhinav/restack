package main

import (
	"bytes"
	"testing"

	"github.com/abhinav/restack/internal/testwriter"
	"github.com/stretchr/testify/assert"
)

func TestRun_Version(t *testing.T) {
	var stdout bytes.Buffer
	opts := options{
		Stdout: &stdout,
		Stderr: testwriter.New(t),
	}
	assert.NoError(t, run(&opts, []string{"-version"}))
	assert.NotEmpty(t, stdout.String(),
		"stdout should contain version information")
}

func TestRun_CommandErrors(t *testing.T) {
	tests := []struct {
		name string
		give []string
		want string
	}{
		{
			name: "no arguments",
			want: "no command specified",
		},
		{
			name: "unknown command",
			give: []string{"foo"},
			want: `unrecognized command "foo"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			opts := &options{
				Stdout: testwriter.New(t),
				Stderr: &stderr,
			}

			assert.ErrorContains(t, run(opts, tt.give), tt.want)
			assert.NotEmpty(t, stderr.String(),
				"stderr should contain usage")
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
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestNewSetup(t *testing.T) {
	var stdout, stderr bytes.Buffer

	_, err := newSetup(&options{
		Stdout: &stdout,
		Stderr: &stderr,
	}, nil)
	assert.NoError(t, err)
	assert.Empty(t, stdout.String(), "stdout should be empty")
	assert.Empty(t, stderr.String(), "stderr should be empty")
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
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got.Editor, "unexpected editor")
			assert.Empty(t, stdout.String(), "stdout should be empty")
			assert.Empty(t, stderr.String(), "stderr should be empty")
		})
	}
}

func TestNewEdit_FileParsing(t *testing.T) {
	tests := []struct {
		name string
		args []string

		want    string
		wantErr string
	}{
		{
			name: "path specified",
			args: []string{"foo"},
			want: "foo",
		},
		{
			name:    "no path specified",
			args:    []string{"-e", "foo"},
			wantErr: "no file specified",
		},
		{
			name:    "too many paths",
			args:    []string{"foo", "bar"},
			wantErr: `too many arguments: ["foo" "bar"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newEdit(&options{
				Stdout: new(bytes.Buffer),
				Stderr: new(bytes.Buffer),
				Getenv: func(string) string { return "" },
			}, tt.args)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got.Path, "unexpected path")
		})
	}
}
