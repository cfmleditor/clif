package resolve

import (
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/conv"
	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// An untyped argument holds what its callers pass. ContentBox's
// `createSite( required setup )` is called as `createSite( arguments.setup )`
// from a method whose own argument is a `SetupData`; `populateModule( module,
// config )` from code that holds a `Module`. Nothing in the function says so,
// and every `arguments.setup.x()` in it was "has no component ref".
//
// argumentFromCallers reads the answer from the workspace: every call of the
// function that is the function's own call (a bare call in its file or a
// subclass, or a call on a receiver that resolves to it), the argument's
// expression read from the tokens (so a call split over lines reads), and
// that expression typed as a receiver is — `new X()`, a variable or a dotted
// name, a single call. The argument has a type only when **every** caller that
// passes it gets one, and they are the alternatives; a caller whose receiver
// cannot be placed, or whose expression cannot be typed, leaves it untyped
// rather than guessed from the others. A caller that omits the argument says
// nothing about it. Callers the workspace does not hold (a test elsewhere,
// a dynamic invoke) are unseen: this is an inference from the code present.
//
// It runs only where the resolver was given the files to search
// (Resolver.InferArgsFiles, set by a batch scan once its index is complete);
// the editor's per-keystroke paths never pay for the caller index.

// callerIndex is, for each name followed by an open parenthesis anywhere in
// the files, the files it appears in.
type callerIndex struct {
	byName map[string][]string
}

// IndexCallerFile adds the call-shaped names in content to the batch caller
// index while the unresolved scan already has the file bytes in hand. It
// avoids reading every workspace file again when the first untyped argument
// asks for its callers.
func (r *Resolver) IndexCallerFile(path, content string) {
	if len(r.InferArgsFiles) == 0 {
		return
	}

	seen := callerNames(content)

	r.mu.Lock()
	if r.callerIdx == nil {
		r.callerIdx = &callerIndex{byName: map[string][]string{}}
	}

	for name := range seen {
		r.callerIdx.byName[name] = append(r.callerIdx.byName[name], path)
	}
	r.mu.Unlock()
}

func callerNames(content string) map[string]bool {
	seen := map[string]bool{}

	for open := strings.IndexByte(content, '('); open >= 0; {
		end := open
		for end > 0 && callerSpace(content[end-1]) {
			end--
		}

		start := end
		for start > 0 && callerWord(content[start-1]) {
			start--
		}

		if start < end {
			addCallerName(seen, content[start:end])
		}

		next := open + 1

		rel := strings.IndexByte(content[next:], '(')
		if rel < 0 {
			break
		}

		open = next + rel
	}

	// A name ending a quoted string is recorded too, as quotedCallerKey:
	// createObject( "component", "a.Templates" ) and getInstance( "Templates" )
	// make an object without a `Templates(` for the scan above to see.
	for i := range len(content) {
		if c := content[i]; c != '"' && c != '\'' {
			continue
		}

		start := i
		for start > 0 && callerWord(content[start-1]) {
			start--
		}

		if start < i {
			addCallerName(seen, quotedCallerKey(content[start:i]))
		}
	}

	return seen
}

// quotedCallerKey is the caller-index key for name ending a quoted string.
func quotedCallerKey(name string) string { return "\"" + name }

func addCallerName(seen map[string]bool, name string) {
	const stackName = 128

	var folded [stackName]byte

	if len(name) > len(folded) {
		normalized := strings.ToLower(name)
		if !seen[normalized] {
			if normalized == name {
				normalized = strings.Clone(normalized)
			}

			seen[normalized] = true
		}

		return
	}

	changed := false

	for i := range len(name) {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
			changed = true
		}

		folded[i] = c
	}

	if !changed {
		if !seen[name] {
			seen[strings.Clone(name)] = true
		}

		return
	}

	if seen[string(folded[:len(name)])] {
		return
	}

	seen[string(folded[:len(name)])] = true
}

func callerSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func callerWord(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '$'
}

// owner is the resolver that holds the shared caches: a per-caller view
// delegates to the one it was made from.
func (r *Resolver) owner() *Resolver {
	if r.indexer != nil {
		return r.indexer
	}

	return r
}

