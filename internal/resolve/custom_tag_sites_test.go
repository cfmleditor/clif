package resolve

import (
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

// A template found is kept only while it exists, and a miss only briefly
// outside a batch scan: a tag directory can change with no event saying so.
func TestTagFileInForgetsWhatChanged(t *testing.T) {
	dir := t.TempDir()
	r := &Resolver{FS: vfs.OS{}}

	if f := r.tagFileIn(dir, "nav", false); f != "" {
		t.Fatalf("before the file: %q", f)
	}

	writeFiles(t, dir, map[string]string{"nav.cfm": ``})

	if f := r.tagFileIn(dir, "nav", false); f != "" {
		t.Fatalf("a fresh miss is kept, got %q", f)
	}

	for k, e := range r.tagFiles {
		e.missed = e.missed.Add(-2 * tagMissTTL)
		r.tagFiles[k] = e
	}

	want := filepath.Join(dir, "nav.cfm")
	if f := r.tagFileIn(dir, "nav", false); f != want {
		t.Fatalf("after the miss expired: %q, want %q", f, want)
	}

	if err := os.Remove(want); err != nil {
		t.Fatal(err)
	}

	if f := r.tagFileIn(dir, "nav", false); f != "" {
		t.Fatalf("after the file went: %q", f)
	}
}
