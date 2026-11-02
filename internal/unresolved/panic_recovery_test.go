package unresolved

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

// panicReadFS panics reading one file, as a bad file might on its way
// through a scan.
type panicReadFS struct {
	vfs.FS
	bad string
}

func (p panicReadFS) ReadFile(path string) ([]byte, error) {
	if filepath.Base(path) == p.bad {
		panic("a bad file")
	}

	return p.FS.ReadFile(path)
}

func (p panicReadFS) ReadFileString(path string) (string, error) {
	data, err := p.ReadFile(path)

	return string(data), err
}

// A panic on one file ends neither the scan nor the process it runs in (the
// editor's export runs a scan inside the language server), and the file is
// reported as one the scan could not check rather than left out.
func TestAScanSurvivesAPanicOnOneFile(t *testing.T) {
	dir := t.TempDir()
	good, bad := filepath.Join(dir, "good.cfm"), filepath.Join(dir, "bad.cfm")

	for _, f := range []string{good, bad} {
		if err := os.WriteFile(f, []byte(`<cfset x = 1>`), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	rep := Scan(panicReadFS{FS: vfs.OS{}, bad: "bad.cfm"}, []string{good, bad}, nil, &Options{})

	var failed []Call

	for _, c := range rep.Calls {
		if strings.HasPrefix(c.Reason, ScanFailedReason) {
			failed = append(failed, c)
		}
	}

	if len(failed) != 1 || failed[0].File != bad {
		t.Fatalf("entries for a failed scan = %v, want one for bad.cfm", failed)
	}
}