func (r *Resolver) callerFiles(name string) []string {
	o := r.owner()

	o.mu.RLock()
	idx := o.callerIdx
	o.mu.RUnlock()

	if idx == nil {
		idx = o.buildCallerIndexOnce()
	}

	key := strings.ToLower(name)

	o.mu.Lock()
	files := idx.byName[key]
	slices.Sort(files)
	files = slices.Compact(files)
	idx.byName[key] = files
	files = slices.Clone(files)
	o.mu.Unlock()

	return files
}

// buildCallerIndexOnce builds the caller index unless another worker has, or
// is: without the lock every worker that asked before the first build finished
// read and scanned every file itself, and the code map's scan of tassweb spent
// 95s of CPU (58% of the build) building the same index over and over.
func (r *Resolver) buildCallerIndexOnce() *callerIndex {
	r.callerBuild.Lock()
	defer r.callerBuild.Unlock()

	r.mu.RLock()
	idx := r.callerIdx
	r.mu.RUnlock()

	if idx != nil {
		return idx
	}

	return r.buildCallerIndex()
}

// ShareCallerIndex builds one caller index over the files the resolvers search
// (their InferArgsFiles, which must be the same list), in parallel, and gives
// it to every one of them. A workspace of several configs has a resolver per
// config, and each built the same index from the same files: callerNames
// depends on a file's text alone. The returned function gives each resolver
// back the index it had, for a resolver that outlives the run (the editor's).
func ShareCallerIndex(rs []*Resolver) (restore func()) {
	if len(rs) == 0 || len(rs[0].InferArgsFiles) == 0 {
		return func() {}
	}

	files := rs[0].InferArgsFiles
	names := make([]map[string]bool, len(files))

	var wg sync.WaitGroup

	work := make(chan int)

	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for i := range work {
				if text, err := rs[0].readText(files[i]); err == nil {
					names[i] = callerNames(text)
				}
			}
		})
	}

	for i := range files {
		work <- i
	}

	close(work)
	wg.Wait()

	idx := &callerIndex{byName: map[string][]string{}}

	for i, seen := range names {
		for name := range seen {
			idx.byName[name] = append(idx.byName[name], files[i])
		}
	}

	prev := make([]*callerIndex, len(rs))

	for i, r := range rs {
		o := r.owner()

		o.mu.Lock()
		prev[i], o.callerIdx = o.callerIdx, idx
		o.mu.Unlock()
	}

	return func() {
		for i, r := range rs {
			o := r.owner()

			o.mu.Lock()
			if o.callerIdx == idx {
				// argCache holds answers read from idx, which this resolver
				// would not have given with its own.
				o.callerIdx, o.argCache = prev[i], nil
			}
			o.mu.Unlock()
		}
	}
}

func (r *Resolver) buildCallerIndex() *callerIndex {
	idx := &callerIndex{byName: map[string][]string{}}

	for _, path := range r.InferArgsFiles {
		text, err := r.readText(path)
		if err != nil {
			continue
		}

		for name := range callerNames(text) {
			idx.byName[name] = append(idx.byName[name], path)
		}
	}

	r.mu.Lock()
	if r.callerIdx == nil {
		r.callerIdx = idx
	}

	idx = r.callerIdx
	r.mu.Unlock()

	return idx
}

// argumentFromCallers is the component(s) the untyped argument variable names
// holds, from its callers, or "".
func (r *Resolver) argumentFromCallers(variable, caller string, pr *parser.ParseResult, ctx lookupCtx) string {
	o := r.owner()
	if len(o.InferArgsFiles) == 0 || pr == nil || caller == "" || ctx.depth >= maxAssignedDepth {
		return ""
	}

	name := variable
	if rest, ok := strings.CutPrefix(strings.ToLower(variable), "arguments."); ok {
		name = variable[len(variable)-len(rest):]
	}

	if name == "" || strings.ContainsAny(name, ".[(") {
		return ""
	}

	fi := slices.IndexFunc(pr.Funcs, func(f parser.FunctionDef) bool { return strings.EqualFold(f.Name, caller) })
	if fi < 0 {
		return ""
	}

	fd := &pr.Funcs[fi]

	pos := slices.IndexFunc(fd.Arguments, func(a parser.Argument) bool { return strings.EqualFold(a.Name, name) })
	if pos < 0 {
		return ""
	}

	switch strings.ToLower(fd.Arguments[pos].Type) {
	case "", "any", "component", "object":
	default:
		return ""
	}

	key := pr.URI.Path() + "\x00" + strconv.Itoa(int(fd.Line)) + "\x00" + strings.ToLower(name) + "\x00" + strconv.Itoa(ctx.depth)

	o.mu.RLock()
	cached, ok := o.argCache[key]
	o.mu.RUnlock()

	if ok {
		return cached
	}

	answer := r.inferArgument(fd, pos, pr.URI.Path(), ctx)

	o.mu.Lock()
	if o.argCache == nil || len(o.argCache) >= 8192 {
		o.argCache = map[string]string{}
	}

	o.argCache[key] = answer
	o.mu.Unlock()

	return answer
}

