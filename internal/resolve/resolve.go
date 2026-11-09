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
	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/nativetype"
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
	BeanPaths          map[string]string // configured factory-managed bean roots
	StartupFiles       []string          // configured startup templates, absolute; see startup.go
	ExpressionMappings map[string]string
	Index              *index.Index
	Resolvers          []parser.Resolver
	// PropertyResolvers, ServicePropertyResolvers and InterpolateAllText are
	// what a parse of a file in this workspace takes beyond the resolvers and
	// mappings above; see ParseFile.
	PropertyResolvers        []parser.PropertyResolver
	ServicePropertyResolvers map[string]string
	InterpolateAllText       bool // features.outputContextInterpolation turned off
	// IndexTracksFiles says a change to a file on disk reaches Index as a
	// write, which moves its generation: the LSP re-indexes a saved or
	// watched file. Source-backed caches then trust the generation rather
	// than stat the file on every lookup, which on a large component was the
	// largest flat cost of a reparse. A batch scan, which reads the disk
	// as it goes, leaves it false.
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
	callerTokens    spanCache[parser.Token]      // caller file URI → tokens for its current content
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
	appRootCache        map[string]string                   // dir → Application.cfc root
	appNameCache        map[string]bool                     // dir → an Application file there names an application
	slugCache           map[string]string                   // dir → its box.json slug, "" for none
	resolveCache        sync.Map                            // component+"\t"+baseDir → file path; read on every call, so lock-free
	includeCache        map[string]string                   // include path+"\t"+including directory → template
	interfaceCache      map[string]bool                     // component file → whether it declares an interface
	dirCache            *cfpath.DirCache                    // directory listings behind those resolutions
	incGraph            *includeGraph                       // the index's cfincludes, rebuilt when they change
	implicitCache       map[string]string                   // path → ImplicitExtends(path)
	helpers             *helperSet                          // application helper templates, per set of config files
	wb                  *wireboxWorkspace                   // what ModuleConfig.cfc and config/WireBox.cfc say about ids, per set of files
	fw1Scopes           map[string]fw1Scope                 // nearest application's source-defined injection scope
	startupCache        map[string][]startupAssign          // app root → its startup templates' shared-scope assignments
	sourceInfos         map[string]sourceInfo               // source-checked method bodies; refreshed when bytes change
	ctlPathCache        map[string]controllerPaths          // dir → its files' writes of Wheels' controllerPath
	includerCache       map[string]string                   // template, name, depth and include generation → includerHeld's answer
	returnCache         returnCache                         // ReturnComponentOf answers, for one index generation
	funcCache           funcCache                           // lookupFunc answers, for one index generation
	funcMu              sync.RWMutex                        // guards funcCache alone; see lookupFunc
	funcLookups         sync.Map                            // funcLookupKey → FuncLookup's answer, in a batch scan
	computedIncludes    map[string]computedIncludeFile      // file → its computed includes (computedIncludesOf)
	callerIdx           *callerIndex                        // name → files calling it, built once for argumentFromCallers
	argCache            map[string]string                   // an argument → what its callers pass (argumentFromCallers)
	requestWriterIdx    map[string][]string                 // request key → files writing it (requestWriters)
	ownedKeys           map[string]string                   // file and function → what its owned-key factory returns (ownedKeyReturn)
	keyPrefixes         map[string]string                   // literal key prefix → what every factory spelling it stores, "" when they differ (prefixComponent)
	handlerCache        map[string]*parser.ParseResult      // handler path → its parse (handlerParse)
	templateNames       map[string][]string                 // template file name → its paths, for a computed include (templatesNamed)
	uniqueNames         *uniqueNameCache                    // bare component name → its only file (uniqueComponentNamed)
	fw1ActionCache      map[string][]fw1Action              // controllers dir + section.item → its actions (fw1Actions)
	fw1IncluderCache    map[string]map[string][]string      // application dir → view → the views rendering it (fw1Includers)
	boundEntityCache    map[string]string                   // service path → the entity it binds (boundEntity)
	wrapperHosts        map[string][]string                 // template path → Wheels wrapper hosts (wheelsTemplateHosts)
	wrapperCalls        map[string][]wrapperCall            // caller path + wrapper → its wrapper calls (wheelsWrapperCalls)
	extraIncludeHosts   map[string][]includeHost            // template path → includes the graph cannot see (frameworkIncludeHosts)
	muraEventFiles      map[string]bool                     // file → Mura calls it with an event (handsMuraEvent)
	lazyGetters         map[string]string                   // file and function → what its lazy getter returns (lazyGetterReturn)
	lazyFiles           map[string]bool                     // file → it holds an isObject() guard at all
	viewArgCache        map[string]map[string][]viewArgSite // module → view → its renders (viewArgSites)
	muraSites           []muraObjectSite                    // the workspace's dspObject_include calls (muraObjectSites)
	muraSitesBuilt      bool                                // muraSites has been read
	muraSets            map[muraSetsKey]bool                // template, name and scope → the template sets it (muraTemplateSets)
	collectionFields    map[string]string                   // file, field and index generation → emptyCollectionField's kind
	leafSubs            map[string]map[string][]string      // mappings scope → base file and index generation → leafSubclasses
	tagSites            map[string][]customTagSite          // custom tag name → the tags calling it (customTagSites)
	tagTemplates        map[string]int                      // custom tag name → how many templates answer to it
	interceptSiteList   []interceptSite                     // the workspace's announces (interceptSites)
	interceptSitesBuilt bool                                // interceptSiteList has been read
	handoffs            handoffIndex                        // handler actions by the view each renders (viewActions)
	registryTexts       map[string]registryText             // includer path → its registry facts while its stamp holds (includeRegistries)
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

// acceptsScopeReference reports whether a call qualified by a scope word is a
// call of a function the scope holds, which a call's receiver cannot be: a
// scope is not a component.
func (r *Resolver) acceptsScopeReference(call *parser.CallSite, pr *parser.ParseResult, tr *callTrace) bool {
	funcName := call.FuncName
	variable := call.Variable

	// Bare VARIABLES qualifier (no further dot, e.g. "VARIABLES.someName(...)") —
	// a scope can't itself be a component ref, so treating "VARIABLES" as the thing
	// needing a ComponentRef is a category error. What's actually being called is a
	// property assigned onto the scope (e.g. "VARIABLES.someName = ARGUMENTS.callback;
	// ... VARIABLES.someName(...)"), a function-reference value passed in and stored —
	// same reasoning as the ARGUMENTS-as-function-reference case below, just for a
	// VARIABLES-scoped property instead of a bare argument. Accept if funcName was
	// ever assigned in VARIABLES scope anywhere in the file (assignments inside any
	// function are visible from every other function, unlike a var-scoped local).
	if strings.EqualFold(variable, "VARIABLES") {
		tr.addf("VARIABLES.%s called as a function reference — checking for a VARIABLES-scoped assignment", funcName)

		if pr.HasScopedAssignment(parser.ScopeVariables, funcName) {
			tr.addf("%q was assigned in VARIABLES scope — treating as a call through a function-reference property", funcName)

			return true
		}
	}

	// ARGUMENTS qualifier — funcName is being called as a function-reference argument.
	// These are dynamic (type="any"), so we can't verify them statically; accept if the
	// argument is declared in the enclosing function.
	if strings.EqualFold(variable, "ARGUMENTS") {
		tr.addf("ARGUMENTS.%s called as a function reference — checking caller %q's argument list", funcName, call.Caller)

		if declaresArgument(pr, call.Caller, funcName) {
			return true
		}
	}

	return false
}

