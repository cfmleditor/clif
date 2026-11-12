// A function body read on its own, as the whole parse's walk reads it.

package parser

import (
	"slices"
	"strings"
	"sync"
)

// bodyState is what a parse knows of its function bodies: the declarations
// its walk recorded in them, which of them an edit has changed since, and,
// for a parse walking one body (ParseOptions.Body), which body that is.
//
// The lazy readers (FuncVars, FuncRefs, FuncCalls, HasScopedAssignment) used
// to read a body with walkers of their own: the script parser over the
// body's raw text, whether it was script or tags, with none of the parse's
// region split, output gate, island seeding or typing. They now answer from
// the parse's walk, and where it did not walk a body, or an edit has changed
// one, from the parse's walk of that body alone (readBody).
type bodyState struct {
	// only is the body a ParseOptions.Body parse walks, and context the
	// parse it reads that body of; both nil for a whole parse.
	only    *FuncScope
	context *ParseResult

	// mu guards everything below. It is taken before funcVarsMu and
	// funcRefsMu, and after pr.mu, never the other way.
	mu sync.Mutex

	// vars is every declaration the walk recorded in a script or tag region,
	// on file lines and in the order it met them; tag marks a tag region's.
	// An edit inside a function moves the lines after it (shift) and a walk
	// of a body replaces that body's (splice).
	vars []walkVar
	// byLine is vars' indexes in line order, built on the first question
	// when vars is not in line order already (sorted).
	byLine []int32
	sorted bool
	// varsWalked is true when vars holds the whole file's: a whole parse
	// that walked every region. settled is true when the parse also typed
	// what the bodies assign (a parse that is not Shallow), so its function
	// refs, links and calls are a whole parse's.
	varsWalked, settled bool
	// vars and refs are, for the declarations and for the refs, links and
	// calls, the bodies to read again: those an edit changed (stale) and,
	// where the parse did not read them, those read alone since (done).
	varsOf, refsOf bodyMarks
	// members is the bodies read alone whose refs a member write types:
	// the member-binding pass reads the whole file, so their refs are a
	// whole parse's, made on the first question about refs.
	members []FuncScope

	// anyScoped is HasScopedAssignment's answer per scope, from vars.
	anyScoped map[Scope][]string

	// outputAt is the output state the walk had where a tag function opens
	// on a line, where it was not the empty one.
	outputAt map[int]outputState

	// regions, lines, tagScopes and hasTag are the file's split and tag
	// function scopes for the content regionsOf, for a body's walk to start
	// from: an edit inside a function changes Content and not pr.Regions.
	regions   []Region
	lines     LineTable
	tagScopes []FuncScope
	hasTag    bool
	regionsOf string
	scopesOf  string
	// comments is, for each of regions, where the <!--- comments in it open
	// and end, nil until a body's walk asks (commentAround).
	comments [][][2]int
}

// walkVar is a declaration the walk recorded, and whether it was in a tag
// region.
type walkVar struct {
	VarDef

	tag bool
}

// keep records a region's declarations, on file lines, in walk order.
func (b *bodyState) keep(vars []VarDef, tag bool) {
	for _, v := range vars {
		b.vars = append(b.vars, walkVar{VarDef: v, tag: tag})
	}
}

// functionOutput is the output state the tag walk had at a function it
// opened on line.
type functionOutput struct {
	line  int
	state outputState
}

// noteOutput keeps the output state at a tag function opening on line.
func (b *bodyState) noteOutput(line int, state outputState) {
	if b.outputAt == nil {
		b.outputAt = map[int]outputState{}
	}

	b.outputAt[line] = state
}

// walked notes, once a parse's walk is done, whether it read the whole file's
// bodies.
func (b *bodyState) walked(pr *ParseResult) {
	b.byLine, b.sorted, b.anyScoped, b.varsOf, b.refsOf, b.members = nil, false, nil, bodyMarks{}, bodyMarks{}, nil
	b.varsWalked = b.only == nil && !pr.scopesOnly
	b.settled = false
}

// bodyMarks is which bodies one aspect of a parse's answers must be read
// again for: stale, the bodies an edit changed, and done, the bodies read
// alone since, which for a parse that did not read an aspect at all are the
// only ones it has.
type bodyMarks struct {
	stale, done []FuncScope
}

