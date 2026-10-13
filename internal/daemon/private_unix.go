//go:build unix

package daemon

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"syscall"
)

// prepareSocketDir creates the directory the daemon's socket goes in and
// refuses it unless only this user can reach it. Without that, on a machine
// where it falls back to a shared /tmp, another user could create the
// directory first and own every socket in it: a socket they put at the path an
// editor dials is a language server of theirs, answering that editor's
// requests and sending it edits to apply.
//
// A directory this user owns but left open is closed rather than refused: it
// is this user's to fix, and refusing would leave daemon mode off for good.
func prepareSocketDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	info, err := ownDirInfo(dir)
	if err != nil {
		return err
	}

	if info.Mode().Perm()&0o077 != 0 {
		return os.Chmod(dir, 0o700) //nolint:gosec // a directory needs its search bit
	}

	return nil
}

// checkSocket says whether the socket at path is one this user's daemon made,
// in a directory nobody else can reach, before an editor's session is handed
// to it. It only checks: a proxy that finds the directory open goes on to be
// the daemon, and prepareSocketDir closes it there.
func checkSocket(path string) error {
	info, err := ownDirInfo(dirOf(path))
	if err != nil {
		return err
	}

	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("socket directory %s is %v; only its owner may use it", dirOf(path), info.Mode().Perm())
	}

	sock, err := os.Lstat(path)
	if err != nil {
		return err
	}

	if sock.Mode().Type() != fs.ModeSocket {
		return fmt.Errorf("%s is not a socket", path)
	}

	return ownedByMe(path, sock)
}

// ownDirInfo is dir's own entry, refused unless it is a directory (a symlink
// is not, whatever it points at) owned by this user.
func ownDirInfo(dir string) (fs.FileInfo, error) {
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("socket directory %s is not a directory", dir)
	}

	return info, ownedByMe(dir, info)
}

func ownedByMe(path string, info fs.FileInfo) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot tell who owns %s", path)
	}

	if uid := int(st.Uid); uid != os.Getuid() {
		return fmt.Errorf("%s belongs to uid %d, not this user (%d)", path, uid, os.Getuid())
	}

	return nil
}

// peerAllowed says whether the process at the other end of c runs as this
// user. The private directory already keeps everyone else from the socket;
// this holds even if the directory's permissions are loosened under a running
// daemon.
func peerAllowed(c net.Conn) error {
	uid, err := peerUID(c)
	if errors.Is(err, errNoPeerCredentials) {
		return nil
	}

	if err != nil {
		return err
	}

	if uid != os.Getuid() {
		return fmt.Errorf("connection from uid %d, not this user (%d)", uid, os.Getuid())
	}

	return nil
}
