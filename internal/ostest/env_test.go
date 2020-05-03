package ostest

import (
	"os"
	"strconv"
	"testing"
	"time"
)

func TestSetenv(t *testing.T) {
	t.Run("previously unset", func(t *testing.T) {
		name := strconv.Itoa(int(time.Now().Unix()))

		ft := fakeT{T: t}
		Setenv(&ft, name, "FOO")

		if got := os.Getenv(name); got != "FOO" {
			t.Errorf("envvar mismatch %q=%q, want %q", name, got, "FOO")
		}

		ft.runCleanups()

		if got, ok := os.LookupEnv(name); ok || len(got) > 0 {
			t.Errorf("envvar should be unset, got %q=%q", name, got)
		}
	})

	t.Run("previous value", func(t *testing.T) {
		name := strconv.Itoa(int(time.Now().Unix()))

		os.Setenv(name, "FOO")
		defer os.Unsetenv(name)

		ft := fakeT{T: t}
		Setenv(&ft, name, "BAR")

		if got := os.Getenv(name); got != "BAR" {
			t.Errorf("envvar mismatch %q=%q, want %q", name, got, "BAR")
		}

		ft.runCleanups()

		if got := os.Getenv(name); got != "FOO" {
			t.Errorf("envvar mismatch %q=%q, want %q", name, got, "FOO")
		}
	})
}

func TestUnsetenv(t *testing.T) {
	name := strconv.Itoa(int(time.Now().Unix()))

	os.Setenv(name, "FOO")
	defer os.Unsetenv(name)

	ft := fakeT{T: t}
	Unsetenv(&ft, name)

	if got, ok := os.LookupEnv(name); ok || len(got) > 0 {
		t.Errorf("envvar should be unset, got %q=%q", name, got)
	}

	ft.runCleanups()

	if got := os.Getenv(name); got != "FOO" {
		t.Errorf("envvar mismatch %q=%q, want %q", name, got, "FOO")
	}
}