// shift moves the marks for an edit on editLine.
func (m *bodyMarks) shift(editLine, delta int) {
	for i := range m.stale {
		m.stale[i] = shiftScope(m.stale[i], editLine, delta)
	}

	for i := range m.done {
		m.done[i] = shiftScope(m.done[i], editLine, delta)
	}
}

// change marks the body on lines start through end changed. A body already
// marked, or inside one that is, is not marked again: nothing clears a mark
// until the body is read, and an edit marks the body it is in, so a run of
// keystrokes in one function with nothing asking about it added a mark each,
// and every later edit shifted them all.
func (m *bodyMarks) change(body FuncScope) {
	if _, ok := covering(m.stale, body); !ok {
		m.stale = slices.DeleteFunc(m.stale, func(r FuncScope) bool { return r.Start >= body.Start && r.End <= body.End })
		m.stale = append(m.stale, body)
	}

	m.done = slices.DeleteFunc(m.done, func(r FuncScope) bool { return r.End >= body.Start && r.Start <= body.End })
}

// read marks body read alone.
func (m *bodyMarks) read(body FuncScope) {
	m.stale = slices.DeleteFunc(m.stale, func(r FuncScope) bool { return r.Start >= body.Start && r.End <= body.End })
	m.done = slices.DeleteFunc(m.done, func(r FuncScope) bool { return r.Start >= body.Start && r.End <= body.End })
	m.done = append(m.done, body)
}

// due is the body to read again for scope, if any: the changed one holding
// it, or, when the parse did not read this aspect (read false), the body
// holding it unless it was read alone since.
func (pr *ParseResult) due(m *bodyMarks, read bool, scope FuncScope) (FuncScope, bool) {
	if body, ok := covering(m.stale, scope); ok {
		return body, true
	}

	if !read {
		if _, ok := covering(m.done, scope); !ok {
			return pr.bodyOf(scope), true
		}
	}

	return FuncScope{}, false
}

// bodyRegions is the file's regions cut to body's lines: each region's text
// where it meets them, starting on the line it then starts on.
//
// A script function's doc comment, above its first line, is read with it, as
// the whole walk reads it: its hints type the arguments. A tag region cut
// inside a <!--- comment the whole walk is still in starts where the comment
// ends, so a function the scope finder found in commented-out text is a
// comment to this walk too.
func (pr *ParseResult) bodyRegions(body *FuncScope) []Region {
	from, to := pr.lineOffsets(body.Start, body.End)
	if from < 0 {
		return nil
	}

	var cut []Region

	for i, r := range pr.Regions {
		end := r.Offset + len(r.Text)

		lo, hi := max(r.Offset, from), min(end, to)
		if lo >= hi {
			continue
		}

		switch {
		case r.Kind == RegionScript && lo > r.Offset:
			lo = max(r.Offset, pr.docCommentStart(lo))
		case r.Kind != RegionScript && lo > r.Offset:
			lo = min(pr.commentAround(i, lo), hi)
		}

		if lo >= hi {
			continue
		}

		start := r.StartLine
		if lo > r.Offset {
			start = pr.lineAt(lo)
		}

		// From the region's own text, not the file's: a script region holds
		// its tag islands as blanks, and the island is a region of its own.
		cut = append(cut, Region{Kind: r.Kind, StartLine: start, Offset: lo, Text: r.Text[lo-r.Offset : hi-r.Offset], Island: r.Island})
	}

	return cut
}

// docCommentStart is where the line holding the /* opening a block comment
// that ends on the lines just above the line starting at off begins, past
// blank lines and // comments; off itself when there is none.
func (pr *ParseResult) docCommentStart(off int) int {
	lines := pr.lineStartsTable()

	for line := lines.LineOf(off) - 1; line >= 0; line-- {
		from, to := lines.Span(line, line, len(pr.Content))
		text := strings.TrimSpace(pr.Content[from:to])

		switch {
		case text == "", strings.HasPrefix(text, "//"):
			continue
		case !strings.HasSuffix(text, "*/"):
			return off
		}

		// Block comments do not nest: the comment opens at the first /*
		// after the comment before it closed.
		closeAt := strings.LastIndex(pr.Content[:to], "*/")
		prev := max(strings.LastIndex(pr.Content[:max(closeAt, 0)], "*/")+2, 0)

		open := strings.Index(pr.Content[prev:max(closeAt, prev)], "/*")
		if open < 0 {
			return off
		}

		open += prev

		start, _ := lines.Start(lines.LineOf(open))

		return start
	}

	return off
}