// matchReceiverResolvers is the component the configured resolvers give a
// receiver nothing in the file types: by the variable's name, then by the
// whole line's text (a chain like x.method().prop.func()). soft is the
// component when a dynamicIfMissing resolver gave it. accepted says a resolver
// marked noFollow gave it, which takes the call without looking for the method; the
// caller records the target.
func (r *Resolver) matchReceiverResolvers(call *parser.CallSite, tr *callTrace) (comp, softComp string, accepted bool) {
	variable := call.Variable

	var noFollow, soft bool

	comp, noFollow, soft = r.matchResolver(variable, tr, func(c, desc string, nf bool) {
		tr.addf("resolved %q to %q via componentResolver matching the variable name [%s] (noFollow=%v)",
			variable, c, desc, nf)
	})
	if soft {
		softComp = comp
	}

	if noFollow && comp != "" {
		return comp, softComp, true
	}

	if comp == "" && call.Text != "" {
		comp, noFollow, soft = r.matchResolver(call.Text, tr, func(c, desc string, nf bool) {
			tr.addf("resolved %q to %q via componentResolver matching the full line text %q [%s] (noFollow=%v)",
				variable, c, call.Text, desc, nf)
		})
		if soft {
			softComp = comp
		}

		if noFollow && comp != "" {
			return comp, softComp, true
		}
	}

	return comp, softComp, false
}

// lateReceiverComponent is the answer for a receiver every other step left
// untyped: a loop element, a view's prc or rc, then what its arguments' callers
// pass and what init stores. They come last so an inference never turns a call
// something else accepted into a finding.
func (r *Resolver) lateReceiverComponent(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	variable := call.Variable
	funcName := call.FuncName

	comp := ""

	if comp == "" {
		comp = r.loopElement(call, pr, baseDir, tr)
	}

	if comp == "" {
		comp = r.viewPrc(variable, funcName, pr, tr)
	}

	if comp == "" {
		comp = r.fw1ViewRc(variable, funcName, pr, tr)
	}

	// Last, an untyped argument is what every caller passes it, when the scan
	// was given the files to look through. After every other answer, so an
	// inference never turns a call something else accepted into a finding.
	if comp == "" {
		// callers: a caller passing on an argument of its own is typed by its
		// own callers in turn (Mura's gateways take what the manager forwards).
		if comp = r.argumentFromCallers(variable, call.Caller, pr, lookupCtx{callers: true}); comp != "" {
			tr.addf("resolved %q to %q: what every caller of %s passes it", variable, comp, call.Caller)
		}
	}

	// A receiver built from an argument (`var item = arguments.iterator
	// .next()`, `var result = arguments.filter`) is looked up again with
	// inference allowed inside it: only here, for a call nothing else
	// answered, so the cost is that of the calls left unresolved. Without
	// files to infer from it would only repeat the lookup.
	if comp == "" && len(r.InferArgsFiles) > 0 {
		if comp, _ = r.receiverComponentD(variable, call.Line, call.Caller, funcName, pr, baseDir, nil, lookupCtx{call: call, callers: true}); comp != "" {
			tr.addf("resolved %q to %q: looked up again with what its arguments' callers pass", variable, comp)
		}
	}

	if comp == "" {
		if comp = r.initArgMember(variable, pr, lookupCtx{}); comp != "" {
			tr.addf("resolved %q to %q: init stores its argument there, and every construction passes it that", variable, comp)
		}
	}

	return comp
}

// hopCallReturn is what a hop's own call returns: the configured bean
// rules' answer for a factory call (the container's getBean( "content" ),
// whose construction the policy spells), or the expression's own.
func (r *Resolver) hopCallReturn(fd *parser.FunctionDef, expression, baseDir, comp string) string {
	if rc, _, _ := parser.ResolveFromCallMatch(expression, r.Resolvers); rc != "" {
		return rc
	}

	return r.expressionReturn(fd, expression, baseDir, comp)
}

// mockHop is what a chain holds at a hop as far as MockBox is concerned. A hop
// that makes a mock makes one of the class it names, whatever it is called on
// (getMockBox().createMock( "X" ).init()), and made reports it: the hop is
// done. A decoration that reports on a mock ($callLog()) leaves the rest of the
// chain dynamic. A mock reaching a hop is its class, or dynamic where that
// names none.
func (r *Resolver) mockHop(comp, hop, expression string, callHop bool, baseDir string, tr *callTrace) (string, bool) {
	if mock := parser.MockCall(expression); callHop && mock != "" {
		tr.addf("chain hop %q makes a MockBox mock: %q", hop, mock)

		return mock, true
	}

	if parser.IsMockReport(hop) {
		tr.addf("chain hop %q reports on a MockBox mock and does not return it", hop)

		return "$any", false
	}

	if cls, ok := strings.CutPrefix(comp, parser.MockPrefix); ok {
		if r.ComponentPath(cls, baseDir) == "" {
			return "$any", false
		}

		return cls, false
	}

	return comp, false
}

// walkHops follows call.Chain from comp through each hop's return type. It
// reports the component the chain reaches, or done with the answer when a hop
// settles the call itself.
func (r *Resolver) walkHops(comp, softComp string, call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) (reached, soft, reason string, done bool) {
	funcName := call.FuncName

	for i, hop := range call.Chain {
		// A hop carries its call's arguments when the parser could read
		// them; every check and every reason below is about its name.
		expression, callHop := parser.CallExpression(hop)
		if callHop {
			hop = parser.CallHopName(hop)
		}

		var made bool
		if comp, made = r.mockHop(comp, hop, expression, callHop, baseDir, tr); made {
			continue
		}

		// A hop that returned "$any" makes the rest of the chain dynamic.
		// Walking on asked "$any" for the next method, which it can never
		// define: getSandBox("x").getAttendanceObj().getLog() was reported
		// as getAttendanceObj missing from $any. The accept after the walk
		// is the answer for a dynamic receiver.
		if comp == "$any" {
			tr.addf("chain hop %q is on a dynamic ($any) value — the rest of the chain is dynamic", hop)

			break
		}

		// A registry family is its members' set, which no single hop's
		// declared return type can name: what it returns is an array of the
		// element the members' own files prove, and the chain past the
		// getter runs on that array, not on the family. Dynamic, once the
		// hop itself is every family's method.
		if paths := familyPaths(comp); len(paths) > 0 {
			hopCall := *call
			hopCall.FuncName = hop

			if why := r.checkFamiliesMethod(familyRefs(comp), &hopCall, pr, tr); why != "" {
				return comp, softComp, why + " (chain to '" + funcName + "')", true
			}

			tr.addf("chain hop %q is on a registry family (%s) — the rest of the chain is dynamic", hop, displayFamilyMembers(paths))
			tr.hit(TargetDynamic, "", nil)

			return comp, softComp, "", true
		}

		if strings.HasPrefix(comp, "$builtin.") {
			ret, ok := docs.LookupBuiltinMethodReturn(comp, hop)
			if !ok {
				return comp, softComp, "method '" + hop + "' not found on builtin " + comp[9:] + " (chain to '" + funcName + "')", true
			}

			tr.addf("native chain hop %q on %q returns %q", hop, comp, ret)
			comp = ret

			continue
		}

		if property, ok := parser.PropertyName(hop); ok {
			ret, soft, reason, done := r.propertyHop(comp, property, call, i, pr, baseDir, tr)
			if done {
				return comp, softComp, reason, true
			}

			if soft {
				softComp = ret
			}

			comp = ret

			continue
		}

		fd := r.ResolveFunc(comp, hop, baseDir)

		if fd == nil {
			if next := r.undeclaredHopReturn(comp, hop, baseDir, tr); next != "" {
				comp = next

				continue
			}

			return comp, softComp, r.missingChainHop(comp, softComp, hop, funcName, pr, baseDir, tr), true
		}

		ret, noFollow, soft := r.hopReturn(comp, hop, fd, baseDir, tr)
		if ret == "" && !callHop {
			ret = r.expressionReturn(fd, factoryCallExpression(pr.Content, hop, int(call.Line)), baseDir, comp)
		}

		if ret == "" && callHop {
			ret = r.hopCallReturn(fd, expression, baseDir, comp)
		}

		if soft {
			softComp = ret
		}

		if noFollow && ret != "" {
			tr.hit(TargetDynamic, ret, nil)

			return comp, softComp, "", true
		}

		if ret == "" {
			next, reason, done := r.untypedHop(fd, comp, hop, call, i, tr)
			if done {
				return comp, softComp, reason, true
			}

			ret = next
		}

		comp = ret
	}

	return comp, softComp, "", false
}

