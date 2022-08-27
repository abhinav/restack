package test

// T is a subset of the testing.T interface.
type T interface {
	Cleanup(func())
	Errorf(string, ...interface{})
	FailNow()
	Helper()
	Logf(string, ...interface{})
	TempDir() string
}
