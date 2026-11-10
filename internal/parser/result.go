package parser

import (
	"cmp"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/log"
	"go.lsp.dev/uri"
)

// Logger is an optional interface for parse diagnostics.
type Logger = log.Logger

// ParseResult caches a single parse of a file. It extracts function signatures
// and component refs eagerly, but defers function body parsing until requested.
type ParseResult struct {
	imports map[string]string // from the file's `import`, for a function body parsed later

	URI                 uri.URI
	Content             string
	Regions             []Region
	Funcs               []FunctionDef
	ComponentRefs       []ComponentRef
	Scopes              []FuncScope
	Extends             string                                  // dot-path of parent component (from extends attribute)
	Accessors           bool                                    // component has accessors=true
	Persistent          bool                                    // true if component has persistent="true" (ORM entity)
	Properties          []propertyDef                           // parsed property declarations
	Delegates           []Delegate                              // WireBox delegations the component declares
	Links               []DocumentLink                          // file path references extracted during shallow scan
	Calls               []CallSite                              // function call sites (when FindCalls is set)
	Instantiations      []Instantiation                         // components built, with calls extracted; a fresh parse's lines
	log                 Logger                                  // optional logger for timing and errors
	Resolvers           []Resolver                              // optional component resolvers for RHS matching
	resolverSet         *ResolverSet                            // pre-grouped resolvers for fast matching
	PropertyResolvers   []PropertyResolver                      // optional property-to-component resolvers
	managedSetterLookup func(string) string                     // current property metadata applied to setter injection
	ConstructorLookup   func(string) string                     // known factory constructor dependency identity
	SetterLookup        func(string) string                     // managed bean setter dependency → component
	BeanLookup          func(string) string                     // optional bean name → dot-path lookup
	PropertyBeanLookup  func(string, map[string]string) string  // optional factory property eligibility and identity
	BuiltinReturnLookup func(string) string                     // optional: builtin function → return component
	FuncLookup          func(component, funcName string) string // optional: resolve method return type from external components
	expressionMappings  map[string]string                       // runtime expression → static value substitutions
	expressionKeys      []string                                // expressionMappings' keys, in the order they apply
	// ServicePropertyResolvers maps a "@serviceproperty" annotation kind (e.g. "package",
	// "service", "controller") to a dot-path template containing "${name}". Lets a project
	// document the real component type of a generically-typed (e.g. <cfargument type="struct">)
	// constructor-injected dependency via a "<!--- @serviceproperty varName kind|name --->"
	// comment at each use site, instead of (or in addition to) a componentResolver.
	ServicePropertyResolvers map[string]string
	extractLinks             bool        // whether to extract links during global scan
	extractCalls             bool        // whether to extract all call sites during parsing
	findCalls                []string    // function names to scan for
	scanAllScopes            bool        // scan all lines including function bodies
	shallow                  bool        // minimal parse mode
	interpolateAll           bool        // ParseOptions.InterpolateAllText
	outputScanned            bool        // importPrefixes is computed
	output                   outputState // where the tag walk stands in the output contexts
	importPrefixes           []string    // <cfimport prefix="..."> values, lowercased

	// contentLineIdx is the line index ClassifyRegions built over Content on its
	// way to the regions, kept so extractSignatures does not build a second one
	// over the same string. nil for a script file, which never needs it.
	contentLineIdx []int32

	// Lazy global var caches (protected by mu).
	mu            sync.Mutex
	globalVars    []string
	variablesVars []string
	thisVars      []string
	scopedDone    bool // the three above, from one scan
	// tagScopeVars is, for each tag region by its offset, the variables- and
	// this-scope declarations the parse's own walk recorded there, kept so
	// computeScopedVars need not walk the region again; tagScopeVarsOf is the
	// Content they were read from, since an edit inside a function moves lines
	// without reparsing the regions.
	tagScopeVars   map[int][]VarDef
	tagScopeVarsOf string
	// collectionTags is, for each tag region by its offset, where the parse's
	// walk met the tags collectWrites reads, valid while tagScopeVarsOf is.
	collectionTags map[int][]int
	// textLinks is DocumentLinks' answer, for the content textLinksOf.
	textLinks   []DocumentLink
	textLinksOf string
	// regionsOf is the Content Regions and contentLineIdx were split from: an
	// edit inside a function changes Content and leaves them.
	regionsOf string
	// customTags is CustomTagCalls' answer, for the content customTagsOf and
	// the prefixes customTagsKey.
	customTags    []CustomTagCall
	customTagsOf  string
	customTagsKey string
	allVars       []VarDef
	declareScans  map[string]bool // mayDeclare's answers, by lowercased name
	allVarsDone   bool

	// anyScopedVars caches, per Scope, every name ever assigned in that scope
	// anywhere in the file (see HasScopedAssignment) — unlike variablesVars/thisVars,
	// which only look outside functions and inside init().
	anyScopedVars map[Scope][]string

	// memberSets are the `a.m = …` assignments met; see AssignsMember.
	memberSets []pendingCall
	// memberSetsByName indexes memberSets by lowercased member name, built on
	// AssignsMember's first question and rebuilt when memberSets has changed
	// length; a reparse that resets memberSets resets it too. memberSetsMu
	// guards it.
	memberSetsMu     sync.Mutex
	memberSetsByName map[string][]int
	memberSetsCount  int

	// refsByName indexes ComponentRefs by lowercased variable name, built on
	// RefsNamed's first question and rebuilt when ComponentRefs is no longer
	// the slice it was built from (refsBuiltFrom, refsBuiltLen). refsMu
	// guards it.
	refsMu        sync.Mutex
	refsByName    map[string][]int
	refsBuiltFrom *ComponentRef
	refsBuiltLen  int

	// funcVars caches per-function variable lists keyed by "start:end".
	funcVarsMu sync.Mutex
	funcVars   map[string][]string
	// funcVarSets holds each function's funcVars as a set, for FuncDeclares,
	// with the list it was built from so a list replaced is noticed.
	funcVarSets map[string]funcVarSet

	// callsByName indexes the calls AllCalls returns by lowercased name, for
	// CallsNamed: where each is, in Calls (key "") or a funcCallsMap bucket.
	// callsMu guards it; callsFrom, callsLen and callsBuckets are what it was
	// built from. A reparse resets it.
	callsMu      sync.Mutex
	callsByName  map[string][]callAt
	callsFrom    *CallSite
	callsLen     int
	callsBuckets int

	// memoMu guards memo, answers a caller derived from this parse and keyed
	// as it chose (see Memo). Every edit clears it (resetGlobalCaches).
	memoMu sync.Mutex
	memo   map[string]any

	// scopeAtMu guards scopeAt, FuncScopeAt's index over Scopes, and the
	// Scopes it was built from.
	scopeAtMu    sync.Mutex
	scopeAt      func(line int) int
	scopeAtFrom  *FuncScope
	scopeAtCount int

	// funcRefs caches per-function component refs keyed by "start:end".
	memberSnapshot        *ParseResult // member refs computed once per document version
	memberSnapshotContent string
	// refIdx is firstRef's index while resolvePendingCalls runs, nil
	// otherwise; see firstRef.
	refIdx       map[*ComponentRef]*refNames
	funcRefsMu   sync.Mutex
	funcRefsMap  map[string][]ComponentRef
	funcLinksMap map[string][]DocumentLink
	funcCallsMap map[string][]CallSite // per-function call sites keyed by "start:end"
	// flow is the blocks a full parse recorded, for comparing the branches
	// that reach a return; nil once resolvePendingCalls has read it.
	flow *flowBlocks

	// writesMemo, while set, holds collectionWrites' answer for the passes
	// in extractSignatures that share it.
	writesMemo *writesMemo

	// tagWrites keeps the collection writes read from each <cfset> from one
	// signature pass to the next; see expressionWriteCache.
	tagWrites *expressionWriteCache

	// lineStarts holds the byte offset of each line of lineStartsContent, so
	// a function body is sliced without walking the file from its start.
	lineStartsMu      sync.Mutex
	lineStarts        []int
	lineStartsContent string
}

// ParseOptions configures optional parse behaviour.
type ParseOptions struct {
	Logger    Logger
	Resolvers []Resolver
	// ResolverSet, built from Resolvers, lends a parse its index and memo of
	// answers (ResolverSet.ForParse), so the parses of one workspace share
	// them. Optional; a set built from another slice is ignored.
	ResolverSet              *ResolverSet
	PropertyResolvers        []PropertyResolver
	ConstructorLookup        func(name string) string // optional, only for known factory construction
	SetterLookup             func(name string) string // optional, only for factory-managed components
	BeanLookup               func(name string) string // optional: resolve bean name → dot-path
	PropertyBeanLookup       func(name string, attrs map[string]string) string
	BuiltinReturnLookup      func(name string) string                // optional: resolve builtin function → return component
	FuncLookup               func(component, funcName string) string // optional: resolve method return type from external components
	ExpressionMappings       map[string]string                       // runtime expression → static value substitutions
	ServicePropertyResolvers map[string]string                       // "@serviceproperty" annotation kind → dot-path template
	ExtractLinks             bool                                    // extract document links during global scan
	ExtractCalls             bool                                    // extract all variable.method() call sites during parsing
	FindCalls                []string                                // function names to find call sites for
	ScanAllScopes            bool                                    // scan all lines including function bodies (for refs/deps)
	Shallow                  bool                                    // minimal parse: signatures only, no refs/properties/args
	// InterpolateAllText scans #...# in all of a tag file's text, as the
	// parser did before it learned where ColdFusion evaluates it (see
	// outputContext), and reads a tag-free .cfm template as CFScript. It is
	// the features.outputContextInterpolation switch turned off.
	InterpolateAllText bool
}

