package testwriter

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/abhinav/restack/internal/test"
	"github.com/stretchr/testify/assert"
)

type fakeT struct {
	test.T

	buff bytes.Buffer
}

func (t *fakeT) Helper() {}

func (t *fakeT) Logf(msg string, args ...interface{}) {
	fmt.Fprintf(&t.buff, msg, args...)
}

func TestWriter(t *testing.T) {
	var ft fakeT
	w := New(&ft)

	fmt.Fprintf(w, "foo\nbar\nbaz\n")
	fmt.Fprintln(w, "qux")

	want := "foo\nbar\nbaz\nqux\n"
	assert.Equal(t, want, ft.buff.String(), "logged output mismatch")
}