func (r *Resolver) inferArgument(fd *parser.FunctionDef, pos int, file string, ctx lookupCtx) string {
	if strings.EqualFold(fd.Name, "init") && strings.EqualFold(filepath.Ext(file), ".cfc") {
		return r.inferInitArgument(fd, pos, file, ctx)
	}

	files := r.callerFiles(fd.Name)
	files = append(files, r.callerFiles(quotedCallerKey(fd.Name))...)
	slices.Sort(files)

	files = slices.Compact(files)
	if len(files) == 0 || exceedsCallerLimit(len(files), r.CallerInferenceLimits.MaxCandidateFiles) {
		return ""
	}

	argName := fd.Arguments[pos].Name
	budget := callerFileBudget{limit: r.CallerInferenceLimits.MaxCallerFiles}

	var (
		comps []string
		sites int
	)

	// Files are read in the order their parses finish: the answer does not
	// depend on the order (every caller must agree, and the alternatives are
	// sorted), so the loop never waits on one slow parse while others are
	// ready, and stopping early stops the parsing. With limits configured it
	// does (a forwarded argument counts as a site until it is found to be
	// one), so then they are read in file order.
	next, stop := r.parseCallers(files)
	defer stop()

	for got, more := next(); more; got, more = next() {
		path, hpr := got.path, got.pr
		if hpr == nil {
			return ""
		}

		var tokens []parser.Token

		// Only a call to the function, or to invoke(), can pass it the
		// argument; and the directory is the file's, not the call's.
		calls := hpr.CallsNamed(fd.Name, "invoke")
		dir := filepath.Dir(path)

		for ci := range calls {
			call := &calls[ci]

			invoke := r.builtinInvoke(call, hpr, dir)
			if !invoke && !strings.EqualFold(call.FuncName, fd.Name) {
				continue
			}

			expr, passed, ours, known := r.callerArgument(call, hpr, path, fd, file, dir, pos, argName, ctx)

			if !ours {
				if known || !invoke && r.callerOmitsArgument(call, hpr, pos, argName) {
					continue
				}

				return ""
			}

			if !budget.take(path) {
				return ""
			}

			if tokens == nil {
				tokens = r.fileTokens(hpr)
			}

			if !invoke {
				expr, passed = callSiteArgument(tokens, call, pos, argName)
			}

			if !passed {
				continue
			}

			sites++
			if exceedsCallerLimit(sites, r.CallerInferenceLimits.MaxCallSites) {
				return ""
			}

			comp := r.argumentExprComponent(expr, call.Line, call.Caller, hpr, dir, ctx)
			if comp == "" && forwardedArg(expr) {
				sites--

				continue
			}

			if comp == "" {
				return ""
			}

			comps = r.appendArgumentComponents(comps, comp, dir)
		}
	}

	if sites == 0 {
		return ""
	}

	slices.Sort(comps)

	return strings.Join(comps, "|")
}

// appendArgumentComponents keeps the union of the proven caller alternatives.
func (r *Resolver) appendArgumentComponents(comps []string, comp, dir string) []string {
	for alt := range strings.SplitSeq(r.pathsOf(comp, dir), "|") {
		if !containsFold(comps, alt) {
			comps = append(comps, alt)
		}
	}

	return comps
}

