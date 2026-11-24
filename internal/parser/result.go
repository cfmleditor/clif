package parser

import (
	"sync"
	"time"

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
	Implements          []string                                // interfaces named in the implements attribute
	Accessors           bool                                    // component has accessors=true
	Persistent          bool                                    // true if component has persistent="true" (ORM entity)
	Properties          []propertyDef                           // parsed property declarations
	Delegates           []Delegate                              // WireBox delegations the component declares
	Links               []DocumentLink                          // with ExtractLinks, the links outside any function, from the walk's record
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
	findCalls                []string    // ParseOptions.FindCalls, lowercased
	scanAllScopes            bool        // scan all lines including function bodies
	shallow                  bool        // minimal parse mode
	interpolateAll           bool        // ParseOptions.InterpolateAllText
	keepVars                 bool        // ParseOptions.KeepVars
	scopesOnly               bool        // ParseOptions.ScopesOnly
	fileRefsOnly             bool        // ParseOptions.FileRefsOnly
	foundVars                [][]VarDef  // with keepVars, each region's declarations, on file lines
	outputScanned            bool        // importPrefixes is computed
	output                   outputState // where the tag walk stands in the output contexts
	importPrefixes           []string    // <cfimport prefix="..."> values, lowercased

	// contentLineIdx is the line index ClassifyRegions built over Content on its
	// way to the regions, kept so extractSignatures does not build a second one
	// over the same string. nil for a script file, which never needs it.
	contentLineIdx LineTable

	// Lazy global var caches (protected by mu).
	mu            sync.Mutex
	globalVars    []string
	variablesVars []string
	thisVars      []string
	scopedDone    bool // the three above, from one scan
	// collectionTags is, for each tag region by its offset, where the parse's
	// walk met the tags collectWrites reads; collectionTagsOf is the Content
	// they were read from, since an edit inside a function moves lines without
	// reparsing the regions.
	collectionTags   map[int][]int
	collectionTagsOf string
	// bodies is what the walk read of the function bodies, and what is still
	// to be read again after an edit (body_walk.go).
	bodies bodyState
	// textLinks is DocumentLinks' answer, for the content textLinksOf.
	textLinks   []DocumentLink
	textLinksOf string
	// recorded is the file references the walk recorded, or read since for
	// the content it names (file_refs.go); refsRec is the record the walk is
	// making, while it runs.
	recorded *fileRefRecord
	refsRec  *fileRefRecord
	// regionsOf is the Content Regions and contentLineIdx were split from: an
	// edit inside a function changes Content and leaves them.
	regionsOf string
	// spans is FunctionSpans' answer, behind a pointer since most parses are
	// never asked for it.
	spans *functionSpans
	// assignments is Assignments' answer, recorded during the parse when
	// ParseOptions.RecordAssignments asked; nil otherwise.
	assignments *assignments
	// sharedAssignments is ParseOptions.SharedAssignments.
	sharedAssignments func(content string) []Assignment
	// customTags is CustomTagCalls' answer, for the content customTagsOf and
	// the prefixes customTagsKey.
	customTags    []CustomTagCall
	customTagsOf  string
	customTagsKey string
	// structure is StructureSpans' answer, for the content structureOf.
	structure    []Span
	structureOf  string
	allVars      []VarDef
	declareScans map[string]bool // mayDeclare's answers, by lowercased name
	allVarsDone  bool

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

	// funcVars caches per-function variable lists keyed by function (fnKey).
	funcVarsMu sync.Mutex
	funcVars   map[fnKey][]string
	// funcVarSets holds each function's funcVars as a set, for FuncDeclares,
	// with the list it was built from so a list replaced is noticed.
	funcVarSets map[fnKey]funcVarSet

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

	// funcRefs caches per-function component refs keyed by function (fnKey).
	memberSnapshot        *ParseResult // member refs computed once per document version
	memberSnapshotContent string
	// refIdx is firstRef's index while resolvePendingCalls runs, nil
	// otherwise; see firstRef.
	refIdx       map[*ComponentRef]*refNames
	funcRefsMu   sync.Mutex
	funcRefsMap  map[fnKey][]ComponentRef
	funcLinksMap map[fnKey][]DocumentLink
	funcCallsMap map[fnKey][]CallSite // per-function call sites keyed by function
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
	lineStarts        LineTable
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
	// ExtractLinks records every path the file names and every custom tag it
	// calls as the walk goes, not only its includes, and files the links in
	// Links and by function.
	ExtractLinks bool
	ExtractCalls bool // extract all variable.method() call sites during parsing
	// FindCalls is the function names to find call sites for, ignoring
	// case, in Calls. With ExtractCalls it filters what that records; without
	// it the walk records these names' calls alone, as ExtractCalls would,
	// with their spans, receivers and chains, and nothing else changes
	// (call_names.go).
	FindCalls     []string
	ScanAllScopes bool // scan all lines including function bodies (for refs/deps)
	Shallow       bool // minimal parse: signatures only, no refs/properties/args
	// InterpolateAllText scans #...# in all of a tag file's text, as the
	// parser did before it learned where ColdFusion evaluates it (see
	// outputContext), and reads a tag-free .cfm template as CFScript. It is
	// the features.outputContextInterpolation switch turned off.
	InterpolateAllText bool
	// KeepVars keeps every variable declaration the walk records, so AllVars
	// answers from this parse rather than walking the file again. ParseVars
	// and an AllVars asked of a parse without it are a Shallow parse with it.
	KeepVars bool
	// ScopesOnly is for a reader that wants the functions' Scopes and
	// FunctionSpans and nothing else: the tag regions are not walked, since
	// the whole-file <cffunction> finder gives their functions' scopes and
	// spans, and the script regions are walked as ever, since their
	// functions are what that walk finds. Every other answer of such a parse
	// is partial. FunctionSpansOf and FindFuncScopes read one; with Shallow,
	// its scopes and spans are a full parse's.
	ScopesOnly bool
	// FileRefsOnly is for a reader that wants the file's references and
	// nothing else: its includes (IncludeSites) and, with ExtractLinks, its
	// links and custom tag calls. Each script region's tokens are scanned for
	// the record alone, with no statement read, and each tag region's tags
	// read by the reference reader without the tag walk; every other answer
	// of such a parse is empty. The text entry points (IncludeSites,
	// ExtractLinks, CustomTagCalls) read one, and a parse asked for what it
	// did not record reads one of its content.
	FileRefsOnly bool
	// Body narrows the walk to one function's lines, Body.Start to Body.End.
	// The file is split into regions and its tag functions found as for the
	// whole file, and each region is walked only where it meets those lines,
	// its walker seeded as the whole walk seeds it there: a <cfscript> island
	// with the tag function around it, a tag region with the file's tag
	// function scopes and its output context. The result holds that
	// function's variables, refs, links and (with ExtractCalls) calls, and
	// those of the functions nested in it, filed as a whole parse files them.
	// The lazy readers (FuncVars, FuncRefs, FuncCalls, HasScopedAssignment)
	// read a function with it where the parse they are asked of did not walk
	// its body, or an edit has changed it since.
	Body *FuncScope
	// context is the parse a lazy reader reads a body of: its regions, tag
	// function scopes and imports seed the walk, and its functions and
	// file-level refs type what the body assigns, as they type it in the
	// whole parse.
	context *ParseResult
	// RecordAssignments records every write the parse reads (Assignments)
	// as it runs, rather than on the first question. Off, nothing is read
	// until Assignments is asked.
	RecordAssignments bool
	// SharedAssignments is how Assignments reads a text's writes when the
	// caller holds them for every parse of that text, a file being parsed once
	// to index and again to scan: what it returns is AssignmentsOf's answer
	// for the parse's file and text, and is shared. Nil reads them from the
	// parse's own regions.
	SharedAssignments func(content string) []Assignment
}

