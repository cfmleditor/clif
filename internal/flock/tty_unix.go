//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package flock

import "golang.org/x/sys/unix"

// inForeground reports whether stdin is a terminal whose foreground process
// group is this process's.
func inForeground() bool {
	fg, err := unix.IoctlGetInt(0, unix.TIOCGPGRP)

	return err == nil && fg == unix.Getpgrp()
}

// takeForeground makes this process's group the terminal's foreground.
func takeForeground() { _ = unix.IoctlSetPointerInt(0, unix.TIOCSPGRP, unix.Getpgrp()) }
