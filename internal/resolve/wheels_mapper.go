package resolve

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.lsp.dev/uri"

	"github.com/cfmleditor/clif/internal/parser"
)

// Wheels' Mapper copies public function references from wheels.mapper.* into
// variables and this. Recognize its cached-plan loader from source, rather than
// treating a directory name or an unused $integrateComponents helper as a mixin.
// The other Wheels loaders have different override rules and are not covered.
func (r *Resolver) wheelsMapperFunc(path, name string) (*parser.FunctionDef, bool) {
	if !strings.EqualFold(filepath.Base(path), "Mapper.cfc") {
		return nil, false
	}

	setup, ok := r.wheelsMapperSetup(path)
	if !ok {
		return nil, false
	}

	source, copyBody, global, folder := setup.source, setup.copyBody, setup.global, setup.folder

	entries, err := r.fs().ReadDir(folder)
	if err != nil || len(entries) > 128 {
		return nil, false
	}

	var found *parser.FunctionDef

	methods := maps.Clone(source.methodMap())
	seenMethods := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".cfc") {
			continue
		}

		file := filepath.Join(folder, entry.Name())
		// The loader's metadata scan is nonrecursive and reads this component's
		// public functions. Do not import includes, inherited or private methods.
		fileMethods := r.sourceMethods(file)
		for key := range fileMethods {
			method := fileMethods[key]
			if !method.public {
				continue
			}

			if seenMethods[key] {
				methods[key] = wheelsMethod{}
			} else {
				methods[key] = method
			}

			seenMethods[key] = true
		}

		method, ok := fileMethods[strings.ToLower(name)]
		if !ok || !method.public {
			continue
		}

		for _, def := range r.EnsureIndexed(file) {
			if strings.EqualFold(def.Name, name) {
				// DirectoryList does not specify an order. An ambiguous override cannot
				// provide a stable definition or return contract.
				if found != nil {
					return nil, true
				}

				found = def
			}
		}
	}

	if found != nil {
		// CFML's struct return annotation is also used for these components.
		// A copied function returning this is bound to Mapper, not its stateless
		// source holder. Keep the index definition untouched for direct calls.
		if wheelsReturnsMapper(strings.ToLower(name), methods) {
			bound := *found
			bound.ReturnComponent = path
			bound.ReturnSources = nil
			bound.CollectionValues = nil

			return &bound, true
		}
	}

	if found == nil && !strings.EqualFold(name, "get") && !strings.EqualFold(name, "controller") {
		// Global is copied first, before mapper package methods. Only add its
		// otherwise missing public API; included UDFs require the Adobe fallback.
		if _, own := source.methodMap()[strings.ToLower(name)]; !own {
			def := r.wheelsPlanFunc(global, name)
			if def != nil && (samePath(def.URI.Path(), global) || strings.Contains(copyBody, wheelsTokens(wheelsGlobalIncludeCopy))) {
				found = def
			}
		}
	}

	return found, found != nil
}

// The cached public-method producer is shared by Mapper and Controller.
func (r *Resolver) wheelsIntegrationPlan(global string) bool {
	plan := r.wheelsPlanFunc(global, "$componentIntegrationPlan")

	builder := r.wheelsPlanFunc(global, "$buildComponentIntegrationPlan")
	if plan == nil || builder == nil {
		return false
	}

	planBody := r.sourceMethods(plan.URI.Path())["$componentintegrationplan"].body
	builderBody := r.sourceMethods(builder.URI.Path())["$buildcomponentintegrationplan"].body
	// Both the producer and consumer must agree on publicMethods. Comments and
	// quoted descriptions cannot supply this evidence: token boundaries are kept.
	for _, evidence := range []string{
		`local.folderPath = ExpandPath("/#Replace(arguments.path, ".", "/", "all")#");`,
		`local.fileList = DirectoryList(local.folderPath, false, "name", "*.cfc");`,
		`local.instance = CreateObject("component", "#arguments.path#.#local.componentName#");`,
		`publicMethods = local.publicMethods`,
		`return local.rv;`,
	} {
		if !strings.Contains(builderBody, wheelsTokens(evidence)) {
			return false
		}
	}

	// The member name and reference must stay paired inside the public filter.
	// Requiring a contiguous body excludes intervening rewrites and escapes.
	split := `if (local.fns[local.f].access == "public") { local.ref = local.instance[local.fns[local.f].name]; ArrayAppend(local.publicMethods, {name = local.fns[local.f].name, ref = local.ref}); }`
	inline := `if (local.fns[local.f].access == "public") { ArrayAppend(local.publicMethods, {name = local.fns[local.f].name, ref = local.instance[local.fns[local.f].name]}); }`

	return (strings.Contains(builderBody, wheelsTokens(split)) || strings.Contains(builderBody, wheelsTokens(inline))) && strings.Contains(planBody, wheelsTokens(`$buildComponentIntegrationPlan(arguments.path)`))
}

