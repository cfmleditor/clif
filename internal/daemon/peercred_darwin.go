package daemon

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// peerUID is the uid of the process at the other end of a Unix socket.
func peerUID(c net.Conn) (int, error) {
	raw, err := rawConn(c)
	if err != nil {
		return 0, err
	}

	var (
		cred    *unix.Xucred
		credErr error
	)

	err = raw.Control(func(fd uintptr) {
		cred, credErr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	})
	if err != nil {
		return 0, err
	}

	if credErr != nil {
		return 0, fmt.Errorf("reading the peer's credentials: %w", credErr)
	}

	return int(cred.Uid), nil
}