// Parse performs a full file parse: extracts function signatures, component refs,
// and function scopes. Function bodies are NOT parsed for variables until requested.
func Parse(fileURI uri.URI, content string, resolvers ...[]Resolver) *ParseResult {
	pr := &ParseResult{
		URI:      fileURI,
		Content:  content,
		funcVars: make(map[string][]string),
	}
	if len(resolvers) > 0 {
		pr.Resolvers = resolvers[0]
	}

	start := time.Now()
	pr.Regions, pr.contentLineIdx = pr.classifyRegions()
	pr.regionsOf = pr.Content
	pr.extractSignatures()
	pr.logDebug("parse", "uri", string(fileURI), "funcs", len(pr.Funcs), "refs", len(pr.ComponentRefs), "dur", time.Since(start))

	return pr
}

// classifyRegions is ClassifyRegionsIdx for this file, which also knows its
// name: a .cfm template with no CF tags that is HTML is text, not CFScript,
// unless InterpolateAllText asks for the old reading.
func (pr *ParseResult) classifyRegions() ([]Region, []int32) {
	if !pr.interpolateAll && isMarkupTemplate(string(pr.URI), pr.Content) {
		return splitCFScriptBlocks(pr.Content)
	}

	return ClassifyRegionsIdx(pr.Content)
}

// IsSoftComponent reports whether comp came, in this parse, from a resolver
// marked dynamicIfMissing: a component whose absence is not a finding.
//
// A framework object an injection names counts too, whichever file declared
// the property — a base class usually does: without the framework's source
// or its stubs, a call on one is not a finding.
func (pr *ParseResult) IsSoftComponent(comp string) bool {
	return pr != nil && pr.resolverSet.isSoft(comp) || isInjectedFrameworkComponent(comp)
}

// outputGate returns the output state the file's tag regions share, its
// cfimport prefixes, computed once, and whether the walk is gated at all.
//
// The gate is not only about calls. Outside an output context the walk finds
// the next tag with a plain search, where inside one it steps over a #...#
// span; ungated, a `#` in a <script> body pairs with one far below and the
// walk steps over real tags between them. A parse without call extraction
// lost Lucee's <cffunction name="doMore"> that way. Tracking the contexts is
// part of the walk now and costs next to nothing, so every parse gates.
func (pr *ParseResult) outputGate() (state *outputState, prefixes []string, gated bool) {
	if pr.interpolateAll {
		return nil, nil, false
	}

	if !pr.outputScanned {
		pr.importPrefixes = cfimportPrefixes(pr.Content)
		pr.outputScanned = true
	}

	return &pr.output, pr.importPrefixes, true
}

// ParseWithOptions performs a full file parse with extended options. opts is
// only read; nil means the zero options.
func ParseWithOptions(fileURI uri.URI, content string, opts *ParseOptions) *ParseResult {
	if opts == nil {
		opts = &ParseOptions{}
	}

	pr := &ParseResult{
		URI:                      fileURI,
		Content:                  content,
		funcVars:                 make(map[string][]string),
		log:                      opts.Logger,
		Resolvers:                opts.Resolvers,
		PropertyResolvers:        opts.PropertyResolvers,
		BeanLookup:               opts.BeanLookup,
		PropertyBeanLookup:       opts.PropertyBeanLookup,
		SetterLookup:             opts.SetterLookup,
		ConstructorLookup:        opts.ConstructorLookup,
		BuiltinReturnLookup:      opts.BuiltinReturnLookup,
		FuncLookup:               opts.FuncLookup,
		expressionMappings:       opts.ExpressionMappings,
		expressionKeys:           ExpressionMappingOrder(opts.ExpressionMappings),
		ServicePropertyResolvers: opts.ServicePropertyResolvers,
		extractLinks:             opts.ExtractLinks,
		extractCalls:             opts.ExtractCalls,
		findCalls:                opts.FindCalls,
		scanAllScopes:            opts.ScanAllScopes,
		shallow:                  opts.Shallow,
		interpolateAll:           opts.InterpolateAllText,
	}
	if len(pr.Resolvers) > 0 {
		if opts.ResolverSet.Serves(pr.Resolvers) {
			pr.resolverSet = opts.ResolverSet.ForParse()
		} else {
			pr.resolverSet = newPrivateResolverSet(pr.Resolvers)
		}
	}

	start := time.Now()
	pr.Regions, pr.contentLineIdx = pr.classifyRegions()
	pr.regionsOf = pr.Content
	pr.extractSignatures()
	pr.logDebug("parse", "uri", string(fileURI), "funcs", len(pr.Funcs), "refs", len(pr.ComponentRefs), "dur", time.Since(start))

	return pr
}

// extractSignatures does a shallow parse: function names/args, component refs, scopes.
// tagFunctionScopes are the scopes of the tag functions of the whole file, and
// whether it has a tag region at all.
//
// Pre-computed from the whole file (not per region) so a function whose body is
// interrupted by a nested <cfscript> block (which splits the file into separate
// Tag/Script/Tag... regions) still gets a correct end line and in-function
// tracking in every region it spans, not just the region containing its
// opening <cffunction> tag.
func (pr *ParseResult) tagFunctionScopes() (scopes []FuncScope, hasTagRegion bool) {
	for _, r := range pr.Regions {
		if r.Kind == RegionTag {
			return findTagFuncScopesIdx(pr.Content, 0, pr.contentLineIdx), true
		}
	}

	return nil, false
}

// mergeRegions parses each region of the file into the result, in file order,
// and returns the calls whose receivers the later passes still have to type.
func (pr *ParseResult) mergeRegions(tagScopes []FuncScope) []pendingCall {
	var allPendingCalls []pendingCall

	// The locals of a tag function a region split, carried to the regions
	// after the split.
	var open openLocals

	// The tag regions share one output state, in file order, from the top.
	pr.output = outputState{}
	pr.tagScopeVars, pr.tagScopeVarsOf, pr.collectionTags = nil, pr.Content, nil

	for _, r := range pr.Regions {
		// A RegionSkip is a literal <script> block, left out of the script
		// regions so its JavaScript is never fed to the CFScript scanner. Its
		// `#...#` spans are CFML all the same — a <script> body inside
		// <cfoutput> is where a page writes `var id = "#o.getContentID()#";` —
		// and dropping the region dropped those with it.
		//
		// It goes to the tag parser below, which needs no mode of its own:
		// findScriptSkipSpans only makes a span of a block that holds no `<cf`
		// tag at all, so the walk can find nothing in one *but* its
		// interpolation. A flag restricting it to the spans measured
		// identically over the corpus and against tag-shaped text inside a JS
		// string, which is the case it was written for — and which is not a
		// skip region, because it contains `<cf`.
		if r.Kind == RegionSkip && !pr.extractCalls {
			continue
		}

		if r.Kind == RegionScript {
			allPendingCalls = append(allPendingCalls, pr.mergeScriptRegion(&r, tagScopes, &open)...)
		} else {
			allPendingCalls = append(allPendingCalls, pr.mergeTagRegion(&r, tagScopes, &open)...)
		}
	}

	return allPendingCalls
}

// addTagScopes adds the tag functions' scopes the regions did not record. They
// are keyed by start line rather than found by a scan of pr.Scopes per
// candidate: both lists are one entry per function, so the pair was quadratic
// in a component's method count.
func (pr *ParseResult) addTagScopes(tagScopes []FuncScope) {
	seen := make(map[int]struct{}, len(pr.Scopes))
	for _, existing := range pr.Scopes {
		seen[existing.Start] = struct{}{}
	}

	for _, sc := range tagScopes {
		if _, dup := seen[sc.Start]; dup {
			continue
		}

		seen[sc.Start] = struct{}{}

		pr.Scopes = append(pr.Scopes, sc)
	}
}

// settleSignatures is the second half of a full parse, once every region is in:
// the expression mappings and service properties, the accessors and delegates
// properties generate, the member writes and factory and collection returns,
// and the calls left untyped.
func (pr *ParseResult) settleSignatures(allPendingCalls []pendingCall) {
	pr.applyExpressionMappings()
	pr.applyServiceProperties()
	pr.generatePropertyAccessors()
	pr.collectDelegates()
	pr.appendResolverRefs()
	// Both passes read the same writes, and nothing between them changes
	// the regions or scopes those are read from.
	pr.writesMemo = &writesMemo{}
	hasBindings := pr.hasMemberBinding(pr.Content)
	pr.applyMemberBindings(hasBindings)
	pr.applyFactoryReturnCalls(allPendingCalls)
	pr.applyCollectionReturns(allPendingCalls)
	pr.writesMemo.release()
	pr.writesMemo = nil
	pr.resolvePendingCalls(allPendingCalls)
	pr.flow = nil
	pr.applyChainedReturnLookup()

	if hasBindings {
		pr.memberSnapshot = &ParseResult{funcRefsMap: maps.Clone(pr.funcRefsMap)}
		pr.memberSnapshotContent = pr.Content
	}
}

