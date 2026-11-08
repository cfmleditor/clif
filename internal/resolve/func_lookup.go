// Looking a function up in a file and through its extends chain, includes and delegates; a file's extends.

package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/uri"
)

// EnsureIndexed ensures a CFC file is indexed, loading from disk if needed.
//
// The "needed" test is whether the index holds the file (HasFile), not whether
// it holds any functions for it. Those differ for a component that legitimately
// declares none — a property-only bean, a DTO, a `this`-scope struct — and
// asking the second question re-read and re-parsed every such file on every
// single lookup, since re-indexing it produced the same empty result.
func (r *Resolver) EnsureIndexed(cfcPath string) []*parser.FunctionDef {
	idx, cfcURI, ok := r.ensureFileIndexed(cfcPath)
	if !ok {
		return nil
	}

	return idx.FunctionsForFile(cfcURI)
}

// funcInFile is the first function of cfcPath's named name, indexing the file
// first as EnsureIndexed does.
func (r *Resolver) funcInFile(cfcPath, name string) *parser.FunctionDef {
	idx, cfcURI, ok := r.ensureFileIndexed(cfcPath)
	if !ok {
		return nil
	}

	return idx.FunctionInFile(cfcURI, name)
}

// ensureFileIndexed indexes cfcPath if it is not yet, and says which index
// holds it under which URI; false when it cannot be read.
func (r *Resolver) ensureFileIndexed(cfcPath string) (*index.Index, uri.URI, bool) {
	if r.indexer != nil {
		return r.indexer.ensureFileIndexed(cfcPath)
	}

	cfcURI := cfpath.ToURI(cfcPath)

	if !r.Index.HasFile(cfcURI) {
		text, err := r.readText(cfcPath)
		if err != nil {
			return nil, "", false
		}

		// Indexed as a batch run indexes it (IndexForBatch), so a component's entry
		// does not depend on whether a scan or a lookup reached it first.
		r.Index.IndexParsed(cfcURI, text, r.ParseFile(cfcPath, text, IndexSpec()))
	}

	return r.Index, cfcURI, true
}

// LookupFuncWithExtends searches for a function in cfcPath, walking the
// extends chain, and then the methods WireBox delegates into any component of
// the chain.
func (r *Resolver) LookupFuncWithExtends(cfcPath, funcName string) *parser.FunctionDef {
	return r.lookupFunc(cfcPath, funcName, 0)
}

// maxDelegateDepth bounds a delegate's own delegates: two components that
// delegate to each other must not recurse forever.
const maxDelegateDepth = 3

// funcCache holds lookupFunc's answers. Producer evaluation and caller
// inference ask the same component for the same method again and again (97%
// of the 405,000 lookups a cfwheels scan made), and a lookup in a Wheels
// controller or model walks the mixin plan each time. An answer reads the
// index and the sources it names, so it is kept only where a source cannot
// change unseen:
//
//   - in a batch scan (SnapshotScan), whose files do not change while it runs:
//     the scan's index writes only add files it had not indexed yet, which a
//     lookup needing them indexes itself, so answers hold for the whole scan;
//   - in the editor (IndexTracksFiles), where every edit and watched-file
//     event is an index write: an answer holds for one index generation, and
//     one computed while the generation moved is not kept, since a write it
//     raced may have changed what it read.
//
// InvalidatePaths clears it. Callers share the definitions it returns, as
// they share the index's, and must copy one before binding it to a name.
type funcCache struct {
	gen     uint64
	entries map[funcKey]*parser.FunctionDef
}

// funcKey spells the method as the caller did: the lookup ignores case, so two
// spellings are two entries with one answer, and no key is lowercased.
type funcKey struct {
	path, name string
	depth      int
}

// maxFuncCache bounds the editor's funcCache: past it, it starts again. A
// batch scan's is bounded by the methods it asks about.
const maxFuncCache = 1 << 16

func (r *Resolver) lookupFunc(cfcPath, funcName string, depth int) *parser.FunctionDef {
	if r.Index == nil || !r.IndexTracksFiles && !r.SnapshotScan {
		return r.lookupFuncUncached(cfcPath, funcName, depth)
	}

	gen := r.Index.Generation()
	key := funcKey{path: cfcPath, name: funcName, depth: depth}

	// funcCache has a lock of its own. Under r.mu, which some sixty caches
	// share, every miss's write here blocked every reader of all of them, and
	// it was the most contended lock in a batch scan: 0.8s of tassweb's
	// workers' time waiting.
	r.funcMu.RLock()
	def, ok := r.funcCache.entries[key]
	ok = ok && (r.SnapshotScan || r.funcCache.gen == gen)
	r.funcMu.RUnlock()

	if ok {
		return def
	}

	def = r.lookupFuncUncached(cfcPath, funcName, depth)

	if !r.SnapshotScan && r.Index.Generation() != gen {
		return def
	}

	r.funcMu.Lock()
	if r.funcCache.entries == nil || !r.SnapshotScan && (r.funcCache.gen != gen || len(r.funcCache.entries) >= maxFuncCache) {
		r.funcCache = funcCache{gen: gen, entries: map[funcKey]*parser.FunctionDef{}}
	}

	r.funcCache.entries[key] = def
	r.funcMu.Unlock()

	return def
}