// Global's known integration helpers are declared directly or in literal
// includes. Do not re-enter arbitrary inheritance/mixin lookup while checking
// the loader itself (Global can otherwise form a cycle back to Mapper).
func (r *Resolver) wheelsPlanFunc(global, name string) *parser.FunctionDef {
	var found *parser.FunctionDef

	_, own := r.sourceMethods(global)[strings.ToLower(name)]
	if own {
		for _, def := range r.EnsureIndexed(global) {
			if strings.EqualFold(def.Name, name) {
				found = def

				break
			}
		}
	} else {
		found = r.includedFunc(global, name)
	}

	if found == nil || !r.sourceMethods(found.URI.Path())[strings.ToLower(name)].public {
		return nil
	}

	return found
}

type wheelsMethod struct {
	body              string
	public            bool
	literalDefaults   bool
	noArgsReturn      string
	selfDefaultParam  string
	selfExcludedModes []string
}

// sourceInfo is what is read once from a file on disk and kept: its method
// bodies, producer plans and the facts derived from them. It began as Wheels'
// method-body check and every producer evaluation now reads it, Wheels or not.
// It is not merged with sourceFacts: that one reads the text a parse holds,
// which in the editor is the open buffer, where this one reads the disk.
type sourceInfo struct {
	associations map[string]parser.FunctionDef
	content      string
	derived      *sourceDerived // facts read from content on first use

	// size and modTime are the file's stamp when content was read. An
	// unchanged stamp serves the cache without reading the file again: this
	// is asked for every untyped return lookup, on the keystroke path.
	size    int64
	modTime time.Time

	// gen is the index generation the stamp was last checked at, when the
	// index tracks the disk.
	gen uint64
}

// Cache lexical work, not definitions or directory listings. Re-read bytes
// when the file's size or modification time moves, so a changed loader/access
// modifier is not accepted by an old policy; EnsureIndexed supplies the
// current definition from the shared index.
func (r *Resolver) sourceInfo(path string) sourceInfo {
	if r.indexer != nil {
		return r.indexer.sourceInfo(path)
	}

	var gen uint64
	if r.IndexTracksFiles && r.Index != nil {
		gen = r.Index.Generation()
	}

	r.mu.RLock()
	cached, ok := r.sourceInfos[path]
	r.mu.RUnlock()

	// Nothing the index has seen has changed since the stamp was checked, and
	// the index sees every change: see IndexTracksFiles.
	if ok && r.IndexTracksFiles && r.Index != nil && cached.gen == gen {
		return cached
	}

	info, err := r.fs().Stat(path)
	if err != nil {
		return sourceInfo{}
	}

	if ok && cached.size == info.Size() && cached.modTime.Equal(info.ModTime()) {
		if r.IndexTracksFiles {
			cached.gen = gen
			r.storeSourceInfo(path, &cached)
		}

		return cached
	}

	text, err := r.readText(path)
	if err != nil {
		return sourceInfo{}
	}

	content := text

	result := cached
	if !ok || cached.content != content {
		result = sourceInfo{content: content, derived: &sourceDerived{path: path}}
		result.associations = wheelsAssociations(content, func() *parser.ParseResult { return result.plainParse(path) })
	}

	result.size, result.modTime, result.gen = info.Size(), info.ModTime(), gen

	return r.adoptSourceInfo(path, result)
}

