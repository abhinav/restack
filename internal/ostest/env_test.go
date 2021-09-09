package ostest

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetenv(t *testing.T) {
	t.Run("previously unset", func(t *testing.T) {
		name := strconv.Itoa(int(time.Now().Unix()))

		ft := fakeT{T: t}
		Setenv(&ft, name, "FOO")

		assert.Equal(t, "FOO", os.Getenv(name),
			"envvar %q mismatch", name)

		ft.runCleanups()

		got, ok := os.LookupEnv(name)
		assert.False(t, ok, "envvar %q must be unset, got %q", name, got)
	})

	t.Run("previous value", func(t *testing.T) {
		name := strconv.Itoa(int(time.Now().Unix()))

		os.Setenv(name, "FOO")
		defer os.Unsetenv(name)

		ft := fakeT{T: t}
		Setenv(&ft, name, "BAR")

		assert.Equal(t, "BAR", os.Getenv(name),
			"envvar %q mismatch", name)

		ft.runCleanups()

		assert.Equal(t, "FOO", os.Getenv(name),
			"envvar %q mismatch after cleanup", name)
	})
}

func TestUnsetenv(t *testing.T) {
	name := strconv.Itoa(int(time.Now().Unix()))

	os.Setenv(name, "FOO")
	defer os.Unsetenv(name)

	ft := fakeT{T: t}
	Unsetenv(&ft, name)

	got, ok := os.LookupEnv(name)
	assert.False(t, ok, "envvar %q should be unset, got %q", name, got)

	ft.runCleanups()

	assert.Equal(t, "FOO", os.Getenv(name),
		"envvar %q mismatch after cleanup", name)
}