func (pr *ParseResult) extractSignatures() {
	defer func() {
		if r := recover(); r != nil {
			log.Recovered(pr.log, "parse panic in extractSignatures", r, "uri", string(pr.URI))
		}
	}()

	pr.prepareManagedSetterLookup()

	if pr.FuncLookup != nil {
		lookup := pr.FuncLookup
		pr.FuncLookup = memoFuncLookup(lookup)

		defer func() { pr.FuncLookup = lookup }()
	}

	if !pr.shallow {
		pr.flow = newFlowBlocks(0)
	}

	tagScopes, hasTagRegion := pr.tagFunctionScopes()

	allPendingCalls := pr.mergeRegions(tagScopes)

	// Detect tag-based function scopes from full content (handles functions
	// that span region boundaries, e.g. containing <cfscript> blocks).
	//
	// tagScopes and hasTagRegion are the ones computed at the top of this
	// function. findTagFuncScopes used to run here as well, over the same
	// pr.Content, under a guard testing the same condition — 18% of everything a
	// tag parse allocated, spent scanning the file a second time for an answer
	// already in hand.
	if hasTagRegion {
		pr.addTagScopes(tagScopes)
	}

	// Sort scopes by start line to match function order.
	sortScopes(pr.Scopes)

	if pr.extractCalls {
		pr.fillCallers()
	}

	pr.markStructReturns(allPendingCalls)
	pr.discardPrimitiveReturnComponents()

	// Generate synthetic accessor functions for properties (skip if explicit function exists).
	if pr.shallow {
		// A shallow parse cannot afford FuncLookup, and a ref typed by the
		// first call of a longer chain is wrong, so such a ref is dynamic.
		dropChainRest(pr, dynamicIfTyped)

		return
	}

	pr.settleSignatures(allPendingCalls)
}

// markStructReturns flags each function one of whose returns is a struct
// literal or structNew(), from the return expressions both parsers record.
func (pr *ParseResult) markStructReturns(calls []pendingCall) {
	var byKey map[string][]int

	for i := range calls {
		c := &calls[i]
		if !c.returnExpr || c.funcKey == "" {
			continue
		}

		expr := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(c.varName), "/"))
		if !strings.HasPrefix(expr, "{") && !lex.HasPrefixFold(expr, "structnew(") {
			continue
		}

		if byKey == nil {
			byKey = make(map[string][]int, len(pr.Funcs))
			scopeAt := scopeIndex(pr.Scopes, true)

			for j := range pr.Funcs {
				if k := scopeAt(int(pr.Funcs[j].Line)); k >= 0 {
					scope := pr.Scopes[k]
					key := funcKey(scope.Start, scope.End)
					byKey[key] = append(byKey[key], j)
				}
			}
		}

		for _, j := range byKey[c.funcKey] {
			pr.Funcs[j].returnsStruct = true
		}
	}
}

// discardPrimitiveReturnComponents keeps an inference from overriding a
// declared primitive return type. Without this, a string method's local
// assigned from a receiver call can make the method and its callers return
// the receiver's component. Generic any/component/struct declarations still
// infer.
func (pr *ParseResult) discardPrimitiveReturnComponents() {
	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		f.ReturnComponent = pr.componentReturnFor(f, f.ReturnComponent)
	}
}

// componentReturnFor also guards returns settled after resolver refs become
// available. A runtime-created component remains dynamic; it must not acquire
// a concrete component type.
//
// `struct` admits a component instance, because to the engine a component is
// a struct, so it keeps the inference as `any` does. tassweb leans on that
// throughout: kernel2's GetPageTools, GetLockBroker and getController all
// declare struct and hand back components, and treating struct as primitive
// left every caller of getTASSWebUser() holding nothing.
func (pr *ParseResult) componentReturnFor(f *FunctionDef, comp string) string {
	if strings.EqualFold(f.ReturnType, "struct") {
		if pr.replaceExpressions(comp) == "$any" {
			return "$any"
		}

		// One path returns a plain struct: Wheels' mapper writes
		// `if (flag) { return {}; } return this;`.
		if f.returnsStruct {
			return ""
		}

		return comp
	}

	if f.ReturnType != "" && !strings.EqualFold(f.ReturnType, "any") && !looksLikeCFCType(f.ReturnType) {
		if pr.replaceExpressions(comp) == "$any" {
			return "$any"
		}

		return ""
	}

	return comp
}

// openLocals are the names declared local — `var`, `local.` — in the tag
// function still open where a region ends, keyed by the function's file-level
// funcKey. A <cfscript> block inside a <cffunction> splits the file into
// regions, and the parser of each region starts with only the function's
// arguments as its locals: `<cfset var conn = "">` above the block, and
// `conn = uri.openConnection();` inside it read as an unscoped assignment,
// which is a variables-scope one, so the variable was filed for the whole
// component.
type openLocals struct {
	key   string
	names []string
}

// namesFor is the names declared in the function keyed key, if they are the
// ones held.
func (o *openLocals) namesFor(key string) []string {
	if o.key != key {
		return nil
	}

	return o.names
}

// mergeScriptRegion parses a script region and merges what it found into pr,
// returning its pending calls for resolvePendingCalls.
func (pr *ParseResult) mergeScriptRegion(r *Region, tagScopes []FuncScope, open *openLocals) []pendingCall {
	sp := newScriptParser(r.Text, string(pr.URI), r.StartLine, pr.Resolvers).asCFScript()
	sp.baseOffset = r.Offset
	sp.resolverSet = pr.resolverSet
	sp.extractLinks = pr.extractLinks
	sp.extractCalls = pr.extractCalls
	sp.builtinReturnLookup = pr.BuiltinReturnLookup
	sp.argumentTypes = pr.argumentTypeLookup()

	if pr.flow != nil {
		sp.flow = pr.flow.region(r.Offset, 0)
	}

	// If this <cfscript> region sits inside a tag <cffunction> body
	// (nested script island — ClassifyRegions splits the file there),
	// seed inFunc so refs/pending calls in it route to that function's
	// scope instead of global. scriptParser bakes baseLine into its own
	// keys, so the seed must be the function's absolute funcKey — unlike
	// the tag-region continuation seed below, which stays region-relative.
	seed := -1

	if s, ok := pr.regionTagScope(tagScopes, r.StartLine); ok {
		sp.inFunc = funcKey(s.Start, s.End)
		seed, sp.funcs = pr.seedRegionFunction(s.Start, s.Name)
		sp.localVarSet = make(map[string]bool)

		for _, arg := range pr.argsOfFuncAt(s.Start) {
			sp.localVarSet[strings.ToLower(arg.Name)] = true
		}

		for _, name := range open.namesFor(sp.inFunc) {
			sp.localVarSet[strings.ToLower(name)] = true
		}
	}

	key := sp.inFunc

	sp.parse()

	// What the block declared local stays local in the regions after it.
	if key != "" {
		names := slices.Clone(open.namesFor(key))
		for name := range sp.localVarSet {
			names = append(names, name)
		}

		*open = openLocals{key: key, names: names}
	}

	if seed >= 0 {
		pr.Funcs[seed] = sp.funcs[0]
		sp.funcs = sp.funcs[1:]
	}

	pr.Funcs = append(pr.Funcs, sp.funcs...)
	pr.ComponentRefs = append(pr.ComponentRefs, sp.componentRefs...)
	pr.Scopes = append(pr.Scopes, sp.scopes...)
	pr.Properties = append(pr.Properties, sp.properties...)

	if len(sp.imports) > 0 {
		pr.imports = maps.Clone(sp.imports)
	}

	// Merge function-scoped refs from script parser
	for k, refs := range sp.funcRefs {
		pr.funcRefsMap = appendKeyed(pr.funcRefsMap, k, refs)
	}

	// Merge links from script parser
	pr.Links = append(pr.Links, sp.links...)
	for k, links := range sp.funcLinks {
		pr.funcLinksMap = appendKeyed(pr.funcLinksMap, k, links)
	}

	// Merge calls from script parser
	pr.Calls = append(pr.Calls, sp.calls...)
	for k, calls := range sp.funcCalls {
		pr.funcCallsMap = appendKeyed(pr.funcCallsMap, k, calls)
	}

	pr.Instantiations = append(pr.Instantiations, sp.instantiationsFound()...)

	pr.dropCallsIndex()

	if sp.extends != "" {
		pr.Extends = sp.extends
	}

	pr.Accessors = pr.Accessors || sp.accessors

	if sp.persistent {
		pr.Persistent = true
	}

	return sp.pendingCalls
}

