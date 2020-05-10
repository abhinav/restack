package ostest

import "os"

// Chdir changes the working directory to the given directory for the duration
// of the current test.
//
// The old working directory is restored when the test exits.
func Chdir(t T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %q: %v", dir, err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("chdir old %q: %v", oldDir)
		}
	})
}
