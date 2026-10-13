//go:build unix

package vfs

import "syscall"

// openFlags keeps opening a FIFO from waiting for a writer.
const openFlags = syscall.O_NONBLOCK
