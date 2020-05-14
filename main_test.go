package restack

import (
	"os"
	"testing"

	"github.com/abhinav/restack/internal/editorfake"
)

func TestMain(m *testing.M) {
	editorfake.TryMain()

	os.Exit(m.Run())
}