func (r *Resolver) lookupFuncUncached(cfcPath, funcName string, depth int) *parser.FunctionDef {
	seen := make(map[string]bool)

	var chain []string

	for cfcPath != "" && !seen[cfcPath] {
		seen[cfcPath] = true
		chain = append(chain, cfcPath)
		cfcURI := cfpath.ToURI(cfcPath)

		// Mapper copies its package methods over existing members. An ambiguous
		// copy must not fall back to the definition it may have overwritten.
		if d, copied := r.wheelsMapperFunc(cfcPath, funcName); copied {
			return d
		}

		if d := r.funcInFile(cfcPath, funcName); d != nil {
			return d
		}

		ext, ok := r.extendsOf(cfcPath, cfcURI)
		if !ok || ext == "" {
			break
		}

		baseDir := filepath.Dir(cfcPath)
		cfcPath = r.ComponentPath(ext, baseDir)
	}

	// A template a component includes declares its functions into the
	// component: Wheels' Global.cfc is little but includes of global/*.cfm,
	// and application.wo.$simpleLock() is one of theirs.
	for _, p := range chain {
		if d := r.includedFunc(p, funcName); d != nil {
			return d
		}
	}

	if d := r.wheelsObjectMixinFunc(chain, funcName); d != nil {
		return d
	}

	if d := r.wheelsTestGlobalFunc(chain, funcName); d != nil {
		return d
	}

	// A delegated method never replaces one the object has (Injector.cfc
	// processDelegation skips a name the target already holds), so the whole
	// chain is searched first.
	for _, p := range chain {
		if d := r.delegatedFunc(r.delegatesOf(p), filepath.Dir(p), funcName, depth); d != nil {
			return d
		}
	}

	if d := r.presideSuperClassFunc(chain, funcName, depth); d != nil {
		return d
	}

	if d := r.legacyColdBoxFunc(chain, funcName, depth); d != nil {
		return d
	}

	// A Wheels association's methods: `hasMany( "comments" )` in config()
	// gives the model comments(), commentCount(), newComment() and the rest.
	if len(chain) > 0 && r.isWheelsModel(chain[0]) {
		if d := r.associationFunc(chain[0], funcName); d != nil {
			return d
		}
	}

	if d := r.mementoFunc(chain, funcName); d != nil {
		return d
	}

	return nil
}

// maxIncludeDepth bounds how far includedFunc follows templates that
// include templates.
const maxIncludeDepth = 4

// includedFunc is funcName as declared by a template cfcPath includes, or by
// one those include, maxIncludeDepth deep. Only a component's own include
// statements are read, from the index, rather than the workspace's include
// graph: that graph is rebuilt whenever an include is indexed, which a lazy
// lookup during a scan would do once per component.
func (r *Resolver) includedFunc(cfcPath, funcName string) *parser.FunctionDef {
	seen := map[string]bool{pathKey(cfcPath): true}
	queue := []string{cfcPath}

	for depth := 0; depth < maxIncludeDepth && len(queue) > 0; depth++ {
		var next []string

		for _, from := range queue {
			if !r.Index.HasFile(cfpath.ToURI(from)) {
				r.EnsureIndexed(from)
			}

			for _, raw := range r.Index.IncludesForFile(cfpath.ToURI(from)) {
				target := r.IncludePath(raw, from)
				if target == "" || seen[pathKey(target)] {
					continue
				}

				seen[pathKey(target)] = true

				for _, d := range r.EnsureIndexed(target) {
					if strings.EqualFold(d.Name, funcName) {
						return d
					}
				}

				next = append(next, target)
			}
		}

		queue = next
	}

	return nil
}