// undeclaredHopReturn is what a chain hop comp does not declare returns when
// something still says, or "". A decoration returns the mock it is called
// on, so the chain goes on from the same component: model.$( "getCache", c
// ).getFoo() is checked against model's class. Slatwall's BaseService
// answers this.newProduct() and its family with an entity.
func (r *Resolver) undeclaredHopReturn(comp, hop, baseDir string, tr *callTrace) string {
	if mockDecoration(hop) {
		tr.addf("chain hop %q is a method MockBox adds to a mock, and returns the mock — %q", hop, comp)

		return comp
	}

	e := r.slatwallGatewayReturn(r.ComponentPath(comp, baseDir), hop, baseDir)
	if e != "" {
		tr.addf("chain hop %q is answered by Slatwall's BaseService onMissingMethod, which returns %q", hop, e)
	}

	return e
}

// untypedHop decides a chain hop whose function declares no component: what
// it returns when something else says (next), or the call's answer (done).
func (r *Resolver) untypedHop(fd *parser.FunctionDef, comp, hop string, call *parser.CallSite, i int, tr *callTrace) (next, reason string, done bool) {
	funcName := call.FuncName

	// What an untyped hop returns is a mock when the next call on it is one
	// MockBox adds: `c.getRequestService().$( "getContext", x )`.
	after := funcName
	if i+1 < len(call.Chain) {
		after = parser.CallHopName(call.Chain[i+1])
	}

	if mockDecoration(after) {
		tr.addf("chain hop %q declares no component, and %q is called on what it returns — a mock, so the rest of the chain is dynamic", hop, after)
		tr.hit(TargetDynamic, "", nil)

		return "", "", true
	}

	if reason, done := r.closureStructHops(fd, comp, hop, call, i, tr); done {
		return "", reason, true
	}

	if ret := r.initArgGetter(fd); ret != "" {
		tr.addf("chain hop %q returns the property init stores from its argument, which every construction passes as %q", hop, ret)

		return ret, "", false
	}

	if r.engineValueReturn(fd, comp) {
		tr.addf("chain hop %q returns what a built-in function hands back — the rest of the chain is dynamic", hop)
		tr.hit(TargetDynamic, "", nil)

		return "", "", true
	}

	return "", "method '" + hop + "' in " + displayComponent(comp) + " has no component return type (chain to '" + funcName + "')", true
}

// propertyHop is walkHops for a property hop: what the property holds, from
// the component's typed properties or else a componentResolver matched
// against the property read, or done with the answer when that settles the
// call.
func (r *Resolver) propertyHop(comp, property string, call *parser.CallSite, i int, pr *parser.ParseResult, baseDir string, tr *callTrace) (ret string, soft bool, reason string, done bool) {
	if ret = r.publicPropertyComponent(comp, property, baseDir); ret != "" {
		return ret, false, "", false
	}

	var noFollow bool

	// The parse may have folded the calls before the property into
	// call.Component, so the source line is asked too.
	for _, text := range []string{propertyHopText(call, i, property), propertyReadText(lineOfContent(pr.Content, int(call.Line)), property)} {
		if text == "" {
			continue
		}

		if ret, noFollow, soft = r.propertyResolver(text, property, tr); ret != "" {
			break
		}
	}

	switch {
	case ret == "":
		return "", false, "property '" + property + "' in " + displayComponent(comp) + " has no component type (chain to '" + call.FuncName + "')", true
	case noFollow:
		tr.hit(TargetDynamic, ret, nil)

		return "", false, "", true
	}

	return ret, soft, "", false
}

// propertyHopText is a property hop as a resolver sees it: the call or
// variable the property is read from, then the property. A struct a factory
// fills at run time is typed nowhere in the source — tassweb's
// getSubServices() is one key per subservice, each a component created from
// a list — so the property's component can only come from configuration,
// `{"match": "getSubServices().$1", "resolve": "subservices.${1:lower}.service"}`,
// as a method hop's can from one matched against `hop()`.
func propertyHopText(call *parser.CallSite, i int, property string) string {
	if i == 0 {
		return call.Variable + "." + property
	}

	prev := call.Chain[i-1]
	if name, ok := parser.PropertyName(prev); ok {
		return propertyHopText(call, i-1, name) + "." + property
	}

	expression, callHop := parser.CallExpression(prev)
	if !callHop {
		expression = prev + "()"
	}

	return expression + "." + property
}

// propertyResolver is the componentResolver answer for a property read, or
// "" when the answer does not depend on the property. A resolver's prefix is
// found anywhere in the text, so the catch-all `get$1()` matches
// `foo.getBar().baz` on its `getBar()` and would type baz as getBar's
// component; an answer the text without the property gives too is that.
func (r *Resolver) propertyResolver(text, property string, tr *callTrace) (comp string, noFollow, soft bool) {
	comp, noFollow, soft = r.matchResolver(text, nil, nil)
	if comp == "" {
		return "", false, false
	}

	receiver := strings.TrimSuffix(text, text[len(text)-len(property):])
	receiver = strings.TrimSuffix(receiver, ".")

	if without, _, _ := r.matchResolver(receiver, nil, nil); without == comp {
		return "", false, false
	}

	_, _, idx := parser.ResolveFromCallMatch(text, r.Resolvers)
	tr.addf("property hop %q: resolved %q to %q via %s", property, text, comp, r.describeResolver(idx))

	return comp, noFollow, soft
}

// lineOfContent is the 0-based line n of content, or "".
func lineOfContent(content string, n int) string {
	for range n {
		i := strings.IndexByte(content, '\n')
		if i < 0 {
			return ""
		}

		content = content[i+1:]
	}

	line, _, _ := strings.Cut(content, "\n")

	return line
}

