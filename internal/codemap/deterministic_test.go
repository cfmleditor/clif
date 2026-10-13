package codemap_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

// The same workspace gives the same map. ContentBox keeps a RailoDBInfo.cfc in
// several patch directories, and a patch that has none calls `new
// RailoDBInfo()`: every copy is equally near, and the edge went to whichever
// the index — filled by a parallel scan — listed first, so it moved from run to
// run and the island numbering moved with it. Ten copies and ten builds make
// a lucky pass impossible.
func TestTheSameWorkspaceGivesTheSameMap(t *testing.T) {
	root := t.TempDir()

	write := func(rel, src string) string {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return p
	}

	var files []string

	for i := range 10 {
		files = append(files, write(fmt.Sprintf("patches/p%d/Widget.cfc", i), "component {\n\tfunction run() {}\n}\n"))
	}

	files = append(files, write("patches/zz/Caller.cfc", "component {\n\tfunction go() {\n\t\treturn new Widget().run();\n\t}\n}\n"))

	for range 10 {
		fsys := vfs.OS{}
		m := codemap.Build(&codemap.Options{
			Root: root, Files: files, FS: fsys, Workers: 4,
			Resolver: &resolve.Resolver{FS: fsys, Index: index.New(), WorkspaceFolders: []string{root}},
		})

		var to []string

		for _, e := range m.Edges {
			if e.From == "patches/zz/Caller.cfc::go" && e.Kind == codemap.EdgeCalls {
				to = append(to, e.To)
			}
		}

		if len(to) != 1 || to[0] != "patches/p0/Widget.cfc::run" {
			t.Fatalf("go's call edges: got %v, want [patches/p0/Widget.cfc::run]", to)
		}
	}
}

// A page calling a function its includer declares resolves through the
// include graph, so the includer's includes must be known before any page is
// scanned. They were recorded only when something indexed the includer during
// the parallel scan, so the edge came and went with goroutine order: Lucee's
// admin pages differed by a dozen edges between runs of one build. Scanning
// the page first, with one worker, is the order that lost it every time.
func TestAPageReachesItsIncludersFunctionsInAnyOrder(t *testing.T) {
	root := t.TempDir()

	write := func(rel, src string) string {
		p := filepath.Join(root, rel)
		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return p
	}

	parent := write("bundle.cfm", "<cffunction name=\"byteFormat\"><cfreturn 1></cffunction>\n<cfinclude template=\"bundle.create.cfm\">\n")
	child := write("bundle.create.cfm", "<cfset size = byteFormat(10)>\n")

	for _, files := range [][]string{{child, parent}, {parent, child}} {
		for _, workers := range []int{1, 4} {
			fsys := vfs.OS{}
			m := codemap.Build(&codemap.Options{
				Root: root, Files: files, FS: fsys, Workers: workers,
				Resolver: &resolve.Resolver{FS: fsys, Index: index.New(), WorkspaceFolders: []string{root}},
			})

			found := false

			for _, e := range m.Edges {
				if e.From == "bundle.create.cfm" && e.To == "bundle.cfm::byteformat" && e.Kind == codemap.EdgeCalls {
					found = true
				}
			}

			if !found {
				t.Errorf("files %v, %d workers: no edge from bundle.create.cfm to the byteFormat its includer declares", []string{filepath.Base(files[0]), filepath.Base(files[1])}, workers)
			}
		}
	}
}