// delegatedFunc is the method one of delegates gives the component under
// name: the delegate target's own method, found with its extends chain and
// its own delegates. dir is the declaring file's, for the target's id.
func (r *Resolver) delegatedFunc(delegates []parser.Delegate, dir, name string, depth int) *parser.FunctionDef {
	if depth >= maxDelegateDepth {
		return nil
	}

	for i := range delegates {
		d := &delegates[i]

		inner := d.DelegatedMethod(name)
		if inner == "" {
			continue
		}

		target := r.ComponentPath(parser.DelegateTarget(d.Target), dir)
		if target == "" {
			continue
		}

		if def := r.lookupFunc(target, inner, depth+1); def != nil {
			return def
		}
	}

	return nil
}

// delegatesOf is the WireBox delegations cfcPath declares, from the index or,
// failing that, the file itself — recorded with its extends, as
// declaredExtendsOf does.
func (r *Resolver) delegatesOf(cfcPath string) []parser.Delegate {
	cfcURI := cfpath.ToURI(cfcPath)
	if ds, ok := r.Index.DelegatesForFile(cfcURI); ok {
		return ds
	}

	text, err := r.readText(cfcPath)
	if err != nil {
		return nil
	}

	pr := parser.Parse(cfcURI, text)
	r.Index.SetDelegates(cfcURI, pr.Delegates)

	return pr.Delegates
}

// mockDecoration reports whether name is a method MockBox adds to a mock
// (parser.IsMockDecoration). A test mocks a real component in place —
// prepareMock( event ), getMockRequestContext() — and then calls these on it,
// so a component that is otherwise known is the wrong place to look for
// them: event.$( "getValue" ) was "method '$' not found in RequestContext".
func mockDecoration(name string) bool {
	return parser.IsMockDecoration(name)
}

// fileMissingBase is MissingBase for the file's own extends chain, and
// whether a framework preset implied a link of it — the file's own base or
// one further up. A break reached through an implied link is the framework's
// source not being in the workspace, which callers accept as dynamic.
func (r *Resolver) fileMissingBase(pr *parser.ParseResult, baseDir string) (base string, implied bool) {
	ext := r.fileExtends(pr)

	return r.chainBreak(ext, baseDir, ext != "" && impliedBase(pr))
}

// impliedBase reports whether the file's extends chain is one a framework
// preset implied rather than one the file wrote. A preset's resolvers answer
// a component missing from the workspace as dynamic — the framework's source
// is simply not checked out — and a base it implies does the same, so a view
// in a ColdBox app without ColdBox on disk is not a list of findings its own
// code never caused. A base the file names is still reported when it breaks.
func impliedBase(pr *parser.ParseResult) bool {
	return pr.Extends == ""
}

// fileExtends is what the parsed file extends: its own extends attribute, or
// the component a framework preset says a file at its path extends. Every
// reader of the extends chain goes through it, so a handler with no extends
// resolves a bare getInstance() exactly as one declaring EventHandler does.
func (r *Resolver) fileExtends(pr *parser.ParseResult) string {
	if pr.Extends != "" || r.ImplicitExtends == nil {
		return pr.Extends
	}

	return r.extendsFor("", cfpath.FromURI(string(pr.URI)))
}

// extendsFor is declared, or ImplicitExtends' answer for path when declared is
// empty. Remembered per path: bare-call resolution asks once per call site.
func (r *Resolver) extendsFor(declared, path string) string {
	if declared != "" || r.ImplicitExtends == nil || path == "" {
		return declared
	}

	r.mu.RLock()
	ext, ok := r.implicitCache[path]
	r.mu.RUnlock()

	if ok {
		return ext
	}

	ext = r.ImplicitExtends(path)
	if ext == parser.MuraAdminBase {
		ext = r.muraAdminBase(path)
	} else if fallback, ok := strings.CutPrefix(ext, parser.ApplicationBase); ok {
		ext = r.applicationBase(path, fallback)
	}

	r.mu.Lock()
	if r.implicitCache == nil {
		r.implicitCache = make(map[string]string)
	}

	r.implicitCache[path] = ext
	r.mu.Unlock()

	return ext
}

// applicationBase is the base parser.ApplicationBase names for path: the
// Application.cfc governing it, when that file is the framework instance, and
// fallback otherwise. It is the framework instance when its chain declares
// FW/1's view() and buildURL(): an Application.cfc that extends the framework
// is the object FW/1 renders a view inside, so a view calls its functions
// unqualified. Mura's admin/Application.cfc extends its embedded FW/1 1.x and
// declares rbKey(), which every admin view calls.
func (r *Resolver) applicationBase(path, fallback string) string {
	root := r.FindApplicationRoot(filepath.Dir(path))
	if root == "" {
		return fallback
	}

	app := filepath.Join(root, "Application.cfc")
	if info, err := r.fs().Stat(app); err != nil || info.IsDir() || pathKey(app) == pathKey(path) {
		return fallback
	}

	if r.ResolveFunc(app, "view", root) == nil || r.ResolveFunc(app, "buildURL", root) == nil {
		return fallback
	}

	return app
}