// adoptSourceInfo stores src for path and returns it, unless another caller
// stored the same content first: then that one is kept and returned, with
// src's stamp. Two workers that miss a source together each read it, and the
// one that stored last used to replace the other's, so whatever the first had
// derived (its parses, its spans, its bodies) was derived again; a MasaCMS
// scan parsed some files a third time over.
func (r *Resolver) adoptSourceInfo(path string, src sourceInfo) sourceInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sourceInfos == nil {
		r.sourceInfos = make(map[string]sourceInfo)
	}

	if held, ok := r.sourceInfos[path]; ok && held.derived != nil && held.content == src.content {
		held.size, held.modTime, held.gen = src.size, src.modTime, src.gen
		src = held
	}

	r.sourceInfos[path] = src

	return src
}

// sourceDerived holds what is read from a source's content once, the first
// time it is asked for, and shared by every copy of its sourceInfo.
type sourceDerived struct {
	uncommentedOnce sync.Once
	uncommented     string
	producersOnce   sync.Once
	producers       *producerPlans
	requestOnce     sync.Once
	requestWrites   map[string]int
	startupOnce     sync.Once
	startup         []parser.StartupVariableAssignment
	fieldsOnce      sync.Once
	fields          variablesReferences
	methodsOnce     sync.Once
	methods         map[string]wheelsMethod
	tagBodiesOnce   sync.Once
	tagBodies       map[string]string
	elementOnce     sync.Once
	elementReads    bool
	plainOnce       sync.Once
	plain           *parser.ParseResult
	plainMade       atomic.Pointer[parser.ParseResult] // plain, once made
	selfOnce        sync.Once
	mentionsSelf    bool
	wrappersOnce    sync.Once
	wrappers        map[string]parser.IncludeWrapper
	spansOnce       sync.Once
	spans           []parser.FunctionSpan
	bodiesOnce      sync.Once
	bodyReaders     atomic.Int32 // the readers that have the bodies (doneWithBodies)
	bodies          [][]parser.Token
	assignmentsOnce sync.Once
	assigned        []parser.Assignment
	path            string // the file content was read from, for its extension
}

// functionSpansOf is the functions the parse finds in content, read from
// file just now: its source's, read once, when that holds the same text.
func (r *Resolver) functionSpansOf(file, content string) []parser.FunctionSpan {
	if source := r.sourceInfo(file); source.derived != nil && source.content == content {
		return source.functionSpans()
	}

	return parser.FunctionSpansOf(file, content)
}

// functionSpans is the source's functions as the parse finds them
// (parser.FunctionSpansOf, the functions-only reading of the parse's walk),
// read once per source: the method maps and plans below are built from them
// rather than finding the functions again.
func (s *sourceInfo) functionSpans() []parser.FunctionSpan {
	if s.derived == nil {
		return parser.FunctionSpansOf("", s.content)
	}

	// A source whose plain parse is made already holds its spans there (a
	// full parse's); reading them on their own parses the text again. One
	// not yet made keeps the lighter spans-only reading, since many sources
	// asked for their functions are never asked for that parse.
	s.derived.spansOnce.Do(func() {
		if plain := s.derived.plainMade.Load(); plain != nil {
			s.derived.spans = plain.FunctionSpans()
		} else {
			s.derived.spans = parser.FunctionSpansOf(s.derived.path, s.content)
		}
	})

	return s.derived.spans
}

// functionBodies is the script tokens of each of the source's function
// bodies (functionBodyTokens), by the index of its span, read once: the
// Wheels method reader and the producer plans read the same bodies, and each
// tokenized them for itself.
//
// They are dropped once both readers have them: a source's readers keep
// strings, not tokens, and kept for every source a scan reads the bodies were
// 10.7MB of MasaCMS's peak heap. Nearly every source asked for one reader asks
// for the other (167 of 168 on MasaCMS). A reader that comes after the drop
// finds nil and tokenizes the bodies itself.
func (s *sourceInfo) functionBodies() [][]parser.Token {
	s.derived.bodiesOnce.Do(func() { s.derived.bodies = functionBodyTokens(s.content, s.functionSpans()) })

	return s.derived.bodies
}

// doneWithBodies records that one of the two readers of the function bodies
// has them, and drops them when it is the second.
func (s *sourceInfo) doneWithBodies() {
	if s.derived.bodyReaders.Add(1) == 2 {
		s.derived.bodies = nil
	}
}