// propertyReadText is the expression that reads property on line, from the
// start of its receiver through the property: `a.getB( x ).prop` for a line
// holding `a.getB( x ).prop.c()`. "" when the line reads it nowhere, or more
// than once, since the two reads could be of different receivers.
func propertyReadText(line, property string) string {
	end := -1

	for from := 0; ; {
		i := parser.IndexFoldFrom(line, "."+property, from)
		if i < 0 {
			break
		}

		from = i + 1

		after := i + 1 + len(property)
		if after < len(line) && lex.IsIdent(line[after]) {
			continue
		}

		if end >= 0 {
			return ""
		}

		end = after
	}

	if end < 0 {
		return ""
	}

	start := end - len(property) - 1

	for start > 0 {
		j := start

		if line[j-1] == ')' {
			depth := 0

			for j > 0 {
				j--

				switch line[j] {
				case ')':
					depth++
				case '(':
					depth--
				}

				if depth == 0 {
					break
				}
			}

			if depth != 0 {
				return ""
			}
		}

		k := j
		for k > 0 && lex.IsIdent(line[k-1]) {
			k--
		}

		if k == j {
			// An argument list with no name before it was not walked
			// correctly (a parenthesis inside a string); decline.
			if j < start {
				return ""
			}

			break
		}

		start = k
		if start == 0 || line[start-1] != '.' {
			break
		}

		start--
	}

	return strings.TrimPrefix(line[start:end], ".")
}

// hopReturn is chainHopReturn, and failing that what the receiver binds the
// method to: a cborm service's entity, a Wheels model's own type.
func (r *Resolver) hopReturn(comp, hop string, fd *parser.FunctionDef, baseDir string, tr *callTrace) (ret string, noFollow, soft bool) {
	ret, noFollow, soft = r.chainHopReturn(comp, hop, fd, tr)
	if sub := r.selfTyped(comp, baseDir, fd, ret); sub != "" {
		tr.addf("chain hop %q returns the class declaring it, and is called on its subclass %q", hop, sub)
		ret = sub
	}

	if ret == "" {
		if e := r.receiverReturn(r.ComponentPath(comp, baseDir), hop); e != "" {
			tr.addf("chain hop %q returns what it is called on binds it to: %q", hop, e)
			ret = e
		}
	}

	return ret, noFollow, soft
}

// resolveBareCall is canResolveCall for an unqualified call: this file, a
// function-reference local, the extends chain, cfinclude, and a
// variables-scope function reference.
func (r *Resolver) resolveBareCall(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	funcName := call.FuncName

	tr.addf("unqualified call to %q — checking same file", funcName)

	if r.bareCallInComponent(call, pr, baseDir, tr) || r.bareCallThroughContext(call, pr, baseDir, tr) {
		return ""
	}

	if r.fileExtends(pr) != "" {
		if base, implied := r.fileMissingBase(pr, baseDir); base != "" {
			if implied {
				tr.addf("the framework base %s implies does not resolve (%s) — accepted as dynamic", r.fileExtends(pr), base)
				tr.hit(TargetDynamic, "", nil)

				return ""
			}

			return MissingBaseReason(base)
		}

		return "not found in extends chain"
	}

	return "no qualifier, not in file"
}

// bareCallInComponent reports whether an unqualified call is to a function the
// component has of its own or by inheritance, recording where it landed: this
// file, a function-reference local, the extends chain, the framework's virtual
// base, a WireBox delegate, or PresideSuperClass.
func (r *Resolver) bareCallInComponent(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) bool {
	funcName := call.FuncName

	for i := range pr.Funcs {
		if strings.EqualFold(pr.Funcs[i].Name, funcName) {
			tr.hit(TargetSameFile, "", &pr.Funcs[i])
			tr.addf("found %q defined in this file", funcName)

			return true
		}
	}

	// A bare call whose name matches a declared local variable or argument in
	// the enclosing function is a call through a function-reference value (e.g.
	// "var columnfn = ARGUMENTS.extend[x].column; ... columnfn(...)"), not a call
	// to a missing/undefined function — CFML supports first-class function
	// values assigned to locals/arguments. This is a genuinely different case
	// from "no qualifier, not in file": the identifier IS declared, its value is
	// just dynamic (unknown until runtime), so — same reasoning as the
	// ARGUMENTS.x-as-function-reference case below — there's nothing further to
	// verify statically, and it should not be reported as if it were a missing
	// function.
	if scope := pr.FuncScopeAt(int(call.Line)); scope.Start != -1 {
		if pr.FuncDeclares(scope.Start, scope.End, funcName) {
			tr.addf("%q is a declared local variable/argument in the enclosing function — treating as a call through a function-reference value, not a missing function", funcName)
			tr.hit(TargetDynamic, "", nil)

			return true
		}
	}

	// Check extends chain
	if r.fileExtends(pr) != "" {
		tr.addf("not in this file — checking extends chain (%s)", r.fileExtends(pr))

		if def := r.inheritedFunc(pr, funcName, baseDir); def != nil {
			tr.hit(TargetExtends, r.fileExtends(pr), def)
			tr.addf("found %q in extends chain", funcName)

			return true
		}
	}

	if def, base := r.virtualBaseFunc(pr, funcName); def != nil {
		tr.hit(TargetExtends, base, def)
		tr.addf("found %q on %s, which the framework gives this component by virtual inheritance", funcName, base)

		return true
	}

	if def := r.delegatedFunc(pr.Delegates, baseDir, funcName, 0); def != nil {
		tr.hit(TargetExtends, "", def)
		tr.addf("found %q among the methods a WireBox delegate gives this component", funcName)

		return true
	}

	if pr.URI.IsFile() && strings.HasPrefix(funcName, "$") {
		if def := r.presideSuperClassFunc(r.extendsChainOf(pr.URI.Path()), funcName, 0); def != nil {
			tr.hit(TargetExtends, "", def)
			tr.addf("found %q on PresideSuperClass, which Preside's WireBox gives a presideService component", funcName)

			return true
		}
	}

	return false
}

// bareCallThroughContext reports whether an unqualified call is to a function
// the component reaches from outside itself, recording where it landed: a
// cfinclude, a Wheels host or include wrapper, a variables-scope function
// reference, a module helper, an onMissingMethod, or what every subclass
// declares.
func (r *Resolver) bareCallThroughContext(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) bool {
	funcName := call.FuncName

	if def, via := r.findThroughIncludes(pr, funcName); def != nil {
		tr.hit(TargetInclude, "", def)
		tr.addf("found %q through cfinclude, in %s", funcName, via)

		return true
	}

	if def := r.wheelsMixinHostFunc(pr, funcName, call.Caller); def != nil {
		tr.hit(TargetExtends, "", def)
		tr.addf("found %q on the Wheels host that integrates this component's public methods", funcName)

		return true
	}

	if def := r.wheelsWrappedTemplateFunc(pr, funcName); def != nil {
		tr.hit(TargetInclude, "", def)
		tr.addf("found %q on the component that includes this template through a Wheels include wrapper", funcName)

		return true
	}

	// CFML looks a bare name up in the variables scope, so a bare call to
	// one the file assigns there — VARIABLES.render = ARGUMENTS.render —
	// is a call through that function-reference property, the same as the
	// qualified VARIABLES.render() accepted below. The parser records
	// VARIABLES.name() as a bare call wherever it reads it as a member of
	// this component, which is what left `#VARIABLES._renderTemplate()#`
	// in a string "no qualifier, not in file".
	if pr.HasScopedAssignment(parser.ScopeVariables, funcName) {
		tr.addf("%q is assigned in VARIABLES scope — a call through a function-reference property", funcName)
		tr.hit(TargetDynamic, "", nil)

		return true
	}

	if module, ok := r.moduleHelper(pr, funcName, baseDir); ok {
		tr.addf("%q is a helper the %s module mixes into ColdBox components", funcName, module)
		tr.hit(TargetDynamic, "", nil)

		return true
	}

	if r.thisCallAnswered(call, pr, baseDir) {
		tr.addf("%q is called through this., and the component answers a missing method with onMissingMethod", funcName)
		tr.hit(TargetDynamic, "", nil)

		return true
	}

	if comps := r.subclassesDeclare(pr, funcName, tr); comps != "" {
		tr.hit(TargetDynamic, comps, nil)

		return true
	}

	return false
}

