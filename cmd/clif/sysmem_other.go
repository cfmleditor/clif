//go:build !darwin && !linux && !windows

package main

// systemMemory is unknown here, so the batch defaults stand.
func systemMemory() uint64 { return 0 }
