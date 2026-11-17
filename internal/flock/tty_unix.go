//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package flock

import (
	"os"

	"golang.org/x/sys/unix"
)

// InForeground reports whether this process's group is its terminal's
// foreground group: then a Ctrl-C or hang-up it gets came from the terminal,
// which sent it to that whole group, its children included (or, rarely, from
// a kill aimed at it alone, which cannot be told apart). Without a terminal,
// or in the background, no terminal sent it.
func InForeground() bool {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return false
	}

	defer func() { _ = tty.Close() }()

	fg, err := unix.IoctlGetInt(int(tty.Fd()), unix.TIOCGPGRP)

	return err == nil && fg == unix.Getpgrp()
}