// commentAround is lo, or, when lo is inside a <!--- comment in the i'th of
// the regions a body's walk cuts, where the comment ends: a walk of the
// region would read everything up to there as comment. The tag function
// finder reads comments by a rule of its own, so it can find a function the
// walk reads as commented out: in a comment holding a nested opener, or
// after `<!------->`, which by the comment rule opens a comment that never
// closes. The comments are read once per split, in the context's regions.
func (pr *ParseResult) commentAround(i, lo int) int {
	ctx := pr.bodies.context
	if ctx == nil {
		ctx = pr
	}

	b := &ctx.bodies
	if len(b.comments) != len(pr.Regions) {
		b.comments = make([][][2]int, len(pr.Regions))
	}

	spans := b.comments[i]
	if spans == nil {
		r := &pr.Regions[i]
		end := r.Offset + len(r.Text)
		spans = [][2]int{}

		for pos := r.Offset; pos < end; {
			at := strings.Index(pr.Content[pos:end], "<!---")
			if at < 0 {
				break
			}

			at += pos
			pos = SkipCFMLComment(pr.Content, at)
			spans = append(spans, [2]int{at, pos})
		}

		b.comments[i] = spans
	}

	j, _ := slices.BinarySearchFunc(spans, lo, func(c [2]int, lo int) int { return c[0] - lo })
	if j > 0 && spans[j-1][1] > lo {
		return spans[j-1][1]
	}

	return lo
}

// outputAtBody is the output state the whole walk has where a body's first
// tag region starts: what it noted at the function opening there, and for a
// function it did not note (one an edit added), the component's own output
// attribute, the one before it having closed.
func (pr *ParseResult) outputAtBody(regions []Region) outputState {
	i := slices.IndexFunc(regions, func(r Region) bool { return r.Kind != RegionScript })
	if i < 0 || pr.interpolateAll {
		return outputState{}
	}

	if ctx := pr.bodies.context; ctx != nil {
		if o, ok := ctx.bodies.outputAt[pr.bodies.only.Start]; ok {
			return o
		}
	}

	var o outputState

	if at := IndexFold(pr.Content, "<cfcomponent"); at >= 0 && at < regions[i].Offset {
		// The component's tag alone: tag ends over the whole file look for
		// its last '>' first.
		if end := tagEndIndex(pr.Content[at:]); end >= 0 {
			tag := pr.Content[at : at+end+1]
			ends := newTagEnds(tag)
			o.track(&ends, 0)
			o.depth, o.fnOut = 0, false
		}
	}

	return o
}

// keepSplit keeps a whole parse's region split and tag function scopes for
// the walks of its bodies, which an edit inside a function carries over
// (shiftSplit) where it cannot have changed them.
func (pr *ParseResult) keepSplit(tagScopes []FuncScope, hasTag bool) {
	b := &pr.bodies
	b.regions, b.lines, b.regionsOf, b.comments = pr.Regions, pr.contentLineIdx, pr.Content, nil
	b.tagScopes, b.hasTag, b.scopesOf = tagScopes, hasTag, pr.Content
}

// splitChars are the bytes an edit inside a function must not hold, removed
// or inserted, for the file's region split and tag functions to be the ones
// before it, moved: tags and comments (<, >, !, -), script comments (/, *),
// and strings and their expressions (", ', #), which decide where a
// <cfscript> block ends.
const splitChars = `<>!-/*"'#`

