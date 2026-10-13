//go:build unix

package config

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// The lookups Untrusted makes, as variables so a test can play another owner
// without running as root.
var (
	currentUID = os.Getuid
	currentGID = os.Getgid

	fileOwner = func(info fs.FileInfo) (uid, gid int64, ok bool) {
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return 0, 0, false
		}

		return int64(st.Uid), int64(st.Gid), true
	}

	// privateGroup reports whether gid is the current user's own group, the
	// one-member group named after the user that Linux distributions give
	// each account (user private groups). Their default umask is 002, so a
	// checkout there is group-writable by nobody but its owner.
	privateGroup = func(gid int64) bool {
		u, err := user.Current()
		if err != nil {
			return false
		}

		g, err := user.LookupGroupId(strconv.FormatInt(gid, 10))

		return err == nil && g.Name == u.Username
	}
)

// Untrusted says why the config file info describes must not be read, or ""
// when it may: it must be owned by the current user or by root, and no other
// user may be able to write it. A group-writable file passes only when the
// group is the user's private group. A file whose owner the filesystem does
// not report (an editor's virtual filesystem) is not checked.
func Untrusted(info fs.FileInfo) string {
	uid, gid, ok := fileOwner(info)
	if !ok {
		return ""
	}

	me := int64(currentUID())
	if uid != me && uid != 0 {
		return fmt.Sprintf("owned by uid %d, not by the current user (uid %d) or root", uid, me)
	}

	perm := info.Mode().Perm()
	if perm&0o002 != 0 {
		return fmt.Sprintf("writable by other users (mode %04o)", perm)
	}

	if perm&0o020 != 0 && (gid != int64(currentGID()) || !privateGroup(gid)) {
		return fmt.Sprintf("writable by its group, gid %d (mode %04o)", gid, perm)
	}

	return ""
}
