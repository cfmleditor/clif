//go:build darwin

package flock

import (
	"strconv"

	"golang.org/x/sys/unix"
)

// startTime is when pid started (sysctl kern.proc.pid), to the microsecond:
// enough to tell a process from a later one given its PID.
func startTime(pid int) (string, bool) {
	p, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil || int(p.Proc.P_pid) != pid {
		return "", false
	}

	t := p.Proc.P_starttime

	return strconv.FormatInt(t.Sec, 10) + "." + strconv.FormatInt(int64(t.Usec), 10), true
}
