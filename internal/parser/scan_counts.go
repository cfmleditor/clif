package parser

import "sync/atomic"

// scanCounts are the tokens the scanner produced and the parses run, counted
// only in a build with the passcount tag (countScans), so that a test can
// hold a batch scan to how many times it reads its files. In any other
// build countScans is false and the counting compiles away: an atomic add on
// every token contends across a scan's workers, and showed as a third more
// CPU when the counters were measured by hand.
var scanCounts struct{ tokens, parses atomic.Int64 }

// ScanCounts is the tokens scanned and the parses run since the last
// ResetScanCounts; both zero unless built with the passcount tag.
func ScanCounts() (tokens, parses int64) {
	return scanCounts.tokens.Load(), scanCounts.parses.Load()
}

// ResetScanCounts sets ScanCounts back to zero.
func ResetScanCounts() {
	scanCounts.tokens.Store(0)
	scanCounts.parses.Store(0)
}

// CountsScans is whether this build counts (the passcount tag).
func CountsScans() bool { return countScans }