// shiftSplit carries the kept split and tag function scopes over an edit of
// old, the content before it, at startLine:startChar that replaced removed
// bytes with newText, when the edit lies inside one region and holds none of
// splitChars: the region holding it grows or shrinks, and those after it
// move. Otherwise the next body walk splits the file again. Asked of every
// keystroke inside a function, it is what keeps a body walk from splitting
// a large file again for each.
func (pr *ParseResult) shiftSplit(old string, startOff, removed int, newText string) {
	b := &pr.bodies
	b.mu.Lock()
	defer b.mu.Unlock()

	content := pr.Content
	endOff := startOff + removed

	ok := b.regionsOf == old && b.scopesOf == old && b.regions != nil && startOff > 0 && endOff <= len(old) &&
		!strings.ContainsAny(old[startOff:endOff], splitChars) && !strings.ContainsAny(newText, splitChars) &&
		strings.HasPrefix(content[startOff:], newText)
	if !ok {
		return
	}

	delta := len(newText) - removed
	lineDelta := strings.Count(newText, "\n") - strings.Count(old[startOff:endOff], "\n")
	editLine := strings.Count(old[:startOff], "\n")

	// A file with tag islands is split again: a script region holds its
	// islands as blanks, which text cut from content would not.
	for i := range b.regions {
		if b.regions[i].Island {
			return
		}
	}

	regions := make([]Region, len(b.regions))
	holders := 0

	for i, r := range b.regions {
		end := r.Offset + len(r.Text)

		switch {
		case r.Offset > startOff:
			if r.Offset < endOff {
				return
			}

			r.Offset += delta
			r.StartLine += lineDelta
			r.Text = content[r.Offset : r.Offset+len(r.Text)]
		case startOff >= r.Offset && endOff <= end:
			holders++
			r.Text = content[r.Offset : end+delta]
		case end > startOff:
			return
		default:
			r.Text = content[r.Offset:end]
		}

		regions[i] = r
	}

	if holders != 1 {
		return
	}

	scopes := slices.Clone(b.tagScopes)
	for i := range scopes {
		scopes[i] = shiftScope(scopes[i], editLine, lineDelta)
	}

	// The comments open and close where they did, moved: the edit holds no
	// comment, though it may be in one.
	for _, spans := range b.comments {
		for j := range spans {
			switch {
			case spans[j][0] > startOff:
				spans[j][0] += delta
				spans[j][1] += delta
			case spans[j][1] > startOff:
				spans[j][1] += delta
			}
		}
	}

	b.regions, b.lines, b.regionsOf = regions, nil, content
	b.tagScopes, b.scopesOf = scopes, content
}

// walkRegions is the file's regions as the parse splits Content now, for a
// body's walk. b.mu is held by the reader that started the walk.
func (pr *ParseResult) walkRegions() ([]Region, LineTable) {
	b := &pr.bodies
	if b.regionsOf == pr.Content && b.regions != nil && b.lines == nil {
		return b.regions, pr.lineStartsTable()
	}

	if b.regionsOf != pr.Content || b.regions == nil {
		if pr.regionsOf == pr.Content && pr.Regions != nil {
			b.regions, b.lines = pr.Regions, pr.contentLineIdx
		} else {
			b.regions, b.lines = pr.classifyRegions()

			// The split counted the lines of the content an edit made; the
			// line table built for that content next is the same one.
			if b.lines != nil {
				pr.lineStartsMu.Lock()
				if pr.lineStartsContent != pr.Content {
					pr.lineStarts, pr.lineStartsContent = b.lines, pr.Content
				}
				pr.lineStartsMu.Unlock()
			}
		}

		b.regionsOf, b.comments = pr.Content, nil
	}

	return b.regions, b.lines
}

// walkTagScopes is tagFunctionScopes for Content now, for a body's walk. b.mu
// is held.
func (pr *ParseResult) walkTagScopes() ([]FuncScope, bool) {
	b := &pr.bodies
	if b.scopesOf != pr.Content {
		regions, lines := pr.walkRegions()
		b.tagScopes, b.hasTag = nil, false

		for _, r := range regions {
			if r.Kind == RegionTag {
				b.tagScopes, b.hasTag = findTagFuncScopesIdx(pr.Content, 0, lines, regions), true

				break
			}
		}

		b.scopesOf = pr.Content
	}

	return b.tagScopes, b.hasTag
}