// sourcePlans is the source's producer plans (newProducerPlans), built on
// first use: a source is made for nearly every file a lookup touches, and
// few are asked what their methods produce. nil for a source never read.
func sourcePlans(s sourceInfo) *producerPlans {
	if s.derived == nil {
		return nil
	}

	s.derived.producersOnce.Do(func() {
		s.derived.producers = newProducerPlansFrom(s.content, s.functionSpans(), s.functionBodies())
		s.doneWithBodies()
	})

	return s.derived.producers
}

// methodMap is the source's function bodies (wheelsMethods), read once per
// source on first use: most sources are never asked for one, and reading them
// copies every body.
func (s *sourceInfo) methodMap() map[string]wheelsMethod {
	if s.derived == nil {
		return wheelsMethods(s.content, s.functionSpans())
	}

	s.derived.methodsOnce.Do(func() {
		s.derived.methods = wheelsMethodsFrom(s.content, s.functionSpans(), s.functionBodies())
		s.doneWithBodies()
	})

	return s.derived.methods
}

// includeWrappers is the source's include wrappers (parser.IncludeWrappers),
// read once per source.
func (s *sourceInfo) includeWrappers() map[string]parser.IncludeWrapper {
	if s.derived == nil {
		return parser.IncludeWrappers(s.content)
	}

	s.derived.wrappersOnce.Do(func() { s.derived.wrappers = parser.IncludeWrappers(s.content) })

	return s.derived.wrappers
}

// hasSelfDefaultParam reports whether fd's body ends in a dispatch on a
// parameter defaulting to "self" (defaultSelfReturnModes). That needs the string
// "self" in the file, so a file without it is not asked for its function bodies,
// which every return lookup would otherwise read.
func (r *Resolver) hasSelfDefaultParam(fd *parser.FunctionDef) bool {
	source := r.sourceInfo(fd.URI.Path())
	if !source.mentionsSelf() {
		return false
	}

	return source.methodMap()[strings.ToLower(fd.Name)].selfDefaultParam != ""
}

// mentionsSelf reports whether the source spells the string "self" in either
// quote, read once per source: hasSelfDefaultParam is asked on every return
// lookup, and searching the whole file each time was over a second of a
// tassweb scan.
func (s *sourceInfo) mentionsSelf() bool {
	read := func() bool { return mentions(s.content, `"self"`) || mentions(s.content, `'self'`) }
	if s.derived == nil {
		return read()
	}

	s.derived.selfOnce.Do(func() { s.derived.mentionsSelf = read() })

	return s.derived.mentionsSelf
}

// sourceMethods is sourceInfo(path)'s function bodies.
func (r *Resolver) sourceMethods(path string) map[string]wheelsMethod {
	source := r.sourceInfo(path)

	return source.methodMap()
}

// requestWriteCounts is how many writes of each request key the parse reads
// in the source (requestWriteCounts), counted once per source.
func (s *sourceInfo) requestWriteCounts() map[string]int {
	if s.derived == nil {
		return requestWritesIn("", s.content)
	}

	s.derived.requestOnce.Do(func() {
		var inserts []parsedCall
		if mentions(s.content, "structInsert") {
			inserts = parsedCalls(s.plainParse(s.derived.path), "structInsert")
		}

		s.derived.requestWrites = requestWriteCounts(s.assignments(), inserts)
	})

	return s.derived.requestWrites
}

// withoutComments is the source with its comments blanked (withoutComments),
// read once per source: a forwarded field is checked against it for every
// call that reads it, and each check tokenized the whole file again (on Mura,
// nearly twice its text in a scan).
func (s *sourceInfo) withoutComments() string {
	if s.derived == nil {
		return withoutComments(s.content)
	}

	s.derived.uncommentedOnce.Do(func() { s.derived.uncommented = withoutComments(s.content) })

	return s.derived.uncommented
}

// variablesReferences is how the source refers to its variables scope,
// counted once per source (countVariablesReferences).
func (s *sourceInfo) variablesReferences() variablesReferences {
	if s.derived == nil {
		return countVariablesReferences(allTokens(s.content))
	}

	s.derived.fieldsOnce.Do(func() { s.derived.fields = countVariablesReferences(allTokens(s.content)) })

	return s.derived.fields
}

