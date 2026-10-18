package resolve

import (
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// A loop variable holds an element of the collection it iterates:
// `for ( var c in getComments() )` and `<cfloop array="#comments#" index="c">`
// give c an entity whenever the collection's element type is in the source. The
// parser files a loop variable as a declaration with no component, so a call on
// it read as one on an untyped variable.
//
// The collection's element type comes from what the source states:
//
//   - a persistent entity's one-to-many or many-to-many property, through its
//     generated getter (FunctionDef.ElementComponent), whether the loop calls the
//     getter or reads the property;
//   - a cborm service bound to an entity, whose getAll() returns an array of it,
//     unless it is asked for `properties`, when it returns structs;
//   - a local variable assigned from either, by its nearest preceding assignment
//     in the function, the rule a receiver's own ref follows.
//
// Anything else gives no element type and the variable stays untyped.

var (
	loopTagRe      = regexp.MustCompile(`(?is)<(/?)cfloop\b([^>]*)>`)
	loopAttrRe     = regexp.MustCompile(`(?i)\b(array|collection|struct|index|item)\s*=\s*["']([^"']*)["']`)
	loopPathRe     = regexp.MustCompile(`^(?:(local|variables|this|arguments)\.)?(\w+)$`)
	indexedReadRe  = regexp.MustCompile(`^((?:local|variables|this|arguments)\.[\w.]+|[A-Za-z_]\w*)\s*\[[^\]]*\]$`)
	maxElementHops = 4
)

// loopElement is the component call's receiver holds as the variable of a loop
// around the call, or "".
func (r *Resolver) loopElement(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	name := strings.TrimPrefix(strings.ToLower(call.Variable), "local.")
	if name == "" || strings.ContainsAny(name, ".[") || pr == nil {
		return ""
	}

	start, end, caller := 0, -1, ""
	if scope, ok := enclosingScope(pr, call.Line); ok {
		start, end, caller = scope.Start, scope.End, scope.Name
	}

	collection, header, ok := enclosingLoop(r.loopsOf(pr)[name], name, int(call.Line), start, end)
	if !ok {
		return ""
	}

	comp := r.elementOf(collection, header, start, caller, pr, baseDir, 0, "")
	if comp != "" {
		tr.addf("%q is the variable of a loop over %q, whose elements are %q", call.Variable, collection, comp)
	}

	return comp
}

// enclosingScope is the innermost function scope holding line.
func enclosingScope(pr *parser.ParseResult, line uint32) (parser.FuncScope, bool) {
	var (
		best  parser.FuncScope
		found bool
	)

	for _, scope := range pr.Scopes {
		if int(line) < scope.Start || int(line) > scope.End {
			continue
		}

		if !found || scope.End-scope.Start < best.End-best.Start {
			best, found = scope, true
		}
	}

	return best, found
}

type loopSpan struct {
	name, collection   string // the loop variable, lowercased without local., and what it iterates
	header, start, end int
}

// enclosingLoop finds, among loops, the innermost one between lines start
// and end (end < 0 for the rest of the file) that binds name and holds line in
// its body, and returns the collection it iterates and the line of its header.
func enclosingLoop(loops []loopSpan, name string, line, start, end int) (collection string, header int, ok bool) {
	var best *loopSpan

	for i := range loops {
		s := &loops[i]
		if s.name != name || s.start > line || s.end < line || s.header < start || end >= 0 && s.header > end {
			continue
		}

		if best == nil || s.end-s.start < best.end-best.start {
			best = s
		}
	}

	if best == nil {
		return "", 0, false
	}

	return best.collection, best.header, true
}

// loopsByName is a file's loops by the variable they bind.
type loopsByName map[string][]loopSpan

// loopsOf is every loop in pr's file by the variable it binds, found once per
// version of its content: the lookup runs for every untyped receiver, and
// finding them anew each time tokenised the whole file per call. Grouped,
// because walking every loop in the file per receiver was quadratic in a
// function of thousands of loops.
func (r *Resolver) loopsOf(pr *parser.ParseResult) loopsByName {
	o := r.owner()

	return o.loopCache.get(&o.mu, pr, func() []loopsByName {
		byName := loopsByName{}
		for _, s := range append(scriptLoops(o.fileTokens(pr)), tagLoops(pr.Content)...) {
			byName[s.name] = append(byName[s.name], s)
		}

		return []loopsByName{byName}
	})[0]
}

// fileSpans is what a per-file cache holds: the spans found in a file and the
// text they were found in.
//
// The caches were keyed by file and a hash of its text, so every edit of an
// open document added an entry and none replaced one, and each entry's spans
// were slices of the text it was found in: up to 4,096 versions of a document
// kept alive, ~2.8MB each for a 65,000-line component. Keyed by file, a new
// version replaces the last, and the text kept is the one the file's parse
// holds anyway. Comparing it costs nothing while it is the same string, where
// the hash read the whole file on every lookup.
type fileSpans[T any] struct {
	content string
	spans   []T
}

// spanCache is a per-file cache of fileSpans, guarded by the resolver's mu.
type spanCache[T any] struct {
	files map[string]fileSpans[T]
	held  int // spans held across files, for getWithin
}

// get answers find for pr from the cache while pr's text is what it was last
// found in. mu guards c.
func (c *spanCache[T]) get(mu *sync.RWMutex, pr *parser.ParseResult, find func() []T) []T {
	return c.getWithin(mu, pr, find, 0)
}

// getWithin is get holding at most maxHeld spans across all files, when
// maxHeld is positive, as well as at most 4,096 files: the token cache holds
// whole files' tokens. A file whose spans alone exceed maxHeld is answered
// but not kept: kept, it would empty the cache for itself and be emptied out
// by the next file, re-reading every file each time.
func (c *spanCache[T]) getWithin(mu *sync.RWMutex, pr *parser.ParseResult, find func() []T, maxHeld int) []T {
	key := string(pr.URI)

	mu.RLock()

	e, ok := c.files[key]

	mu.RUnlock()

	if ok && e.content == pr.Content {
		return e.spans
	}

	spans := find()

	if maxHeld > 0 && len(spans) > maxHeld {
		return spans
	}

	mu.Lock()
	if c.files == nil || len(c.files) >= 4096 || maxHeld > 0 && c.held+len(spans) > maxHeld {
		c.files = make(map[string]fileSpans[T])
		c.held = 0
	}

	c.held += len(spans) - len(c.files[key].spans)
	c.files[key] = fileSpans[T]{content: pr.Content, spans: spans}
	mu.Unlock()

	return spans
}

// scriptLoops are the `for ( [var] name in collection ) { … }` loops in content.
func scriptLoops(tokens []parser.Token) []loopSpan {
	var out []loopSpan

	for i := 0; i+4 < len(tokens); i++ {
		if !strings.EqualFold(tokens[i].Value, "for") || tokens[i+1].Kind != parser.TokLParen {
			continue
		}

		j := i + 2
		if strings.EqualFold(tokens[j].Value, "var") {
			j++
		}

		if j+1 >= len(tokens) || tokens[j].Kind != parser.TokIdent || !strings.EqualFold(tokens[j+1].Value, "in") {
			continue
		}

		closeParen := producerGroupEnd(tokens, i+1, parser.TokLParen, parser.TokRParen)
		if closeParen < 0 || closeParen+1 >= len(tokens) || tokens[closeParen+1].Kind != parser.TokLBrace {
			continue
		}

		closeBrace := producerGroupEnd(tokens, closeParen+1, parser.TokLBrace, parser.TokRBrace)
		if closeBrace < 0 {
			continue
		}

		out = append(out, loopSpan{
			name:       strings.TrimPrefix(strings.ToLower(tokens[j].Value), "local."),
			collection: producerText(tokens[j+2 : closeParen]),
			header:     tokens[i].Line,
			start:      tokens[closeParen+1].Line,
			end:        tokens[closeBrace].Line,
		})
	}

	return out
}

// tagLoops are the `<cfloop array="#collection#" index|item="name">` loops in
// content, each to its matching </cfloop>.
func tagLoops(content string) []loopSpan {
	type open struct {
		span  loopSpan
		match bool
	}

	var (
		out   []loopSpan
		stack []open
	)

	lines := newLineIndex(content)

	for _, m := range loopTagRe.FindAllStringSubmatchIndex(content, -1) {
		line := lines.lineOf(m[0])

		if m[3] > m[2] { // </cfloop>
			if len(stack) == 0 {
				continue
			}

			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			if top.match {
				top.span.end = line
				out = append(out, top.span)
			}

			continue
		}

		attrs := map[string]string{}
		for _, a := range loopAttrRe.FindAllStringSubmatch(content[m[4]:m[5]], -1) {
			attrs[strings.ToLower(a[1])] = a[2]
		}

		variable := attrs["index"]
		if collection := attrs["collection"]; collection != "" {
			// A collection loop's item is the value it holds, and its index
			// the key — a string, which types no component. Without an item
			// the loop's variable stays unkeyed.
			variable = attrs["item"]

			stack = append(stack, open{span: loopSpan{
				name:       strings.TrimPrefix(strings.ToLower(variable), "local."),
				collection: strings.Trim(collection, "#"), header: line, start: line,
			}, match: strings.ContainsAny(collection, "#") || attrs["item"] != ""})

			continue
		}

		if variable == "" {
			variable = attrs["item"]
		}

		if attrs["item"] != "" {
			// A loop that names an item holds the element there: an array
			// loop's index, when an item is named too, is the element's
			// position, not the element.
			variable = attrs["item"]
		}

		array := strings.TrimSpace(attrs["array"])
		match := array != "" && variable != "" &&
			strings.HasPrefix(array, "#") && strings.HasSuffix(array, "#") && strings.Count(array, "#") == 2

		stack = append(stack, open{span: loopSpan{
			name:       strings.TrimPrefix(strings.ToLower(variable), "local."),
			collection: strings.Trim(array, "#"), header: line, start: line,
		}, match: match})
	}

	return out
}

func allTokens(content string) []parser.Token {
	sc := parser.NewScanner(content)

	// About one token per six bytes of CFML: sized up front, a file's tokens
	// are not copied through every doubling on the way.
	tokens := make([]parser.Token, 0, len(content)/6)

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			return tokens
		}

		tokens = append(tokens, tok)
	}
}

