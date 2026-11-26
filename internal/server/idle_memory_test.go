package server

import (
	"testing"
	"time"
)

// heapGarbage keeps the memory the test allocates reachable until it is let go.
var heapGarbage [][]byte

// Memory freed during a burst of work is returned once the server goes idle,
// and not while messages keep arriving.
func TestIdleServerReturnsFreedMemory(t *testing.T) {
	old := idleReleaseAfter

	// Wide next to the 60 ms between messages below, so a loaded machine that
	// delays a sleep several times over does not look idle.
	idleReleaseAfter = 250 * time.Millisecond

	t.Cleanup(func() { idleReleaseAfter = old })

	idleRelease.Lock()
	idleRelease.timer, idleRelease.releases = nil, 0
	idleRelease.Unlock()

	// Free well over idleReleaseMin, and all of it at once: five blocks held
	// together and then let go. Allocated and dropped one at a time, the runtime
	// reuses the same space for each, and what the heap holds when the release
	// looks (it counts the heap less what was returned) came to 34 to 98 MB
	// depending on timing, on either side of the 64 MB minimum, so the test
	// passed or failed with the load on the machine.
	for range 5 {
		b := make([]byte, 32<<20)
		b[0] = 1
		heapGarbage = append(heapGarbage, b)
	}

	heapGarbage = nil

	// Messages go on for longer than the idle window, each well inside it (60 ms
	// against 250): a countdown that a message did not restart would run out
	// during them.
	for range 8 {
		noteActivity()
		time.Sleep(60 * time.Millisecond)
	}

	releases := func() int {
		idleRelease.Lock()
		defer idleRelease.Unlock()

		return idleRelease.releases
	}

	if busy := releases(); busy != 0 {
		t.Fatalf("released %d times while messages were arriving", busy)
	}

	// Once idle: waited for, not slept past. The release forces a collection,
	// which takes as long as the machine lets it.
	deadline := time.Now().Add(10 * time.Second)

	for releases() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if idle := releases(); idle != 1 {
		t.Fatalf("released %d times once idle, want 1", idle)
	}

	// And once for the stretch of idleness, not again while it goes on.
	time.Sleep(2 * idleReleaseAfter)

	if again := releases(); again != 1 {
		t.Errorf("released %d times over one idle stretch, want 1", again)
	}
}
