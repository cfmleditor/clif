package resolve

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestForEachFileSurvivesAPanicOnOneFile(t *testing.T) {
	files := make([]string, 50)
	for i := range files {
		files[i] = fmt.Sprintf("f%02d.cfm", i)
	}

	done := make([]bool, len(files))

	failed := forEachFile(files, func(i int, file string) {
		if file == "f07.cfm" {
			panic("a bad file")
		}

		done[i] = true
	})

	for i, ok := range done {
		if ok == (i == 7) {
			t.Fatalf("file %d: done %v", i, ok)
		}
	}

	if len(failed) != 1 || failed[0] != "f07.cfm" {
		t.Fatalf("failed = %v, want [f07.cfm]", failed)
	}
}

// panicFS panics on reading one file, as a bad file might on the way through
// a scan.
type panicFS struct {
	vfs.FS
	bad string
}

func (p panicFS) ReadFile(path string) ([]byte, error) {
	if filepath.Base(path) == p.bad {
		panic("a bad file")
	}

	return p.FS.ReadFile(path)
}

// A file whose names could not be read is a caller of every name, so caller
// inference parses it rather than typing an argument from the rest.
func TestAFileTheCallerIndexCouldNotReadIsACallerOfEverything(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"good.cfm": `<cfset svc.save( thing )>`,
		"bad.cfm":  `<cfset svc.save( other )>`,
	})

	good, bad := filepath.Join(dir, "good.cfm"), filepath.Join(dir, "bad.cfm")
	r := &Resolver{FS: panicFS{FS: vfs.OS{}, bad: "bad.cfm"}, Index: index.New(), InferArgsFiles: []string{good, bad}}

	restore := ShareCallerIndex([]*Resolver{r})
	defer restore()

	if files := r.callerFiles("save"); !slices.Contains(files, bad) || !slices.Contains(files, good) {
		t.Fatalf("callers of save = %v, want good.cfm and the unread bad.cfm", files)
	}
}

// A file whose tag calls could not be read may call any tag, so no tag's
// callers are known.
func TestATagScanThatCouldNotReadAFileKnowsNoCallers(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"tags/widget.cfm": `<cfoutput>#attributes.svc.go()#</cfoutput>`,
		"good.cfm":        `<cf_widget svc="#x#">`,
		"bad.cfm":         `<cf_widget svc="#y#">`,
	})

	files := []string{filepath.Join(dir, "good.cfm"), filepath.Join(dir, "bad.cfm")}
	r := &Resolver{FS: panicFS{FS: vfs.OS{}, bad: "bad.cfm"}, Index: index.New(), InferArgsFiles: files}

	if sites, _ := r.customTagSites(); len(sites["widget"]) != 0 {
		t.Fatalf("widget's callers = %v: a file that could not be read was taken as calling nothing", sites["widget"])
	}
}
