package server

import (
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// Freed memory goes back to the OS slowly on its own, so after a stretch of
// editing a daemon showed its peaks for long after: over 1,500 tassweb files
// opened, edited and closed, resident size rose to 330 MB while the live heap
// was 34 MB. Once no message has arrived for idleReleaseAfter, the freed
// memory the runtime still holds is returned in one go, as indexWorkspace does
// after indexing. That forces a collection, so it runs only when idle, once
// per idle stretch, and only when the heap holds at least idleReleaseMin; the release is the process's, which is
// right for a daemon serving several sessions.
var (
	idleReleaseAfter = 5 * time.Second

	idleRelease struct {
		sync.Mutex
		timer    *time.Timer
		releases int // for tests
	}
)

const idleReleaseMin = 64 << 20

// noteActivity restarts the idle countdown; every message calls it.
func noteActivity() {
	idleRelease.Lock()
	defer idleRelease.Unlock()

	if idleRelease.timer == nil {
		idleRelease.timer = time.AfterFunc(idleReleaseAfter, releaseIdleMemory)

		return
	}

	idleRelease.timer.Reset(idleReleaseAfter)
}

func releaseIdleMemory() {
	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	// What the heap holds from the OS: what a collection then frees is
	// returned, and memory freed since the last collection is still counted
	// as in use until one runs, so the idle spans alone would miss it.
	if m.HeapSys-m.HeapReleased < idleReleaseMin {
		return
	}

	debug.FreeOSMemory()

	idleRelease.Lock()
	idleRelease.releases++
	idleRelease.Unlock()
}
