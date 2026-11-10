package resolve

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

// A file indexed in the scan is not read again for its custom tag calls: the
// calls are kept from the read indexing made. The file here cannot be read
// at all, so a second read would leave it an opaque caller.
func TestCustomTagSitesReuseTheIndexingRead(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "page.cfm")
	writeFiles(t, dir, map[string]string{"page.cfm": ``})

	r := &Resolver{FS: &panicFS{FS: vfs.OS{}, bad: "page.cfm"}, Index: index.New(), InferArgsFiles: []string{page}}
	r.IndexForBatch(page, `<cf_widget svc="#x#">`)

	sites, _ := r.customTagSites()
	if len(sites["widget"]) != 1 || sites["widget"][0].opaque || sites["widget"][0].attrs["svc"] != "#x#" {
		t.Fatalf("widget's callers = %+v, want the one call indexing read", sites["widget"])
	}
}

// The calls are read by the parser's custom tag scanner: a commented-out call
// is none, an end tag is not a second call, attributes may have space around
// the = or no quotes, a script call is an opaque caller (its arguments are
// expressions, not #expr# attributes), and so is one passing
// attributeCollection.
func TestCustomTagCallsInReadsTheScannersCalls(t *testing.T) {
	content := `<!--- <cf_widget svc="#old#"> --->
<cf_widget svc = "#a#" mode=fast>body</cf_widget>
<cfmodule template="tags/widget.cfm" svc='#b#'>
<cf_widget attributeCollection="#args#">
<cfscript>cf_widget(svc = c);</cfscript>`

	sites := customTagCallsIn(content, "/p.cfm")["widget"]
	if len(sites) != 4 {
		t.Fatalf("got %d calls, want 4: %+v", len(sites), sites)
	}

	if s := sites[0]; s.line != 1 || s.opaque || s.attrs["svc"] != "#a#" || s.attrs["mode"] != "fast" {
		t.Errorf("the tag call: %+v", s)
	}

	if s := sites[1]; s.line != 2 || s.opaque || s.attrs["svc"] != "#b#" {
		t.Errorf("the cfmodule call: %+v", s)
	}

	if !sites[2].opaque || !sites[3].opaque || sites[3].line != 4 {
		t.Errorf("attributeCollection and script calls: %+v, %+v; want both opaque", sites[2], sites[3])
	}
}

func TestWholeHashExpr(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"\t#rc.bean# ", "rc.bean"}, {"#a##b#", ""}, {"x#a#", ""}, {"##", ""}, {"#a#", "a"},
	} {
		got, ok := wholeHashExpr(tc.raw)
		if ok != (tc.want != "") || ok && got != tc.want {
			t.Errorf("wholeHashExpr(%q) = %q, %v; want %q", tc.raw, got, ok, tc.want)
		}
	}
}

func TestWithoutTags(t *testing.T) {
	for in, want := range map[string]string{
		`a<cfparam name="attributes.x" default="#y#">b<CFPARAM name="z">c`: "abc",
		`<cfparameter x="1"><cfparam`:                                      `<cfparameter x="1"><cfparam`,
		`no tags`:                                                          `no tags`,
	} {
		if got := withoutTags(in, "cfparam"); got != want {
			t.Errorf("withoutTags(%q) = %q, want %q", in, got, want)
		}
	}
}

