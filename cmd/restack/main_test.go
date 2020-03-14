package main

import (
	"bytes"
	"testing"

	"github.com/abhinav/restack/internal/testutil"
)

func TestRun_NoArgs(t *testing.T) {
	var stderr bytes.Buffer
	err := run(
		nil,               /* args */
		new(bytes.Buffer), /* stdin */
		testutil.NewWriter(t),
		&stderr,
	)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if stderr.Len() == 0 {
		t.Errorf("stderr should contain usage")
	}
}
