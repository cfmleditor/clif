//go:build !wasm

package daemon

import (
	"context"
	"io"
	"net"
	"os"
	"sync"
)

// Proxy connects to an existing daemon socket and bridges it to stdio.
// Returns nil on clean shutdown. A socket another user could have put there
// is not dialled (checkSocket): the editor's session would be theirs to
// answer.
func Proxy(sockPath string) error {
	if err := checkSocket(sockPath); err != nil {
		return err
	}

	// Proxy has no context of its own: it runs for the life of the editor's
	// connection and ends when either side closes.
	var d net.Dialer

	conn, err := d.DialContext(context.Background(), "unix", sockPath)
	if err != nil {
		return err
	}

	var once sync.Once

	closeConn := func() { once.Do(func() { _ = conn.Close() }) }
	defer closeConn()

	var wg sync.WaitGroup

	wg.Add(2)

	// stdin → socket
	go func() {
		defer wg.Done()

		_, _ = io.Copy(conn, os.Stdin)

		closeConn()
	}()

	// socket → stdout
	go func() {
		defer wg.Done()

		_, _ = io.Copy(os.Stdout, conn)
		// Daemon closed the socket; close stdin to unblock the other goroutine
		_ = os.Stdin.Close()
	}()

	wg.Wait()

	return nil
}
