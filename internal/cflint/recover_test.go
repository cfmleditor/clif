package cflint

import (
	"strings"
	"sync"
	"testing"
)

// A panic in a lint worker becomes the run's error instead of escaping the
// goroutine, outside any request's recovery, and ending the language server.
func TestAWorkerPanicBecomesTheRunsError(t *testing.T) {
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
	)

	wg.Go(func() {
		defer recoverInto(&mu, func(err error) { firstErr = err }, "cflint scan")

		mu.Lock()
		defer mu.Unlock()

		mapReport()
	})

	wg.Wait()

	if firstErr == nil || !strings.Contains(firstErr.Error(), "a bad report") {
		t.Fatalf("firstErr = %v, want the panic", firstErr)
	}
}

// mapReport stands in for a worker's report mapping that panics.
func mapReport() { panic("a bad report") }
