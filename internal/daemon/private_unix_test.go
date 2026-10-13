//go:build unix

package daemon

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/index"
	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/server"
)

// Without XDG_RUNTIME_DIR the socket's directory is named for the user, so
// users sharing /tmp do not share one directory whoever made it first owns.
func TestSocketDirIsPerUserWithoutARuntimeDir(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")

	want := "clif-" + strconv.Itoa(os.Getuid())
	if got := filepath.Base(socketDir()); got != want {
		t.Errorf("socketDir() = %q, want a directory named %q", socketDir(), want)
	}
}

// A directory this user owns but left open is closed; a symlink and a
// directory someone else owns are refused.
func TestSocketDirMustBePrivate(t *testing.T) {
	base := filepath.Dir(shortSock(t))

	open := filepath.Join(base, "open")
	if err := os.Mkdir(open, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(open, 0o777); err != nil {
		t.Fatal(err)
	}

	if err := prepareSocketDir(open); err != nil {
		t.Fatalf("prepareSocketDir on this user's open directory = %v, want it closed", err)
	}

	if info, err := os.Stat(open); err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("directory left at %v (%v), want 0700", info.Mode().Perm(), err)
	}

	link := filepath.Join(base, "link")
	if err := os.Symlink(open, link); err != nil {
		t.Fatal(err)
	}

	if err := prepareSocketDir(link); err == nil {
		t.Error("prepareSocketDir accepted a symlink, which another user could have pointed anywhere")
	}

	if os.Getuid() != 0 {
		if err := prepareSocketDir("/"); err == nil || !strings.Contains(err.Error(), "belongs to uid") {
			t.Errorf("prepareSocketDir on a directory root owns = %v, want it refused", err)
		}
	}
}

// Proxy does not dial a socket in a directory others can reach: whoever could
// put it there would be answering the editor.
func TestProxyRefusesASocketOthersCouldHavePlaced(t *testing.T) {
	sock := shortSock(t)

	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", sock)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = ln.Close() }()

	dialled := make(chan struct{}, 1)

	go func() {
		if c, err := ln.Accept(); err == nil {
			dialled <- struct{}{}

			_ = c.Close()
		}
	}()

	if err := os.Chmod(filepath.Dir(sock), 0o777); err != nil {
		t.Fatal(err)
	}

	if err := Proxy(sock); err == nil {
		t.Fatal("Proxy dialled a socket in a world-writable directory")
	}

	// Not a socket at all.
	notSock := filepath.Join(filepath.Dir(shortSock(t)), "plain")
	if err := os.WriteFile(notSock, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Proxy(notSock); err == nil || !strings.Contains(err.Error(), "not a socket") {
		t.Errorf("Proxy on a plain file = %v, want it refused", err)
	}

	select {
	case <-dialled:
		t.Error("the refused socket received a connection")
	case <-time.After(50 * time.Millisecond):
	}
}

// The daemon's socket is the user's alone, and so is its directory, whatever
// the umask.
func TestServeMakesItsSocketPrivate(t *testing.T) {
	sock := shortSock(t)
	if err := os.Chmod(filepath.Dir(sock), 0o755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	settings := server.Settings{}
	done := make(chan error, 1)

	go func() { done <- Serve(ctx, sock, cflog.NewLogger(false), index.New(), nil, &settings) }()

	waitForSocket(t, sock)

	if info, err := os.Stat(sock); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("socket mode = %v (%v), want 0600", info.Mode().Perm(), err)
	}

	if info, err := os.Stat(filepath.Dir(sock)); err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("socket directory mode = %v (%v), want 0700", info.Mode().Perm(), err)
	}

	cancel()

	if err := <-done; err != nil {
		t.Errorf("Serve = %v", err)
	}
}

// The daemon can tell who is connecting, which is what lets it refuse anyone
// but its own user.
func TestPeerCredentialsNameThisUser(t *testing.T) {
	sock := shortSock(t)

	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", sock)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = ln.Close() }()

	go func() {
		if c, err := (&net.Dialer{}).DialContext(t.Context(), "unix", sock); err == nil {
			time.Sleep(100 * time.Millisecond)

			_ = c.Close()
		}
	}()

	c, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = c.Close() }()

	uid, err := peerUID(c)
	if errors.Is(err, errNoPeerCredentials) {
		t.Skip("no peer credentials on this platform")
	}

	if err != nil || uid != os.Getuid() {
		t.Errorf("peerUID = %d, %v; want %d", uid, err, os.Getuid())
	}

	if err := peerAllowed(c); err != nil {
		t.Errorf("peerAllowed refused this user: %v", err)
	}
}
