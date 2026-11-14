// The component a function returns: declared, inferred, documented or named.

package resolve

import (
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// chainHopReturn is the component a chain hop's method returns: its declared
// or inferred component, a dotted return type, the receiver for an untyped
// init(), or a componentResolver matching the hop. It reports the resolver's
// noFollow and whether the resolver is dynamicIfMissing, for the last.
//
// callText is the hop's call as written when it passes arguments, "" for a hop
// with none: a resolver is matched against that, so `event( "x" )` is not the
// `event()` that returns the event object.
func (r *Resolver) chainHopReturn(comp, hop, callText string, fd *parser.FunctionDef, tr *callTrace) (ret string, noFollow, soft bool) {
	ret = r.ReturnComponentOf(fd)
	if ret != "" {
		tr.addf("chain hop %q on %q: returns %q (declared %q)", hop, comp, ret, fd.ReturnType)
	}

	// An init() that declares nothing returns the object it was called
	// on: the CFC constructor convention, and what the parser assumes
	// when it types a variable assigned through one. Falling through to
	// the resolvers offered them init(), which nothing can answer.
	if ret == "" && strings.EqualFold(hop, "init") {
		ret = comp

		tr.addf("chain hop %q on %q: an untyped init() returns the object it is called on", hop, comp)
	}

	if ret != "" {
		return ret, false, false
	}

	// The real function's declared return type isn't a component (e.g.
	// a generic returntype="struct" on a factory method that actually
	// returns a specific component instance) — fall back to a
	// componentResolver matching the hop's own call shape, same as the
	// non-chain "altComp" fallback below.
	text := hop + "()"
	if callText != "" {
		text = callText
	}

	return r.matchResolver(text, tr, func(c, _ string, nf bool) {
		tr.addf("chain hop %q on %q: no declared return type — componentResolver matched %q(): %q (noFollow=%v)", hop, comp, hop, c, nf)
	})
}

// ReturnComponentOf is the component fd returns: the one the parse inferred
// from its return statements, a dotted return type, or a bare-word return type
// naming a component beside the declaring one. Every question about what a
// function returns asks this, so a chain hop and a variable assigned from the
// same call cannot disagree.
//
// An inferred `$any` does not outrank a declared component, dotted or a bare
// word naming one beside the declaring file. It means the body
// builds its result from a runtime expression (`getCustomObject(type="tassui")`
// goes through `createObject("component", "customobjects.#type#")`), while
// `returntype="pkg.tassui"` is a contract the engine enforces. Any other
// inferred component is kept, since it may be the more specific of the two.
func (r *Resolver) ReturnComponentOf(fd *parser.FunctionDef) string {
	if r.Index == nil || fd == nil {
		budget := 128

		return r.returnComponentOf(fd, 0, &budget)
	}

	// The generation is read before the answer is computed, so a write made
	// meanwhile moves it past the one the answer is stored under.
	gen := r.Index.Generation()

	var stamp fileStamp

	if fd.URI.IsFile() {
		stamp = r.returnStamp(gen, fd.URI.Path())
	}

	r.mu.RLock()
	entry, ok := r.returnCache.entries[fd]
	ok = ok && r.returnCache.gen == gen
	r.mu.RUnlock()

	if ok && entry.size == stamp.size && entry.modTime.Equal(stamp.modTime) {
		return entry.component
	}

	budget := 128
	component := r.returnComponentOf(fd, 0, &budget)

	r.mu.Lock()
	if r.returnCache.gen != gen || r.returnCache.entries == nil || len(r.returnCache.entries) >= maxReturnCache {
		r.returnCache = returnCache{gen: gen, entries: make(map[*parser.FunctionDef]returnEntry), stamps: r.returnCache.stampsFor(gen)}
	}

	r.returnCache.entries[fd] = returnEntry{component: component, size: stamp.size, modTime: stamp.modTime}
	r.mu.Unlock()

	return component
}

// getterReturn is what a getter of a field returns: a shared-scope field the
// startup files type, a lazily built one, or one its constructor is handed.
func (r *Resolver) getterReturn(fd *parser.FunctionDef) string {
	if ret := r.sharedGetterReturn(fd); ret != "" {
		return ret
	}

	if ret := r.lazyGetterReturn(fd); ret != "" {
		return ret
	}

	if ret := r.ownedKeyReturn(fd); ret != "" {
		return ret
	}

	if ret := r.presideObjectReturn(fd); ret != "" {
		return ret
	}

	if ret := r.forwardedFieldReturn(fd); ret != "" {
		if p := r.ComponentPath(ret, filepath.Dir(fd.URI.Path())); p != "" {
			return p
		}

		return ret
	}

	return ""
}

func (r *Resolver) returnComponentOf(fd *parser.FunctionDef, depth int, budget *int) string {
	if depth > 8 {
		return ""
	}

	if ret := r.abstractOverridesReturn(fd, depth, budget); ret != "" {
		return ret
	}

	if native := r.declaredCollectionReturn(fd); native != "" {
		return native
	}

	if fd.ReturnComponent == "" || fd.ReturnComponent == "$any" {
		if ret := r.wheelsFixedMapperReturn(fd); ret != "" {
			return ret
		}
	}

	if ret, done := r.producerReturn(fd, depth, budget); done {
		return ret
	}

	if len(fd.ReturnSources) > 0 {
		return r.sourcesReturn(fd, depth, budget)
	}

	if ret, done := r.declaredReturn(fd); done {
		return ret
	}

	if p := r.bareReturnComponent(fd); p != "" {
		return p
	}

	// An inferred $any, with nothing declared, is still the answer; a doc
	// comment is not a contract, so it does not outrank one.
	if fd.ReturnComponent != "" {
		return fd.ReturnComponent
	}

	if ret := r.returnViaComponent(fd, depth, budget); ret != "" {
		return ret
	}

	if ret := r.docReturnComponent(fd); ret != "" {
		return ret
	}

	return r.aboveDeclaring(fd, fd.ReturnType)
}

// producerReturn is the component a function that declares none returns, as
// the plan of its body evaluates (producer_*.go), when that is the answer:
// done says whether the declared contracts below it in returnComponentOf are
// skipped, which an answer of "" can ask for too.
func (r *Resolver) producerReturn(fd *parser.FunctionDef, depth int, budget *int) (ret string, done bool) {
	if !fd.URI.IsFile() || fd.ReturnType != "" && !strings.EqualFold(fd.ReturnType, "any") && !strings.EqualFold(fd.ReturnType, "component") && !strings.EqualFold(fd.ReturnType, "object") {
		return "", false
	}

	// A final mode dispatch has no context-free component return. The
	// caller must select its branch before a partial parser inference can
	// supply one of the alternatives as if it were unconditional.
	if r.hasSelfDefaultParam(fd) {
		return "", true
	}

	if ret := r.getterReturn(fd); ret != "" {
		return ret, true
	}

	method := r.producerFor(fd)

	// A value the function adds members to before returning it is not
	// its component, documented or inferred: ColdBox's execute() returns a
	// RequestContext with getRenderedContent() and two more attached.
	if method != nil && producerAugmentsReturn(method.body) && !r.producerScalarSelfSetter(method, fd, "") {
		return "", true
	}

	if method != nil && (producerNeedsSpecialization(method, fd) || (fd.ReturnComponent == "" || fd.ReturnComponent == "$any") && len(fd.ReturnSources) == 0 && fd.DocReturn == "") {
		eval := producerEvaluation{resolver: r, fd: fd, baseDir: filepath.Dir(fd.URI.Path()), depth: depth, budget: budget}

		ret := eval.call(fd, "", nil, false).component()

		// An unproven answer withholds the declared contracts below only
		// when the body guards on its arguments (producerSensitive): then
		// the return depends on the call. A return merely derived from a
		// call does not say that, and a documented @return still applies.
		// ColdBox's getContext() has no semicolons; read as a plan, its
		// lock block hid the returns, and it lost its documented type.
		if ret == "" {
			ret = r.unevaluatedReturn(fd, depth, budget)
		}

		if ret != "" || producerNeedsSpecialization(method, fd) && (fd.DocReturn == "" || producerSensitive(method.body)) {
			return ret, true
		}
	}

	return "", false
}

// sourcesReturn is the answer for a function with return statements the
// parser recorded the sources of: a declared type wins, else what the
// collection those sources build holds.
func (r *Resolver) sourcesReturn(fd *parser.FunctionDef, depth int, budget *int) string {
	switch strings.ToLower(fd.ReturnType) {
	case "", "any", "component", "object":
	default:
		if strings.Contains(fd.ReturnType, ".") {
			return r.declaredReturnType(fd)
		}

		return r.bareReturnComponent(fd)
	}

	return r.collectionReturnOf(fd, depth, budget)
}

// declaredReturn is the component a function names in its own return: a
// runtime path (dynamic), a component, or a dotted return type, each read
// beside the declaring file. done is false when it declares none of those.
func (r *Resolver) declaredReturn(fd *parser.FunctionDef) (ret string, done bool) {
	switch {
	case strings.Contains(fd.ReturnComponent, "#"):
		// `return new "builders.#name#.Builder"()`: a runtime path no
		// expressionMappings entry covered is dynamic, as it is assigned to
		// a variable (parser replaceExpressions).
		return "$any", true
	case fd.ReturnComponent != "" && fd.ReturnComponent != "$any":
		// `return new Expectation( … )` names the component beside the
		// declaring file, as a bare return type does; read from the caller's
		// directory it named nothing, and every expect( x ).toBe() in a
		// spec extending the Wheels test BaseSpec was a missing component.
		if p := r.besideDeclaring(fd, fd.ReturnComponent); p != "" {
			return p, true
		}

		return fd.ReturnComponent, true
	case strings.Contains(fd.ReturnType, "."):
		// A relative path is the declaring file's too: ColdBox's AsyncManager
		// returns `new tasks.Future()`, which is system/async/tasks/Future.cfc
		// and nothing from the caller's directory.
		if p := r.besideDeclaring(fd, fd.ReturnType); p != "" {
			return p, true
		}

		if native := nativetype.Java(fd.ReturnType); native != "" {
			return native, true
		}

		return fd.ReturnType, true
	}

	return "", false
}

// docReturnComponent is the component fd's doc comment says it returns, when
// its declaration says nothing more — no type, `any` or `component` — and
// only where the documented path names a file. ColdBox documents most return
// types and declares few, and some of its documented paths are wrong
// (execute()'s names coldbox.system.context.RequestContext): taken on trust,
// one of those would be reported as a component that does not exist. The
// bundled stubs read the same comments (cmd/cfstubgen docReturn).
func (r *Resolver) docReturnComponent(fd *parser.FunctionDef) string {
	switch strings.ToLower(fd.ReturnType) {
	case "", "any", "component":
	default:
		return ""
	}

	if fd.DocReturn == "" {
		return ""
	}

	if native := nativetype.Java(fd.DocReturn); native != "" {
		return native
	}

	p := r.besideDeclaring(fd, fd.DocReturn)

	if p == "" {
		file := cfpath.FromURI(string(fd.URI))
		if file == "" {
			return ""
		}

		p = r.ComponentPath(fd.DocReturn, filepath.Dir(file))
	}

	return p
}

// declaresInterface reports whether content declares an interface: a line
// starting with the word interface (after any whitespace), or a <cfinterface
// tag, case-insensitively, as the expression
//
//	(?im)^\s*interface\b|<cfinterface\b
//
// matched it. It looks only where the word is: that expression, run over every
// component a missing method was checked on, was the largest regular
// expression left in a tassweb scan. TestDeclaresInterfaceMatchesTheExpression
// holds the two together.
func declaresInterface(content string) bool {
	for at := 0; ; {
		i := parser.IndexFoldFrom(content, "interface", at)
		if i < 0 {
			return false
		}

		at = i + 1

		end := i + len("interface")
		if end < len(content) && lex.IsWord(content[end]) {
			continue
		}

		if i >= 3 && strings.EqualFold(content[i-3:i], "<cf") {
			return true
		}

		// ^\s* in multi-line mode: only whitespace before it on its line, or
		// a run of whitespace back to the start of the file.
		j := i
		for j > 0 && lex.IsRegexpSpace(content[j-1]) && content[j-1] != '\n' {
			j--
		}

		if j == 0 || content[j-1] == '\n' {
			return true
		}
	}
}

// isInterface reports whether the file at path declares an interface.
func (r *Resolver) isInterface(path string) bool {
	if path == "" {
		return false
	}

	r.mu.RLock()
	answer, ok := r.interfaceCache[path]
	r.mu.RUnlock()

	if ok {
		return answer
	}

	// Read and matched once per path: it is asked for every call checked on
	// a component that lacks the method, and the whole-file regex was 6% of
	// an unresolved scan. InvalidatePaths drops it with the path caches.
	text, err := r.readText(path)
	answer = err == nil && declaresInterface(text)

	r.mu.Lock()
	if r.interfaceCache == nil {
		r.interfaceCache = map[string]bool{}
	}

	r.interfaceCache[path] = answer
	r.mu.Unlock()

	return answer
}

// FuncLookup is the parser's hook for what a method of a component returns,
// resolved from baseDir: ParseOptions.FuncLookup, which types a variable
// assigned from a call on another component.
func (r *Resolver) FuncLookup(baseDir string) func(component, funcName string) string {
	lookup := r.funcLookupUncached(baseDir)
	if !r.SnapshotScan {
		return lookup
	}

	// In a batch scan the answer is a function of the caller's application,
	// the directory, the component and the name alone (see funcCache for why
	// it holds for the whole scan), and the files of one directory ask the
	// same questions: each parse memoised only its own, which left 2.3s of
	// tassweb's scan in lookups another file beside it had already made. The
	// cache is the owner's, shared by every caller-context view, and a view
	// keeps its own application's mappings whatever directory it is asked
	// from, so the application is part of the key: without it, whichever
	// view asked first answered for all of them, and a test suite's
	// `/root` mapping came and went with the scan's goroutine order.
	cache := &r.owner().funcLookups
	app := r.forCaller(baseDir).callerApplication

	return func(component, funcName string) string {
		key := funcLookupKey{app: app, dir: baseDir, component: component, name: funcName}
		if answer, ok := cache.Load(key); ok {
			s, _ := answer.(string)

			return s
		}

		answer := lookup(component, funcName)
		cache.Store(key, answer)

		return answer
	}
}

// funcLookupKey is one FuncLookup question in a batch scan.
type funcLookupKey struct{ app, dir, component, name string }

func (r *Resolver) funcLookupUncached(baseDir string) func(component, funcName string) string {
	r = r.forCaller(baseDir)

	return func(component, funcName string) string {
		if property, ok := parser.PropertyName(funcName); ok {
			return r.publicPropertyComponent(component, property, baseDir)
		}

		expression, callHop := parser.CallExpression(funcName)
		if callHop {
			// A factory call the configured bean rules name: the
			// container's getBean( "content" ), whose construction the
			// policy spells.
			if comp, _, _ := parser.ResolveFromCallMatch(expression, r.Resolvers); comp != "" {
				return comp
			}

			funcName = wheelsCallName(expression)
		}

		if ret, ok := docs.LookupBuiltinMethodReturn(component, funcName); ok {
			return ret
		}

		fd := r.ResolveFunc(component, funcName, baseDir)
		if fd == nil {
			return r.slatwallGatewayReturn(r.ComponentPath(component, baseDir), funcName, baseDir)
		}

		if ret := r.ReturnComponentOf(fd); ret != "" {
			if sub := r.selfTyped(component, baseDir, fd, ret); sub != "" {
				return sub
			}

			return ret
		}

		if callHop {
			// A factory call the configured bean rules name: the container's
			// getBean( "x" ), whose construction the policy spells.
			if comp, _, _ := parser.ResolveFromCallMatch(expression, r.Resolvers); comp != "" {
				return comp
			}

			if ret := r.expressionReturn(fd, expression, baseDir, component); ret != "" {
				return ret
			}
		}

		if callHop {
			funcName = parser.CallHop(expression)
		}

		return r.receiverReturn(r.ComponentPath(component, baseDir), funcName)
	}
}

// cfmlTypes are the return types CFML itself defines. A component of one of
// these names beside the declaring file does not make returntype="query" a
// component: the engine's type wins, as it does at runtime.
var cfmlTypes = map[string]bool{
	"any": true, "array": true, "binary": true, "boolean": true, "component": true,
	"date": true, "datetime": true, "email": true, "function": true, "closure": true,
	"guid": true, "numeric": true, "integer": true, "number": true, "query": true,
	"string": true, "struct": true, "uuid": true, "variablename": true, "void": true,
	"xml": true, "object": true, "regex": true, "url": true, "double": true,
}

// bareReturnComponent is the component a bare-word return type names:
// ColdBox declares `ColdBoxScheduledTask function task( name )`, and CFML
// finds that component beside the declaring one. Only there — the resolver's
// wider search would take any file of the name anywhere in the workspace, and
// a bare word is far more often a type than a component, which is why a
// dotted return type was the only one taken. The answer is the file's path,
// as a method returning `this` is typed.
func (r *Resolver) bareReturnComponent(fd *parser.FunctionDef) string {
	t := strings.TrimSpace(fd.ReturnType)
	if cfmlTypes[strings.ToLower(t)] {
		return ""
	}

	return r.besideDeclaring(fd, t)
}

// namedType is the component a bare argument type t, declared by fd's file,
// stands for: the file beside it, or else the one component of that name in
// the workspace. The engine checks a component argument by the component's name,
// so qb's grammars declare `required QueryBuilder query` for a
// QueryBuilder.cfc in another directory. A bare return type is still read
// beside the declaring file only (TestABareReturnTypeNamesTheComponentBesideIt).
func (r *Resolver) namedType(fd *parser.FunctionDef, t string) string {
	if p := r.besideDeclaring(fd, t); p != "" {
		return p
	}

	t = strings.TrimSpace(t)
	if t == "" || strings.ContainsAny(t, "/\\[]<> $#:.") {
		return ""
	}

	return r.uniqueComponentNamed(t)
}

// uniqueNameCache holds uniqueComponentNamed's answers for one index
// generation.
type uniqueNameCache struct {
	names map[string]string
	gen   uint64
	mu    sync.Mutex
}

// uniqueComponentNamed is the one component in the workspace whose file is
// named name, or "" when there is none or more than one: a name two files
// answer to is not settled by which is nearer.
func (r *Resolver) uniqueComponentNamed(name string) string {
	if r.Index == nil {
		return ""
	}

	r.mu.Lock()
	if r.uniqueNames == nil {
		r.uniqueNames = &uniqueNameCache{}
	}

	c := r.uniqueNames
	r.mu.Unlock()

	gen := r.Index.Generation()
	key := strings.ToLower(name)

	c.mu.Lock()
	if c.gen != gen || c.names == nil {
		c.names, c.gen = map[string]string{}, gen
	}

	answer, ok := c.names[key]
	c.mu.Unlock()

	if ok {
		return answer
	}

	files := slices.DeleteFunc(r.Index.FindFilesByBasename(name), frameworkapi.IsStub)
	if len(files) == 1 {
		answer = files[0]
	}

	c.mu.Lock()
	if c.gen == gen {
		c.names[key] = answer
	}
	c.mu.Unlock()

	return answer
}

// besideDeclaring is the file a bare component name t names beside the file
// that declares fd, or "".
func (r *Resolver) besideDeclaring(fd *parser.FunctionDef, t string) string {
	t = strings.TrimSpace(t)
	if t == "" || r.FS == nil || strings.ContainsAny(t, "/\\[]<> $#:") || filepath.IsAbs(t) {
		return ""
	}

	file := cfpath.FromURI(string(fd.URI))
	if file == "" {
		return ""
	}

	return cfpath.ResolvePathCached(t, filepath.Dir(file), nil, r.dirs())
}

// aboveDeclaring is the file a bare component name t names in the nearest
// directory above the file that declares fd, within the workspace folder
// holding it, or "". CFML looks a bare name up in the component's own
// directory (besideDeclaring) and then from the application's root, and a
// library's helpers name its main component that way: spreadsheet-cfml's
// helpers/base.cfc declares `Spreadsheet function library()` with
// Spreadsheet.cfc one level up, and the 188 calls chained on library() were
// "no component return type". It is the last answer returnComponentOf tries.
func (r *Resolver) aboveDeclaring(fd *parser.FunctionDef, t string) string {
	t = strings.TrimSpace(t)
	if t == "" || r.FS == nil || cfmlTypes[strings.ToLower(t)] || strings.ContainsAny(t, "/\\[]<> $#:.") || filepath.IsAbs(t) {
		return ""
	}

	file := cfpath.FromURI(string(fd.URI))
	if file == "" {
		return ""
	}

	dir := filepath.Dir(file)

	folder := r.workspaceFolderOf(dir)
	if folder == "" || dir == folder {
		return ""
	}

	for d := filepath.Dir(dir); ; d = filepath.Dir(d) {
		if p := cfpath.ResolvePathCached(t, d, nil, r.dirs()); p != "" {
			return p
		}

		if d == folder || filepath.Dir(d) == d {
			return ""
		}
	}
}

// workspaceFolderOf is the longest workspace folder dir lies in, or "".
func (r *Resolver) workspaceFolderOf(dir string) string {
	best := ""

	for _, folder := range r.WorkspaceFolders {
		folder = filepath.Clean(folder)
		if (dir == folder || strings.HasPrefix(dir, folder+string(filepath.Separator))) && len(folder) > len(best) {
			best = folder
		}
	}

	return best
}

// bareArgumentComponent is the component a bare argument type names, as a
// bare return type is read (namedType). A CFML type name, and a word naming
// no component, is not one.
func (r *Resolver) bareArgumentComponent(t string, pr *parser.ParseResult) string {
	if t == "" || cfmlTypes[strings.ToLower(t)] {
		return ""
	}

	return r.namedType(&parser.FunctionDef{URI: pr.URI}, t)
}