// elementOf is the component each element of the collection expression holds,
// as written on line header of the function between start and header, or "".
func (r *Resolver) elementOf(expression string, header, start int, caller string, pr *parser.ParseResult, baseDir string, depth int, leaf string) string {
	expression = strings.TrimSpace(producerText(producerTokens(expression)))
	expression = strings.ReplaceAll(strings.ReplaceAll(expression, " . ", "."), " (", "(")

	if depth > maxElementHops || expression == "" {
		return ""
	}

	// A registry's own element read: a struct a ComponentListPackageAsStruct
	// recipe fills, whose values are the package's family components.
	if name, ok := registryExprName(expression); ok {
		if fact, found := r.registryFamily(name, pr); found {
			return familyComp(fact)
		}
	}

	self := pr.URI.Path()

	if receiver, calls, ok := callChain(expression); ok {
		return r.collectionExpressionElement(receiver, calls, header, start, caller, pr, baseDir, depth, leaf)
	}

	// A view's prc.X is what the handler action rendering the view assigns,
	// and its args.X what the renders of it pass.
	if name, ok := prcMember(expression); ok {
		return r.viewPrcElement(name, pr, depth)
	}

	if scope, name, ok := strings.Cut(expression, "."); ok && strings.EqualFold(scope, "args") {
		return r.viewArgsElement(name, pr, depth)
	}

	// A field of the struct a local's call returned: results.comments.
	if f := structFieldRe.FindStringSubmatch(expression); f != nil && !isScopeWord(f[1]) {
		if rhs, ok := r.localAssignmentIn(pr, f[1], start, header); ok {
			return r.fieldElement(rhs, f[2], header, caller, pr, baseDir, leaf)
		}

		return ""
	}

	// The same through a key the subclass spells: results[ variables.entityPlural ],
	// where each subclass sets entityPlural to the name its service's struct uses.
	if f := keyedFieldRe.FindStringSubmatch(expression); f != nil && !isScopeWord(f[1]) && leaf != "" {
		key := r.leafLiteral(leaf, pr.URI.Path(), f[2])
		if key == "" {
			return ""
		}

		if rhs, ok := r.localAssignmentIn(pr, f[1], start, header); ok {
			return r.fieldElement(rhs, key, header, caller, pr, baseDir, leaf)
		}

		return ""
	}

	m := loopPathRe.FindStringSubmatch(expression)
	if m == nil {
		// An element read under a key — fields[idx], a struct-key element
		// of a local — is the element of the collection the read indexes:
		// the read's own brackets name no chain and no member. Only what
		// the collection proves answers; a dynamic guess is no answer.
		if k := indexedReadRe.FindStringSubmatch(expression); k != nil && !isScopeWord(k[1]) {
			comp := r.elementOf(k[1], header, start, caller, pr, baseDir, depth+1, leaf)
			if comp != "" && (!strings.HasPrefix(comp, "$") || strings.HasPrefix(comp, familyPrefix)) {
				return comp
			}

			return ""
		}

		return ""
	}

	scope, name := strings.ToLower(m[1]), m[2]

	if scope == "" || scope == "local" {
		if rhs, ok := r.localAssignmentIn(pr, name, start, header); ok {
			// A registry read under a name no recipe spells: the container's
			// name a list-loop item's concatenation makes, whose turns each
			// name one of the template's own containers.
			if comp := r.indirectRegistryPickup(rhs, header, pr); comp != "" {
				return comp
			}

			// A member's own getter that copies its own fields —
			// `var fields=duplicate(variables.fields)` in an overridden
			// getCustomFields — walks the member's own literal of it, and
			// the loop over the copy reads as the getter's element proves.
			if comp := r.memberCopyElement(pr, rhs, name, header, self); comp != "" {
				return comp
			}

			return r.elementOf(rhs, header, start, caller, pr, baseDir, depth+1, leaf)
		}

		if scope == "local" {
			return ""
		}
	}

	if scope == "arguments" {
		return ""
	}

	// The component's own property, read through the variable it fills.
	return r.methodElement(self, "get"+name, "", baseDir)
}

