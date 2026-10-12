package server

import (
	"testing"
	"time"
)

// Memory freed during a burst of work is returned once the server goes idle,
// and not while messages keep arriving.
func TestIdleServerReturnsFreedMemory(t *testing.T) {
	old := idleReleaseAfter
	idleReleaseAfter = 50 * time.Millisecond

	t.Cleanup(func() { idleReleaseAfter = old })

	idleRelease.Lock()
	idleRelease.timer, idleRelease.releases = nil, 0
	idleRelease.Unlock()

	// Free well over idleReleaseMin.
	for range 8 {
		b := make([]byte, 32<<20)
		b[0] = 1
	}

	for range 5 {
		noteActivity()
		time.Sleep(20 * time.Millisecond)
	}

	idleRelease.Lock()
	busy := idleRelease.releases
	idleRelease.Unlock()

	if busy != 0 {
		t.Fatalf("released %d times while messages were arriving", busy)
	}

	time.Sleep(200 * time.Millisecond)

	idleRelease.Lock()
	idle := idleRelease.releases
	idleRelease.Unlock()

	if idle != 1 {
		t.Errorf("released %d times once idle, want 1", idle)
	}
}
