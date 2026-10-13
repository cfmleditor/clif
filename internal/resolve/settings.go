package resolve

import (
	"path/filepath"
	"runtime"
	"sync"

	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// Settings is everything a batch tool decides a resolver with: the .clif.json
// (or the framework a project is assumed to be) reduced to plain fields.
//
// It exists so `unresolved`, `explain` and the code map build their resolver in
// one place. They each built their own, and each had drifted: on Masa CMS, with
// no config, `unresolved` assumed the mura preset and added the startup beans'
// resolvers and the map did neither, so 18,185 calls the one resolved the other
// reported — 21,962 against 3,786.
type Settings struct {
	CallerInferenceLimits    config.CallerInferenceLimits
	Resolvers                []parser.Resolver
	Mappings                 map[string]string
	StartupFiles             []string
	ExpressionMappings       map[string]string
	PropertyResolvers        []parser.PropertyResolver
	ServicePropertyResolvers map[string]string
	BeanPaths                map[string]string // configured beanPaths; Application.cfc's are added
	WorkspaceFolders         []string
	ImplicitExtends          func(path string) string
	HelperScope              func(path string) bool
	Stubs                    *frameworkapi.Set
	NativeEngine             string // "", "generic" or "lucee"; empty detects it from the layout
	// InterpolateAllText is features.outputContextInterpolation turned off.
	InterpolateAllText bool

	// InferArgsFiles are the files to search for the callers of a function, to
	// type an argument from what they pass it; empty leaves it off. It must be
	// set before files are indexed, which is when the caller index is filled.
	InferArgsFiles []string
	// SpeculativeReceivers is the opt-in fallback map a speculative scan adds.
	SpeculativeReceivers map[SpeculativeReceiver]string
}

// NewBatchResolver is the resolver a batch run checks calls with, over idx.
//
// A batch run (a scan, an explain, the code map) reads files that do not change
// while it runs, so the resolver is a snapshot scan. The bean-resolver rules
// are settled here, before anything is indexed, because indexing parses with
// them. Files are indexed afterwards through [Resolver.IndexForBatch].
func NewBatchResolver(fsys vfs.FS, idx *index.Index, s *Settings) *Resolver {
	// Files do not change under a batch run, which is what SnapshotScan says and
	// what lets the filesystem answer a stat or a read once.
	r := NewResolver(vfs.CacheMetadata(fsys), idx, s)
	r.SnapshotScan = true
	r.InferArgsFiles = s.InferArgsFiles
	r.SpeculativeReceivers = s.SpeculativeReceivers
	r.Resolvers = r.BeanResolvers(s.BeanPaths)

	return r
}

// NewResolver is the resolver the settings describe, over idx. A batch run
// goes through [NewBatchResolver], which adds what only a run over files that
// do not change can have; the editor's live resolver adds its own (it tracks
// the files it is told about) and is the other caller. What the settings mean
// for resolution, including recognising a Lucee checkout, is decided here and
// so is the same in both.
func NewResolver(fsys vfs.FS, idx *index.Index, s *Settings) *Resolver {
	r := &Resolver{
		NativeEngine:          s.NativeEngine,
		FS:                    fsys,
		Index:                 idx,
		CallerInferenceLimits: s.CallerInferenceLimits,
		Resolvers:             s.Resolvers,
		Mappings:              s.Mappings,
		StartupFiles:          s.StartupFiles,
		BeanPaths:             s.BeanPaths,
		ExpressionMappings:    s.ExpressionMappings,
		WorkspaceFolders:      s.WorkspaceFolders,
		ImplicitExtends:       s.ImplicitExtends,
		HelperScope:           s.HelperScope,
		Stubs:                 s.Stubs,

		PropertyResolvers:        s.PropertyResolvers,
		ServicePropertyResolvers: s.ServicePropertyResolvers,
		InterpolateAllText:       s.InterpolateAllText,
	}

	if r.NativeEngine == "" && detectsLucee(fsys, s.WorkspaceFolders) {
		r.NativeEngine = "lucee"
	}

	return r
}

// Settings is what the resolver was built from, for a tool that builds another
// run from it (the editor's export of the unresolved report). It carries the
// resolution settings only; what a parse takes is carried by the resolver too (see ParseFile).
func (r *Resolver) Settings() *Settings {
	return &Settings{
		NativeEngine:          r.NativeEngine,
		CallerInferenceLimits: r.CallerInferenceLimits,
		Resolvers:             r.Resolvers,
		Mappings:              r.Mappings,
		StartupFiles:          r.StartupFiles,
		ExpressionMappings:    r.ExpressionMappings,
		BeanPaths:             r.BeanPaths,
		WorkspaceFolders:      r.WorkspaceFolders,
		ImplicitExtends:       r.ImplicitExtends,
		HelperScope:           r.HelperScope,
		Stubs:                 r.Stubs,

		PropertyResolvers:        r.PropertyResolvers,
		ServicePropertyResolvers: r.ServicePropertyResolvers,
		InterpolateAllText:       r.InterpolateAllText,
	}
}

// detectsLucee reports whether every root is laid out as a Lucee checkout.
func detectsLucee(fsys vfs.FS, roots []string) bool {
	if len(roots) == 0 {
		return false
	}

	for _, root := range roots {
		if cfpath.AssumedProjectMappings(root, fsys.ReadFile)["admin"] == "" {
			return false
		}
	}

	return true
}

// LoadBeans gives the resolver's index the bean map the server would build for
// the same workspace, so an injected property is typed the same way in a batch
// report as in the editor.
func (r *Resolver) LoadBeans(beanPaths map[string]string) {
	appDirs := make([]string, 0, len(r.WorkspaceFolders))
	for _, root := range r.WorkspaceFolders {
		appDirs = append(appDirs, r.FindApplicationRoot(root))
	}

	if all := cfpath.BeanPathsFor(beanPaths, appDirs); len(all) > 0 {
		r.Index.SetBeans(cfpath.BuildBeanMap(all, r.FS))
	}
}

// ParseSpec says what a caller wants from a parse of a file; the resolver
// supplies how the workspace types it. This is the one way a file in a
// workspace is parsed: the editor's live parse, the unresolved scan, the code
// map, references and the dependency graph each assembled their own parser
// options from the resolver's lookups, field for field, and a lookup added to
// one was missing from the others.
type ParseSpec struct {
	// Logger receives the parse's timing and errors.
	Logger parser.Logger
	// FindCalls limits the call sites recorded to these function names.
	FindCalls []string
	// Calls extracts call sites; AllScopes extracts them from function bodies
	// too, which every report and graph wants and the editor's cache does not.
	Calls     bool
	AllScopes bool
	// Links extracts document links.
	Links bool
	// Shallow reads signatures only, with no resolver and no lookup: the cheap
	// pass an ordinary file's index entry needs.
	Shallow bool
	// ByResolversOnly types a call by the configured resolvers alone, and not
	// by what the called function declares it returns. A scan wants the
	// declared return (`getService("X")` returns any, so it is `$any`); a
	// dependency graph wants the component the resolver names.
	ByResolversOnly bool
}

// IndexSpec is what indexing a component asks of a parse: its own signatures and
// refs, typed by the configured rules but not by another file's declared
// returns (see IndexForBatch).
func IndexSpec() ParseSpec { return ParseSpec{ByResolversOnly: true} }

// BatchSpec is what a run over every scope of every file asks of a parse.
func BatchSpec() ParseSpec { return ParseSpec{Calls: true, AllScopes: true} }

// ParseFile parses a file of this workspace as the workspace types it: its
// resolvers, mappings and rules, and the lookups that type an injected
// dependency or a function's return. file is a path.
func (r *Resolver) ParseFile(file, content string, spec ParseSpec) *parser.ParseResult {
	fileURI := cfpath.ToURI(file)

	if spec.Shallow {
		return parser.ParseWithOptions(fileURI, content, &parser.ParseOptions{Shallow: true})
	}

	po := &parser.ParseOptions{
		Logger:                   spec.Logger,
		Resolvers:                r.Resolvers,
		ExpressionMappings:       r.ExpressionMappings,
		ServicePropertyResolvers: r.ServicePropertyResolvers,
		PropertyResolvers:        r.PropertyResolvers,
		BeanLookup:               r.InjectionBeanLookup(file),
		PropertyBeanLookup:       r.InjectionPropertyLookup(file),
		SetterLookup:             r.SetterLookup(file),
		ConstructorLookup:        r.ConstructorLookup(file),
		InterpolateAllText:       r.InterpolateAllText,
		ExtractCalls:             spec.Calls,
		ScanAllScopes:            spec.AllScopes,
		ExtractLinks:             spec.Links,
		FindCalls:                spec.FindCalls,
	}

	if !spec.ByResolversOnly {
		po.FuncLookup = r.FuncLookup(filepath.Dir(file))
		po.BuiltinReturnLookup = r.BuiltinReturnLookupFor(file)
	}

	pr := parser.ParseWithOptions(fileURI, content, po)

	// The parse memoises the lookup for its own passes and hands the plain one
	// back, so a function parsed lazily later asks the resolver afresh.
	if po.FuncLookup != nil {
		pr.FuncLookup = po.FuncLookup
	}

	return pr
}

// IndexForBatch indexes one file the way every batch run must: a template only
// records its includes (a page that includes a helper can call what the helper
// declares), and a component is parsed as [Resolver.ParseFile] parses it, so
// what it returns and what its injected properties hold are typed as the scan
// will type them. It returns a template's include paths.
func (r *Resolver) IndexForBatch(f, content string) (includes []string) {
	r.IndexCallerFile(f, content)

	if !cfpath.IsCFCFile(f) {
		includes = parser.ExtractIncludes(content)
		r.Index.SetIncludes(cfpath.ToURI(f), includes)

		return includes
	}

	// The same parse as every other, typing a call by the configured resolvers
	// and not by what another file's function declares it returns: indexing runs
	// in parallel, and a return read from a file not yet indexed would make the
	// index depend on which worker got there first.
	r.Index.IndexParsed(cfpath.ToURI(f), content, r.ParseFile(f, content, IndexSpec()))

	return nil
}

// IndexFiles indexes every file for a batch run, in parallel: each is read and
// parsed on its own, and what a parse looks up (the bean map and the DI
// policies) is settled before the first starts. Serially this was most of a
// scan's wall time on a large workspace.
//
// A file that cannot be read, or is binary, is skipped. visit, when given, is
// called from the workers with each indexed file's content and, for a template,
// its include paths; it must be safe to call concurrently.
func (r *Resolver) IndexFiles(files []string, visit func(f, content string, includes []string)) {
	work := make(chan string)

	var wg sync.WaitGroup

	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for f := range work {
				content, err := vfs.ReadString(r.FS, f)
				if err != nil || cfpath.IsBinary(content) {
					continue
				}

				includes := r.IndexForBatch(f, content)

				if visit != nil {
					visit(f, content, includes)
				}
			}
		})
	}

	for _, f := range files {
		work <- f
	}

	close(work)
	wg.Wait()
}