// memberCopyElement is the element the member's own getter proves for a
// variable it duplicates out of the member's own fields: `var
// fields=duplicate(variables.fields)` in an overridden getCustomFields,
// whose loop the page reads through. Only a copy of the same name reads.
func (r *Resolver) memberCopyElement(pr *parser.ParseResult, rhs, name string, header int, self string) string {
	src, dup := duplicatePlain(rhs)
	if !dup || !strings.EqualFold(strings.TrimPrefix(strings.ToLower(src), "variables."), name) {
		return ""
	}

	fn, ok := enclosingScope(pr, conv.Uint32(header))
	if !ok {
		return ""
	}

	set, _ := r.memberArrayElements(self, fn.Name)

	return set
}

// methodElement is the element type of what method returns on comp.
func (r *Resolver) methodElement(comp, method, arguments, baseDir string) string {
	// A registry family's getter: the element its files' array literals
	// prove, or untyped. A multi-family receiver's element is the union of
	// its families' elements — the receiver holds exactly one family — and
	// it is an answer only when every family proves one.
	if refs := familyRefs(comp); len(refs) > 0 {
		var elements []string

		for _, ref := range refs {
			set := r.familyLoopElement(ref.members, method)
			if set == "" {
				return ""
			}

			for alt := range strings.SplitSeq(set, "|") {
				if !slices.Contains(elements, alt) {
					elements = append(elements, alt)
				}
			}
		}

		slices.Sort(elements)

		return strings.Join(elements, "|")
	}

	if comp == "" || strings.HasPrefix(comp, "$") {
		return ""
	}

	fd := r.ResolveFunc(comp, method, baseDir)
	if fd == nil {
		return ""
	}

	if fd.ElementComponent != "" {
		return r.withSubclasses(r.ComponentPath(fd.ElementComponent, filepath.Dir(fd.URI.Path())))
	}

	if fd.CollectionValues != nil {
		return "$builtin.native.string"
	}

	// cborm: a service bound to an entity returns an array of it from getAll(),
	// and structs when asked for some of its properties; and from a finder of
	// its own built on a criteria list() (returnsCriteriaList).
	if strings.EqualFold(method, "getAll") && !strings.Contains(strings.ToLower(arguments), "properties") || r.returnsCriteriaList(fd) {
		if entity := r.boundEntity(r.ComponentPath(comp, baseDir)); entity != "" {
			return r.withSubclasses(r.ComponentPath(entity, baseDir))
		}
	}

	// The collection the function's own body builds: an array the body
	// starts empty, appends to, and returns — the appended values' common
	// component is the element.
	if elem := r.builtCollectionElement(fd); elem != "" {
		return elem
	}

	// The collection the function's field holds: the function returns the
	// field, and the file's own assignments fill it — a local assigned an
	// empty array, appended with values, assigned to the field.
	if elem := r.builtFieldElement(fd); elem != "" {
		return elem
	}

	return ""
}

