package main

import (
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

func TestBatchMemoryIsSizedFromTheMachine(t *testing.T) {
	prevLimit, prevBudget := debug.SetMemoryLimit(-1), vfs.ContentBudget()

	t.Cleanup(func() {
		debug.SetMemoryLimit(prevLimit)
		vfs.SetContentBudget(prevBudget)
	})

	// Set so that t.Setenv restores them afterwards; unset for the cases.
	t.Setenv("GOGC", "")
	t.Setenv("GOMEMLIMIT", "")

	for _, key := range []string{"GOGC", "GOMEMLIMIT"} {
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}

	for _, c := range []struct {
		total         uint64
		limit, budget int64
	}{
		{32 << 30, 8 << 30, 256 << 20},
		{8 << 30, 2 << 30, 128 << 20},
		{2 << 30, 512 << 20, 64 << 20},
	} {
		debug.SetMemoryLimit(math.MaxInt64)
		tuneBatchMemory(c.total)

		if got := debug.SetMemoryLimit(-1); got != c.limit {
			t.Errorf("%dGB: memory limit %d, want %d", c.total>>30, got, c.limit)
		}

		if got := vfs.ContentBudget(); got != c.budget {
			t.Errorf("%dGB: content budget %d, want %d", c.total>>30, got, c.budget)
		}
	}

	// Unknown memory leaves everything as it was.
	debug.SetMemoryLimit(math.MaxInt64)
	vfs.SetContentBudget(1 << 20)
	tuneBatchMemory(0)

	if debug.SetMemoryLimit(-1) != math.MaxInt64 || vfs.ContentBudget() != 1<<20 {
		t.Error("unknown memory changed the limit or the budget")
	}

	// GOGC or GOMEMLIMIT is the user's say: no limit of ours on top.
	t.Setenv("GOGC", "100")
	debug.SetMemoryLimit(math.MaxInt64)
	tuneBatchMemory(32 << 30)

	if got := debug.SetMemoryLimit(-1); got != math.MaxInt64 {
		t.Errorf("GOGC is set, so no memory limit should be set, got %d", got)
	}
}

func TestSystemMemoryIsKnownWhereItCanBeRead(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("not read on", runtime.GOOS)
	}

	if systemMemory() < 256<<20 {
		t.Errorf("systemMemory() = %d", systemMemory())
	}
}