func (r *Resolver) callerArgument(call *parser.CallSite, pr *parser.ParseResult, path string, fd *parser.FunctionDef, file, dir string, pos int, name string, ctx lookupCtx) (expr []parser.Token, passed, ours, known bool) {
	if r.builtinInvoke(call, pr, dir) {
		return r.invokeArgument(call, pr, fd, pos, name, dir, ctx)
	}

	ours, known = r.callIsTo(call, pr, path, fd, file, dir, ctx)

	return nil, false, ours, known
}

// Byte identity prevents two calls on the same line from sharing the first
// call's arguments. A spanless ambiguous occurrence cannot supply evidence.
func callSiteArgument(tokens []parser.Token, call *parser.CallSite, pos int, name string) ([]parser.Token, bool) {
	selected := -1

	// The tokens are the file's in order, so the call's own are found by
	// search: walking every token of the file for each call was quadratic
	// in a file making thousands of calls. A call with a span is the token
	// at its offset; one without is among its line's.
	from := sort.Search(len(tokens), func(i int) bool { return tokens[i].Line >= int(call.Line) })
	if call.HasSpan {
		from = sort.Search(len(tokens), func(i int) bool { return tokens[i].Offset >= int(call.StartByte) })
	}

	for i := from; i+1 < len(tokens); i++ {
		if call.HasSpan && tokens[i].Offset != int(call.StartByte) || !call.HasSpan && tokens[i].Line != int(call.Line) {
			break
		}

		if tokens[i].Kind != parser.TokIdent || !strings.EqualFold(tokens[i].Value, call.FuncName) || tokens[i+1].Kind != parser.TokLParen {
			continue
		}

		if i > 0 && strings.EqualFold(tokens[i-1].Value, "function") {
			continue
		}

		if call.HasSpan {
			return callArgumentAt(tokens, i, pos, name)
		}

		if selected >= 0 {
			return nil, false
		}

		selected = i
	}

	if selected >= 0 {
		return callArgumentAt(tokens, selected, pos, name)
	}

	return nil, false
}

// callIsTo reports whether call, made in the file at path, is a call of fd:
// a bare or this./variables. call in fd's file or a component extending it, or
// a call on a receiver that resolves to a component whose function of that
// name is fd. known is false when the receiver cannot be placed at all, so
// the call may or may not be fd's.
func (r *Resolver) callIsTo(call *parser.CallSite, hpr *parser.ParseResult, path string, fd *parser.FunctionDef, file, dir string, ctx lookupCtx) (ours, known bool) {
	// The parse may already have placed the receiver, as canResolveCall reads
	// first: a chained call on what a factory returns
	// (`$.getBean( "userManager" ).update( … )`) or on `new X()`, whose
	// variable is untyped or empty.
	if comp := call.Component; comp != "" && !strings.HasPrefix(comp, "$") {
		return r.componentCallIs(comp, call.FuncName, fd, dir), true
	}

	if call.Variable == "" || strings.EqualFold(call.Variable, "this") || strings.EqualFold(call.Variable, "variables") || call.This {
		return cfpath.SamePath(path, file) || r.descendsFrom(path, file), true
	}

	// super.f() is a call of an ancestor's f: fd's, when fd is in one.
	if strings.EqualFold(call.Variable, "super") {
		return r.descendsFrom(path, file), true
	}

	comp, _ := r.receiverComponentD(call.Variable, call.Line, call.Caller, call.FuncName, hpr, dir, nil, lookupCtx{depth: ctx.depth + 1})

	// Then a componentResolver on the variable's name, as canResolveCall
	// tries next: the mura preset's `$` is a MuraScope, and `$.dspObjects()`
	// is the renderer's dspObjects, reached through the scope's
	// onMissingMethod, not a call of the utility function of that name.
	if comp == "" {
		comp, _ = parser.ResolveFromCallFull(call.Variable, r.Resolvers)
	}

	if comp == "" || strings.HasPrefix(comp, "$") {
		// A receiver that cannot hold fd's component is known not to call fd:
		// MessageDigest's md.update() is not a DAO's update().
		return false, r.cannotHold(call, hpr, fd)
	}

	return r.componentCallIs(comp, call.FuncName, fd, dir), true
}

// javaObjectRe is an expression that makes a Java object: createObject with
// the java type, or Lucee's new java:, whatever is chained on it.
var javaObjectRe = regexp.MustCompile(`(?is)^(?:createObject\s*\(\s*["']java["']|new\s+java:)`)

