package main

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// systemMemory is the memory this process may use in bytes, 0 when unknown:
// the machine's, or a container's cgroup limit when that is lower, since
// sysinfo reports the host's memory inside a container.
func systemMemory() uint64 {
	var info unix.Sysinfo_t
	if err := unix.Sysinfo(&info); err != nil {
		return 0
	}

	total := uint64(info.Totalram) * uint64(info.Unit)

	for _, path := range []string{"/sys/fs/cgroup/memory.max", "/sys/fs/cgroup/memory/memory.limit_in_bytes"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		if limit, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil && limit > 0 && limit < total {
			total = limit
		}
	}

	return total
}
