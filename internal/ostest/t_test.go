package ostest

import "testing"

type fakeT struct {
	*testing.T

	cleanups []func()
}

func (t *fakeT) Cleanup(f func()) {
	t.cleanups = append(t.cleanups, f)
}

func (t *fakeT) runCleanups() {
	for _, f := range t.cleanups {
		defer f()
	}
}
