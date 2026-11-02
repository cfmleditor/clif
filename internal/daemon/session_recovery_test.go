package daemon

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/index"
	cflog "github.com/cfmleditor/clif/internal/log"
)

// A panic setting up one editor's session ends that session, not the daemon
// every other editor is connected to. A nil Settings panics in Apply.
func TestADaemonSessionPanicEndsOnlyThatSession(t *testing.T) {
	sock := shortSock(t)

	ctx := t.Context()

	go func() { _ = Serve(ctx, sock, cflog.NewLogger(false), index.New(), nil, nil) }()

	waitForSocket(t, sock)

	for i := range 2 {
		c, err := (&net.Dialer{}).DialContext(ctx, "unix", sock)
		if err != nil {
			t.Fatalf("connection %d refused: the daemon did not survive the first session's panic: %v", i, err)
		}

		// The session panics and its connection is closed: a read ends.
		_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))

		if _, err := io.ReadAll(c); err != nil {
			t.Fatalf("connection %d: %v", i, err)
		}

		_ = c.Close()
	}
}
