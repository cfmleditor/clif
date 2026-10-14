package codemap_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/unresolved"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestTheMapResolvesWhatUnresolvedResolves: a component in a bean root is
// handed its dependency by a setter, and `variables.service.run()` is a call on
// it. `unresolved` types the property from the setter (the resolver's setter,
// constructor and injected-bean lookups reach its parse and its index); the map
// parsed with none of them, so the same call was an unresolved edge. On Masa CMS
// that was 3,134 calls. Both are built from one [resolve.Settings] here, as the
// commands build them, and must count the same unresolved calls.
func TestTheMapResolvesWhatUnresolvedResolves(t *testing.T) {
	dir := t.TempDir()
	beans := filepath.Join(dir, "beans")

	var files []string

	for name, src := range map[string]string{
		"beans/Service.cfc": `component { function run(){} }`,
		"beans/Consumer.cfc": `component accessors=true {
property name="service";
function setService( service ){ variables.service = arguments.service; }
function go(){ variables.service.run(); variables.service.missing(); }
}`,
	} {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, p)
	}

	beanPaths := map[string]string{"": beans}

	rep := unresolved.Scan(vfs.OS{}, files, nil, &unresolved.Options{WorkspaceFolders: []string{dir}, BeanPaths: beanPaths})

	fsys := vfs.OS{}
	idx := index.New()
	r := resolve.NewBatchResolver(fsys, idx, &resolve.Settings{WorkspaceFolders: []string{dir}, BeanPaths: beanPaths})
	r.LoadBeans(beanPaths)

	m := codemap.Build(&codemap.Options{Root: dir, Files: files, FS: fsys, Resolver: r, Workers: 2})

	if m.Stats.Unresolved != len(rep.Calls) {
		t.Errorf("unresolved reports %d calls, the map %d: %+v", len(rep.Calls), m.Stats.Unresolved, rep.Calls)
	}

	if len(rep.Calls) != 1 {
		t.Fatalf("the fixture should leave exactly the call to a missing method unresolved: %+v", rep.Calls)
	}
}