// startupAssignments is the source's component-body variable assignments,
// parsed once per source rather than on every producer evaluation.
func (s *sourceInfo) startupAssignments(file uri.URI) []parser.StartupVariableAssignment {
	if s.derived == nil {
		return parser.Parse(file, s.content).StartupVariableAssignments()
	}

	// The plain parse, which other lookups of the source share.
	s.derived.startupOnce.Do(func() { s.derived.startup = s.plainParse(file.Path()).StartupVariableAssignments() })

	return s.derived.startup
}

func (r *Resolver) storeSourceInfo(path string, src *sourceInfo) {
	r.mu.Lock()
	if r.sourceInfos == nil {
		r.sourceInfos = make(map[string]sourceInfo)
	}

	r.sourceInfos[path] = *src
	r.mu.Unlock()
}

// wheelsMethods is the source's script methods by lowercased name, read from
// the functions the parse found (spans, its FunctionSpans of content): each
// function declared as a statement of the component or of a template's
// script. Tag functions are left out by design, as are named functions
// nested in another's body and functions assigned to a name, which no Wheels
// loader copies; and so is a declaration with no body.
func wheelsMethods(content string, spans []parser.FunctionSpan) map[string]wheelsMethod {
	return wheelsMethodsFrom(content, spans, nil)
}

// wheelsMethodsFrom is wheelsMethods with bodies, each span's body tokens
// (functionBodyTokens), or nil to read them here.
func wheelsMethodsFrom(content string, spans []parser.FunctionSpan, bodies [][]parser.Token) map[string]wheelsMethod {
	methods := make(map[string]wheelsMethod)

	for i := range spans {
		s := &spans[i]
		if s.Kind != parser.FunctionDeclared || s.Body < 0 || s.Params < 0 {
			continue
		}

		var body []parser.Token
		if bodies != nil {
			body = bodies[i]
		} else {
			body = scriptTokens(s.BodyText(content))
		}

		if method, ok := wheelsReadMethodFrom(content, s, body); ok {
			methods[strings.ToLower(s.Name)] = method
		}
	}

	return methods
}

// wheelsReadMethod reads the method s declares in content: public unless it
// states another access, its parameters, and its body as wheelsTokens text.
func wheelsReadMethod(content string, s *parser.FunctionSpan) (wheelsMethod, bool) {
	return wheelsReadMethodFrom(content, s, scriptTokens(s.BodyText(content)))
}

// wheelsReadMethodFrom is wheelsReadMethod given body, the body's script
// tokens.
func wheelsReadMethodFrom(content string, s *parser.FunctionSpan, bodyTokens []parser.Token) (wheelsMethod, bool) {
	// The list with its closing parenthesis, which beanArguments reads to.
	params := beanArguments(parser.NewScriptScanner(content[s.Params : s.ParamsEnd+1]))
	if params == nil {
		return wheelsMethod{}, false
	}

	public := s.Access == "" || strings.EqualFold(s.Access, "public")

	var (
		body   strings.Builder
		tokens []parser.Token
	)

	for _, tok := range bodyTokens {
		wheelsWriteToken(&body, tok)

		if len(tokens) <= 2048 {
			tokens = append(tokens, tok)
		}
	}

	text := body.String()
	parameter, modes := defaultSelfReturnModes(params, text)

	return wheelsMethod{
		body: text, public: public, literalDefaults: wheelsLiteralDefaults(params), noArgsReturn: absentArgumentReturn(params, tokens),
		selfDefaultParam: parameter, selfExcludedModes: modes,
	}, true
}

func wheelsLiteralDefaults(params [][]parser.Token) bool {
	method := &producerMethod{defaults: map[string]string{}}

	for i, param := range params {
		eq := slices.IndexFunc(param, func(token parser.Token) bool { return token.Kind == parser.TokEquals })
		if eq >= 0 {
			method.defaults[producerPosition(i)] = producerText(param[eq+1:])
		}
	}

	return instanceLiteralDefaults(method)
}

