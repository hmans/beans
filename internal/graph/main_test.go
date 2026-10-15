package graph

import (
	"os"
	"testing"

	"github.com/hmans/beans/internal/gittest"
)

func TestMain(m *testing.M) {
	gittest.Isolate()
	os.Exit(m.Run())
}
