package testwriter

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type fakeT struct {
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
	if diff := cmp.Diff(want, ft.buff.String()); len(diff) > 0 {
		t.Errorf("logged output mismatch: (-want, +got):\n%s", diff)
	}

}
