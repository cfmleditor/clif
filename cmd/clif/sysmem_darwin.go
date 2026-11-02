package main

import "golang.org/x/sys/unix"

// systemMemory is the machine's physical memory in bytes, 0 when unknown.
func systemMemory() uint64 {
	n, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0
	}

	return n
}