// withSubclasses is the entity at path and every component in the workspace
// that extends it, directly or not, as alternatives: an ORM collection of an
// entity holds its subclasses too. ContentBox's subscriber holds
// BaseSubscription entities and calls getRelatedContent() on the comment ones,
// a CommentSubscription method, under `case "Comment"`. A method any of them
// declares is found.
func (r *Resolver) withSubclasses(path string) string {
	if path == "" || r.Index == nil {
		return path
	}

	all := []string{path}

	for i := 0; i < len(all) && len(all) < 16; i++ {
		base := all[i]
		name := strings.TrimSuffix(filepath.Base(base), filepath.Ext(base))

		for _, u := range r.Index.FilesExtendingName(name) {
			sub := cfpath.FromURI(u)

			if slices.ContainsFunc(all, func(p string) bool { return cfpath.SamePath(p, sub) }) {
				continue
			}

			if r.descendsFrom(sub, base) {
				all = append(all, sub)
			}
		}
	}

	// The entity itself last: a missing method's reason names the last
	// alternative, and the collection is declared as the entity.
	return strings.Join(append(all[1:], all[0]), "|")
}

// localAssignment is the right-hand side of the last assignment to name on a
// line from start to before header, when its parentheses balance. A statement
// that runs on over the next lines (a call with its arguments one to a line,
// ContentBox's `var results = svc.search(` …) is joined, up to the line
// before header and at most maxAssignmentLines of them.
func localAssignment(content, name string, start, header int) (string, bool) {
	rhs, _, ok := localAssignmentAt(content, name, start, header)

	return rhs, ok
}

