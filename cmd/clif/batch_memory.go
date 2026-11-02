package main

import (
	"os"
	"runtime/debug"

	"github.com/cfmleditor/clif/internal/vfs"
)

// A subcommand sizes its memory from the machine's (systemMemory), so a
// large workspace runs at full speed where memory is plentiful and gives some
// speed back where it is not. The editor's server keeps Go's GC defaults, its
// heap being small and its latency what matters; its batch file cache, used
// by the unresolved export, is capped at 128MB (limitLongLivedCaches).
//
// The garbage collector keeps Go's default target, under which the heap grows
// to twice what is live before a collection, and gets a soft limit of three
// quarters of the machine's memory: only a run nearing it collects sooner.
// The limit is there to keep a run inside the machine, not to make it small:
// a quarter put a 2GB container's limit at 512MB, under the 600MB a tassweb
// scan holds live, where Go collects without pause. A fixed lower target was
// measured on tassweb's `unresolved` scan (GOGC=75: 190MB less resident
// memory for 3% more time; GOGC=50: 300MB for 10%), and is a cost worth
// paying only where the memory is short. GOGC or GOMEMLIMIT, when set, say
// what the user wants instead.
//
// The batch file cache is sized the same way (vfs.SetContentBudget): a
// sixty-fourth of the machine's memory, from 64MB to 256MB, so 16GB and up
// holds the whole of tassweb's 141MB of source.
const (
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
		debug.SetMemoryLimit(int64(min(total/4*3, 1<<62)))
	}

	vfs.SetContentBudget(int64(min(max(total/contentShare, minContentBudget), maxContentBudget)))
}