// computedComponentRe is createObject of a component whose path is computed
// but whose last segment, the file's name, is literal.
var computedComponentRe = regexp.MustCompile(`(?is)^createObject\s*\(\s*["']component["']\s*,\s*["'][^"']*#[^"']*\.(\w+)["']\s*\)$`)

// cannotHold reports whether the variable call is made on is, at the call, a
// local assigned something that cannot be fd's component: a Java object, or a
// component whose computed path names a file of another name — Mura's
// createObject("component","plugins.#dir#.plugin.plugin") is a plugin.cfc,
// unless a file of that name in the workspace extends fd's component.
func (r *Resolver) cannotHold(call *parser.CallSite, pr *parser.ParseResult, fd *parser.FunctionDef) bool {
	name := strings.TrimPrefix(strings.ToLower(call.Variable), "local.")
	if name == "" || strings.ContainsAny(name, ".[(") || pr == nil {
		return false
	}

	start := 0
	if scope, ok := enclosingScope(pr, call.Line); ok {
		start = scope.Start
	}

	rhs, ok := r.localAssignmentIn(pr, name, start, int(call.Line))
	if !ok {
		return false
	}

	rhs = strings.TrimSpace(rhs)
	if javaObjectRe.MatchString(rhs) {
		return true
	}

	m := computedComponentRe.FindStringSubmatch(rhs)
	if m == nil || !fd.URI.IsFile() {
		return false
	}

	file := fd.URI.Path()
	if strings.EqualFold(m[1], strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))) || r.Index == nil {
		return false
	}

	for _, p := range r.Index.FindFilesByBasename(m[1]) {
		if r.descendsFrom(p, file) {
			return false
		}
	}

	return true
}

// componentCallIs reports whether funcName called on comp (or any of its
// alternatives) is fd.
func (r *Resolver) componentCallIs(comp, funcName string, fd *parser.FunctionDef, dir string) bool {
	for alt := range strings.SplitSeq(comp, "|") {
		if other := r.ResolveFunc(alt, funcName, dir); other != nil && other.URI.Path() == fd.URI.Path() && other.Line == fd.Line {
			return true
		}
	}

	return false
}

// callArgumentAt is callArgument for the call whose name is tokens[i].
func callArgumentAt(tokens []parser.Token, i, pos int, argName string) (expr []parser.Token, passed bool) {
	end := producerGroupEnd(tokens, i+1, parser.TokLParen, parser.TokRParen)
	if end < 0 {
		return nil, false
	}

	positional := 0
	forwards := false
	unknownCollection := false

	for _, piece := range producerSplit(tokens[i+2 : end]) {
		if len(piece) > 2 && piece[0].Kind == parser.TokIdent && (piece[1].Kind == parser.TokEquals || piece[1].Kind == parser.TokColon) {
			if strings.EqualFold(piece[0].Value, argName) {
				return piece[2:], true
			}

			if strings.EqualFold(piece[0].Value, "argumentCollection") {
				if len(piece) == 3 && strings.EqualFold(piece[2].Value, "arguments") {
					forwards = true
				} else {
					unknownCollection = true
				}
			}

			continue
		}

		if positional == pos {
			return piece, len(piece) > 0
		}

		positional++
	}

	// `f( argumentCollection = arguments )` hands over the caller's own
	// arguments: Mura's contentRenderer sets `arguments.renderer = this` and
	// forwards to its utility's dspObject( renderer ). What the caller's
	// arguments.name holds is the argument.
	if unknownCollection {
		return nil, true
	}

	if forwards {
		line := tokens[i].Line

		return []parser.Token{
			{Kind: parser.TokIdent, Value: "arguments", Line: line, Offset: forwardedOffset},
			{Kind: parser.TokDot, Value: ".", Line: line, Offset: forwardedOffset},
			{Kind: parser.TokIdent, Value: argName, Line: line, Offset: forwardedOffset},
		}, true
	}

	return nil, false
}

// forwardedOffset marks the tokens callArgumentAt makes for a forwarded
// argument; no real token has a negative offset.
const forwardedOffset = -1

// forwardedArg reports whether expr is a forwarded argument. One that cannot
// be typed is skipped, as a call that does not pass the argument always was:
// cfwheels' SpyTenantMigrator forwards its own untyped migrator to super.
func forwardedArg(expr []parser.Token) bool {
	return len(expr) > 0 && expr[0].Offset == forwardedOffset
}

