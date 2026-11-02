package resolve

import (
	"fmt"
	"path/filepath"
	"slices"
	"sync"
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
// a scan: always, or with once, only the first time.
type panicFS struct {
	vfs.FS
	bad  string
	once bool

	mu      sync.Mutex
	tripped bool
}

func (p *panicFS) ReadFile(path string) ([]byte, error) {
	if filepath.Base(path) == p.bad {
		p.mu.Lock()
		trip := !p.once || !p.tripped
		p.tripped = true
		p.mu.Unlock()

		if trip {
			panic("a bad file")
		}
	}

	return p.FS.ReadFile(path)
}

// A file whose names could not be read is a caller of every name, so caller
// inference parses it rather than typing an argument from the rest.
func TestAFileTheCallerIndexCouldNotReadIsACallerOfEverything(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"good.cfm": `<cfset svc.save( thing )>`,
		// bad.cfm calls something else, so only its failed read can make
		// it a caller of save.
		"bad.cfm": `<cfset svc.run( other )>`,
	})

	good, bad := filepath.Join(dir, "good.cfm"), filepath.Join(dir, "bad.cfm")
	r := &Resolver{FS: &panicFS{FS: vfs.OS{}, bad: "bad.cfm"}, Index: index.New(), InferArgsFiles: []string{good, bad}}

	restore := ShareCallerIndex([]*Resolver{r})
	defer restore()

	if r.callerIdx == nil || !slices.Equal(r.callerIdx.failed, []string{bad}) {
		t.Fatal("the index did not record bad.cfm as a file it could not read")
	}

	if files := r.callerFiles("save"); !slices.Contains(files, bad) || !slices.Contains(files, good) {
		t.Fatalf("callers of save = %v, want good.cfm and the unread bad.cfm", files)
	}
}

// A file whose tag calls and tag names cannot be read may call any tag, so no
// tag's callers are known.
func TestATagScanThatCouldNotReadAFileKnowsNoCallers(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"tags/widget.cfm": `<cfoutput>#attributes.svc.go()#</cfoutput>`,
		"good.cfm":        `<cf_widget svc="#x#">`,
		"bad.cfm":         `<cf_widget svc="#y#">`,
	})

	files := []string{filepath.Join(dir, "good.cfm"), filepath.Join(dir, "bad.cfm")}
	r := &Resolver{FS: &panicFS{FS: vfs.OS{}, bad: "bad.cfm"}, Index: index.New(), InferArgsFiles: files}

	if sites, _ := r.customTagSites(); len(sites["widget"]) != 0 {
		t.Fatalf("widget's callers = %v: a file that could not be read was taken as calling nothing", sites["widget"])
	}
}

// A file whose calls could not be read but whose tag names can is an opaque
// caller of the tags it names, and leaves every other tag's callers known.
func TestAFailedFileIsAnOpaqueCallerOfTheTagsItNames(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"tags/widget.cfm": `<cfoutput>#attributes.svc.go()#</cfoutput>`,
		"tags/gadget.cfm": `<cfoutput>#attributes.svc.go()#</cfoutput>`,
		"good.cfm":        `<cf_widget svc="#x#"><cf_gadget svc="#x#">`,
		"bad.cfm":         `<cf_widget svc="#y#">`,
	})

	good, bad := filepath.Join(dir, "good.cfm"), filepath.Join(dir, "bad.cfm")
	r := &Resolver{FS: &panicFS{FS: vfs.OS{}, bad: "bad.cfm", once: true}, Index: index.New(), InferArgsFiles: []string{good, bad}}

	sites, _ := r.customTagSites()

	if len(sites["gadget"]) != 1 || sites["gadget"][0].file != good {
		t.Fatalf("gadget's callers = %v: a tag the failed file does not name lost its callers", sites["gadget"])
	}

	if !slices.ContainsFunc(sites["widget"], func(s customTagSite) bool { return s.file == bad && s.opaque }) {
		t.Fatalf("widget's callers = %v, want an opaque one from bad.cfm", sites["widget"])
	}
}