// extendsOf reports what cfcPath extends, reading and parsing the file only if
// the index cannot say.
//
// This used to read and fully parse the component every time, to take one
// field off the result and discard the rest. That is the single most expensive
// thing the open path did: LookupFuncWithExtends runs once per unresolved call
// site, so a component with hundreds of them re-read and re-parsed the same
// bases hundreds of times. On a 540KB service.cfc, textDocument/didOpen
// allocated 35.8MB and took 57.7ms; through the index it is 13.2MB and 16.7ms.
// The bare parse of that file is 2.25MB, which is what says the cost was never
// the parser.
//
// It is also why allocation tracked call sites rather than file size — a
// persist.cfc three times larger allocated a third as much per byte.
//
// Nothing needs to invalidate this. The record lives in the index and
// removeFileEntries drops it, so any re-index forgets it and the next call
// re-establishes it from the file as it then stands.
func (r *Resolver) extendsOf(cfcPath string, cfcURI uri.URI) (string, bool) {
	ext, ok := r.declaredExtendsOf(cfcPath, cfcURI)
	if !ok {
		return "", false
	}

	return r.extendsFor(ext, cfcPath), true
}

// declaredExtendsOf is what cfcPath's own extends attribute says, before a
// framework preset's implied base: extendsOf's lookup, for the walk that has
// to know which links a file wrote.
func (r *Resolver) declaredExtendsOf(cfcPath string, cfcURI uri.URI) (string, bool) {
	if ext, ok := r.Index.ExtendsForFile(cfcURI); ok {
		return ext, true
	}

	text, err := r.readText(cfcPath)
	if err != nil {
		return "", false
	}

	ext := parser.Parse(cfcURI, text).Extends
	r.Index.SetExtends(cfcURI, ext)

	return ext, true
}

// nearestEntity is the file an ORM entity name names from baseDir. A file in
// the caller's own application declaring it as its entityname comes first,
// the nearest of them: within one application a declared name is the entity,
// and a nearer persistent component merely called the same is not. Otherwise
// it is the nearest of every candidate, declared or called after the file,
// so one application's declaration does not hide another's component; a
// declared one wins a tie (EntityCandidates lists them first).
func (r *Resolver) nearestEntity(name, baseDir string) string {
	if r.Index == nil {
		return ""
	}

	declared, all := r.Index.EntityChoices(name)

	return r.pickEntity(declared, all, baseDir)
}

// pickEntity is the entity file baseDir means among candidates (declared is
// the subset declaring the name as their entityname). Within the caller's own
// application, the one whose nearest Application.cfc is the caller's, so a
// nested application's entities are not the caller's: the nearest declared
// one, else the nearest of any. Otherwise the nearest of every candidate.
// Every lookup of an entity by name goes through it, so entityNew(), a bare
// component name and Slatwall's gateway methods agree about one.
func (r *Resolver) pickEntity(declared, candidates []uri.URI, baseDir string) string {
	if len(candidates) == 0 {
		return ""
	}

	if root := r.FindApplicationRoot(baseDir); root != "" {
		for _, group := range [][]uri.URI{declared, candidates} {
			if p := r.nearestOf(r.inApplication(group, root), baseDir); p != "" {
				return p
			}
		}
	}

	return r.nearestOf(candidates, baseDir)
}

// inApplication is the files of list whose nearest Application.cfc is in
// root, compared as every path here is (cfpath.SamePath, ignoring case): a
// root walked from an editor's URI and one from a path can spell one folder
// differently, a Windows drive letter or a case-insensitive volume.
func (r *Resolver) inApplication(list []uri.URI, root string) []uri.URI {
	var out []uri.URI

	for _, u := range list {
		if cfpath.SamePath(r.FindApplicationRoot(filepath.Dir(cfpath.FromURI(string(u)))), root) {
			out = append(out, u)
		}
	}

	return out
}

// nearestOf is the candidate nearest baseDir, the first among equals: sorted
// candidates make that the lowest path, so the answer does not depend on the
// order a parallel scan indexed them in.
func (r *Resolver) nearestOf(candidates []uri.URI, baseDir string) string {
	from := string(cfpath.ToURI(filepath.Join(baseDir, "x.cfc")))
	best, bestDist := "", -1

	for _, u := range candidates {
		if d := cfpath.URIDistance(from, string(u)); bestDist < 0 || d < bestDist {
			best, bestDist = cfpath.FromURI(string(u)), d
		}
	}

	return best
}