// localAssignmentAt is localAssignment with the source line retained for
// consumers that must prove the assignment reaches a particular use.
func localAssignmentAt(content, name string, start, header int) (string, int, bool) {
	start = max(start, 0)

	// The lines before header, walked backwards in place: copying them out,
	// often from the top of the file, was a tenth of what a tassweb scan
	// allocated.
	region, end := content, strings.Count(content, "\n")+1
	if at, ok := lineOffset(content, header); ok {
		region, end = content[:max(at-1, 0)], header
	}

	stop := len(region)

	for i := end - 1; i >= start; i-- {
		from := strings.LastIndexByte(region[:stop], '\n') + 1
		line := region[from:stop]
		next := stop + 1 // where line i+1 starts
		stop = max(from-1, 0)

		assignedName, value, matched := matchSourceAssignment(line)
		if !matched || !strings.EqualFold(assignedName, name) {
			continue
		}

		rhs := trimAssignmentRHS(value)

		for j := i + 1; strings.Count(rhs, "(") > strings.Count(rhs, ")") && j < end && j <= i+maxAssignmentLines; j++ {
			following := region[min(next, len(region)):]
			if k := strings.IndexByte(following, '\n'); k >= 0 {
				following = following[:k]
			}

			next += len(following) + 1
			rhs += " " + strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(following), ";"), "/"))
			rhs = strings.TrimSpace(rhs)
		}

		if strings.Count(rhs, "(") != strings.Count(rhs, ")") {
			return "", 0, false
		}

		return rhs, i, true
	}

	return "", 0, false
}

const maxAssignmentLines = 24

// sourceText is strings.Join(strings.Split(content, "\n")[from:to], "\n"),
// clipped to the lines content has: the text of those lines, as a substring
// of content rather than a copy. "" when the window holds no line.
func sourceText(content string, from, to int) string {
	from = max(from, 0)

	begin, ok := lineOffset(content, from)
	if !ok || to <= from {
		return ""
	}

	end, ok := lineOffset(content, to)
	if !ok {
		return content[begin:]
	}

	return content[begin : end-1]
}

// lineIndex is the offsets of a text's line breaks, so the 0-based line of
// any offset is a binary search rather than a count from the top: counting
// from the top for each match in a file is quadratic in its length.
type lineIndex []int

func newLineIndex(s string) lineIndex {
	var idx lineIndex

	for at := 0; ; {
		i := strings.IndexByte(s[at:], '\n')
		if i < 0 {
			return idx
		}

		idx = append(idx, at+i)
		at += i + 1
	}
}

// lineOf is strings.Count(s[:offset], "\n").
func (l lineIndex) lineOf(offset int) int {
	return sort.SearchInts(l, offset)
}

// contentBeforeLine is strings.Join(strings.Split(content, "\n")[:n], "\n").
func contentBeforeLine(content string, n int) string {
	at, ok := lineOffset(content, n)
	if !ok {
		return content
	}

	if at == 0 {
		return ""
	}

	return content[:at-1]
}

func isScopeWord(s string) bool {
	switch strings.ToLower(s) {
	case "local", "variables", "this", "arguments", "prc", "rc", "session", "application", "request":
		return true
	default:
		return false
	}
}