// defaultSelfReturnModes recognizes a final literal-mode dispatch whose
// default parameter value selects `return this`. Work before the final dispatch
// may be too dynamic for a producer plan (beanORM.loadBy builds SQL with
// savecontent). The prefix must still exclude earlier returns and uses of the
// selected parameter or arguments scope that could change the final dispatch.
func defaultSelfReturnModes(params [][]parser.Token, body string) (string, []string) {
	for _, param := range params {
		eq := slices.IndexFunc(param, func(t parser.Token) bool { return t.Kind == parser.TokEquals })
		if eq < 1 || eq+2 != len(param) || param[eq-1].Kind != parser.TokIdent || param[eq+1].Kind != parser.TokString || !strings.EqualFold(strings.Trim(param[eq+1].Value, "\"'"), "self") {
			continue
		}

		name := strings.ToLower(param[eq-1].Value)

		start, found := selfDispatchStart(body, name)
		if !found || !selfDispatchPrefixSafe(body[:start], name) {
			continue
		}

		match := body[start:]
		if !strings.Contains(match, name+"\teq\t\"self\"") && !strings.Contains(match, name+"\t=\t=\t\"self\"") {
			var modes []string
			for _, mode := range selfDispatchModes(match, name) {
				modes = append(modes, strings.ToLower(mode))
			}

			return name, modes
		}
	}

	return "", nil
}

// A suffix match alone says nothing about the code that precedes it. Keep the
// dispatch at method scope and reject returns and any parameter/scope use in
// the prefix, including passing the arguments scope to another function.
func selfDispatchPrefixSafe(prefix, name string) bool {
	tokens := producerTokens(prefix)
	if tokens == nil {
		return false
	}

	depth := 0

	for i, token := range tokens {
		if producerUnbracedControl(tokens, i) {
			return false
		}

		if token.Kind == parser.TokIdent && (strings.EqualFold(token.Value, "return") || strings.EqualFold(token.Value, "arguments") || strings.EqualFold(token.Value, name)) {
			return false
		}

		switch token.Kind {
		case parser.TokLBrace:
			depth++
		case parser.TokRBrace:
			depth--
		default:
		}

		if depth < 0 {
			return false
		}
	}

	return depth == 0
}

func wheelsTokens(source string) string {
	scanner := parser.NewSourceScanner(source)

	var result strings.Builder

	for {
		tok := scanner.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			return result.String()
		}

		wheelsWriteToken(&result, tok)
	}
}

func wheelsWriteToken(dst *strings.Builder, tok parser.Token) {
	value := tok.Value
	if tok.Kind == parser.TokString && len(value) >= 2 {
		quote := value[:1]
		value = `"` + strings.ReplaceAll(value[1:len(value)-1], quote+quote, quote) + `"`
	}

	dst.WriteString(strings.ToLower(value))
	dst.WriteByte('\t')
}

// Follow only self calls in return expressions, with bounded recursion. This
// covers scope()/resource() wrappers and $match's recursive member()/end()
// branch without assigning a Mapper type to an arbitrary struct return.
func wheelsReturnsMapper(name string, methods map[string]wheelsMethod) bool {
	anchored := false
	budget := 64

	var visit func(string, int, map[string]bool) bool

	visit = func(name string, depth int, active map[string]bool) bool {
		if active[name] {
			return true
		}

		if depth > 8 || budget <= 0 {
			return false
		}

		budget--

		body := methods[name].body
		if body == "" {
			return false
		}

		active[name] = true
		defer delete(active, name)

		scanner := parser.NewSourceScanner(body)
		returns := 0

		for {
			tok := scanner.NextSkipComments()
			if tok.Kind == parser.TokEOF {
				return returns > 0
			}

			if tok.Kind != parser.TokIdent || !strings.EqualFold(tok.Value, "return") {
				continue
			}

			returns++

			first := scanner.NextSkipComments()
			if strings.EqualFold(first.Value, "this") && scanner.PeekSkipComments().Kind == parser.TokSemicolon {
				anchored = true

				continue
			}

			if strings.EqualFold(first.Value, "this") || strings.EqualFold(first.Value, "variables") {
				if scanner.NextSkipComments().Kind != parser.TokDot {
					return false
				}

				first = scanner.NextSkipComments()
			}

			for {
				if first.Kind != parser.TokIdent || scanner.NextSkipComments().Kind != parser.TokLParen {
					return false
				}

				if beanArguments(scanner) == nil || !visit(strings.ToLower(first.Value), depth+1, active) {
					return false
				}

				next := scanner.NextSkipComments()
				if next.Kind == parser.TokSemicolon {
					break
				}

				if next.Kind != parser.TokDot {
					return false
				}

				first = scanner.NextSkipComments()
			}
		}
	}

	return visit(name, 0, make(map[string]bool)) && anchored
}

