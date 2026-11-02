package main

import (
	"os"
	"runtime/debug"

	"github.com/cfmleditor/clif/internal/vfs"
)

// A subcommand sizes its memory from the machine's (systemMemory), so a
// large workspace runs at full speed where memory is plentiful and gives some
// speed back where it is not. The editor's server is not tuned: its heap is
// small and its latency is what matters.
//
// The garbage collector keeps Go's default target, under which the heap grows
// to twice what is live before a collection, and gets a soft limit of
// memoryLimitShare of the machine's memory: only a run nearing it collects
// sooner. A fixed lower target was measured on tassweb's `unresolved` scan
// (GOGC=75: 190MB less resident memory for 3% more time; GOGC=50: 300MB for
// 10%), and is a cost worth paying only where the memory is short. GOGC or
// GOMEMLIMIT, when set, say what the user wants instead.
//
// The batch file cache is sized the same way (vfs.SetContentBudget): a
// sixty-fourth of the machine's memory, from 64MB to 256MB, so 16GB and up
// holds the whole of tassweb's 141MB of source.
const (
	memoryLimitShare = 4
	contentShare     = 64
	minContentBudget = 64 << 20
	maxContentBudget = 256 << 20
)

// tuneBatchMemory applies the above for a machine with total bytes of
// memory; 0 (unknown) leaves the defaults.
func tuneBatchMemory(total uint64) {
	if total == 0 {
		return
	}

	_, gogc := os.LookupEnv("GOGC")
	_, limit := os.LookupEnv("GOMEMLIMIT")

	if !gogc && !limit {
		debug.SetMemoryLimit(int64(min(total/memoryLimitShare, 1<<62)))
	}

	vfs.SetContentBudget(int64(min(max(total/contentShare, minContentBudget), maxContentBudget)))
}