// newRegionTagParser is a tag parser set up to read region r of the file: the
// result's resolvers and lookups, the file's tag function scopes, the output
// state the tag regions share, and the flow blocks at the region's offset.
func (pr *ParseResult) newRegionTagParser(r *Region, tagScopes []FuncScope) *tagParser {
	tp := newTagParser(r.Text, string(pr.URI))
	tp.resolvers = pr.Resolvers
	tp.resolverSet = pr.resolverSet
	tp.extractLinks = pr.extractLinks
	tp.extractCalls = pr.extractCalls
	tp.builtinReturnLookup = pr.BuiltinReturnLookup
	tp.argumentTypes = pr.argumentTypeLookup()
	tp.baseLine = r.StartLine
	tp.knownScopes = tagScopes
	tp.output, tp.importPrefixes, tp.gated = pr.outputGate()
	tp.srcOffset = r.Offset

	if pr.flow != nil {
		tp.flow = pr.flow.region(r.Offset, conv.Uint32(r.StartLine))
	}

	return tp
}

// mergeTagFindings adds the links, function links, function refs and pending
// calls a region's tag parser found, with their lines and function keys moved
// from the region's to the file's by the region's start line.
func (pr *ParseResult) mergeTagFindings(tp *tagParser, startLine int) {
	// Merge links from tag parser
	if startLine > 0 {
		for i := range tp.links {
			tp.links[i].Line += conv.Uint32(startLine)
		}
	}

	pr.Links = append(pr.Links, tp.links...)
	for k, links := range tp.funcLinks {
		if startLine > 0 {
			k = shiftFuncKey(k, startLine)

			for i := range links {
				links[i].Line += conv.Uint32(startLine)
			}
		}

		pr.funcLinksMap = appendKeyed(pr.funcLinksMap, k, links)
	}

	// Merge function-scoped refs from tag parser
	for k, refs := range tp.funcRefs {
		// Offset the key and ref lines by region start line
		if startLine > 0 {
			k = shiftFuncKey(k, startLine)

			for i := range refs {
				refs[i].Line += conv.Uint32(startLine)
			}
		}

		pr.funcRefsMap = appendKeyed(pr.funcRefsMap, k, refs)
	}

	// Collect pending calls from tag parser (offset lines and funcKey)
	for i := range tp.pendingCalls {
		tp.pendingCalls[i].line += conv.Uint32(startLine)
		if startLine > 0 && tp.pendingCalls[i].funcKey != "" {
			tp.pendingCalls[i].funcKey = shiftFuncKey(tp.pendingCalls[i].funcKey, startLine)
		}
	}
}

// mergeTagRegion parses a tag region and merges what it found into pr, with
// its lines and function keys moved from region-relative to file lines,
// returning its pending calls for resolvePendingCalls.
func (pr *ParseResult) mergeTagRegion(r *Region, tagScopes []FuncScope, open *openLocals) []pendingCall {
	tp := pr.newRegionTagParser(r, tagScopes)

	// If this region starts partway through a function whose opening
	// <cffunction> tag was in an earlier region (interrupted by a
	// nested <cfscript> region split), seed inFunc/localVars so refs
	// in this region still route to function scope instead of global.
	seed := -1

	if s, ok := pr.regionTagScope(tagScopes, r.StartLine); ok {
		tp.inFunc = funcKey(s.Start-r.StartLine, s.End-r.StartLine)
		seed, tp.funcs = pr.seedRegionFunction(s.Start, s.Name)
		tp.localVars, tp.localSet = nil, nil

		for _, arg := range pr.argsOfFuncAt(s.Start) {
			tp.markVarLocal(arg.Name)
		}

		for _, name := range open.namesFor(funcKey(s.Start, s.End)) {
			tp.markVarLocal(name)
		}
	}

	seededReturn := seed >= 0 && tp.funcs[0].returnVar != ""

	tp.parse()

	if seed >= 0 {
		// A return this region found for the function it continues is at a
		// line within the region, like everything else it recorded.
		if !seededReturn && tp.funcs[0].returnVar != "" {
			tp.funcs[0].returnLine += conv.Uint32(r.StartLine)
		}

		pr.Funcs[seed] = tp.funcs[0]
		tp.funcs = tp.funcs[1:]
	}

	// The function still open where the region ends continues in the next.
	if tp.inFunc != "" {
		*open = openLocals{key: shiftFuncKey(tp.inFunc, r.StartLine), names: slices.Clone(tp.localVars)}
	}

	for i := range tp.funcs {
		tp.funcs[i].Line += conv.Uint32(r.StartLine)
		tp.funcs[i].returnLine += conv.Uint32(r.StartLine)
	}

	for i := range tp.componentRefs {
		tp.componentRefs[i].Line += conv.Uint32(r.StartLine)
	}

	for i := range tp.properties {
		tp.properties[i].line += conv.Uint32(r.StartLine)
	}

	pr.Funcs = append(pr.Funcs, tp.funcs...)
	pr.ComponentRefs = append(pr.ComponentRefs, tp.componentRefs...)
	pr.Properties = append(pr.Properties, tp.properties...)
	pr.keepRegionFacts(r, tp)

	pr.mergeTagFindings(tp, r.StartLine)

	pr.mergeTagCalls(tp, r.StartLine)

	pr.dropCallsIndex()

	// Tag scopes are computed from full content after region processing
	// to handle functions spanning region boundaries.
	if tp.extends != "" {
		pr.Extends = tp.extends
	}

	pr.Accessors = pr.Accessors || tp.accessors

	if tp.persistent {
		pr.Persistent = true
	}

	return tp.pendingCalls
}

// mergeTagCalls takes the calls and instantiations a tag region's parser
// recorded, its lines moved from the region's to the file's.
func (pr *ParseResult) mergeTagCalls(tp *tagParser, startLine int) {
	if startLine > 0 {
		for i := range tp.calls {
			tp.calls[i].Line += conv.Uint32(startLine)
		}
	}

	pr.Calls = append(pr.Calls, tp.calls...)

	for _, in := range tp.instantiations {
		in.Line += conv.Uint32(startLine)
		pr.Instantiations = append(pr.Instantiations, in)
	}

	for k, calls := range tp.funcCalls {
		if startLine > 0 {
			k = shiftFuncKey(k, startLine)

			for i := range calls {
				calls[i].Line += conv.Uint32(startLine)
			}
		}

		pr.funcCallsMap = appendKeyed(pr.funcCallsMap, k, calls)
	}
}

// enclosingTagScope is the tag function a region starting at startLine opens
// inside of: one whose opening <cffunction> is in an earlier region, the file
// having been split around a nested <cfscript>.
func enclosingTagScope(tagScopes []FuncScope, startLine int) (FuncScope, bool) {
	for _, s := range tagScopes {
		if s.Start < startLine && startLine <= s.End {
			return s, true
		}
	}

	return FuncScope{}, false
}

// argsOfFuncAt is the argument list of the first function declared at line.
func (pr *ParseResult) argsOfFuncAt(line int) []Argument {
	for i := range pr.Funcs {
		if int(pr.Funcs[i].Line) == line {
			return pr.Funcs[i].Arguments
		}
	}

	return nil
}

// shiftFuncKey moves a "start:end" function key by delta lines.
func shiftFuncKey(k string, delta int) string {
	parts := strings.SplitN(k, ":", 2)
	if len(parts) != 2 {
		return k
	}

	return funcKey(atoi(parts[0])+delta, atoi(parts[1])+delta)
}

// appendKeyed appends v to m[k], making the map on first use, and returns it.
func appendKeyed[T any](m map[string][]T, k string, v []T) map[string][]T {
	if m == nil {
		m = make(map[string][]T)
	}

	m[k] = append(m[k], v...)

	return m
}

// servicePropertyAnnotations reads a "@serviceproperty varName kind|name" annotation inside a
// CFML tag comment, e.g. "<!--- @serviceproperty donorObj package|sandbox.tasscom.components.donor --->".
// This is a project-specific documentation convention (not standard CFML/CFC syntax) for
// annotating the real component type of a dependency that's declared with a generic
// <cfargument type="struct"> (CF doesn't enforce component types on constructor args, so
// projects sometimes type them loosely and document the real type in a comment instead).
// varName allows "." and "[...]" (e.g. "result.data[i]") so the same convention can
// annotate a bracket-indexed receiver — a case CanResolveCall otherwise can't resolve at
// all without a global componentResolver keyed on the bare receiver text, which risks
// matching an unrelated same-named receiver anywhere else in the workspace. An in-source
// annotation is scoped to this one file, so it doesn't carry that risk.