const wheelsGlobalIncludeCopy = `if (StructKeyExists(arguments.componentInstance, "$frameworkGlobalFunctionNames")) {
 local.includeNames=arguments.componentInstance.$frameworkGlobalFunctionNames();
 local.includeCount=ArrayLen(local.includeNames);
 for(local.n=1;local.n<=local.includeCount;local.n++) {
  local.functionName=local.includeNames[local.n];
  if(!StructKeyExists(this,local.functionName) && (!ListFindNoCase(local.excludeList,local.functionName) || FindNoCase("wheels.mapper",local.componentName))) {
   variables[local.functionName]=arguments.componentInstance[local.functionName];
   this[local.functionName]=arguments.componentInstance[local.functionName];
  }
 }
}`

type wheelsMapperLoader struct {
	source                   sourceInfo
	copyBody, global, folder string
}

// wheelsMapperSetup reports whether path is Wheels' Mapper.cfc as pinned,
// with what its loader reads: its source, the copy body, wheels.Global and
// the mapper package folder.
func (r *Resolver) wheelsMapperSetup(path string) (wheelsMapperLoader, bool) {
	if !strings.EqualFold(filepath.Base(path), "Mapper.cfc") {
		return wheelsMapperLoader{}, false
	}

	source := r.sourceInfo(path)
	if source.methodMap()["init"].body != wheelsTokens(`local.globalComponent = createObject("wheels.Global"); $integrateFunctions(local.globalComponent); $integrateComponents("wheels.mapper"); return this;`) ||
		source.methodMap()["$integratecomponents"].body != wheelsTokens(`local.plan = $componentIntegrationPlan(arguments.path); local.iEnd = ArrayLen(local.plan); for (local.i = 1; local.i <= local.iEnd; local.i++) { $integrateFunctions(local.plan[local.i].instance, local.plan[local.i].publicMethods); }`) {
		return wheelsMapperLoader{}, false
	}

	copyBody := source.methodMap()["$integratefunctions"].body
	if !strings.HasPrefix(copyBody, wheelsTokens(`if (ArrayLen(arguments.publicMethods)) { local.iEnd = ArrayLen(arguments.publicMethods); for (local.i = 1; local.i <= local.iEnd; local.i++) { local.m = arguments.publicMethods[local.i]; variables[local.m.name] = local.m.ref; this[local.m.name] = local.m.ref; } return; }`)) {
		return wheelsMapperLoader{}, false
	}
	// init first copies Global's methods; otherwise the integration-plan call
	// below has no implementation on Mapper. Require that fallback path too.
	for _, evidence := range []string{
		`local.meta = getMetaData(arguments.componentInstance);`,
		`local.excludeList = "get,controller";`,
		`if (local.method.access == "public" && (!listFindNoCase(local.excludeList, local.functionName) || findNoCase("wheels.mapper", local.componentName))) { variables[local.functionName] = componentInstance[local.functionName]; this[local.functionName] = componentInstance[local.functionName]; }`,
	} {
		if !strings.Contains(copyBody, wheelsTokens(evidence)) {
			return wheelsMapperLoader{}, false
		}
	}

	global := r.ComponentPath("wheels.Global", filepath.Dir(path))
	if global == "" {
		return wheelsMapperLoader{}, false
	}

	if !r.wheelsIntegrationPlan(global) {
		return wheelsMapperLoader{}, false
	}

	// ExpandPath in the loader uses the active wheels mapping. Resolve an actual
	// source component as an anchor, and reject a basename fallback elsewhere.
	anchor := r.ComponentPath("wheels.mapper.mapping", filepath.Dir(path))
	if anchor == "" || !strings.EqualFold(filepath.Base(filepath.Dir(anchor)), "mapper") || !samePath(filepath.Dir(filepath.Dir(anchor)), filepath.Dir(path)) {
		return wheelsMapperLoader{}, false
	}

	folder := filepath.Dir(anchor)

	return wheelsMapperLoader{source: source, copyBody: copyBody, global: global, folder: folder}, true
}
