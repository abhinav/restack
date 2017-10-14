package restack

import (
	"io"
	"io/ioutil"
	"os"
)

// FS provides access to the filesystem.
type FS interface {
	// Returns true if a file exists at the provided path.
	FileExists(path string) bool

	// MkdirAll creates a directory at the provided path and any needed parent
	// directories.
	//
	// See os.MkdirAll.
	MkdirAll(path string) error

	// Starts reading the file at the provided path.
	//
	// See os.Open.
	ReadFile(path string) (io.ReadCloser, error)

	// Starts writing to the file at the given path, truncating it if it
	// already exists.
	//
	// See os.Create.
	WriteFile(path string) (io.WriteCloser, error)

	// Start writing to the file at the given path, truncating it if it
	// already exists. The file will be created with permissions 0755.
	//
	// See os.OpenFile.
	WriteExecutableFile(path string) (io.WriteCloser, error)

	// Renames old to new.
	//
	// See os.Rename.
	Rename(old, new string) error

	// Creates a temporary directory somewhere on the system and returns an
	// absolute path to it.
	//
	// It's the caller's responsibility to clean this directory up.
	TempDir(prefix string) (string, error)

	// RemoveAll removes the file/directory at the given path and if it's a
	// directory, all its descendants.
	//
	// See os.RemoveAll.
	RemoveAll(path string) error
}

// DefaultFilesystem is the real underlying filesystem.
var DefaultFilesystem FS = defaultFS{}

type defaultFS struct{}

func (defaultFS) MkdirAll(path string) error {
	return os.MkdirAll(path, 0777)
}

func (defaultFS) FileExists(path string) bool {
	info, err := os.Stat(path)
	return !os.IsNotExist(err) && !info.IsDir()
}

func (defaultFS) ReadFile(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	return f, err
}

func (defaultFS) WriteFile(path string) (io.WriteCloser, error) {
	f, err := os.Create(path)
	return f, err
}

func (defaultFS) WriteExecutableFile(path string) (io.WriteCloser, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	return f, err
}

func (defaultFS) Rename(old, new string) error {
	return os.Rename(old, new)
}

func (defaultFS) TempDir(prefix string) (string, error) {
	return ioutil.TempDir("", prefix)
}

func (defaultFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}