// settleBody types what a body's walk left pending as the whole parse types
// it (resolvePendingCalls): the calls assigned, with the file's functions'
// return types, and as receivers the file-level refs of the file, its other
// lines' as the whole parse found them and the body's as this walk did, in
// file order; the calls left untyped are looked at again while a round types
// one. It runs after the runtime expressions (applyExpressionMappings) and
// before the chains (applyChainedReturnLookup), in the whole parse's order.
// The passes that read the whole file (member bindings, factory and
// collection returns, service properties) are left to the whole parse; a
// body with a member binding takes its refs from one (fileBody).
func (pr *ParseResult) settleBody(calls []pendingCall) {
	funcs := pr

	pr.applyExpressionMappings()

	// The file's other refs stand around the body's while it is typed, and
	// are taken out again after.
	var before, after []ComponentRef

	own := len(pr.ComponentRefs)

	if ctx := pr.bodies.context; ctx != nil {
		funcs = ctx

		for i := range ctx.ComponentRefs {
			switch line := int(ctx.ComponentRefs[i].Line); {
			case line < pr.bodies.only.Start:
				before = append(before, ctx.ComponentRefs[i])
			case line > pr.bodies.only.End:
				after = append(after, ctx.ComponentRefs[i])
			}
		}

		pr.ComponentRefs = slices.Concat(before, pr.ComponentRefs, after)
	}

	if len(calls) > 0 {
		returns := funcs.returnsCalled(calls)
		untyped, typed := pr.typePendingCalls(calls, returns, nil)

		for range maxReturnRounds {
			if typed == 0 || len(untyped) == 0 {
				break
			}

			var still []int

			for _, j := range untyped {
				if c := &calls[j]; !pr.addPendingRef(c, pr.pendingComponent(c, returns, nil)) {
					still = append(still, j)
				}
			}

			typed, untyped = len(untyped)-len(still), still
		}
	}

	pr.flow = nil
	pr.applyChainedReturnLookup()

	if len(before)+len(after) > 0 {
		refs := pr.ComponentRefs
		pr.ComponentRefs = slices.Concat(refs[len(before):len(before)+own], refs[len(before)+own+len(after):])
	}
}

// returnsCalled is funcReturns limited to the functions calls name, on any
// receiver: one typed as this file (`node = this; node = node.parent()`) calls
// the file's own.
func (pr *ParseResult) returnsCalled(calls []pendingCall) map[string]string {
	out := map[string]string{}

	for i := range calls {
		if c := &calls[i]; c.funcName != "" && !c.returnExpr {
			out[strings.ToLower(c.funcName)] = ""
		}
	}

	for i := range pr.Funcs {
		f := &pr.Funcs[i]

		key := strings.ToLower(f.Name)
		if _, ok := out[key]; !ok {
			continue
		}

		comp := f.ReturnComponent
		if comp == "" && isComponentType(f.ReturnType) {
			comp = f.ReturnType
		}

		if comp != "" {
			out[key] = comp
		}
	}

	for k, v := range out {
		if v == "" {
			delete(out, k)
		}
	}

	return out
}

// bodyOf is the body readBody walks for scope: the outermost function holding
// it, so a function nested in another is read as the whole walk reads it,
// from the function it is written in.
func (pr *ParseResult) bodyOf(scope FuncScope) FuncScope {
	outer := scope

	for _, s := range pr.Scopes {
		if s.Start <= outer.Start && s.End >= outer.End && (s.Start < outer.Start || s.End > outer.End) {
			outer = s
		}
	}

	return outer
}

// within reports whether the function keyed k lies on body's lines.
func within(k fnKey, body FuncScope) bool {
	return k.in && int(k.start) >= body.Start && int(k.end) <= body.End
}

// covering is the range in ranges holding scope.
func covering(ranges []FuncScope, scope FuncScope) (FuncScope, bool) {
	for _, r := range ranges {
		if r.Start <= scope.Start && scope.End <= r.End {
			return r, true
		}
	}

	return FuncScope{}, false
}

// readBody makes the parse's answers for scope current: when an edit changed
// its body, or the parse did not read it, it walks the body holding it on its
// own (ParseOptions.Body) and files what that walk found in place of what the
// parse had. refs asks for the refs, links and calls as well as the
// declarations: a Shallow parse read the declarations but did not type the
// refs, and a walk for the declarations alone does not type them either.
// b.mu is held.
func (pr *ParseResult) readBody(scope FuncScope, refs bool) {
	b := &pr.bodies

	if refs {
		if body, ok := pr.due(&b.refsOf, b.settled, scope); ok {
			walk := pr.walkBody(body, false, pr.extractCalls)
			pr.fileRefs(body, walk)
			pr.fileVars(body, walk)
			b.refsOf.read(body)
			b.varsOf.read(body)
		}

		pr.typeMembers(scope)

		return
	}

	if body, ok := pr.due(&b.varsOf, b.varsWalked, scope); ok {
		pr.fileVars(body, pr.walkBody(body, true, false))
		b.varsOf.read(body)
	}
}

