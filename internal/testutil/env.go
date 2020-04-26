package testutil

import "os"

// Setenv changes an environment variable's value for the duration of the
// current test.
//
// It automatically restores the previous value, if any, after the test
// finishes.
func Setenv(t TestingT, k, v string) {
	t.Helper()

	var cleanup func()
	if oldv, ok := os.LookupEnv(k); ok {
		cleanup = func() { os.Setenv(k, oldv) }
	} else {
		cleanup = func() { os.Unsetenv(k) }
	}

	os.Setenv(k, v)
	t.Cleanup(cleanup)
}

// Unsetenv unsets the given environment variable for the duration of the
// current test.
//
// It automatically restores the previous value, if any, after the test
// finishes.
func Unsetenv(t TestingT, k string) {
	t.Helper()

	var cleanup func()
	if oldv, ok := os.LookupEnv(k); ok {
		cleanup = func() { os.Setenv(k, oldv) }
	}

	os.Unsetenv(k)
	if cleanup != nil {
		t.Cleanup(cleanup)
	}
}
