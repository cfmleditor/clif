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
	fsys = vfs.CacheMetadata(fsys)

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
		SnapshotScan:          true,
		InferArgsFiles:        s.InferArgsFiles,
		SpeculativeReceivers:  s.SpeculativeReceivers,
	}

	if r.NativeEngine == "" && detectsLucee(fsys, s.WorkspaceFolders) {
		r.NativeEngine = "lucee"
	}

	r.Resolvers = r.BeanResolvers(s.BeanPaths)

	return r
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

// IndexForBatch indexes one file, and returns a template's include paths the way every batch run must: a template only
// records its includes (a page that includes a helper can call what the helper
// declares), and a component is parsed with the resolver's own rules, so what
// it returns and what its injected properties hold are typed as the scan will
// type them. The code map indexed with the default parse options and typed a
// component's returns differently from the scan that checked the same calls.
func (r *Resolver) IndexForBatch(f, content string, propertyResolvers []parser.PropertyResolver) (includes []string) {
	r.IndexCallerFile(f, content)

	fileURI := cfpath.ToURI(f)

	if !cfpath.IsCFCFile(f) {
		includes = parser.ExtractIncludes(content)
		r.Index.SetIncludes(fileURI, includes)

		return includes
	}

	r.Index.IndexFileWithOptions(fileURI, content, &parser.ParseOptions{
		Resolvers:          r.Resolvers,
		SetterLookup:       r.SetterLookup(f),
		ConstructorLookup:  r.ConstructorLookup(f),
		BeanLookup:         r.InjectionBeanLookup(f),
		PropertyBeanLookup: r.InjectionPropertyLookup(f),
		PropertyResolvers:  propertyResolvers,
	})

	return nil
}

// ParseInputs is what a batch parse takes from the configuration beyond the
// resolver itself.
type ParseInputs struct {
	ExpressionMappings       map[string]string
	ServicePropertyResolvers map[string]string
	PropertyResolvers        []parser.PropertyResolver
	InterpolateAllText       bool
}

// ParseInputs is the part of the settings a parse takes.
func (s *Settings) ParseInputs() *ParseInputs {
	return &ParseInputs{
		ExpressionMappings:       s.ExpressionMappings,
		ServicePropertyResolvers: s.ServicePropertyResolvers,
		PropertyResolvers:        s.PropertyResolvers,
		InterpolateAllText:       s.InterpolateAllText,
	}
}

// BatchParseOptions are the options every batch run parses a file's calls with.
// The setter, constructor and injected-bean lookups are what type
// `variables.configBean` in a component that is handed it through init() or a
// setter; the code map left them out and reported 3,134 calls on Masa CMS that
// `unresolved` resolved. The caller adds what only it needs (links, say).
func (r *Resolver) BatchParseOptions(file string, in *ParseInputs) *parser.ParseOptions {
	return &parser.ParseOptions{
		Resolvers:                r.Resolvers,
		ExpressionMappings:       in.ExpressionMappings,
		ServicePropertyResolvers: in.ServicePropertyResolvers,
		PropertyResolvers:        in.PropertyResolvers,
		BeanLookup:               r.InjectionBeanLookup(file),
		PropertyBeanLookup:       r.InjectionPropertyLookup(file),
		SetterLookup:             r.SetterLookup(file),
		ConstructorLookup:        r.ConstructorLookup(file),
		InterpolateAllText:       in.InterpolateAllText,
		ExtractCalls:             true,
		ScanAllScopes:            true,
		FuncLookup:               r.FuncLookup(filepath.Dir(file)),
		BuiltinReturnLookup:      r.BuiltinReturnLookupFor(file),
	}
}

// IndexFiles indexes every file for a batch run, in parallel: each is read and
// parsed on its own, and what a parse looks up (the bean map and the DI
// policies) is settled before the first starts. Serially this was most of a
// scan's wall time on a large workspace.
//
// A file that cannot be read, or is binary, is skipped. visit, when given, is
// called from the workers with each indexed file's content and, for a template,
// its include paths; it must be safe to call concurrently.
func (r *Resolver) IndexFiles(files []string, propertyResolvers []parser.PropertyResolver, visit func(f, content string, includes []string)) {
	work := make(chan string)

	var wg sync.WaitGroup

	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for f := range work {
				content, err := vfs.ReadString(r.FS, f)
				if err != nil || cfpath.IsBinary(content) {
					continue
				}

				includes := r.IndexForBatch(f, content, propertyResolvers)

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