// argumentExprComponent types the expression a caller passes: `new X( … )`, a
// variable or dotted name, or one call. "" for anything else.
func (r *Resolver) argumentExprComponent(expr []parser.Token, line uint32, caller string, pr *parser.ParseResult, dir string, ctx lookupCtx) string {
	comp := r.argumentExprComponentSource(expr, line, caller, pr, dir, ctx)
	// A parser fallback can retain an interpolated constructor path. It is
	// runtime evidence, not a literal component supplied by a caller.
	if strings.Contains(comp, "#") {
		return ""
	}

	return comp
}

func (r *Resolver) argumentExprComponentSource(expr []parser.Token, line uint32, caller string, pr *parser.ParseResult, dir string, ctx lookupCtx) string {
	expr = producerUnwrap(expr)
	if len(expr) == 0 {
		return ""
	}

	next := lookupCtx{depth: ctx.depth + 1}

	// `this` is the calling component, or any component extending it: Mura's
	// contentRenderer hands itself to its utility, and a theme's renderer
	// extends it.
	if len(expr) == 1 && strings.EqualFold(expr[0].Value, "this") {
		if pr.URI.IsFile() && strings.EqualFold(filepath.Ext(pr.URI.Path()), ".cfc") {
			return r.withSubclasses(pr.URI.Path())
		}

		return ""
	}

	if path := producerPath(expr); path != "" {
		name := strings.ReplaceAll(producerText(expr), " ", "")

		comp, _ := r.receiverComponentD(name, line, caller, "", pr, dir, nil, next)
		if comp == "" {
			// A configured resolver types a name too (prc.oCurrentSite).
			comp, _, _ = parser.ResolveFromCallMatch(name, r.Resolvers)
		}

		if comp == "" {
			// An argument the caller passes on is typed by its own callers.
			comp = r.argumentFromCallers(name, caller, pr, next)
		}

		if comp == "" {
			comp = r.argumentAliasComponent(name, line, caller, pr, dir, next)
		}

		if comp == "" || strings.HasPrefix(comp, "$") {
			return ""
		}

		return comp
	}

	text := strings.ReplaceAll(strings.ReplaceAll(producerText(expr), " . ", "."), " (", "(")

	// A construction is the component only when it is the whole expression,
	// or has init() on it: `new Repo().one()` passes what one() returns, and
	// was typed as a Repo.
	if strings.EqualFold(expr[0].Value, "new") {
		dotted := createdOnly(text)
		if dotted == "" {
			return r.constructionChain(expr, dir)
		}

		if p := r.ComponentPath(dotted, dir); p != "" {
			return p
		}

		return dotted
	}

	return r.typeCallExpr(text, line, caller, pr, dir, next, false)
}

// constructionChain is what a chain of calls on a construction returns:
// `new Repo().one()` is what Repo's one() returns, each later call on what the
// one before it returns. "" when the chain is not that shape or a hop returns
// no component.
func (r *Resolver) constructionChain(expr []parser.Token, dir string) string {
	open := slices.IndexFunc(expr, func(t parser.Token) bool { return t.Kind == parser.TokLParen })
	if open < 2 {
		return ""
	}

	closing := matchingParen(expr, open)
	if closing < 0 || closing+1 >= len(expr) || expr[closing+1].Kind != parser.TokDot {
		return ""
	}

	comp := r.ComponentPath(strings.ReplaceAll(producerText(expr[1:open]), " ", ""), dir)
	if comp == "" {
		return ""
	}

	// The chain past the construction, read as calls on a receiver.
	rest := strings.ReplaceAll(strings.ReplaceAll(producerText(expr[closing+1:]), " . ", "."), " (", "(")

	_, calls, ok := callChain("x" + rest)
	if !ok {
		return ""
	}

	lookup := r.FuncLookup(dir)

	for _, c := range calls {
		if comp == "" || strings.HasPrefix(comp, "$") {
			return ""
		}

		comp = hopReturns(lookup, comp, c, false)
	}

	if strings.HasPrefix(comp, "$") {
		return ""
	}

	return comp
}