// applyServiceProperties scans the whole file for "@serviceproperty" annotations and
// registers a ComponentRef for the annotated variable at the annotation's line, resolved
// via the per-kind dot-path template in pr.ServicePropertyResolvers (e.g.
// "service" -> "tassweb.packages.${name}.service" turns "@serviceproperty objGLJournal
// service|gljournal" into a ref pointing at "tassweb.packages.gljournal.service"). A no-op
// when ServicePropertyResolvers isn't configured, so projects not using the convention pay
// no parsing cost. Runs after pr.Scopes is finalized so each ref can be routed to the
// enclosing function's scope, same as any other ref — this is what makes the annotation
// work like a real (if file-scanned rather than scanner-tracked) assignment: a later
// re-annotation of the same variable name overrides an earlier one, via the existing
// nearest-preceding-ComponentRef resolution in resolve.CanResolveCall.
func (pr *ParseResult) applyServiceProperties() {
	if len(pr.ServicePropertyResolvers) == 0 {
		return
	}

	// The matches come in order, so the line count is carried between them.
	lines := lineCounter{text: pr.Content}

	for _, m := range servicePropertyAnnotations(pr.Content) {
		varName := m.name
		kind := strings.ToLower(m.kind)
		name := m.target

		tmpl, ok := pr.ServicePropertyResolvers[kind]
		if !ok {
			continue
		}

		component := strings.ReplaceAll(tmpl, "${name}", name)
		line := conv.Uint32(lines.before(m.start))

		// resolve.CanResolveCall strips a receiver down to the text after its last
		// top-level "." before comparing against ComponentRef.Variable (e.g.
		// "VARIABLES.donorObj" is looked up as "donorObj"; StripReceiverScope skips any
		// "." inside a "[...]" subscript, so "linkMap[arguments.startSource]" is left
		// unchanged rather than being cut to "startSource]"). Apply the same stripping
		// here so an annotation written against the natural, full receiver text at the
		// call site (e.g. "result.data[i]", matching what a reader actually sees in the
		// source) is stored under the same key CanResolveCall will look it up with.
		lookupVar := StripReceiverScope(varName)

		ref := ComponentRef{
			Variable: lookupVar, Component: component,
			URI: pr.URI, Line: line,
		}

		if fs := findFuncScope(int(line), pr.Scopes); fs.Start != -1 {
			if pr.funcRefsMap == nil {
				pr.funcRefsMap = make(map[string][]ComponentRef)
			}

			key := funcKey(fs.Start, fs.End)
			pr.funcRefsMap[key] = append(pr.funcRefsMap[key], ref)
		} else {
			pr.ComponentRefs = append(pr.ComponentRefs, ref)
		}
	}
}

// applyChainedReturnLookup re-checks component refs created from a receiver.method()
// call (see ComponentRef.ChainBase/ChainMethod) against FuncLookup, overriding the
// componentResolver's guess on the call-site text with the callee's own declared
// return type when FuncLookup can verify it (e.g. a Java stub's getInstance()
// modeling its real return type). Runs last so a verified answer always wins.
//
// It then walks ChainRest, the calls chained after the one that typed the ref,
// so "chart = b.width(1).height(2).build()" types chart by build rather than
// by width. A call it cannot type makes the variable dynamic: keeping the type
// of an earlier call is the wrong answer this exists to stop, and it reported
// every method of the real value as missing from the builder.
func (pr *ParseResult) applyChainedReturnLookup() {
	if pr.FuncLookup == nil {
		dropChainRest(pr, dynamicIfTyped)

		return
	}

	lookupBase := func(baseVar string, scopedRefs []ComponentRef) string {
		for i := range scopedRefs {
			ref := &scopedRefs[i]

			if strings.EqualFold(ref.Variable, baseVar) {
				return ref.Component
			}
		}

		for i := range pr.ComponentRefs {
			ref := &pr.ComponentRefs[i]

			if strings.EqualFold(ref.Variable, baseVar) {
				return ref.Component
			}
		}

		return ""
	}

	override := func(refs []ComponentRef, scoped []ComponentRef) {
		for i := range refs {
			if refs[i].ChainBase == "" {
				continue
			}

			baseComp := lookupBase(refs[i].ChainBase, scoped)
			if baseComp == "" {
				continue
			}

			if ret := pr.FuncLookup(baseComp, refs[i].ChainMethod); ret != "" {
				refs[i].Component = ret
			}
		}
	}

	override(pr.ComponentRefs, nil)

	for k, refs := range pr.funcRefsMap {
		override(refs, refs)
		pr.funcRefsMap[k] = refs
	}

	dropChainRest(pr, pr.walkChainRest)
}

// walkChainRest follows comp through the calls in rest by their declared
// return types. init() is taken to return the object it is called on, which is
// the convention for a CFC constructor and what a java stub's constructor
// models.
func (pr *ParseResult) walkChainRest(comp string, rest []string) string {
	for _, hop := range rest {
		name := callHopName(hop)
		switch {
		case comp == "" || comp == "$any":
			return comp

		case strings.EqualFold(name, "init"), IsMockDecoration(name):
			// A MockBox decoration returns the mock it is called on.
			continue
		}

		ret := pr.FuncLookup(comp, hop)
		if ret == "" && name != hop {
			ret = pr.FuncLookup(comp, name)
		}

		if ret == "" {
			return "$any"
		}

		comp = ret
	}

	return comp
}

// dynamicIfTyped is the answer without FuncLookup: a chain whose rest holds
// any call but init() ends somewhere unknown.
func dynamicIfTyped(comp string, rest []string) string {
	if restTypes(rest) {
		return "$any"
	}

	return comp
}

// restTypes reports whether a chain's rest can change the type it started
// with: anything but init() calls and MockBox decorations, which return what
// they are called on.
func restTypes(rest []string) bool {
	return slices.ContainsFunc(rest, func(h string) bool {
		return !strings.EqualFold(callHopName(h), "init") && !IsMockDecoration(callHopName(h))
	})
}

// settledComponent is ref's Component with any pending chain walked, for a
// reader during the parse that has FuncLookup to hand — resolvePendingCalls,
// typing a variable from the one it was called on — so it reads what the
// finished ref will hold rather than the first call's type.
func (pr *ParseResult) settledComponent(ref *ComponentRef) string {
	if !chainPending(ref) {
		return ref.Component
	}

	if pr.FuncLookup == nil {
		return strictChainComponent(ref, dynamicIfTyped(ref.Component, ref.ChainRest))
	}

	return strictChainComponent(ref, pr.walkChainRest(ref.Component, ref.ChainRest))
}

func strictChainComponent(ref *ComponentRef, component string) string {
	if ref.strictChain && strings.HasPrefix(component, "$") {
		return ""
	}

	return component
}

// chainPending reports whether ref's Component is not yet its answer: it was
// typed by the first call of a longer chain that applyChainedReturnLookup has
// still to walk. A lookup made during the parse must not read it — it would
// hand on the first call's type, which is exactly the wrong answer the walk
// exists to replace — and gets nothing instead, which the resolve step, run
// after the walk, answers from the finished ref.
func chainPending(ref *ComponentRef) bool {
	return restTypes(ref.ChainRest)
}

// dropChainRest applies type to every ref carrying a ChainRest, then clears it.
func dropChainRest(pr *ParseResult, typ func(comp string, rest []string) string) {
	apply := func(refs []ComponentRef) {
		for i := range refs {
			if len(refs[i].ChainRest) == 0 {
				continue
			}

			refs[i].Component = strictChainComponent(&refs[i], typ(refs[i].Component, refs[i].ChainRest))
			refs[i].ChainRest = nil
		}
	}

	apply(pr.ComponentRefs)

	for _, refs := range pr.funcRefsMap {
		apply(refs)
	}
}

// applyExpressionMappings replaces runtime expressions in component paths with static
// values, then treats anything still containing a "#...#" CFML expression as genuinely
// dynamic (e.g. CreateObject("component", "tools.templates.#ARGUMENTS.template#.generator") —
// a runtime-computed path with no configured mapping) and collapses it to "$any" so
// CanResolveCall accepts calls through it instead of surfacing the raw "#...#" text as a
// bogus "not found in #...#" component name. Runs even with no expressionMappings
// configured, since the "$any" fallback is unconditional.
func (pr *ParseResult) applyExpressionMappings() {
	for i := range pr.ComponentRefs {
		pr.ComponentRefs[i].Component = pr.replaceExpressions(pr.ComponentRefs[i].Component)
	}

	for k, refs := range pr.funcRefsMap {
		for i := range refs {
			refs[i].Component = pr.replaceExpressions(refs[i].Component)
		}

		pr.funcRefsMap[k] = refs
	}

	// CallSite.Component can also be baked in directly at parse time (e.g. a bare
	// "x.method()" call whose receiver was resolved inline via lookupComponentRef,
	// or a chained new/createObject) — fix those up too, not just ComponentRefs.
	for i := range pr.Calls {
		pr.Calls[i].Component = pr.replaceExpressions(pr.Calls[i].Component)
	}

	for k, calls := range pr.funcCallsMap {
		for i := range calls {
			calls[i].Component = pr.replaceExpressions(calls[i].Component)
		}

		pr.funcCallsMap[k] = calls
	}

	// A path built at run time builds whatever the program picks; the map
	// draws no edge to it.
	kept := pr.Instantiations[:0]

	for _, in := range pr.Instantiations {
		if in.Component = pr.replaceExpressions(in.Component); in.Component != "$any" {
			kept = append(kept, in)
		}
	}

	pr.Instantiations = kept
}

