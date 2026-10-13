//go:build !unix && !wasm

package daemon

import (
	"net"
	"os"
)

// prepareSocketDir creates the socket's directory. On Windows it is under
// LOCALAPPDATA, which only its user can reach, so there is nothing to check.
func prepareSocketDir(dir string) error {
	return os.MkdirAll(dir, 0o700)
}

func checkSocket(string) error { return nil }

func peerAllowed(net.Conn) error { return nil }