// argumentAliasComponent follows a plain local passed by a caller to the last
// straight-line assignment before that call. Equal lexical brace paths and
// the absence of unbraced controls are required; exhausted scans withhold
// inference. An assignment after the call is excluded by the bounded lookup.
func (r *Resolver) argumentAliasComponent(name string, line uint32, caller string, pr *parser.ParseResult, dir string, ctx lookupCtx) string {
	if pr == nil || ctx.depth >= maxAssignedDepth {
		return ""
	}

	lower := strings.ToLower(name)

	lower = strings.TrimPrefix(lower, "local.")
	if lower == "" || strings.Contains(lower, ".") || isScopeWord(lower) {
		return ""
	}

	start := 0
	if scope, ok := enclosingScope(pr, line); ok {
		start = scope.Start
	}

	rhs, assignedLine, ok := r.localAssignmentAt(pr, lower, start, int(line))
	if !ok || strings.EqualFold(strings.TrimSpace(rhs), name) {
		return ""
	}

	assignedPath, assignedOK := producerBlockPath(pr.Content, start, assignedLine)
	callPath, callOK := producerBlockPath(pr.Content, start, int(line))

	if !assignedOK || !callOK || !slices.Equal(assignedPath, callPath) {
		return ""
	}

	return r.argumentExprComponent(producerTokens(rhs), conv.Uint32(assignedLine), caller, pr, dir, ctx)
}

// producerBlockPath scans only from the enclosing function's first line to
// the target line, so the token budget is spent on the function rather than
// on whatever precedes it in the file. Offsets are relative to that start,
// which is shared by every path compared against this one.
func producerBlockPath(content string, start, line int) ([]int, bool) {
	tokens := producerTokens(sourceText(content, start, line))
	if tokens == nil {
		return nil, false
	}

	var path []int

	for i, token := range tokens {
		if producerUnbracedControl(tokens, i) {
			return nil, false
		}

		switch token.Kind {
		case parser.TokLBrace:
			path = append(path, token.Offset)
		case parser.TokRBrace:
			if len(path) > 0 {
				path = path[:len(path)-1]
			}
		default:
		}
	}

	return path, true
}

// Brace paths cannot distinguish a conditional single statement from an
// unconditional one. Conservatively reject such controls earlier in the
// enclosing function, rather than claim that their assignments reach the call.
func producerUnbracedControl(tokens []parser.Token, i int) bool {
	if tokens[i].Kind != parser.TokIdent {
		return false
	}

	next := i + 1
	switch strings.ToLower(tokens[i].Value) {
	case "if", "for", "while", "switch", "catch":
		if next >= len(tokens) || tokens[next].Kind != parser.TokLParen {
			return true
		}

		end := producerGroupEnd(tokens, next, parser.TokLParen, parser.TokRParen)
		if end < 0 {
			return true
		}

		next = end + 1
	case "else":
		if next < len(tokens) && strings.EqualFold(tokens[next].Value, "if") {
			return false
		}
	case "do", "try", "finally":
	default:
		return false
	}

	return next >= len(tokens) || tokens[next].Kind != parser.TokLBrace
}

// A nonpositive configured limit preserves the default unlimited search.
func exceedsCallerLimit(count, limit int) bool { return limit > 0 && count > limit }

// Calls are visited file by file; count each resolved owner file once.
type callerFileBudget struct {
	last         string
	files, limit int
}

func (b *callerFileBudget) take(file string) bool {
	if b.last != file {
		b.last = file
		b.files++
	}

	return !exceedsCallerLimit(b.files, b.limit)
}

// An unknown receiver cannot supply a type for an argument a complete call
// demonstrably omits. Require exact byte identity and balanced call syntax;
// opaque argumentCollection still counts as supplying the argument.
func (r *Resolver) callerOmitsArgument(call *parser.CallSite, pr *parser.ParseResult, pos int, name string) bool {
	if !call.HasSpan {
		return false
	}

	tokens := r.fileTokens(pr)
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Offset != int(call.StartByte) || tokens[i].Kind != parser.TokIdent || !strings.EqualFold(tokens[i].Value, call.FuncName) || tokens[i+1].Kind != parser.TokLParen {
			continue
		}

		if producerGroupEnd(tokens, i+1, parser.TokLParen, parser.TokRParen) < 0 {
			return false
		}

		_, passed := callArgumentAt(tokens, i, pos, name)

		return !passed
	}

	return false
}