// resolveBareChain is canResolveCall for a call chained onto a bare one,
// `expect( x ).toBe( 1 )`: the first call is looked up as a bare call is —
// this file, its extends chain, what it includes — and the rest of the chain
// is checked on what it returns.
//
// It was resolved as a bare call to the *last* name, which looked for toBe
// among the spec's own methods and reported it "not found in extends chain"
// — once the chain resolved, in every TestBox assertion.
func (r *Resolver) resolveBareChain(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	first := parser.CallHopName(call.Chain[0])

	// The hop carries its arguments when the parser read them. Recovering
	// them from the line instead finds nothing when the name is called
	// twice there.
	expression, ok := parser.CallExpression(call.Chain[0])
	if !ok {
		expression = factoryCallExpression(pr.Content, first, int(call.Line))
	}

	tr.addf("chained on a call to %q — looking it up as an unqualified call", first)

	def := r.bareFunc(first, pr, baseDir, call.Caller)
	if def == nil && call.This && pr.URI.IsFile() {
		if e := r.slatwallGatewayReturn(pr.URI.Path(), call.Chain[0], baseDir); e != "" {
			tr.addf("this.%s() is answered by Slatwall's BaseService onMissingMethod, which returns %q", first, e)

			rest := *call
			rest.Chain = call.Chain[1:]
			rest.Component = e

			return r.canResolveCall(&rest, pr, baseDir, tr)
		}
	}

	if def == nil {
		if answer, ok := r.unboundChainHop(first, expression, call, pr, baseDir, tr); ok {
			return answer
		}

		return "chained on '" + first + "', which is not found (calling '" + call.FuncName + "')"
	}

	ret, noFollow, soft := r.chainHopReturn("this component", first, def, tr)

	// A method returning the class declaring it, called on this subclass,
	// returns the subclass: as hopReturn does for a qualified call.
	if sub := r.selfTyped(pr.URI.Path(), baseDir, def, ret); sub != "" {
		ret = sub
	}

	if ret == "" && expression != "" {
		ret = r.expressionReturn(def, expression, baseDir, pr.URI.Path())
	}

	if ret == "" {
		if e := r.receiverReturn(cfpath.FromURI(string(pr.URI)), first); e != "" {
			tr.addf("%q on this component returns what it binds it to: %q", first, e)
			ret = e
		}
	}

	if noFollow && ret != "" {
		tr.hit(TargetDynamic, ret, nil)

		return ""
	}

	// A dynamicIfMissing resolver's answer that names no file is dynamic here
	// as on every other path; recursing lost the flag, and the next hop
	// reported the component as one that does not exist.
	if soft && ret != "" && !r.componentExists(ret, baseDir) {
		tr.addf("%q names no file and came from a dynamicIfMissing resolver — the rest of the chain is dynamic", ret)
		tr.hit(TargetDynamic, ret, nil)

		return ""
	}

	if ret == "" {
		if reason, done := r.closureStructHops(def, pr.URI.Path(), first, call, 0, tr); done {
			return reason
		}

		ret = r.initArgGetter(def)
		if ret != "" {
			tr.addf("%q returns the property init stores from its argument, which every construction passes as %q", first, ret)
		}
	}

	if ret == "" {
		if r.engineValueReturn(def, pr.URI.Path()) {
			tr.addf("%q returns what a built-in function hands back — the rest of the chain is dynamic", first)
			tr.hit(TargetDynamic, "", nil)

			return ""
		}

		return "method '" + first + "' has no component return type (chain to '" + call.FuncName + "')"
	}

	rest := *call
	rest.Chain = call.Chain[1:]
	rest.Component = ret

	return r.canResolveCall(&rest, pr, baseDir, tr)
}

// bareFunc finds what an unqualified call to name reaches: this file's own
// function, one its extends chain declares, or one a file it includes or is
// included by declares.
func (r *Resolver) bareFunc(name string, pr *parser.ParseResult, baseDir string, caller ...string) *parser.FunctionDef {
	for i := range pr.Funcs {
		if strings.EqualFold(pr.Funcs[i].Name, name) {
			return &pr.Funcs[i]
		}
	}

	if r.fileExtends(pr) != "" {
		if def := r.inheritedFunc(pr, name, baseDir); def != nil {
			return def
		}
	}

	// The same order a call standing alone is looked up in (canResolveCall):
	// a chain headed by $getPresideObject() or a handler's getModel() found
	// nothing where the call alone resolved.
	if def, _ := r.virtualBaseFunc(pr, name); def != nil {
		return def
	}

	if def := r.delegatedFunc(pr.Delegates, baseDir, name, 0); def != nil {
		return def
	}

	if pr.URI.IsFile() && strings.HasPrefix(name, "$") {
		if def := r.presideSuperClassFunc(r.extendsChainOf(pr.URI.Path()), name, 0); def != nil {
			return def
		}
	}

	if def, _ := r.findThroughIncludes(pr, name); def != nil {
		return def
	}

	if def := r.wheelsMixinHostFunc(pr, name, caller...); def != nil {
		return def
	}

	return r.wheelsWrappedTemplateFunc(pr, name)
}

// resolveThisCall is canResolveCall for `this.name()`.
func (r *Resolver) resolveThisCall(funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	tr.addf("'this' qualifier — checking same file")

	for i := range pr.Funcs {
		if strings.EqualFold(pr.Funcs[i].Name, funcName) {
			tr.hit(TargetSameFile, "", &pr.Funcs[i])

			return ""
		}
	}

	if r.fileExtends(pr) != "" {
		tr.addf("not in this file — checking extends chain (%s)", r.fileExtends(pr))

		if def := r.inheritedFunc(pr, funcName, baseDir); def != nil {
			tr.hit(TargetExtends, r.fileExtends(pr), def)

			return ""
		}
	}

	if def := r.delegatedFunc(pr.Delegates, baseDir, funcName, 0); def != nil {
		tr.hit(TargetExtends, "", def)
		tr.addf("found %q among the methods a WireBox delegate gives this component", funcName)

		return ""
	}

	if def, via := r.findThroughIncludes(pr, funcName); def != nil {
		tr.hit(TargetInclude, "", def)
		tr.addf("found %q through cfinclude, in %s", funcName, via)

		return ""
	}

	if comps := r.subclassesDeclare(pr, funcName, tr); comps != "" {
		tr.hit(TargetDynamic, comps, nil)

		return ""
	}

	if r.fileExtends(pr) != "" {
		if base, implied := r.fileMissingBase(pr, baseDir); base != "" {
			if implied {
				tr.addf("the framework base %s implies does not resolve (%s) — accepted as dynamic", r.fileExtends(pr), base)
				tr.hit(TargetDynamic, "", nil)

				return ""
			}

			return MissingBaseReason(base)
		}
	}

	return "method '" + funcName + "' not found in current component"
}

// resolveSuperCall is canResolveCall for `super.name()`.
func (r *Resolver) resolveSuperCall(funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	if r.fileExtends(pr) == "" {
		return "super used but no extends"
	}

	tr.addf("'super' qualifier — checking extends chain (%s)", r.fileExtends(pr))

	if def := r.ResolveFunc(r.fileExtends(pr), funcName, baseDir); def != nil {
		tr.hit(TargetExtends, r.fileExtends(pr), def)

		return ""
	}

	if base, implied := r.fileMissingBase(pr, baseDir); base != "" {
		if implied {
			tr.addf("the framework base %s implies does not resolve (%s) — accepted as dynamic", r.fileExtends(pr), base)
			tr.hit(TargetDynamic, "", nil)

			return ""
		}

		return MissingBaseReason(base)
	}

	return "not found in parent component"
}

