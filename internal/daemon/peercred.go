//go:build unix

package daemon

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"syscall"
)

// errNoPeerCredentials is a platform that cannot say who is connected.
var errNoPeerCredentials = errors.New("peer credentials not available on this platform")

func rawConn(c net.Conn) (syscall.RawConn, error) {
	sc, ok := c.(syscall.Conn)
	if !ok {
		return nil, fmt.Errorf("%T is not a socket", c)
	}

	return sc.SyscallConn()
}

func dirOf(path string) string { return filepath.Dir(path) }