// fileTokens is pr's tokens, read once and shared by everything that walks a
// whole file's tokens: caller inference and the closure, loop and guard
// spans, which each tokenized the file again (half of a tassweb scan's
// allocation). It shares only lexical work, never inferred types. Caller views
// use the owning resolver's cache, which holds at most maxHeldTokens.
func (r *Resolver) fileTokens(pr *parser.ParseResult) []parser.Token {
	o := r.owner()

	return o.callerTokens.getWithin(&o.mu, pr, func() []parser.Token {
		// allTokens sizes its slice from the file's length; a file of mostly
		// comments or markup has far fewer tokens, and the cache counts the
		// tokens, so the spare capacity is not kept.
		tokens := allTokens(pr.Content)
		if cap(tokens) > len(tokens)+len(tokens)/4 {
			tokens = slices.Clone(tokens)
		}

		return tokens
	}, maxHeldTokens)
}

// maxHeldTokens bounds fileTokens' cache: about 10 MB of tokens, enough for
// the files a scan's workers are reading at once.
const maxHeldTokens = 1 << 18

// parsedFile is one candidate file's parse, as parseCallers hands it over.
type parsedFile struct {
	path string
	pr   *parser.ParseResult
}

// parseCallers hands over each file's parse through next, false when there
// are no more; stop ends the parsing of files not yet started.
//
// Inferring an argument parses every file calling its function, and serially
// that was a chain one goroutine walked while the rest of the scan's workers
// sat idle (wheelstest's two runners took 6s each). A whole-list prefetch
// then parsed every candidate before the first was read, though the loop
// usually stops early: run() is in 675 files of cfwheels' CLI, and its loop
// stops at the fifth. So the files are parsed on prefetchWorkers goroutines
// and handed over as each finishes, one worker per file so none is parsed
// twice for the loop. They are parsed in order, inline, where that costs
// nothing or the order matters: one file, every file already parsed, or
// caller-inference limits configured.
func (r *Resolver) parseCallers(files []string) (next func() (parsedFile, bool), stop func()) {
	limits := r.CallerInferenceLimits
	if len(files) < 2 || limits.MaxCallSites > 0 || limits.MaxCallerFiles > 0 || r.allParsed(files) {
		i := 0

		return func() (parsedFile, bool) {
			if i >= len(files) {
				return parsedFile{}, false
			}

			path := files[i]
			i++

			return parsedFile{path: path, pr: r.recoveredHandlerParse(path)}, true
		}, func() {}
	}

	out := make(chan parsedFile, len(files)) // a worker never blocks on a loop that stopped
	work := make(chan string)
	done := make(chan struct{})

	var wg sync.WaitGroup

	for range min(len(files), prefetchWorkers) {
		wg.Go(func() {
			for path := range work {
				out <- parsedFile{path: path, pr: r.recoveredHandlerParse(path)}
			}
		})
	}

	go func() {
		defer close(work)

		for _, path := range files {
			select {
			case work <- path:
			case <-done:
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(out)
	}()

	var once sync.Once

	return func() (parsedFile, bool) {
		got, ok := <-out

		return got, ok
	}, func() { once.Do(func() { close(done) }) }
}

// allParsed reports whether handlerParse already holds every file.
func (r *Resolver) allParsed(files []string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, path := range files {
		if _, ok := r.handlerCache[pathKey(path)]; !ok {
			return false
		}
	}

	return true
}

// recoveredHandlerParse is handlerParse, with a panic in it recorded and read
// as no parse, which the inference loop takes as "cannot infer". The parse may
// run on a worker goroutine, where nothing the scan recovers on its own would
// catch it and one bad caller file would end the process.
func (r *Resolver) recoveredHandlerParse(path string) (pr *parser.ParseResult) {
	defer func() {
		if rec := recover(); rec != nil {
			cflog.Recovered(nil, "caller parse panic", rec, cflog.String("file", path))

			pr = nil
		}
	}()

	return r.handlerParse(path)
}

// prefetchWorkers bounds parseCallers per inference: nested inferences parse
// too, and the scan is already running a worker per core.
const prefetchWorkers = 4