// Parse performs a full file parse: extracts function signatures, component refs,
// and function scopes. Function bodies are NOT parsed for variables until requested.
func Parse(fileURI uri.URI, content string, resolvers ...[]Resolver) *ParseResult {
	if countScans {
		scanCounts.parses.Add(1)
	}

	pr := &ParseResult{
		URI:      fileURI,
		Content:  content,
		funcVars: make(map[fnKey][]string),
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
func (pr *ParseResult) classifyRegions() ([]Region, LineTable) {
	if !pr.interpolateAll && isMarkupTemplate(string(pr.URI), pr.Content) {
		regions, lines := splitCFScriptBlocks(pr.Content)

		return splitTagIslands(regions), lines
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

	// A reading of the file's references alone (FileRefsOnly) is the walk
	// each reference reader made before the parse recorded them: its tokens
	// count as passes, and it is not counted as a parse.
	if countScans && !opts.FileRefsOnly {
		scanCounts.parses.Add(1)
	}

	pr := &ParseResult{
		URI:                      fileURI,
		Content:                  content,
		funcVars:                 make(map[fnKey][]string),
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
		findCalls:                lowerNames(opts.FindCalls),
		scanAllScopes:            opts.ScanAllScopes,
		shallow:                  opts.Shallow,
		interpolateAll:           opts.InterpolateAllText,
		keepVars:                 opts.KeepVars,
		scopesOnly:               opts.ScopesOnly,
		fileRefsOnly:             opts.FileRefsOnly,
		sharedAssignments:        opts.SharedAssignments,
	}
	if opts.Body != nil {
		body := *opts.Body
		pr.bodies.only, pr.bodies.context = &body, opts.context
	}

	if len(pr.Resolvers) > 0 {
		if opts.ResolverSet.Serves(pr.Resolvers) {
			pr.resolverSet = opts.ResolverSet.ForParse()
		} else {
			pr.resolverSet = newPrivateResolverSet(pr.Resolvers)
		}
	}

	start := time.Now()

	if ctx := pr.bodies.context; ctx != nil {
		pr.Regions, pr.contentLineIdx = ctx.walkRegions()
		pr.lineStarts, pr.lineStartsContent = ctx.lineStartsTable(), pr.Content

		if ctx.outputScanned {
			pr.importPrefixes, pr.outputScanned = ctx.importPrefixes, true
		}
	} else {
		pr.Regions, pr.contentLineIdx = pr.classifyRegions()
	}

	pr.regionsOf = pr.Content
	pr.extractSignatures()

	if opts.RecordAssignments {
		pr.assignments = &assignments{of: content, list: pr.readAssignments()}
	}

	pr.logDebug("parse", "uri", string(fileURI), "funcs", len(pr.Funcs), "refs", len(pr.ComponentRefs), "dur", time.Since(start))

	return pr
}
