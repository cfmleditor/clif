package unresolved

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

func TestScanRefreshesMetadataBetweenRuns(t *testing.T) {
	dir := t.TempDir()
	caller := filepath.Join(dir, "caller.cfm")
	service := filepath.Join(dir, "Service.cfc")

	if err := os.WriteFile(caller, []byte(`<cfscript>svc = new Service(); svc.run();</cfscript>`), 0o600); err != nil {
		t.Fatal(err)
	}

	opt := &Options{WorkspaceFolders: []string{dir}}

	before := Scan(vfs.OS{}, []string{caller}, nil, opt)
	if len(before.Calls) == 0 {
		t.Fatal("missing component unexpectedly resolved")
	}

	if err := os.WriteFile(service, []byte(`component { function run() {} }`), 0o600); err != nil {
		t.Fatal(err)
	}

	after := Scan(vfs.OS{}, []string{caller, service}, nil, opt)
	if len(after.Calls) != 0 {
		t.Fatalf("new scan retained missing component: %+v", after.Calls)
	}
}
