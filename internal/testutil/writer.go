package testutil

import (
	"bytes"
	"io"
)

// Writer writes output to the given TestingT.
type Writer struct {
	t TestingT
}

var _ io.Writer = (*Writer)(nil)

// NewWriter builds a new test Writer.
func NewWriter(t TestingT) *Writer {
	return &Writer{t: t}
}

func (w *Writer) Write(b []byte) (int, error) {
	w.t.Helper()

	// t.Logf will add newlines.
	for _, line := range bytes.Split(b, []byte("\n")) {
		w.t.Logf("%s", line)
	}
	return len(b), nil
}
