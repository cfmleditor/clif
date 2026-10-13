//go:build unix && !linux && !darwin

package daemon

import "net"

// peerUID has no implementation here; the socket's private directory is the
// whole of the check on these platforms, none of which clif releases for.
func peerUID(net.Conn) (int, error) { return 0, errNoPeerCredentials }
