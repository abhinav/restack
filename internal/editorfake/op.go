package editorfake

import (
	"fmt"
	"os"

	"github.com/google/go-cmp/cmp"
)

type state struct {
	Contents string
	Path     string
	ExitCode int
}

// Option configures the behavior of a fake editor.
type Option interface {
	optionType() optionType
	run(*state) error
}

type wantContents struct {
	Contents string
}

// WantContents indicates that the editor should expect exactly the provided
// contents as input. The editor will fail with a non-zero exit code if this
// condition is not met.
func WantContents(c string) Option {
	return &wantContents{c}
}

func (*wantContents) optionType() optionType {
	return optionTypeWantContents
}

func (c *wantContents) run(s *state) error {
	if diff := cmp.Diff(c.Contents, s.Contents); len(diff) > 0 {
		return fmt.Errorf("contents mismatch: (-want, +got)\n%s", diff)
	}
	return nil
}

type giveContents struct {
	Contents string
}

// GiveContents tells the editor to replace the contents of the file with the
// given string.
func GiveContents(c string) Option {
	return &giveContents{c}
}

func (*giveContents) optionType() optionType {
	return optionTypeGiveContents
}

func (c *giveContents) run(s *state) error {
	s.Contents = c.Contents
	if err := os.WriteFile(s.Path, []byte(s.Contents), 0o644); err != nil {
		return fmt.Errorf("write %q: %v", s.Path, err)
	}
	return nil
}

type addPrefix struct {
	Prefix string
}

// AddPrefix makes the editor add a prefix to the contents of the file.
func AddPrefix(prefix string) Option {
	return &addPrefix{prefix}
}

func (*addPrefix) optionType() optionType {
	return optionTypeAddPrefix
}

func (c *addPrefix) run(s *state) error {
	s.Contents = c.Prefix + s.Contents
	if err := os.WriteFile(s.Path, []byte(s.Contents), 0o644); err != nil {
		return fmt.Errorf("write %q: %v", s.Path, err)
	}
	return nil
}

type deleteFile struct{}

// DeleteFile informs the editor to delete the file instead of writing to it.
func DeleteFile() Option {
	return &deleteFile{}
}

func (*deleteFile) optionType() optionType {
	return optionTypeDeleteFile
}

func (c *deleteFile) run(s *state) error {
	if err := os.Remove(s.Path); err != nil {
		return fmt.Errorf("rm %q: %v", s.Path, err)
	}
	return nil
}

type exitCode struct {
	Code int
}

// ExitCode configures the exit code of the editor. This is ignored if any of
// the prior operations failed.
func ExitCode(code int) Option {
	return &exitCode{code}
}

func (*exitCode) optionType() optionType {
	return optionTypeExitCode
}

func (c *exitCode) run(s *state) error {
	s.ExitCode = c.Code
	return nil
}
