package resolve

import (
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestWheelsInlineTestBinder(t *testing.T) {
	dir := t.TempDir()
	files := wheelsFactoryFiles()
	files["wheels/WheelsTest.cfc"] = `component {` + wheelsTestInlineBinder + ` public any function own(){return this;} }`
	writeFiles(t, dir, files)
	got := reasonsWith(t, &Resolver{}, dir, "Spec.cfc")
	expectReasons(t, got, map[string]string{"mapper.route.end": "", "$mapper.route.end": "", "b.route": "", "b.missing": "method 'missing' not found in Mapper", "hiddenGlobal": "not found in extends chain"})
}

func TestWheelsInlineTestBinderBoundaries(t *testing.T) {
	for _, body := range []string{
		`function unused(){` + wheelsTestInlineBinder + `}`,
		`var load=function(){` + wheelsTestInlineBinder + `};`,
		`if (enabled){` + wheelsTestInlineBinder + `}`,
		strings.Replace(wheelsTestInlineBinder, `local.metaIndex[local.key] neq "public"`, `false`, 1),
		strings.Replace(wheelsTestInlineBinder, `variables[local.key] = application.wo[local.key];`, `variables[local.key] = other[local.key];`, 1),
		wheelsTestInlineBinder + `variables.mapper=other;`,
		wheelsTestInlineBinder + `function late(){}` + `this.mapper=other;`,
	} {
		t.Run(body[:min(40, len(body))], func(t *testing.T) {
			dir := t.TempDir()
			files := wheelsFactoryFiles()
			files["wheels/WheelsTest.cfc"] = `component {` + body + `}`
			writeFiles(t, dir, files)

			r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
			if fd := r.ResolveFunc("Spec", "mapper", dir); fd != nil {
				t.Fatalf("unproven binder supplied %v", fd)
			}
		})
	}
}
