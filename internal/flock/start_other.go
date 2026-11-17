//go:build !linux && !darwin

package flock

// startTime is not read here: a PID is taken as its holder's while a process
// has it.
func startTime(int) (string, bool) { return "", false }
