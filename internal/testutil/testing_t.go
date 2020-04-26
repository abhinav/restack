package testutil

// TestingT is a subset of the testing.T interface.
type TestingT interface {
	Name() string

	Helper()

	Logf(string, ...interface{})
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})

	Cleanup(func())
}
