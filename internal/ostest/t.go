package ostest

// T is a subset of the testing.T interface.
type T interface {
	Helper()
	Cleanup(func())
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
}
