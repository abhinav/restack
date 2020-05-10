package testwriter

import (
	"bytes"
	"io"
)

// T is a subset of the testing.T interface.
type T interface {
	Helper()
	Logf(string, ...interface{})
}

// Writer writes output to the given testing.T.
type Writer struct {
	t T
}

var _ io.Writer = (*Writer)(nil)

// New builds a new test Writer.
func New(t T) *Writer {
	return &Writer{t: t}
}

func (w *Writer) Write(b []byte) (int, error) {
	w.t.Helper()

	// Break multi-line input across multiple lines to ensure that
	// everything is decorated with test and file information.
	for _, line := range bytes.Split(b, []byte("\n")) {
		if len(line) > 0 {
			w.t.Logf("%s\n", line)
		} else {
			// For empty lines, avoid printing two newlines.
			// t.Logf splits the input and adds newlines as
			// needed.
			w.t.Logf("")
		}
	}
	return len(b), nil
}