// ExpressionMappingOrder returns the keys of an expressionMappings block in the
// order they are applied: longest first, and by key among equals. Applying
// them in map order let two overlapping keys — `#A#` and `#A#.b` — replace in
// either order from run to run, so one component path named two different
// components, and a code-map edge moved between them. The longer, more
// specific expression goes first, as route aliases do.
func ExpressionMappingOrder(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}

	keys := slices.Collect(maps.Keys(m))
	slices.SortFunc(keys, func(a, b string) int {
		return cmp.Or(cmp.Compare(len(b), len(a)), strings.Compare(a, b))
	})

	return keys
}

func (pr *ParseResult) replaceExpressions(comp string) string {
	if !strings.Contains(comp, "#") {
		return comp
	}

	for _, key := range pr.expressionKeys {
		value := pr.expressionMappings[key]

		for expr := range strings.SplitSeq(key, "|") {
			if expr != "" && strings.Contains(comp, expr) {
				comp = strings.ReplaceAll(comp, expr, value)
			}
		}
	}

	if strings.Contains(comp, "#") {
		return "$any"
	}

	return comp
}

// appendResolverRefs scans content for function call sites (findCalls).
// Resolver refs are now handled by the script/tag parsers during initial parse.
func (pr *ParseResult) appendResolverRefs() {
	if len(pr.findCalls) == 0 {
		return
	}

	if pr.extractCalls {
		// Filter recorded calls by target function names
		pr.filterCallsByName()

		return
	}

	content := pr.Content
	lineNum := 0
	scopeIdx := 0
	currentFunc := ""

	for content != "" {
		nl := strings.IndexByte(content, '\n')

		var line string

		if nl < 0 {
			line = content
			content = ""
		} else {
			line = content[:nl]
			content = content[nl+1:]
		}

		// Track current function scope
		if scopeIdx < len(pr.Scopes) {
			if lineNum > pr.Scopes[scopeIdx].End {
				scopeIdx++
				currentFunc = ""
			}

			if scopeIdx < len(pr.Scopes) && lineNum == pr.Scopes[scopeIdx].Start+1 {
				currentFunc = pr.Scopes[scopeIdx].Name
			}
		}

		pr.scanLineForCalls(line, lineNum, currentFunc)

		lineNum++
	}
}

// filterCallsByName populates pr.Calls with calls from funcCallsMap and global
// calls that match the pr.findCalls target names.
func (pr *ParseResult) filterCallsByName() {
	targets := make(map[string]bool, len(pr.findCalls))
	for _, t := range pr.findCalls {
		targets[strings.ToLower(t)] = true
	}

	// Gather all calls (global + per-function)
	var allCalls []CallSite

	allCalls = append(allCalls, pr.Calls...)

	for _, calls := range pr.funcCallsMap {
		allCalls = append(allCalls, calls...)
	}

	// Filter by target names and resolve components
	pr.Calls = nil

	scopeAt := scopeIndex(pr.Scopes, false)

	for i := range allCalls {
		call := allCalls[i] // a copy: edited below, and kept

		if !targets[strings.ToLower(call.FuncName)] {
			continue
		}

		// Resolve component from variable name
		if call.Variable != "" && call.Component == "" {
			// Strip scope prefix (VARIABLES.service → service)
			resolveVar := call.Variable
			if dotIdx := strings.LastIndexByte(resolveVar, '.'); dotIdx >= 0 {
				resolveVar = resolveVar[dotIdx+1:]
			}

			call.Component = pr.resolveVarComponent(resolveVar)
			call.Resolved = call.Component != ""
		}

		// Determine caller from line
		if call.Caller == "" {
			if k := scopeAt(int(call.Line)); k >= 0 {
				call.Caller = pr.Scopes[k].Name
			}
		}

		pr.Calls = append(pr.Calls, call)
	}
}

// extractLinksFromLine extracts file path references from a single line, in
// linkAttrs order and left to right within each.
func extractLinksFromLine(line string, lineNum int, links *[]DocumentLink) {
	// Quick reject: all link attrs contain '=' or "include " — check for quote presence
	if !strings.ContainsAny(line, "\"'") {
		return
	}

	// Three of the four names end in '=', so a line without one, or whose
	// '=' follow none of them, needs only the "include " search. Most tag
	// lines have an '=' and none of the names; testing the bytes before each
	// '=' replaced three whole-line case-insensitive searches per line.
	hasEq := strings.IndexByte(line, '=') >= 0

	for _, attr := range linkAttrs {
		if attr[len(attr)-1] == '=' && (!hasEq || !precedesEquals(line, attr[:len(attr)-1])) {
			continue
		}

		idx := 0

		for {
			pos := indexFold(line[idx:], attr)
			if pos < 0 {
				break
			}

			pos += idx + len(attr)
			for pos < len(line) && (line[pos] == ' ' || line[pos] == '\t') {
				pos++
			}

			if pos >= len(line) {
				break
			}

			q := line[pos]
			if q != '"' && q != '\'' {
				idx = pos

				continue
			}

			start := pos + 1

			end := strings.IndexByte(line[start:], q)
			if end < 0 {
				break
			}

			end += start
			path := line[start:end]

			if path != "" && !strings.Contains(path, "#") && !strings.Contains(path, "://") {
				*links = append(*links, DocumentLink{
					Path:  path,
					Line:  conv.Uint32(lineNum),
					Start: conv.Uint32(start),
					End:   conv.Uint32(end),
				})
			}

			idx = end + 1
		}
	}
}

// precedesEquals reports whether name, case-insensitively, stands directly
// before some '=' in line.
func precedesEquals(line, name string) bool {
	for i := strings.IndexByte(line, '='); i >= 0; {
		if i >= len(name) && strings.EqualFold(line[i-len(name):i], name) {
			return true
		}

		next := strings.IndexByte(line[i+1:], '=')
		if next < 0 {
			return false
		}

		i += next + 1
	}

	return false
}

// linkAttrs are the attribute names that contain file paths.
var linkAttrs = []string{"template=", "include ", "href=", "action="}

// ExtractLinks scans content for file path references (cfinclude, href, etc.).
func ExtractLinks(content string) []DocumentLink {
	var links []DocumentLink

	lineNum := 0

	for content != "" {
		nl := strings.IndexByte(content, '\n')

		var line string

		if nl < 0 {
			line = content
			content = ""
		} else {
			line = content[:nl]
			content = content[nl+1:]
		}

		for _, attr := range linkAttrs {
			idx := 0

			for {
				pos := indexFold(line[idx:], attr)
				if pos < 0 {
					break
				}

				pos += idx + len(attr)
				// Skip whitespace and find opening quote
				for pos < len(line) && (line[pos] == ' ' || line[pos] == '\t') {
					pos++
				}

				if pos >= len(line) {
					break
				}

				q := line[pos]
				if q != '"' && q != '\'' {
					idx = pos

					continue
				}

				start := pos + 1

				end := strings.IndexByte(line[start:], q)
				if end < 0 {
					break
				}

				end += start
				path := line[start:end]

				if path != "" && !strings.Contains(path, "#") && !strings.Contains(path, "://") {
					links = append(links, DocumentLink{
						Path:  path,
						Line:  uint32(lineNum),
						Start: conv.Uint32(start),
						End:   conv.Uint32(end),
					})
				}

				idx = end + 1
			}
		}

		lineNum++
	}

	return links
}

func sortScopes(scopes []FuncScope) {
	for i := 1; i < len(scopes); i++ {
		for j := i; j > 0 && scopes[j].Start < scopes[j-1].Start; j-- {
			scopes[j], scopes[j-1] = scopes[j-1], scopes[j]
		}
	}
}

// funcKey builds the "start:end" key the lazy per-function caches
// (funcVars/funcRefsMap/funcCallsMap/funcLinksMap) are stored under.
//
// It is on every per-keystroke path -- hover, definition and completion each
// reach FuncVars/FuncRefs, and FuncLinks calls it once per link -- so it is
// written to allocate exactly once, for the returned string. The obvious
// spelling, strings.Join([]string{itoa(start), itoa(end)}, ":"), costs four:
// a string from each itoa, the slice holding them, and the join's result. In a
// profile of the document-link handler that was 600,001 of its 630,284
// objects, 95% of everything it allocated.
//
// A struct key would allocate none at all, and is deliberately not used: the
// zero value of a struct{start, end int} is {0, 0}, which is a real function
// beginning on line 0, so it could not also mean "no enclosing function" the
// way the empty string does for every `inFunc == ""` test in the two parsers.
func funcKey(start, end int) string {
	// Two base-10 ints with sign, plus the separator.
	var buf [2*20 + 1]byte

	b := strconv.AppendInt(buf[:0], int64(start), 10)
	b = append(b, ':')
	b = strconv.AppendInt(b, int64(end), 10)

	return string(b)
}

func atoi(s string) int {
	neg := false

	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}

	n := 0

	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}

	if neg {
		return -n
	}

	return n
}

