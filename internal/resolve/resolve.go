// Package resolve provides component and function resolution logic.
package resolve

import (
	"fmt"
	"maps"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// Resolver resolves component dot-paths to files and functions.
type Resolver struct {
	NativeEngine string // "lucee" enables its private runtime API; empty uses portable contracts

	// SpeculativeReceivers is an immutable, opt-in fallback map from a scan.
	// These guesses do not prove the receiver origin. Nil keeps strict behavior.
	SpeculativeReceivers map[SpeculativeReceiver]string

	Discovery         *DiscoveryDependencies
	configurationOnce sync.Once
	// Context views keep runtime mappings while traversing physical library files.
	callerApplication  string // execution application retained across physical library traversal
	callerMappings     map[string]string
	contextViews       map[string]*Resolver
	indexer            *Resolver
	FS                 vfs.FS
	WorkspaceFolders   []string
	Mappings           map[string]string
	BeanPaths          map[string]string      // configured factory-managed bean roots
	StartupFiles       []string               // configured startup templates, absolute; see startup.go
	CustomTagPaths     []config.CustomTagPath // configured custom tag directories, absolute; see custom_tags.go
	ExpressionMappings map[string]string
	Index              *index.Index
	Resolvers          []parser.Resolver
	// PropertyResolvers, ServicePropertyResolvers and InterpolateAllText are
	// what a parse of a file in this workspace takes beyond the resolvers and
	// mappings above; see ParseFile.
	PropertyResolvers        []parser.PropertyResolver
	ServicePropertyResolvers map[string]string
	InterpolateAllText       bool // features.outputContextInterpolation turned off
	// IndexTracksFiles says indexed disk changes reach the resolver through
	// reindexing or InvalidatePaths. A batch scan, which reads the disk as
	// it goes, leaves it false.
	IndexTracksFiles bool
	// SnapshotScan says the resolver serves one batch scan over files that do
	// not change while it runs (unresolved.NewResolver sets it), so answers
	// read from them may be kept for the scan. See funcCache.
	SnapshotScan bool
	// ImplicitExtends names the component a file extends when it names none,
	// from its path: a framework preset's rule that a ColdBox handler is an
	// EventHandler without saying so. Nil for none. See config.ImplicitExtends.
	ImplicitExtends func(path string) string
	// HelperScope reports whether a framework mixes its helper templates into
	// the file at path: a ColdBox handler, view, layout or interceptor. Nil for
	// none. See helpers.go and config.HelperScope.
	HelperScope func(path string) bool
	// Stubs is the API of the frameworks the configuration names, answering
	// a framework component nothing on disk does: calls on it are checked,
	// and completion and hover see its methods. Nil for none. See
	// internal/frameworkapi.
	Stubs *frameworkapi.Set
	// InferArgsFiles are the files to search for the callers of a function, to
	// type an argument from what they pass (argumentFromCallers). Nil switches
	// it off: a batch scan sets it once its index is complete.
	InferArgsFiles        []string
	CallerInferenceLimits config.CallerInferenceLimits
	stubFS                vfs.FS
	mu                    sync.RWMutex
	pathCacheEpoch        uint64 // advances on path invalidation, under mu
	// The build locks are outside pathCaches: InvalidatePaths replaces that
	// struct whole, and a lock held by a build in progress would be replaced
	// by an unlocked one, which the build's Unlock then aborts on.
	callerBuild     sync.Mutex        // held while callerIdx is built; see buildCallerIndexOnce
	tagSitesBuild   sync.Mutex        // one customTagSites build at a time
	exprKeys        []string          // ExpressionMappings' keys in the order they apply
	resolverSets    sync.Map          // resolverSetKey → *parser.ResolverSet, every parse with that slice sharing it (parseResolverSet)
	mappingsKey     string            // a view's callerMappings as a key; see leafSubclasses
	beanPathsCache  map[string]string // merged application/configured bean roots
	diOnce          sync.Once
	diPolicies      []diPolicy // source-backed DI/1 injection contracts
	discoveringDI   bool       // private policy discovery never re-enters injection lookup
	sourceFacts     spanCache[lexicalSourceFacts]
	loopCache       spanCache[loopsByName]       // file URI → every loop its current text holds (loopsOf)
	typeGuardCache  spanCache[receiverTypeGuard] // exact builtin checks in current source
	closureCache    spanCache[closureSpan]       // file URI → every function literal its current text holds (closuresOf)
	fw1SetViewCache spanCache[fw1SetView]        // file URI → every setView call its current text holds (fw1SetViews)
	registryCache   spanCache[[]registryFact]    // file URI → every registry recipe its current text holds (registriesOf)
	pathCaches                                   // what InvalidatePaths clears
}

// pathCaches are the answers a Resolver keeps that depend on files, paths and
// the index: InvalidatePaths starts them all again at once by resetting this
// struct, so a cache added here cannot be left out of the reset.
type pathCaches struct {
	appRootCache        map[string]string                     // dir → Application.cfc root
	appNameCache        map[string]bool                       // dir → an Application file there names an application
	slugCache           map[string]string                     // dir → its box.json slug, "" for none
	resolveCache        sync.Map                              // component+"\t"+baseDir → file path; read on every call, so lock-free
	includeCache        map[string]string                     // include path+"\t"+including directory → template
	interfaceCache      map[string]bool                       // component file → whether it declares an interface
	dirCache            *cfpath.DirCache                      // directory listings behind those resolutions
	incGraph            *includeGraph                         // the index's cfincludes, rebuilt when they change
	implicitCache       map[string]string                     // path → ImplicitExtends(path)
	helpers             *helperSet                            // application helper templates, per set of config files
	wb                  *wireboxWorkspace                     // what ModuleConfig.cfc and config/WireBox.cfc say about ids, per set of files
	fw1Scopes           map[string]fw1Scope                   // nearest application's source-defined injection scope
	fw1Parses           map[string]*fw1ParseEntry             // readFW1Scope's parses, by file (fw1Parse), on the owner
	startupCache        map[string][]startupAssign            // app root → its startup templates' shared-scope assignments
	startupFileFacts    map[string]*startupFileFacts          // startup source → what its text states (startupFacts), on the owner
	sourceInfos         map[string]sourceInfo                 // source-checked method bodies; refreshed when bytes change
	ctlPathCache        map[string]controllerPaths            // dir → its files' writes of Wheels' controllerPath
	includerCache       map[string]string                     // template, name, depth and include generation → includerHeld's answer
	returnCache         returnCache                           // ReturnComponentOf answers, for one index generation
	funcCache           funcCache                             // lookupFunc answers, for one index generation
	funcMu              sync.RWMutex                          // guards funcCache alone; see lookupFunc
	funcLookups         sync.Map                              // funcLookupKey → FuncLookup's answer, in a batch scan
	computedIncludes    map[string]computedIncludeFile        // file → its computed includes (computedIncludesOf)
	callerIdx           *callerIndex                          // name → files calling it, built once for argumentFromCallers
	argCache            map[string]string                     // an argument → what its callers pass (argumentFromCallers)
	requestWriterIdx    map[string][]string                   // request key → files writing it (requestWriters)
	ownedKeys           map[string]string                     // file and function → what its owned-key factory returns (ownedKeyReturn)
	keyPrefixes         map[string]string                     // literal key prefix → what every factory spelling it stores, "" when they differ (prefixComponent)
	handlerCache        map[string]*parser.ParseResult        // handler path → its parse (handlerParse)
	templateNames       map[string][]string                   // template file name → its paths, for a computed include (templatesNamed)
	uniqueNames         *uniqueNameCache                      // bare component name → its only file (uniqueComponentNamed)
	fw1ActionCache      map[string][]fw1Action                // controllers dir + section.item → its actions (fw1Actions)
	fw1IncluderCache    map[string]map[string][]string        // application dir → view → the views rendering it (fw1Includers)
	boundEntityCache    map[string]string                     // service path → the entity it binds (boundEntity)
	wrapperHosts        map[string][]string                   // template path → Wheels wrapper hosts (wheelsTemplateHosts)
	indexing            map[string]chan struct{}              // path → done, while a lookup indexes it (ensureFileIndexed)
	indexingBy          map[uint64]int                        // goroutine → index parses it is in (indexOnce)
	wrapperCalls        map[string][]wrapperCall              // caller path + wrapper → its wrapper calls (wheelsWrapperCalls)
	extraIncludeHosts   map[string][]includeHost              // template path → includes the graph cannot see (frameworkIncludeHosts)
	muraEventFiles      map[string]bool                       // file → Mura calls it with an event (handsMuraEvent)
	lazyGetters         map[string]string                     // file and function → what its lazy getter returns (lazyGetterReturn)
	lazyFiles           map[string]bool                       // file → it holds an isObject() guard at all
	viewArgCache        map[string]map[string][]viewArgSite   // module → view → its renders (viewArgSites)
	muraSites           []muraObjectSite                      // the workspace's dspObject_include calls (muraObjectSites)
	muraSitesBuilt      bool                                  // muraSites has been read
	muraSiteIdx         sync.Map                              // template path → positions in muraSites that could include it (muraSitesFor)
	muraSets            map[muraSetsKey]bool                  // template, name and scope → the template sets it (muraTemplateSets)
	collectionFields    map[string]string                     // file, field and index generation → emptyCollectionField's kind
	leafSubs            map[string]map[string][]string        // mappings scope → base file and index generation → leafSubclasses
	tagSites            map[string][]customTagSite            // custom tag name → the tags calling it (customTagSites)
	tagTemplates        map[string]int                        // custom tag name → how many templates answer to it
	tagCallsByFile      map[string]map[string][]customTagSite // file → its custom tag calls, read while it was indexed (recordTagCalls)
	tagFiles            map[string]tagFileEntry               // custom tag root, name and depth → its template (tagFileIn)
	tagRoots            map[string][]cfpath.TagRoot           // caller directory → its custom tag roots (customTagRoots)
	interceptSiteList   []interceptSite                       // the workspace's announces (interceptSites)
	interceptSitesBuilt bool                                  // interceptSiteList has been read
	handoffs            handoffIndex                          // handler actions by the view each renders (viewActions)
	registryTexts       map[string]registryText               // includer path → its registry facts while its stamp holds (includeRegistries)
}

// returnCache holds ReturnComponentOf's answers. An answer reads the index and
// the declaring file's source, so it is kept only while the index generation
// is the one it was computed under and the declaring file's stamp is
// unchanged. Keys are the definitions themselves: a re-indexed file gets new
// ones, and a cache that outgrows maxReturnCache (definitions from transient
// parses never repeat) starts again rather than holding them.
type returnCache struct {
	gen     uint64
	entries map[*parser.FunctionDef]returnEntry
	stamps  map[string]fileStamp // declaring file → its stamp, read once per generation
}

type fileStamp struct {
	size    int64
	modTime time.Time
}

// stampsFor keeps the stamps across a reset that stays in one generation,
// which is only the size cap: they describe files, not answers.
func (c returnCache) stampsFor(gen uint64) map[string]fileStamp {
	if c.gen == gen {
		return c.stamps
	}

	return nil
}

// returnStamp is the declaring file's stamp, read from disk at most once per
// index generation. Every call used to stat the file, cache hit or not, and
// that was a tenth of an unresolved scan's CPU. A change on disk the index
// has not seen is therefore noticed at the next index write rather than at
// once; in the editor every edit is one, and a watched-file event is one and
// clears this cache besides.
func (r *Resolver) returnStamp(gen uint64, path string) fileStamp {
	r.mu.RLock()
	stamp, ok := r.returnCache.stamps[path]
	ok = ok && r.returnCache.gen == gen
	r.mu.RUnlock()

	if ok {
		return stamp
	}

	if !r.IndexTracksFiles {
		if info, err := r.fs().Stat(path); err == nil {
			stamp = fileStamp{size: info.Size(), modTime: info.ModTime()}
		}
	}

	r.mu.Lock()
	if r.returnCache.gen != gen || r.returnCache.entries == nil {
		r.returnCache = returnCache{gen: gen, entries: make(map[*parser.FunctionDef]returnEntry)}
	}

	if r.returnCache.stamps == nil {
		r.returnCache.stamps = map[string]fileStamp{}
	}

	r.returnCache.stamps[path] = stamp
	r.mu.Unlock()

	return stamp
}

type returnEntry struct {
	component string
	size      int64
	modTime   time.Time
}

const maxReturnCache = 1 << 16

// ResolveFunc finds a function definition by component path and function name,
// handling pipe-separated alternatives, absolute paths, and the extends chain.
func (r *Resolver) ResolveFunc(component, funcName, baseDir string) *parser.FunctionDef {
	if context := r.forCaller(baseDir); context != r {
		return context.ResolveFunc(component, funcName, baseDir)
	}

	alternatives := []string{component}
	if strings.Contains(component, "|") {
		alternatives = strings.Split(component, "|")
	}

	for _, alt := range alternatives {
		var cfcPath string
		if filepath.IsAbs(alt) {
			cfcPath = alt
		} else {
			cfcPath = r.ComponentPath(alt, baseDir)
		}

		if cfcPath == "" {
			continue
		}

		if d := r.LookupFuncWithExtends(cfcPath, funcName); d != nil {
			return d
		}
	}

	return nil
}

// HasFunction returns true if the component has a function with the given name.
func (r *Resolver) HasFunction(component, funcName, baseDir string) bool {
	cfcPath := r.ComponentPath(component, baseDir)
	if cfcPath == "" {
		return false
	}

	for _, d := range r.EnsureIndexed(cfcPath) {
		if strings.EqualFold(d.Name, funcName) {
			return true
		}
	}

	return false
}

// FindApplicationRoot walks up from dir looking for Application.cfc or Application.cfm.
func (r *Resolver) FindApplicationRoot(dir string) string {
	r.mu.RLock()

	if r.appRootCache != nil {
		if v, ok := r.appRootCache[dir]; ok {
			r.mu.RUnlock()

			return v
		}
	}

	r.mu.RUnlock()

	result := r.findApplicationRootUncached(dir)

	r.mu.Lock()
	if r.appRootCache == nil {
		r.appRootCache = make(map[string]string)
	}

	r.appRootCache[dir] = result
	r.mu.Unlock()

	return result
}

func (r *Resolver) findApplicationRootUncached(dir string) string {
	d := dir

	for {
		for _, name := range []string{"Application.cfc", "Application.cfm"} {
			if _, err := r.fs().Stat(filepath.Join(d, name)); err == nil {
				return d
			}
		}

		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}

		d = parent
	}
}