// A panicked file declines every tag only when it calls cfmodule, whose tag
// is in an attribute; naming the word in prose or a longer name does not.
func TestCallsCFModule(t *testing.T) {
	for text, want := range map[string]bool{
		`<cfmodule template="x.cfm">`:      true,
		`<CFMODULE name="a.b">`:            true,
		`cfmodule ( template = "x.cfm" );`: true,
		`<!--- moved off cfmodule --->`:    false,
		`var cfmoduleName = "x";`:          false,
		`x.cfmodule = 1;`:                  false,
	} {
		if got := callsCFModule(text); got != want {
			t.Errorf("callsCFModule(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestMayCallCustomTag(t *testing.T) {
	for text, want := range map[string]bool{
		`<cf_x>`:                         true,
		`cfmodule(template="x.cfm")`:     true,
		"module\n\ttemplate=\"x.cfm\";":  true,
		`component { this.module = 1; }`: false,
		`// load the module's settings`:  false,
		`submodule template="x"`:         false,
		`// set the module name first`:   false,
	} {
		if got := mayCallCustomTag(text); got != want {
			t.Errorf("mayCallCustomTag(%q) = %v, want %v", text, got, want)
		}
	}
}

// The attributes a site keeps are copied out of the file's text.
func TestCustomTagSitesDoNotHoldTheFile(t *testing.T) {
	content := `<cf_widget svc="#a#">` + strings.Repeat(" ", 1<<12)
	site := customTagCallsIn(content, "/p.cfm")["widget"][0]

	v := site.attrs["svc"]
	if v != "#a#" || unsafe.StringData(v) == unsafe.StringData(content[16:]) {
		t.Fatalf("svc = %q, sharing the file's bytes", v)
	}
}

// A template found is kept only while it exists. A miss holds while the
// directories searched are unchanged, so asking again lists nothing, and a
// template added to one of them is seen at once; a recursive miss is redone
// after recursiveMissTTL.
func TestTagFileInForgetsWhatChanged(t *testing.T) {
	dir := t.TempDir()
	fsys := &readDirCounter{FS: vfs.OS{}}
	r := &Resolver{FS: fsys}

	if f := r.tagFileIn(dir, "nav", false); f != "" {
		t.Fatalf("before the file: %q", f)
	}

	listed := fsys.n

	if f := r.tagFileIn(dir, "nav", false); f != "" || fsys.n != listed {
		t.Fatalf("an unchanged miss: %q, %d more listings, want none", f, fsys.n-listed)
	}

	writeFiles(t, dir, map[string]string{"nav.cfm": ``})

	want := filepath.Join(dir, "nav.cfm")
	if f := r.tagFileIn(dir, "nav", false); f != want {
		t.Fatalf("after the file was added: %q, want %q", f, want)
	}

	if err := os.Remove(want); err != nil {
		t.Fatal(err)
	}

	if f := r.tagFileIn(dir, "nav", false); f != "" {
		t.Fatalf("after the file went: %q", f)
	}

	// Deep below the root, where the root's own listing does not change.
	writeFiles(t, dir, map[string]string{"a/b/keep.txt": ``})

	if f := r.tagFileIn(dir, "deep", true); f != "" {
		t.Fatalf("before the deep file: %q", f)
	}

	writeFiles(t, dir, map[string]string{"a/b/deep.cfm": ``})

	if f := r.tagFileIn(dir, "deep", true); f != "" {
		t.Fatalf("a recursive miss is kept for its TTL, got %q", f)
	}

	for k, e := range r.tagFiles {
		e.missed = e.missed.Add(-2 * recursiveMissTTL)
		r.tagFiles[k] = e
	}

	if want := filepath.Join(dir, "a", "b", "deep.cfm"); r.tagFileIn(dir, "deep", true) != want {
		t.Fatalf("after the recursive miss expired, want %q", want)
	}
}

// readDirCounter counts directory listings.
type readDirCounter struct {
	vfs.FS
	n int
}

func (c *readDirCounter) ReadDir(path string) ([]fs.DirEntry, error) {
	c.n++

	return c.FS.ReadDir(path)
}

// A file that panicked, and so is read again by name, cannot be said to
// name its tags when it calls one by path: through cfmodule, or Lucee's
// module statement.
func TestCallsCFModuleCountsTheModuleStatement(t *testing.T) {
	for text, want := range map[string]bool{
		`module template="widget.cfm" svc=x;`: true,
		"module\n\tname = \"a.b\";":           true,
		`// the module name is set below`:     false,
	} {
		if got := callsCFModule(text); got != want {
			t.Errorf("callsCFModule(%q) = %v, want %v", text, got, want)
		}
	}
}

// One scan's resolvers, one per config, share what each recorded: a file is
// walked for its tag calls once, whichever config indexed it.
func TestShareTagCallsGivesEachResolverEveryRecord(t *testing.T) {
	a := &Resolver{InferArgsFiles: []string{"/a.cfm", "/b.cfm"}}
	b := &Resolver{InferArgsFiles: a.InferArgsFiles}

	a.recordTagCalls("/a.cfm", `<cf_x>`, nil)
	b.recordTagCalls("/b.cfm", `<cf_y>`, nil)
	ShareTagCalls([]*Resolver{a, b})

	for _, r := range []*Resolver{a, b} {
		if len(r.tagCallsByFile["/a.cfm"]["x"]) != 1 || len(r.tagCallsByFile["/b.cfm"]["y"]) != 1 {
			t.Fatalf("records: %+v", r.tagCallsByFile)
		}
	}
}