// lineOffsets converts line numbers to byte offsets: start is where startLine
// begins and end is just past endLine's newline (or the end of the content).
// The line table is built once per content, since every function body's
// parse asks and walking from the top each time made a file with many
// functions quadratic in its length. Comparing the cached content is cheap:
// two strings sharing their bytes compare equal without reading them.
func (pr *ParseResult) lineOffsets(startLine, endLine int) (start, end int) {
	return offsetsFromTable(pr.lineStartsTable(), len(pr.Content), startLine, endLine)
}

// lineAt is the 0-based line holding byte offset off of Content, from the
// same table lineOffsets reads.
func (pr *ParseResult) lineAt(off int) int {
	starts := pr.lineStartsTable()

	return sort.Search(len(starts), func(i int) bool { return starts[i] > off }) - 1
}

func (pr *ParseResult) lineStartsTable() []int {
	pr.lineStartsMu.Lock()
	defer pr.lineStartsMu.Unlock()

	if pr.lineStarts == nil || pr.lineStartsContent != pr.Content {
		pr.lineStarts = lineStartTable(pr.Content)
		pr.lineStartsContent = pr.Content
	}

	return pr.lineStarts
}

func lineStartTable(content string) []int {
	starts := make([]int, 1, strings.Count(content, "\n")+1)

	for i := 0; ; {
		idx := strings.IndexByte(content[i:], '\n')
		if idx < 0 {
			return starts
		}

		i += idx + 1
		starts = append(starts, i)
	}
}

func offsetsFromTable(starts []int, size, startLine, endLine int) (start, end int) {
	if startLine < 0 {
		startLine = 0
	}

	if startLine >= len(starts) {
		return -1, -1
	}

	start = starts[startLine]

	switch {
	case endLine < startLine:
		end = start
	case endLine+1 < len(starts):
		end = starts[endLine+1]
	default:
		end = size
	}

	return start, end
}

func (pr *ParseResult) logDebug(msg string, keysAndValues ...any) {
	if pr.log != nil {
		pr.log.Debug(msg, keysAndValues...)
	}
}

// IsMemberMethod returns true if the method name is a known CFML member function
// on native types (Array, Struct, Query, String, List).
func IsMemberMethod(name string) bool {
	return isMemberMethod(name)
}

// isMemberMethod returns true if the method name is a known CFML member function
// on native types (Array, Struct, Query, String, List).
func isMemberMethod(name string) bool {
	var buf foldScratch

	switch string(buf.lowerFold(name)) {
	// Array
	case "append", "prepend", "clear", "delete", "deleteat", "each", "every", "filter",
		"find", "findall", "findallnocase", "findnocase", "first", "getat", "indexexists",
		"insertat", "isdefined", "isempty", "last", "len", "map", "max", "median", "merge",
		"mid", "min", "new", "pop", "push", "range", "reduce", "reduceright",
		"removeduplicates", "resize", "reverse", "set", "shift", "slice", "some", "sort",
		"splice", "sum", "swap", "tolist", "tostruct", "unshift", "avg", "contains",
		"containsnocase", "addall", "getduplicates", "compact", "rest",
		// Struct
		"copy", "count", "equals", "findkey", "findvalue", "get", "getmetadata",
		"insert", "iscasesensitive", "isordered", "keyarray", "keyexists", "keylist",
		"keytranslate", "listnew", "setmetadata", "toquerystring", "tosorted", "update",
		"valuearray",
		// Query
		"addcolumn", "addrow", "close", "columnarray", "columncount", "columndata",
		"columnexists", "columnlist", "currentrow", "deletecolumn", "deleterow", "execute",
		"getcell", "getcellbyindex", "getresult", "getrow", "lazy",
		"recordcount", "renamecolumn", "rowbyindex", "rowdata", "rowdatabyindex", "rowswap",
		"setcell", "setrow", "convertforgrid",
		// String/List
		"changedelims", "qualifiedtoarray", "qualify", "itemtrim", "trim",
		"valuecount", "valuecountnocase",
		// Common global/Java methods
		"getbytes", "tostring", "hashcode", "getclass", "init", "tobytearray", "tochararray",
		// java.lang.Class / java.lang.reflect methods
		"getcomponenttype", "getname", "getsimplename", "getdeclaredmethods",
		"getmethods", "getdeclaredfields", "getfields", "newinstance",
		"isarray", "isassignablefrom", "isinstance", "getinterfaces",
		"getsuperclass", "forname",
		// JavaScript String/Array/RegExp methods
		"split", "substr", "substring", "indexof", "lastindexof", "tolowercase",
		"touppercase", "charat", "concat", "search", "test", "exec", "join",
		"replace", "match", "startswith", "endswith", "includes", "padstart",
		"padend", "repeat", "charcodeat", "normalize", "trimstart", "trimend",
		// JavaScript DOM/Event/BOM methods
		"focus", "submit", "add", "remove", "item", "preventdefault",
		"stoppropagation", "closest", "on", "configure", "setattribute",
		"getattribute", "removeattribute", "insertcell", "hasownproperty",
		"setdate", "getdate", "getday", "gettime", "getmonth", "setmonth",
		"getfullyear", "setfullyear", "getdocumentelement", "getelementsbytagname",
		"getelementbyid", "getelementsbyclassname", "queryselector", "queryselectorall",
		// jQuery/JS common
		"is", "has", "call", "apply", "bind", "then", "catch", "finally",
		// Common property-like methods (Java/iText objects)
		"width", "height", "length":
		return true
	}

	return false
}

// mayDeclare is mayDeclareVariable for this file, used only while AllVars has
// not been built, and for at most maxDeclareScans names: each answer is a
// search of the whole file, and the resolver asks Declares of many names in
// one file, so past a few names building AllVars once is cheaper.
func (pr *ParseResult) mayDeclare(name string) bool {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.allVarsDone || len(pr.declareScans) >= maxDeclareScans {
		return true
	}

	key := strings.ToLower(name)
	if may, ok := pr.declareScans[key]; ok {
		return may
	}

	may := mayDeclareVariable(pr.Content, name)
	if pr.declareScans == nil {
		pr.declareScans = map[string]bool{}
	}

	pr.declareScans[key] = may

	return may
}

// maxDeclareScans bounds mayDeclare's whole-file searches per file.
const maxDeclareScans = 4

// mayDeclareVariable reports whether content could declare a variable called
// name: whether the name appears anywhere, as a whole word, other than called.
// Every declaration (an assignment, var, a for-in or catch binding, a tag's
// name="…") has something other than "(" after the name, so a name only ever
// called is declared by nothing, and AllVars, a second whole-file parse, is
// not run to find that out: Declares( "structNew" ) asked it of every file
// with a collection seeded by structNew(), 750 MB over a tassweb scan.
func mayDeclareVariable(content, name string) bool {
	if name == "" {
		return true
	}

	for at := 0; ; {
		i := indexFoldFrom(content, name, at)
		if i < 0 {
			return false
		}

		at = i + len(name)

		if i > 0 && isIdentPart(content[i-1]) || at < len(content) && isIdentPart(content[at]) {
			continue
		}

		j := at
		for j < len(content) && (content[j] == ' ' || content[j] == '\t') {
			j++
		}

		if j >= len(content) || content[j] != '(' {
			return true
		}
	}
}

// IsShallow reports whether pr is a shallow parse (ParseOptions.Shallow):
// signatures and extends, but no properties, refs or arguments.
func (pr *ParseResult) IsShallow() bool { return pr.shallow }

// Declares reports whether the file declares name anywhere, case-insensitively:
// a variable in any scope, a property, a function or a function's argument.
// A name it does not declare is one it expects from elsewhere — its base
// component, an injector, the framework — which is what the resolver needs to
// know before blaming a missing base for a receiver with no component.
func (pr *ParseResult) Declares(name string) bool {
	for i := range pr.Funcs {
		if strings.EqualFold(pr.Funcs[i].Name, name) {
			return true
		}

		for j := range pr.Funcs[i].Arguments {
			if strings.EqualFold(pr.Funcs[i].Arguments[j].Name, name) {
				return true
			}
		}
	}

	for i := range pr.Properties {
		if strings.EqualFold(pr.Properties[i].name, name) {
			return true
		}
	}

	if !pr.mayDeclare(name) {
		return false
	}

	for _, v := range pr.AllVars() {
		if strings.EqualFold(v.Name, name) {
			return true
		}
	}

	return false
}

// A script island or tag continuation has no opening function declaration of
// its own. Carry the enclosing typed arguments without publishing a duplicate
// definition; whole-argument assignments must see the same signature throughout.
func (pr *ParseResult) seedRegionFunction(start int, name string) (int, []FunctionDef) {
	for i := range pr.Funcs {
		fn := &pr.Funcs[i]
		if int(fn.Line) != start || !strings.EqualFold(fn.Name, name) {
			continue
		}

		for _, arg := range fn.Arguments {
			if argumentComponentType(&arg) != "" {
				return i, []FunctionDef{*fn}
			}
		}

		break
	}

	return -1, nil
}