// EffectiveMappings returns config mappings merged with Application.cfc mappings.
func (r *Resolver) EffectiveMappings(baseDir string) map[string]string {
	return r.effectiveMappings(baseDir)
}

func (r *Resolver) effectiveMappings(baseDir string) map[string]string {
	if r.indexer != nil {
		return r.callerMappings
	}

	appDir := r.FindApplicationRoot(baseDir)
	if appDir == "" {
		appDir = baseDir
	}

	appMappings := cfpath.LoadAppMappingsWithInitial(appDir, r.Mappings)
	if len(appMappings) == 0 {
		return r.Mappings
	}

	if len(r.Mappings) == 0 {
		return appMappings
	}

	merged := make(map[string]string, len(appMappings)+len(r.Mappings))
	maps.Copy(merged, appMappings)

	for key, value := range r.Mappings {
		for existing := range merged {
			if strings.EqualFold(existing, key) {
				delete(merged, existing)
			}
		}

		merged[key] = value
	}

	return merged
}

// ResolveFromCall resolves a call expression against configured resolvers.
func (r *Resolver) ResolveFromCall(expr string) string {
	return parser.ResolveFromCall(expr, r.Resolvers)
}

// CanResolveCall determines whether a function call can be resolved given the
// parse result context. Returns empty string if resolved, or a reason if not.
func (r *Resolver) CanResolveCall(call *parser.CallSite, pr *parser.ParseResult, baseDir string) string {
	return r.forCaller(baseDir).canResolveCall(call, pr, baseDir, nil)
}