// walkBody is the parse's walk of body alone; unless shallow, typed as the
// parse types it.
func (pr *ParseResult) walkBody(body FuncScope, shallow, calls bool) *ParseResult {
	pr.walkRegions()
	pr.walkTagScopes()

	return ParseWithOptions(pr.URI, pr.Content, &ParseOptions{
		Logger:                   pr.log,
		Resolvers:                pr.Resolvers,
		ResolverSet:              pr.resolverSet,
		PropertyResolvers:        pr.PropertyResolvers,
		ConstructorLookup:        pr.ConstructorLookup,
		SetterLookup:             pr.SetterLookup,
		BeanLookup:               pr.BeanLookup,
		PropertyBeanLookup:       pr.PropertyBeanLookup,
		BuiltinReturnLookup:      pr.BuiltinReturnLookup,
		FuncLookup:               pr.FuncLookup,
		ExpressionMappings:       pr.expressionMappings,
		ServicePropertyResolvers: pr.ServicePropertyResolvers,
		ExtractLinks:             !shallow,
		ExtractCalls:             calls,
		Shallow:                  shallow,
		InterpolateAllText:       pr.interpolateAll,
		Body:                     &body,
		context:                  pr,
	})
}

// bodyKeys maps the keys of the functions walk found to the keys pr has for
// them, which can differ after an edit only in where pr thinks a function
// ends; a function pr has no scope for keeps its own.
func (pr *ParseResult) bodyKeys(body FuncScope, walk *ParseResult) map[fnKey]fnKey {
	keys := make(map[fnKey]fnKey, len(walk.Scopes))

	for _, ws := range walk.Scopes {
		k := funcKey(ws.Start, ws.End)
		keys[k] = k

		for _, s := range pr.Scopes {
			if s.Start == ws.Start && s.Start >= body.Start && s.End <= body.End && strings.EqualFold(s.Name, ws.Name) {
				keys[k] = funcKey(s.Start, s.End)

				break
			}
		}
	}

	return keys
}

// fileRefs files the refs, links and (with ExtractCalls) calls walk, the
// walk of body alone, found in pr, under pr's keys for its functions, in
// place of what pr had for any function on those lines. A body with a member
// write is noted for typeMembers.
func (pr *ParseResult) fileRefs(body FuncScope, walk *ParseResult) {
	keys := pr.bodyKeys(body, walk)

	pr.funcRefsMu.Lock()
	pr.funcRefsMap = refile(pr.funcRefsMap, walk.funcRefsMap, body, keys)
	pr.funcLinksMap = refile(pr.funcLinksMap, walk.funcLinksMap, body, keys)
	pr.funcRefsMu.Unlock()

	if pr.extractCalls {
		pr.funcCallsMap = refile(pr.funcCallsMap, walk.funcCallsMap, body, keys)
		pr.dropCallsIndex()
	}

	if from, to := pr.lineOffsets(body.Start, body.End); from >= 0 && pr.hasMemberBinding(pr.Content[from:to]) {
		pr.bodies.members = append(pr.bodies.members, body)
	}
}

// typeMembers gives the functions of a body read alone whose text writes a
// member, when scope is in one, the refs a whole parse gives them: the
// member-binding pass reads every write in the file.
func (pr *ParseResult) typeMembers(scope FuncScope) {
	b := &pr.bodies

	body, ok := covering(b.members, scope)
	if !ok {
		return
	}

	b.members = slices.DeleteFunc(b.members, func(r FuncScope) bool { return r == body })

	refs := pr.memberRefs()

	pr.funcRefsMu.Lock()
	pr.funcRefsMap = refile(pr.funcRefsMap, refs, body, nil)
	pr.funcRefsMu.Unlock()
}

// fileVars files the declarations walk, the walk of body alone, recorded in
// place of those pr had on its lines, and forgets the variable lists made
// from them.
func (pr *ParseResult) fileVars(body FuncScope, walk *ParseResult) {
	pr.spliceVars(body, walk.bodies.vars)
	pr.bodies.anyScoped = nil

	pr.funcVarsMu.Lock()
	for k := range pr.funcVars {
		if within(k, body) {
			delete(pr.funcVars, k)
			delete(pr.funcVarSets, k)
		}
	}
	pr.funcVarsMu.Unlock()
}

