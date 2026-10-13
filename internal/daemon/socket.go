//go:build !wasm

package daemon

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// SocketPath returns the Unix socket path for this project's daemon.
func (c *Config) SocketPath() string {
	h := sha256.Sum256([]byte(c.Name))
	name := fmt.Sprintf("cfmleditor-%x.sock", h[:8])

	return filepath.Join(socketDir(), name)
}

// socketDir is a directory only this user should be able to reach:
// XDG_RUNTIME_DIR or LOCALAPPDATA, which are each one user's, or else a
// directory named for the uid under the temp folder. The temp folder is
// per-user on macOS but is the shared /tmp on Linux without XDG_RUNTIME_DIR
// (an ssh session without pam_systemd, a container), where one name for every
// user let whoever created it first own every daemon's socket. The uid keeps
// users out of each other's way; prepareSocketDir and checkSocket are what
// make it safe.
func socketDir() string {
	switch runtime.GOOS {
	case "windows":
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "clif")
		}

		return filepath.Join(os.TempDir(), "clif")
	case "darwin":
	default:
		if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
			return filepath.Join(d, "clif")
		}
	}

	return filepath.Join(os.TempDir(), fmt.Sprintf("clif-%d", os.Getuid()))
}
