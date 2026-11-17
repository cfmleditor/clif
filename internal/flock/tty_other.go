//go:build unix && !(darwin || dragonfly || freebsd || linux || netbsd || openbsd)

package flock

// inForeground: the terminal is not handed over here (AIX, Solaris), so the
// command's group stays in the background.
func inForeground() bool { return false }

func takeForeground() {}