// missingBasePrefix starts MissingBaseReason's answer, which MissingBaseOf
// reads back.
const missingBasePrefix = "extends chain breaks at "

// missingBaseSuffix closes MissingBaseReason's answer; MissingBaseOf requires
// the same text.
const missingBaseSuffix = "', which does not resolve"

// MissingBaseReason is canResolveCall's answer for an inherited call when a
// component in the file's extends chain does not resolve. The method may well
// be declared there; what is missing is the base, and a report that blames the
// call names the wrong problem once per call — 56,000 times over the six-project
// corpus, for a dozen missing bases.
func MissingBaseReason(base string) string {
	return missingBasePrefix + "'" + base + missingBaseSuffix
}

// MissingBaseOf reports the base a MissingBaseReason names.
func MissingBaseOf(reason string) (string, bool) {
	rest, ok := strings.CutPrefix(reason, missingBasePrefix+"'")
	if !ok {
		return "", false
	}

	if !strings.HasSuffix(rest, missingBaseSuffix) {
		return "", false
	}

	base := strings.TrimSuffix(rest, missingBaseSuffix)
	// Every producer of MissingBaseReason guards base != "", so ok means a
	// base was found, never none.
	return base, base != ""
}

// inheritedFromMissingBase is the base a receiver with no component is taken
// to come from: the missing link of the file's extends chain, when the
// receiver is an unscoped or variables./this. name the file never declares.
// `print` in a CommandBox command and `$assert` in a TestBox spec are the
// base's, injected or declared there, and with the base missing they are as
// unchecked as its methods. A name in any other scope, or one the file
// declares, is not the base's to explain.
func (r *Resolver) inheritedFromMissingBase(variable string, line uint32, pr *parser.ParseResult, baseDir string) (base string, implied bool) {
	if r.fileExtends(pr) == "" || variable == "" || strings.ContainsAny(variable, "[(") {
		return "", false
	}

	root, rest, _ := strings.Cut(variable, ".")
	if strings.EqualFold(root, "variables") || strings.EqualFold(root, "this") {
		root, _, _ = strings.Cut(rest, ".")
	} else if isScopeName(root) {
		return "", false
	}

	if root == "" || pr.Declares(root) {
		return "", false
	}

	// A closure's locals and parameters and a catch variable are the
	// enclosing function's to the body scan, and not in the file-wide
	// declarations.
	if scope := pr.FuncScopeAt(int(line)); scope.Start != -1 {
		if pr.FuncDeclares(scope.Start, scope.End, root) {
			return "", false
		}
	}

	return r.fileMissingBase(pr, baseDir)
}

// isScopeName reports whether name is a CFML scope other than variables and
// this, whose members a base component does not supply.
func isScopeName(name string) bool {
	switch strings.ToLower(name) {
	case "arguments", "local", "request", "session", "application", "server",
		"url", "form", "cgi", "cookie", "client", "super", "attributes", "caller", "thread":
		return true
	default:
		return false
	}
}

// MissingBase returns the first component in the extends chain starting at
// extends, from a file in baseDir, that does not resolve to a file; "" when
// every link resolves, or the chain loops.
func (r *Resolver) MissingBase(extends, baseDir string) string {
	base, _ := r.chainBreak(extends, baseDir, false)

	return base
}

// chainBreak walks the extends chain from extends and names the first link
// that does not resolve, and whether any link up to it was one a framework
// preset implied rather than one a file wrote — implied says so of the first.
// A list of alternatives is a component a file cannot declare, so it answers
// nothing, unless it was implied: a Wheels view's base is its controller and
// every mixin, and any of them missing is the framework's source missing.
func (r *Resolver) chainBreak(extends, baseDir string, implied bool) (string, bool) {
	seen := make(map[string]bool)

	for extends != "" {
		if strings.Contains(extends, "|") {
			if !implied {
				return "", false
			}

			for alt := range strings.SplitSeq(extends, "|") {
				if base, _ := r.chainBreak(alt, baseDir, true); base != "" {
					return base, true
				}
			}

			return "", false
		}

		p := r.ComponentPath(extends, baseDir)
		if p == "" {
			return extends, implied
		}

		if seen[p] {
			return "", false
		}

		seen[p] = true

		// A chain that reaches a framework's stubs and breaks beyond them is
		// the framework's source not being here — BaseTestCase's TestBox base
		// in a workspace naming ColdBox but not TestBox — not a finding.
		implied = implied || frameworkapi.IsStub(p)

		declared, ok := r.declaredExtendsOf(p, cfpath.ToURI(p))
		if !ok {
			return "", false
		}

		extends, baseDir = declared, filepath.Dir(p)
		if declared == "" {
			extends = r.extendsFor("", p)
			implied = implied || extends != ""
		}
	}

	return "", false
}

// matchResolver tries the componentResolvers against text. It reports the
// component, the resolver's noFollow, and whether that resolver is
// dynamicIfMissing; describe is told about a match, for the trace.
func (r *Resolver) matchResolver(text string, tr *callTrace, describe func(comp, desc string, noFollow bool)) (comp string, noFollow, soft bool) {
	comp, noFollow, idx := parser.ResolveFromCallMatch(text, r.Resolvers)
	soft = idx >= 0 && r.Resolvers[idx].DynamicIfMissing

	if comp != "" && tr != nil {
		describe(comp, r.describeResolver(idx), noFollow)
	}

	return comp, noFollow, soft
}

