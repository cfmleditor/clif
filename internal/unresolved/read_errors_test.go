package unresolved

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

func TestScanReportsReadFailuresSeparatelyFromFindings(t *testing.T) {
	dir := t.TempDir()

	good, bad := filepath.Join(dir, "good.cfm"), filepath.Join(dir, "bad.cfm")
	if err := os.WriteFile(good, []byte("<cfset result = missingService.run()>"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(bad, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Truncate(bad, vfs.MaxFileSize+1); err != nil {
		t.Fatal(err)
	}

	for name, opt := range map[string]Options{
		"strict": {}, "speculative": {Speculative: true}, "open-world": {AssumeOpenWorld: true},
	} {
		t.Run(name, func(t *testing.T) {
			rep := Scan(vfs.OS{}, []string{good, bad}, nil, &opt)
			if !errors.Is(rep.Err, vfs.ErrTooLarge) || !strings.Contains(rep.Err.Error(), bad) {
				t.Fatalf("scan error = %v, want read failure naming %s", rep.Err, bad)
			}

			if rep.Scanned != 2 || rep.Skipped != 1 {
				t.Errorf("scanned %d, skipped %d; want 2 attempted and 1 unreadable", rep.Scanned, rep.Skipped)
			}

			if len(rep.Calls)+len(rep.AssumedCalls) != 1 {
				t.Errorf("calls = %+v, assumptions = %+v; want the readable file's finding or assumption", rep.Calls, rep.AssumedCalls)
			}
		})
	}
}