// ExplainCall runs the same resolution logic as CanResolveCall but also returns a
// human-readable trace of every decision point that produced (or failed to produce)
// the final verdict — which mechanism set the call's component, which componentResolver
// or FuncLookup hop fired, and why the final method check succeeded or failed. Intended
// for the `explain` CLI command; not used on the hot lint path.
func (r *Resolver) ExplainCall(call *parser.CallSite, pr *parser.ParseResult, baseDir string) (string, []string) {
	tr := &callTrace{}
	reason := r.forCaller(baseDir).canResolveCall(call, pr, baseDir, tr)

	return reason, tr.steps
}

// callTrace accumulates what canResolveCall learned on the way to its verdict: a
// human-readable step list for [Resolver.ExplainCall], and the callee it landed on
// for [Resolver.ResolveCallTarget]. A nil *callTrace is always safe to call add or
// hit on (both no-op), so canResolveCall can be shared between the hot lint path
// (tr == nil) and its two introspecting callers without extra branching.
//
// Recording the target here rather than widening canResolveCall's return is
// deliberate: the function has twenty-odd `return ""` sites, and threading a second
// value through every one of them is exactly the kind of hand-maintained parallel
// list this codebase keeps getting bitten by. A recorder lets each site opt in where
// it already opts in to a trace step, and a site that forgets degrades to
// TargetNone — an edge the call graph reports as unresolved — rather than to a wrong
// edge. TestEveryAcceptPathRecordsATarget pins the set that must not forget.
type callTrace struct {
	steps  []string
	target CallTarget
}