// memberRefs is a whole parse's function refs for Content now, made once per
// version of it. funcRefsMu is not held.
func (pr *ParseResult) memberRefs() map[fnKey][]ComponentRef {
	if pr.memberSnapshot == nil || pr.memberSnapshotContent != pr.Content {
		pr.memberSnapshot = ParseWithOptions(pr.URI, pr.Content, &ParseOptions{
			Resolvers: pr.Resolvers, PropertyResolvers: pr.PropertyResolvers,
			FuncLookup: pr.FuncLookup, BeanLookup: pr.BeanLookup,
			SetterLookup: pr.SetterLookup, ConstructorLookup: pr.ConstructorLookup,
			PropertyBeanLookup: pr.PropertyBeanLookup, BuiltinReturnLookup: pr.BuiltinReturnLookup,
			ExpressionMappings: pr.expressionMappings, ServicePropertyResolvers: pr.ServicePropertyResolvers,
		})
		pr.memberSnapshotContent = pr.Content
	}

	return pr.memberSnapshot.funcRefsMap
}

// refile is m with every entry for a function on body's lines replaced by
// what found has for it, under keys' key for it.
func refile[T any](m, found map[fnKey][]T, body FuncScope, keys map[fnKey]fnKey) map[fnKey][]T {
	for k := range m {
		if within(k, body) {
			delete(m, k)
		}
	}

	for k, v := range found {
		if !within(k, body) {
			continue
		}

		if pk, ok := keys[k]; ok {
			k = pk
		}

		if m == nil {
			m = map[fnKey][]T{}
		}

		m[k] = slices.Clone(v)
	}

	return m
}

// spliceVars replaces the declarations on body's lines with found, the walk of
// that body's, where the first of them was.
func (pr *ParseResult) spliceVars(body FuncScope, found []walkVar) {
	b := &pr.bodies
	in := func(v *walkVar) bool { return int(v.Line) >= body.Start && int(v.Line) <= body.End }

	from := slices.IndexFunc(b.vars, func(v walkVar) bool { return in(&v) || int(v.Line) > body.End })
	if from < 0 {
		from = len(b.vars)
	}

	to := from
	for to < len(b.vars) && in(&b.vars[to]) {
		to++
	}

	found = slices.DeleteFunc(found, func(v walkVar) bool { return !in(&v) })

	// The body's declarations are together in walk order, as a walk meets
	// them: replaced where they stand.
	if !slices.ContainsFunc(b.vars[to:], func(v walkVar) bool { return in(&v) }) {
		b.vars, b.byLine, b.sorted = slices.Replace(b.vars, from, to, found...), nil, false

		return
	}

	kept := make([]walkVar, 0, len(b.vars)+len(found))
	kept = append(kept, b.vars[:from]...)
	kept = append(kept, found...)

	for i := from; i < len(b.vars); i++ {
		if !in(&b.vars[i]) {
			kept = append(kept, b.vars[i])
		}
	}

	b.vars, b.byLine, b.sorted = kept, nil, false
}

// varsOn is the declarations the walk recorded on lines start through end, in
// the order it met them. b.mu is held.
func (b *bodyState) varsOn(start, end int) []walkVar {
	if b.byLine == nil && !b.sorted {
		b.sorted = slices.IsSortedFunc(b.vars, func(x, y walkVar) int { return int(x.Line) - int(y.Line) })
	}

	// Walk order is line order but where a walk records a declaration
	// after a later line's (a tag's attributes, a closure's parameters):
	// then the lines are searched through an index.
	if b.sorted {
		lo, _ := slices.BinarySearchFunc(b.vars, start, func(v walkVar, line int) int { return int(v.Line) - line })
		hi := lo

		for hi < len(b.vars) && int(b.vars[hi].Line) <= end {
			hi++
		}

		return b.vars[lo:hi:hi]
	}

	if b.byLine == nil {
		b.byLine = make([]int32, len(b.vars))
		for i := range b.byLine {
			b.byLine[i] = int32(i)
		}

		slices.SortStableFunc(b.byLine, func(i, j int32) int { return int(b.vars[i].Line) - int(b.vars[j].Line) })
	}

	lo, _ := slices.BinarySearchFunc(b.byLine, start, func(i int32, line int) int { return int(b.vars[i].Line) - line })

	var at []int32

	for _, i := range b.byLine[lo:] {
		if int(b.vars[i].Line) > end {
			break
		}

		at = append(at, i)
	}

	slices.Sort(at)

	out := make([]walkVar, len(at))
	for k, i := range at {
		out[k] = b.vars[i]
	}

	return out
}

