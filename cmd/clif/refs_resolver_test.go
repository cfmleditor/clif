package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/refs"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestRefsVerifiesAHitAgainstItsComponent: `clif refs` searched with the
// config's resolvers alone, so any call named `save` was a reference to `save`.
// The editor's find-references checked each against the component the call is
// made on, and the two answered one question differently. Both go through the
// workspace's resolver now: a call on a component that does not define the
// function is not a reference to it.
func TestRefsVerifiesAHitAgainstItsComponent(t *testing.T) {
	dir := t.TempDir()

	for name, src := range map[string]string{
		"Repo.cfc":  `component { function save(){} }`,
		"Other.cfc": `component { function load(){} }`,
		"Page.cfm": `<cfset repo = new Repo()>
<cfset other = new Other()>
<cfset repo.save()>
<cfset other.save()>`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	fsys := vfs.OS{}
	resolver := workspaceResolver([]string{dir}, collectCFMLFiles(fsys, []string{dir}))

	entries, _ := refs.FindCalls(fsys, []string{dir}, "save", resolver)

	var lines []uint32

	for i := range entries {
		if filepath.Base(entries[i].File) == "Page.cfm" {
			lines = append(lines, entries[i].Line)
		}
	}

	if len(lines) != 1 || lines[0] != 2 {
		t.Errorf("want the one call on a Repo (line 3, 0-based 2), got lines %v: %+v", lines, entries)
	}
}
