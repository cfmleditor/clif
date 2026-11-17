//go:build linux

package flock

import (
	"os"
	"strconv"
	"strings"
)

// startTime is when pid started, in clock ticks since boot (/proc/<pid>/stat,
// field 22): enough to tell a process from a later one given its PID.
func startTime(pid int) (string, bool) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat") //nolint:gosec // a /proc path built from a PID
	if err != nil {
		return "", false
	}

	// The command name, field 2, is in parentheses and may hold spaces: the
	// fields are counted from after its closing one.
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return "", false
	}

	f := strings.Fields(s[i+1:])
	if len(f) < 20 { // field 22 is the 20th after the name
		return "", false
	}

	return f[19], true
}
