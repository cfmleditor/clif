package log

import (
	"io"
	"os"
	"strings"
	"testing"
)

// With no logger and no crash file, as in a command-line run, a recovered
// panic is still reported: on stderr, naming the file.
func TestARecoveredPanicIsReportedWithNowhereElseToGo(t *testing.T) {
	crashMu.Lock()
	held := crashFile
	crashFile = nil
	crashMu.Unlock()

	t.Cleanup(func() {
		crashMu.Lock()
		crashFile = held
		crashMu.Unlock()
	})

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	stderr := os.Stderr
	os.Stderr = write

	t.Cleanup(func() { os.Stderr = stderr })

	Recovered(nil, "workspace file scan panic", "a bad file", String("file", "/x/bad.cfm"))

	os.Stderr = stderr

	if err := write.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"workspace file scan panic", "file=/x/bad.cfm", "a bad file"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("stderr = %q, want it to mention %q", out, want)
		}
	}
}