func (t *callTrace) addf(format string, args ...any) {
	if t == nil {
		return
	}

	t.steps = append(t.steps, fmt.Sprintf(format, args...))
}

// hit records where a call site resolved to. def may be nil for the dynamic kinds,
// where there is a component (or not even that) but no definition to point at.
func (t *callTrace) hit(kind TargetKind, component string, def *parser.FunctionDef) {
	if t == nil {
		return
	}

	t.target.Kind = kind
	t.target.Component = component

	if def != nil {
		t.target.URI = def.URI
		t.target.FuncName = def.Name
		t.target.Line = def.Line
	}
}

func (r *Resolver) canResolveCall(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	funcName := call.FuncName
	variable := call.Variable

	// Unqualified call — check same file, then extends chain.
	// Skip if call.Component is already set (e.g. resolved via chained new/createObject).
	if variable == "" && call.Component == "" {
		if len(call.Chain) > 0 {
			return r.resolveBareChain(call, pr, baseDir, tr)
		}

		return r.resolveBareCall(call, pr, baseDir, tr)
	}

	// this. qualifier — refers to the current component
	if strings.EqualFold(variable, "this") {
		return r.resolveThisCall(funcName, pr, baseDir, tr)
	}

	// super. qualifier
	if strings.EqualFold(variable, "super") {
		return r.resolveSuperCall(funcName, pr, baseDir, tr)
	}

	if r.acceptsScopeReference(call, pr, tr) {
		tr.hit(TargetDynamic, "", nil)

		return ""
	}

	// Qualified call — find the component from refs
	comp := call.Component

	// softComp is a component a dynamicIfMissing resolver produced during this
	// resolution; the parse records its own (ParseResult.IsSoftComponent).
	softComp := ""

	if comp != "" {
		tr.addf("call.Component already set to %q (resolved earlier via chained new/createObject)", comp)
	}

	if comp == "" {
		var member bool

		comp, member = r.receiverComponentD(variable, call.Line, call.Caller, funcName, pr, baseDir, tr, lookupCtx{call: call})
		if member {
			tr.hit(TargetMember, "", nil)

			return ""
		}
	}

	if comp == "" {
		var accepted bool

		if comp, softComp, accepted = r.matchReceiverResolvers(call, tr); accepted {
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}
	}

	if comp == "" {
		comp = r.lateReceiverComponent(call, pr, baseDir, tr)
	}

	if comp == "" {
		tr.addf("no ComponentRef and no componentResolver matched %q", variable)
	}

	// Walk any intermediate .method() hops between the resolved base and this
	// call (e.g. "kpg.generateKeyPair().getPublic().getParams()" — comp here is
	// kpg's own component; call.Chain lists "generateKeyPair", "getPublic", each
	// needing its own declared return type applied before checking funcName below).
	if comp != "" && comp != "$any" {
		var (
			reason string
			done   bool
		)

		if comp, softComp, reason, done = r.walkHops(comp, softComp, call, pr, baseDir, tr); done {
			return reason
		}
	}

	return r.checkMethodOn(comp, softComp, call, pr, baseDir, tr)
}
