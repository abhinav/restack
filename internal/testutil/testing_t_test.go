package testutil

import "testing"

// Compile time check to verify interface compliance.
var _ TestingT = (*testing.T)(nil)