// shift moves the declarations after an edit on line editLine by delta lines,
// as shiftAfter moves the scopes and refs.
func (b *bodyState) shift(editLine, delta int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range b.vars {
		if int(b.vars[i].Line) > editLine {
			b.vars[i].Line = uint32(int(b.vars[i].Line) + delta) //nolint:gosec // a line number
		}
	}

	b.sorted = false

	b.varsOf.shift(editLine, delta)
	b.refsOf.shift(editLine, delta)

	for i := range b.members {
		b.members[i] = shiftScope(b.members[i], editLine, delta)
	}

	if len(b.outputAt) > 0 {
		moved := make(map[int]outputState, len(b.outputAt))
		for line, o := range b.outputAt {
			if line > editLine {
				line += delta
			}

			moved[line] = o
		}

		b.outputAt = moved
	}

	b.byLine = nil
}

// shiftScope is s moved as shiftAfter moves a scope for an edit on editLine.
func shiftScope(s FuncScope, editLine, delta int) FuncScope {
	if s.Start > editLine {
		s.Start += delta
		s.End += delta
	} else if s.End >= editLine {
		s.End += delta
	}

	return s
}

// shiftKey is the function key k moved as shiftScope moves its scope.
func shiftKey(k fnKey, editLine, delta int) fnKey {
	if !k.in {
		return k
	}

	s := shiftScope(k.scope(), editLine, delta)

	return funcKey(s.Start, s.End)
}

// shiftKeys is m with its function keys moved for an edit on editLine, and
// each value's lines by move.
func shiftKeys[T any](m map[fnKey][]T, editLine, delta int, move func(*T)) map[fnKey][]T {
	if len(m) == 0 {
		return m
	}

	out := make(map[fnKey][]T, len(m))

	for k, v := range m {
		if move != nil {
			for i := range v {
				move(&v[i])
			}
		}

		out[shiftKey(k, editLine, delta)] = v
	}

	return out
}

// shiftBodies moves what the parse keeps of its bodies for an edit inside a
// function on line editLine that changed the line count by delta: their keys,
// and the lines of what each holds.
func (pr *ParseResult) shiftBodies(editLine, delta int) {
	pr.bodies.shift(editLine, delta)

	line := func(l *uint32) {
		if int(*l) > editLine {
			*l = uint32(int(*l) + delta) //nolint:gosec // a line number
		}
	}

	pr.funcRefsMu.Lock()
	pr.funcRefsMap = shiftKeys(pr.funcRefsMap, editLine, delta, func(r *ComponentRef) { line(&r.Line) })
	pr.funcLinksMap = shiftKeys(pr.funcLinksMap, editLine, delta, func(l *DocumentLink) { line(&l.Line) })
	pr.funcCallsMap = shiftKeys(pr.funcCallsMap, editLine, delta, func(c *CallSite) { line(&c.Line) })
	pr.funcRefsMu.Unlock()
	pr.dropCallsIndex()

	pr.funcVarsMu.Lock()
	pr.funcVars = shiftKeys(pr.funcVars, editLine, delta, nil)
	pr.funcVarSets = nil
	pr.funcVarsMu.Unlock()
}

// staleBody marks the body on lines start through end as changed since the
// parse walked it, and forgets what the parse had for it.
func (pr *ParseResult) staleBody(start, end int) {
	body := FuncScope{Start: start, End: end}

	b := &pr.bodies
	b.mu.Lock()
	b.varsOf.change(body)
	b.refsOf.change(body)
	b.members = slices.DeleteFunc(b.members, func(r FuncScope) bool { return r.End >= start && r.Start <= end })
	b.anyScoped = nil
	b.mu.Unlock()

	pr.funcVarsMu.Lock()
	for k := range pr.funcVars {
		if within(k, body) {
			delete(pr.funcVars, k)
			delete(pr.funcVarSets, k)
		}
	}
	pr.funcVarsMu.Unlock()

	pr.funcRefsMu.Lock()
	for k := range pr.funcRefsMap {
		if within(k, body) {
			delete(pr.funcRefsMap, k)
			delete(pr.funcLinksMap, k)
		}
	}

	pr.memberSnapshot = nil
	pr.funcRefsMu.Unlock()
}

// funcVarNames is the local and arguments names declared on body's lines, in
// the order the walk met them, each once. b.mu is held.
func (b *bodyState) funcVarNames(start, end int) []string {
	var names []string

	seen := map[string]bool{}

	for _, v := range b.varsOn(start, end) {
		if (v.Scope == ScopeLocal || v.Scope == ScopeArguments) && !seen[v.Name] {
			seen[v.Name] = true

			names = append(names, v.Name)
		}
	}

	return names
}