// missingChainHop is canResolveCall's answer when a hop of a chained call is
// not a method of the component the chain has reached.
func (r *Resolver) missingChainHop(comp, softComp, hop, funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	if !r.componentExists(comp, baseDir) {
		if softMissing(comp, softComp, pr) {
			tr.addf("%q names no file and came from a dynamicIfMissing resolver — the rest of the chain is dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		if r.uninstalledModule(comp) {
			tr.addf("%q is a module's model and no such module is in the workspace — the rest of the chain is dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		if _, ok := engineComponent(comp); ok {
			tr.addf("%q is a component the engine provides and its source is not in the workspace — the rest of the chain is dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		if frameworkapi.NamespaceOf(comp) != "" {
			tr.addf("%q is in the %s namespace and the bundled stubs do not hold it — the rest of the chain is dynamic", comp, frameworkapi.NamespaceOf(comp))
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		return "component '" + displayComponent(comp) + "' does not exist (chain hop '" + hop + "' to '" + funcName + "')"
	}

	if mockDecoration(hop) {
		tr.addf("%q is a method MockBox adds to a mock — chain hop accepted, the rest of the chain is dynamic", hop)
		tr.hit(TargetDynamic, comp, nil)

		return ""
	}

	// A hop onMissingMethod answers is as valid as a last call it answers, and
	// what it returns is whatever the dispatcher decides. Lucee's own Http and
	// Query build their setters this way: new Http().setUrl(u).send().
	if r.ResolveFunc(comp, "onMissingMethod", baseDir) != nil {
		tr.addf("%q defines onMissingMethod — chain hop %q accepted, the rest of the chain is dynamic", comp, hop)
		tr.hit(TargetDynamic, comp, nil)

		return ""
	}

	if r.isInterface(r.ComponentPath(comp, baseDir)) {
		tr.addf("%q is an interface, and what implements it may have %q — the rest of the chain is dynamic", comp, hop)
		tr.hit(TargetDynamic, comp, nil)

		return ""
	}

	if base, implied := r.componentMissingBase(comp, baseDir); base != "" {
		if implied {
			tr.addf("%q's framework base does not resolve (%s) — chain hop %q accepted, the rest of the chain is dynamic", comp, base, hop)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		return MissingBaseReason(base)
	}

	return "method '" + hop + "' not found in " + displayComponent(comp) + " (chain to '" + funcName + "')"
}

// componentMissingBase is MissingBase for the chain comp extends: a method
// comp does not declare may be its base's, and when a link of that chain
// names no file the method was never looked for. ContentBox's services extend
// cborm's VirtualEntityService, and without cborm on disk every findWhere,
// save and list was "not found in" the service.
func (r *Resolver) componentMissingBase(comp, baseDir string) (base string, implied bool) {
	if strings.Contains(comp, "|") {
		return "", false
	}

	p := comp
	if !filepath.IsAbs(p) {
		p = r.ComponentPath(comp, baseDir)
	}

	if p == "" {
		return "", false
	}

	declared, ok := r.declaredExtendsOf(p, cfpath.ToURI(p))
	if !ok {
		return "", false
	}

	ext := declared
	if ext == "" {
		ext = r.extendsFor("", p)
	}

	return r.chainBreak(ext, filepath.Dir(p), declared == "" && ext != "")
}

// checkMethodOn is canResolveCall's last step: whether the component the
// receiver resolved to defines the method, with the dynamic, builtin,
// member-method, onMissingMethod and altComp fallbacks.
func (r *Resolver) checkMethodOn(comp, softComp string, call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	funcName, variable := call.FuncName, call.Variable

	if comp == "" {
		return r.receiverlessReason(variable, call.Line, pr, baseDir, tr)
	}

	// A MockBox mock of a class: the class's methods, where the class
	// resolves, and dynamic where it does not — mocking a class of a module
	// the workspace lacks is not a finding.
	if cls, ok := strings.CutPrefix(comp, parser.MockPrefix); ok {
		if r.ComponentPath(cls, baseDir) == "" {
			tr.addf("a mock of %q, which does not resolve — accepted as dynamic", cls)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		tr.addf("a mock of %q — checking the class", cls)
		comp = cls
	}

	// Dynamic return type — method called on a result of a function returning "any".
	if comp == "$any" {
		tr.addf("component is $any (dynamic) — accepted without a method check")
		tr.hit(TargetDynamic, "$any", nil)

		return ""
	}

	if variable != "" && len(call.Chain) == 0 && pr.AssignsMember(variable, funcName, call.Line) {
		tr.hit(TargetDynamic, comp, nil)

		return ""
	}

	// A registry family: the receiver is any one of the package directory's
	// components, so the method must be each member's own, or its base's. A
	// receiver several families name — the container read through a
	// list-loop item's concatenation — holds each of them on one of the
	// loop's turns, so every family must answer the call.
	if refs := familyRefs(comp); len(refs) > 0 {
		return r.checkFamiliesMethod(refs, call, pr, tr)
	}

	// Builtin return type — check method exists in-memory
	if strings.HasPrefix(comp, "$builtin.") {
		builtinName := comp[9:]
		if docs.LookupBuiltinMethod(builtinName, funcName) {
			tr.hit(TargetBuiltin, comp, nil)

			return ""
		}

		return "method '" + funcName + "' not found on builtin " + builtinName
	}

	tr.addf("checking whether %q defines method %q", comp, funcName)

	if r.methodAccepted(comp, call, pr, baseDir, tr) {
		return ""
	}

	return r.missingMethod(comp, softComp, call, pr, baseDir, tr)
}

// receiverlessReason is the verdict on a call whose receiver no step typed:
// accepted as dynamic when it belongs to a framework base that does not
// resolve, the base's reason when the file's own base does not, else that the
// variable has no component ref.
func (r *Resolver) receiverlessReason(variable string, line uint32, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	if base, implied := r.inheritedFromMissingBase(variable, line, pr, baseDir); base != "" {
		if implied {
			tr.addf("%q is the framework base's, which does not resolve (%s) — accepted as dynamic", variable, base)
			tr.hit(TargetDynamic, "", nil)

			return ""
		}

		return MissingBaseReason(base)
	}

	return "variable '" + variable + "' has no component ref"
}

// methodAccepted reports whether comp, which does not make the receiver
// untyped, answers a call of the method, recording where: a method it declares
// or inherits, an assertion a test setup installs, one MockBox adds, a member
// method every value has, onMissingMethod, an interface (which declares less
// than what implements it), or the component a resolver names by the variable.
func (r *Resolver) methodAccepted(comp string, call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) bool {
	funcName, variable := call.FuncName, call.Variable

	if def := r.ResolveFunc(comp, funcName, baseDir); def != nil {
		tr.hit(TargetComponent, comp, def)

		return true
	}

	if def := r.registeredAssertion(comp, call, pr, baseDir); def != nil {
		tr.addf("%q is a literal assertion installed by the selected beforeTests setup", funcName)
		tr.hit(TargetComponent, comp, def)

		return true
	}

	if mockDecoration(funcName) {
		tr.addf("%q is a method MockBox adds to a mock of %q — accepted", funcName, comp)
		tr.hit(TargetDynamic, comp, nil)

		return true
	}

	if parser.IsMemberMethod(funcName) {
		tr.addf("%q is a known member/Java-interop method — accepted without finding it in %q", funcName, comp)
		tr.hit(TargetMember, comp, nil)

		return true
	}

	// Component defines onMissingMethod — any method call is valid.
	if r.ResolveFunc(comp, "onMissingMethod", baseDir) != nil {
		tr.addf("%q defines onMissingMethod — any method call accepted", comp)
		tr.hit(TargetDynamic, comp, nil)

		return true
	}

	// An interface declares less than the object behind it: ColdBox types
	// its cache providers as ICacheProvider and calls getOrSet() on them,
	// which every provider has and the interface does not declare. A method
	// the interface does declare was found above.
	if r.isInterface(r.ComponentPath(comp, baseDir)) {
		tr.addf("%q is an interface, and what implements it may have %q — accepted as dynamic", comp, funcName)
		tr.hit(TargetDynamic, comp, nil)

		return true
	}

	// The ref-derived component didn't have the method. Try the variable-name resolver
	// as a fallback — a pendingCall propagation may have assigned the wrong component
	// (e.g. var objFile = _parent.getFile() inherits _parent's component, but the
	// "objFile" resolver names the real type).
	if altComp, altNoFollow := parser.ResolveFromCallFull(variable, r.Resolvers); altComp != "" && altComp != comp {
		tr.addf("%q not found in %q — trying altComp fallback: componentResolver on variable name gives %q (noFollow=%v)", funcName, comp, altComp, altNoFollow)

		if altNoFollow {
			tr.hit(TargetDynamic, altComp, nil)

			return true
		}

		if altDef := r.ResolveFunc(altComp, funcName, baseDir); altDef != nil {
			tr.hit(TargetComponent, altComp, altDef)

			return true
		}
	}

	return false
}

// missingMethod is checkMethodOn's answer once comp is known not to declare
// funcName: a component that is not there, a base that is not, or a method
// the code checks for or that comp lacks.
func (r *Resolver) missingMethod(comp, softComp string, call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	funcName, variable := call.FuncName, call.Variable

	// A component that names no file is a different finding from a method a
	// real component lacks, and reporting it as the second hid it. It is
	// almost always a componentResolver producing a path — a broad get$1()
	// catch-all turning getInjectorController() into tass.injectorcontroller —
	// so saying so points at the config rather than at the code.
	if !r.componentExists(comp, baseDir) {
		// Unless a resolver marked dynamicIfMissing produced it: that is a
		// broad pattern's guess, and a guess that names nothing says the value
		// is not one of the components it was written for — not that one is
		// missing. A component named any other way is still reported.
		if softMissing(comp, softComp, pr) {
			tr.addf("%q names no file and came from a dynamicIfMissing resolver — accepted as dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		// Nor a model of a module the workspace does not have: that is the
		// module not being installed.
		if r.uninstalledModule(comp) {
			tr.addf("%q is a module's model and no such module is in the workspace — accepted as dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		// Nor a class in a framework's own namespace the bundled stubs lack:
		// the stubs hold the API the presets need, so their silence is not
		// the framework's.
		if frameworkapi.NamespaceOf(comp) != "" {
			tr.addf("%q is in the %s namespace and the bundled stubs do not hold it — accepted as dynamic", comp, frameworkapi.NamespaceOf(comp))
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		// Nor one the engine provides: `new Query()` is Lucee's or Adobe's
		// own component, whose methods are the engine's to answer for.
		if _, ok := engineComponent(comp); ok {
			tr.addf("%q is a component the engine provides and its source is not in the workspace — accepted as dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		return "component '" + displayComponent(comp) + "' does not exist (calling '" + funcName + "')"
	}

	if base, implied := r.componentMissingBase(comp, baseDir); base != "" {
		if implied {
			tr.addf("%q's framework base does not resolve (%s) — accepted as dynamic", comp, base)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		return MissingBaseReason(base)
	}

	// Nor one a type check before the call says the object has.
	for _, c := range guardedInstanceOf(pr, variable, funcName, int(call.Line)) {
		if def := r.ResolveFunc(c, funcName, baseDir); def != nil {
			tr.addf("%q is called only after the code checks %q is a %q, which declares it", funcName, variable, c)
			tr.hit(TargetComponent, c, def)

			return ""
		}
	}

	// Nor a method the code checks for before calling it.
	if guardedByExistsCheck(pr, variable, funcName, int(call.Line)) {
		tr.addf("%q is called only after the code checks %q has it — accepted as dynamic", funcName, variable)
		tr.hit(TargetDynamic, comp, nil)

		return ""
	}

	return "method '" + funcName + "' not found in " + displayComponent(comp)
}

// displayComponent is how a reason names comp. A component named by its file
// — what a function returning `this` returns — is named by the file's name:
// a reason is written into a known-issues file that is committed and read on
// other machines, where an absolute path would not mean anything.
func displayComponent(comp string) string {
	if strings.Contains(comp, "|") {
		// Each alternative of a union, which may mix a file and a dot-path.
		alts := strings.Split(comp, "|")
		for i, alt := range alts {
			alts[i] = displayComponent(alt)
		}

		return strings.Join(alts, "|")
	}

	comp = strings.TrimPrefix(comp, parser.EntityPrefix)

	if filepath.IsAbs(comp) {
		return strings.TrimSuffix(filepath.Base(comp), filepath.Ext(comp))
	}

	return comp
}

// sharedGetterReturn is what a function whose whole body is `return
// application.x` (or another shared scope) returns: what the startup
// templates assign x, the type it already has as a receiver. Mura's
// getPluginManager() and getServiceFactory() are this. Anything else in the
// body withholds it, a call above all, since a call can write the scope:
// TestAbsentOptionalArgumentBoundaries holds the guarded and computed shapes
// to no answer.
func (r *Resolver) sharedGetterReturn(fd *parser.FunctionDef) string {
	method := r.producerFor(fd)
	if method == nil || len(method.body) != 1 || method.body[0].kind != "return" {
		return ""
	}

	path := producerPath(producerUnwrap(producerTokens(method.body[0].expression)))
	if _, shared := splitSharedScope(path); !shared {
		return ""
	}

	dir := filepath.Dir(fd.URI.Path())

	comp := r.startupComponent(path, dir, nil)
	if comp == "" || strings.HasPrefix(comp, "$") {
		return ""
	}

	if p := r.ComponentPath(comp, dir); p != "" {
		return p
	}

	return comp
}

func (r *Resolver) declaredReturnType(fd *parser.FunctionDef) string {
	if path := r.besideDeclaring(fd, fd.ReturnType); path != "" {
		return path
	}

	if native := nativetype.Java(fd.ReturnType); native != "" {
		return native
	}

	return fd.ReturnType
}

// returnViaComponent finishes what the index's parse could not: fd returns
// a variable last assigned `x = recv.m()`, recv holding fd.ReturnVia's
// component, and fd returns what m returns there — "$any" included, as the
// parse would give it with FuncLookup. tassweb's itext.createParagraph
// returns `VARIABLES._itextUtils.createParagraph( … )`, whose branches return
// a string or one of several paragraphs. Depth and budget are carried, so
// two functions returning each other's calls end rather than recurse.
func (r *Resolver) returnViaComponent(fd *parser.FunctionDef, depth int, budget *int) string {
	via := fd.ReturnVia
	if via == nil || !fd.URI.IsFile() || depth >= 8 || *budget <= 0 {
		return ""
	}

	*budget--

	target := r.ResolveFunc(via.Component, via.Method, filepath.Dir(fd.URI.Path()))
	if target == nil {
		return ""
	}

	return r.returnComponentOf(target, depth+1, budget)
}

// unevaluatedReturn is what fd returns when evaluating its body gave no
// answer: what the call it returns returns (returnViaComponent), the element
// of a collection it returns (elementReturn), or "$any"
// when the parse found a branch returning no component (a string or an
// argument passed through), which no choice of the caller's arguments makes
// a component on every path.
func (r *Resolver) unevaluatedReturn(fd *parser.FunctionDef, depth int, budget *int) string {
	if ret := r.returnViaComponent(fd, depth, budget); ret != "" {
		return ret
	}

	if ret := r.elementReturn(fd); ret != "" {
		return ret
	}

	if fd.ReturnsUntyped && fd.ReturnComponent == "$any" {
		return "$any"
	}

	return ""
}

// nativeArgumentFamily is the nativetype family a declared argument type
// names, or "" for a component or a type with no contract.
func nativeArgumentFamily(declared string) string {
	switch t := strings.ToLower(strings.TrimSpace(declared)); t {
	case "query", "struct", "array", "string", "numeric", "boolean", "date", "datetime":
		if t == "datetime" {
			return "date"
		}

		return t
	}

	return ""
}

// topLevelLine reports whether line is outside every function of pr: a
// template's own code, whose assignments run in the order they are written.
func topLevelLine(pr *parser.ParseResult, line uint32) bool {
	for i := range pr.Scopes {
		if int(line) >= pr.Scopes[i].Start && int(line) <= pr.Scopes[i].End {
			return false
		}
	}

	return true
}

// namedRefs is a lookup of the refs for name among refs, run only if the union
// of reaching assignments needs them.
func namedRefs(refs []parser.ComponentRef, name string) func() []*parser.ComponentRef {
	return func() []*parser.ComponentRef {
		var found []*parser.ComponentRef

		for i := range refs {
			if strings.EqualFold(refs[i].Variable, name) {
				found = append(found, &refs[i])
			}
		}

		return found
	}
}