// An island can begin on the opening tag's own line. The declaration must
// already have been parsed: a first tag region on that line is not a continuation.
func (pr *ParseResult) regionTagScope(scopes []FuncScope, line int) (FuncScope, bool) {
	if scope, ok := enclosingTagScope(scopes, line); ok {
		return scope, true
	}

	for _, scope := range scopes {
		if scope.Start != line {
			continue
		}

		for i := range pr.Funcs {
			fn := &pr.Funcs[i]
			if int(fn.Line) == line && strings.EqualFold(fn.Name, scope.Name) {
				return scope, true
			}
		}
	}

	return FuncScope{}, false
}

// prepareManagedSetterLookup reads properties before signatures so declaration
// order cannot change DI/1 eligibility. The metadata parse has no injection
// callbacks, and therefore does not recurse. Global edits rebuild this snapshot.
func (pr *ParseResult) prepareManagedSetterLookup() {
	pr.managedSetterLookup = pr.SetterLookup
	if pr.SetterLookup == nil || pr.PropertyBeanLookup == nil {
		return
	}

	metadata := pr.managedPropertyMetadata()

	properties := make(map[string]map[string]string, len(metadata.Properties))
	for _, prop := range metadata.Properties {
		if !metadata.Accessors && !metadata.Persistent {
			continue
		}

		properties[strings.ToLower(prop.name)] = prop.attrs
	}

	pr.managedSetterLookup = func(name string) string {
		attrs, found := properties[strings.ToLower(name)]
		// setter=false omits the implicit property setter, allowing an explicit one.
		if found && !strings.EqualFold(attrs["setter"], "false") && pr.PropertyBeanLookup(name, attrs) == "" {
			return ""
		}

		return pr.SetterLookup(name)
	}
}

func (pr *ParseResult) propertyBeanComponent(prop *propertyDef) string {
	if pr.PropertyBeanLookup != nil && !pr.Accessors && !pr.Persistent {
		return ""
	}

	if pr.BeanLookup != nil {
		lookup := pr.BeanLookup
		if pr.PropertyBeanLookup != nil {
			lookup = func(name string) string { return pr.PropertyBeanLookup(name, prop.attrs) }
		}

		if inject := prop.attrs["inject"]; inject != "" {
			if comp := lookup(normalizeBeanKey(inject)); comp != "" {
				return comp
			}

			if comp := lookup(extractBeanName(inject)); comp != "" {
				return comp
			}
		}

		if comp := lookup(prop.name); comp != "" {
			return comp
		}
	}

	if pr.PropertyBeanLookup != nil {
		return ""
	}

	inject, ok := prop.attrs["inject"]
	if ok && inject == "" {
		inject = prop.name
	}

	return injectedComponent(inject)
}

func (pr *ParseResult) applyManagedArgumentTypes(name, access string, args []Argument) {
	applySetterArgumentTypes(name, access, args, pr.managedSetterLookup)
	applyConstructorArgumentTypes(name, access, args, pr.ConstructorLookup)
}

func (pr *ParseResult) argumentTypeLookup() func(string, string, []Argument) {
	if pr.SetterLookup == nil && pr.ConstructorLookup == nil {
		return nil
	}

	return pr.applyManagedArgumentTypes
}

func callHopName(hop string) string {
	if expression, ok := CallExpression(hop); ok {
		name, _, _ := strings.Cut(expression, "(")
		_, name, _ = strings.CutLast("."+name, ".")

		return strings.TrimSpace(name)
	}

	return hop
}

// memoFuncLookup answers each (component, method) pair once for the length of
// one signature pass. A component's methods are asked about once per call site,
// and a large file names the same few thousands of times: on a 65,000-line
// component the repeats were most of what a reparse cost. The answer depends on
// the index and the filesystem, which the pass does not change, so it is held
// only while the pass runs and never across an edit.
func memoFuncLookup(lookup func(component, funcName string) string) func(component, funcName string) string {
	type key struct{ component, funcName string }

	var mu sync.Mutex

	seen := make(map[key]string)

	return func(component, funcName string) string {
		k := key{component, funcName}

		mu.Lock()
		answer, ok := seen[k]
		mu.Unlock()

		if ok {
			return answer
		}

		answer = lookup(component, funcName)

		mu.Lock()
		seen[k] = answer
		mu.Unlock()

		return answer
	}
}

// aliasComponent is what `varName = baseVar` copies: what baseVar holds at
// the assignment, read from the function's refs and then the component's, as
// a receiver is. It waits for the member pass, so a member is answered too:
// tassweb's itext.cfc builds its helpers into `temp` in init() and keeps them
// with `VARIABLES._itextUtils = temp.itextUtils`, and every method calling
// one returned an untyped value.
func (pr *ParseResult) aliasComponent(c *pendingCall, globals []ComponentRef) string {
	if globals == nil {
		globals = pr.ComponentRefs
	}

	scope, name := c.baseScope, c.baseVar
	if lower := strings.ToLower(name); strings.HasPrefix(lower, "variables.") || strings.HasPrefix(lower, "this.") || strings.HasPrefix(lower, "local.") {
		_, name, _ = strings.Cut(name, ".")
	}

	var ref *ComponentRef

	if scope == RefAny && c.funcKey != "" {
		ref = refReaching(pr.funcRefsMap[c.funcKey], name, c.line, func(r *ComponentRef) bool { return r.VisibleTo == 0 && !r.HasClosureSpan })
	}

	if ref == nil && !c.baseLocal {
		ref = refReaching(globals, name, c.line, scope.Admits)
	}

	if ref == nil {
		return ""
	}

	return pr.settledComponent(ref)
}

// guessedMethod is the method a ref typed by its receiver's own type was not
// looked up for, which the resolver may look up (FunctionDef.ReturnVia).
// ChainBase stays empty, so applyChainedReturnLookup leaves the ref alone.
func guessedMethod(c *pendingCall) string {
	if !c.baseGuess {
		return ""
	}

	return c.funcName
}

// keepRegionFacts keeps what a tag region's walk found that later passes would
// otherwise walk the region again for: its scope declarations and where it met
// the tags the collection passes read.
func (pr *ParseResult) keepRegionFacts(r *Region, tp *tagParser) {
	pr.keepTagScopeVars(r, tp.vars)

	if pr.collectionTags == nil {
		pr.collectionTags = map[int][]int{}
	}

	pr.collectionTags[r.Offset] = tp.collectionTags
}

// keepTagScopeVars keeps the variables- and this-scope declarations a tag
// region's walk recorded, on file lines, for computeScopedVars. Re-indexing
// after every edit asked for them, and a second walk of the region was 10ms
// of each edit outside a function on a 58,000-line component.
func (pr *ParseResult) keepTagScopeVars(r *Region, vars []VarDef) {
	var kept []VarDef

	for _, v := range vars {
		if v.Scope == ScopeVariables || v.Scope == ScopeThis {
			v.Line += conv.Uint32(r.StartLine)
			kept = append(kept, v)
		}
	}

	if pr.tagScopeVars == nil {
		pr.tagScopeVars = map[int][]VarDef{}
	}

	pr.tagScopeVars[r.Offset] = kept
}

// keptTagScopeVars is what keepTagScopeVars kept for the tag region at
// offset, when the regions were read from the current content.
func (pr *ParseResult) keptTagScopeVars(offset int) ([]VarDef, bool) {
	if pr.tagScopeVars == nil || !pr.regionFactsCurrent() {
		return nil, false
	}

	vars, ok := pr.tagScopeVars[offset]

	return vars, ok
}

// keptCollectionTags is where the parse's walk met the tags collectWrites
// reads in the tag region at offset, when read from the current content.
func (pr *ParseResult) keptCollectionTags(offset int) ([]int, bool) {
	if pr.collectionTags == nil || !pr.regionFactsCurrent() {
		return nil, false
	}

	sites, ok := pr.collectionTags[offset]

	return sites, ok
}

// regionFactsCurrent reports whether what the region merge kept was read from
// the current content. When not, it is dropped, so the old document is not
// kept alive by it.
func (pr *ParseResult) regionFactsCurrent() bool {
	// Equal strings sharing their bytes compare without reading them.
	if pr.tagScopeVarsOf == pr.Content {
		return true
	}

	pr.tagScopeVars, pr.tagScopeVarsOf, pr.collectionTags = nil, "", nil

	return false
}

// DocumentLinks returns the file path references in the document, from its
// text, computed once per content. The editor's parse does not extract links
// (ExtractLinks), since only the document-link request reads them and the
// extraction was 6% of every reparse; this answers that request instead.
// Read from the whole text rather than per region, a link has its column on
// its own line wherever a region started, and one in a <script> block is
// found too.
func (pr *ParseResult) DocumentLinks() []DocumentLink {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.textLinks == nil || pr.textLinksOf != pr.Content {
		pr.textLinks = ExtractLinks(pr.Content)
		if pr.textLinks == nil {
			pr.textLinks = []DocumentLink{}
		}

		pr.textLinksOf = pr.Content
	}

	return pr.textLinks
}
