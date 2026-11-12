// The calls of a parse: per function, by name, in file order, and the callers they are filled with.

package parser

import (
	"slices"
	"sort"
	"strings"
)

// FuncCalls returns all variable.method() calls recorded for a function scope.
// Requires ExtractCalls: true in parse options.
//
// A function that calls nothing has no entry, and that is not the same question
// as "is this a function scope at all". This used to answer both by falling
// back to every call in the file, so a leaf method was handed its siblings'
// calls — `deps` drew an edge out of an empty function, labelled with another
// function's line. Callers wanting the whole file ask AllCalls for it.
//
// Without ExtractCalls the parse recorded no calls, and the function's are
// those of the parse's walk of the body holding it, made for the question.
func (pr *ParseResult) FuncCalls(funcStart, funcEnd int) []CallSite {
	scope := FuncScope{Start: funcStart, End: funcEnd}
	key := funcKey(funcStart, funcEnd)

	b := &pr.bodies
	b.mu.Lock()
	defer b.mu.Unlock()

	if pr.extractCalls {
		pr.readBody(scope, true)

		return pr.funcCallsMap[key]
	}

	// The calls need none of the typing passes but the runtime expressions
	// a call's component is read through.
	body := pr.bodyOf(scope)
	walk := pr.walkBody(body, true, true)
	walk.applyExpressionMappings()

	for k, pk := range pr.bodyKeys(body, walk) {
		if pk == key {
			return walk.funcCallsMap[k]
		}
	}

	return walk.funcCallsMap[key]
}

// AllCalls returns every call site in the file, wherever it was made: the ones
// outside any function and the ones inside each.
//
// This is what a whole-file scan wants — `unresolved`, `explain`, the code map
// and the MCP server each check or attribute every call in a file, and each
// used to ask FuncCalls for a line range covering the file and rely on the key
// missing. Requires ExtractCalls: true; without it there is nothing recorded to
// return.
func (pr *ParseResult) AllCalls() []CallSite {
	if !pr.extractCalls {
		return nil
	}

	n := len(pr.Calls)
	for _, calls := range pr.funcCallsMap {
		n += len(calls)
	}

	all := make([]CallSite, 0, n)
	all = append(all, pr.Calls...)

	for _, calls := range pr.funcCallsMap {
		all = append(all, calls...)
	}

	// The buckets come out of a map, so without this the order changes between
	// runs of one build — and every caller here writes a report a reader is
	// meant to diff against an earlier one.
	sortCallsLikeAllCalls(all)

	return all
}

// InnermostCallAround is the call with the shortest source span that starts
// before functionFrom and ends at or after to: the call a callback written in
// that range is an argument of. It is what a walk of AllCalls for the shortest
// enclosing span finds, without building and sorting a copy of every call in
// the file; spans of equal length are told apart by AllCalls' own order (line,
// then variable, then name). The result is a copy, or nil.
func (pr *ParseResult) InnermostCallAround(functionFrom, to int) *CallSite {
	if !pr.extractCalls {
		return nil
	}

	var best *CallSite

	consider := func(calls []CallSite) {
		for i := range calls {
			c := &calls[i]
			if !c.HasSpan || int(c.StartByte) >= functionFrom || int(c.EndByte) < to {
				continue
			}

			if best == nil || c.EndByte-c.StartByte < best.EndByte-best.StartByte ||
				c.EndByte-c.StartByte == best.EndByte-best.StartByte && callsInAllCallsOrder(c, best) {
				best = c
			}
		}
	}

	consider(pr.Calls)

	for _, calls := range pr.funcCallsMap {
		consider(calls)
	}

	if best == nil {
		return nil
	}

	found := *best

	return &found
}

// callsInAllCallsOrder reports whether a sorts before b in AllCalls.
func callsInAllCallsOrder(a, b *CallSite) bool {
	if a.Line != b.Line {
		return a.Line < b.Line
	}

	if a.Variable != b.Variable {
		return a.Variable < b.Variable
	}

	return a.FuncName < b.FuncName
}

// CallsNamed is AllCalls' answer restricted to calls whose name is one of
// names (compared case-insensitively), in the same order. Argument inference
// asks it for each untyped argument of each caller file, and taking AllCalls
// each time copied and sorted every call in the file to keep a few: with n
// functions called from one file, n² log n.
func (pr *ParseResult) CallsNamed(names ...string) []CallSite {
	if !pr.extractCalls {
		return nil
	}

	matches := func(name string) bool {
		for _, n := range names {
			if strings.EqualFold(name, n) {
				return true
			}
		}

		return false
	}

	var out []CallSite

	at := pr.callsAt(names)
	if at == nil {
		for i := range pr.Calls {
			if matches(pr.Calls[i].FuncName) {
				out = append(out, pr.Calls[i])
			}
		}

		for _, calls := range pr.funcCallsMap {
			for i := range calls {
				if matches(calls[i].FuncName) {
					out = append(out, calls[i])
				}
			}
		}
	} else {
		for _, a := range at {
			calls := pr.Calls
			if a.bucket != "" {
				calls = pr.funcCallsMap[a.bucket]
			}

			if a.i < len(calls) && matches(calls[a.i].FuncName) {
				out = append(out, calls[a.i])
			}
		}
	}

	sortCallsLikeAllCalls(out)

	return out
}

// CallsWritten is the calls of names (compared case-insensitively) in the
// order they are written: by line, then by where each starts. It answers for
// a parse that extracts every call, and for one asked to find these names'
// calls (ParseOptions.FindCalls); a name the parse recorded no calls of has
// none.
func (pr *ParseResult) CallsWritten(names ...string) []CallSite {
	var out []CallSite

	if pr.extractCalls {
		out = pr.CallsNamed(names...)
	} else {
		for i := range pr.Calls {
			if namedIn(names, pr.Calls[i].FuncName) {
				out = append(out, pr.Calls[i])
			}
		}
	}

	sort.SliceStable(out, func(i, j int) bool { return callWrittenBefore(&out[i], &out[j]) })

	return out
}

// dropCallsIndex forgets callsAt's index: a bucket that grows under its key
// changes none of the counts the index is checked against.
func (pr *ParseResult) dropCallsIndex() {
	pr.callsMu.Lock()
	pr.callsByName = nil
	pr.callsMu.Unlock()
}

// callAt is where one call is: Calls[i] when bucket is "", else
// funcCallsMap[bucket][i].
type callAt struct {
	bucket string
	i      int
}

// callsAt is where the calls named names are, through callsByName, or nil
// when a name is not ASCII (EqualFold and lowering then disagree) and the
// caller should walk every call. Asked once per untyped argument of every
// function called from a file, a walk of the file's every call was
// quadratic in a file that calls thousands of them.
func (pr *ParseResult) callsAt(names []string) []callAt {
	for _, n := range names {
		if !isASCII(n) {
			return nil
		}
	}

	var from *CallSite
	if len(pr.Calls) > 0 {
		from = &pr.Calls[0]
	}

	pr.callsMu.Lock()
	defer pr.callsMu.Unlock()

	if pr.callsByName == nil || pr.callsFrom != from || pr.callsLen != len(pr.Calls) || pr.callsBuckets != len(pr.funcCallsMap) {
		pr.callsByName = map[string][]callAt{}

		add := func(bucket string, calls []CallSite) {
			for i := range calls {
				n := strings.ToLower(calls[i].FuncName)
				pr.callsByName[n] = append(pr.callsByName[n], callAt{bucket: bucket, i: i})
			}
		}

		add("", pr.Calls)

		for k, calls := range pr.funcCallsMap {
			add(k, calls)
		}

		pr.callsFrom, pr.callsLen, pr.callsBuckets = from, len(pr.Calls), len(pr.funcCallsMap)
	}

	out := []callAt{}

	for i, n := range names {
		// A name given twice is one name: each call is listed once.
		if slices.ContainsFunc(names[:i], func(m string) bool { return strings.EqualFold(m, n) }) {
			continue
		}

		var buf foldScratch

		out = append(out, pr.callsByName[string(buf.lowerFold(n))]...)
	}

	return out
}

// sortCallsLikeAllCalls orders calls by line, then variable, then name, as
// AllCalls returns them.
func sortCallsLikeAllCalls(all []CallSite) {
	sort.SliceStable(all, func(i, j int) bool { return callsInAllCallsOrder(&all[i], &all[j]) })
}

// fillCallers names the enclosing function of every call recorded without one.
//
// A parser names a call's caller from the functions it has parsed itself, and
// a region cut from a function body holds none: the <cffunction> tag was in an
// earlier region. So the calls after a <cfscript> island or a <script> block in
// a tag function, and every #...# call inside such a block, came back with no
// caller, and the ARGUMENTS lookup for a function-reference call and the
// function column of find-references both lost them. The scopes are the
// file's, so the line answers it for every region alike.
func (pr *ParseResult) fillCallers() {
	scopeAt := scopeIndex(pr.Scopes, false)

	fill := func(calls []CallSite) {
		for i := range calls {
			if calls[i].Caller == "" {
				if k := scopeAt(int(calls[i].Line)); k >= 0 {
					calls[i].Caller = pr.Scopes[k].Name
				}
			}
		}
	}

	fill(pr.Calls)

	for _, calls := range pr.funcCallsMap {
		fill(calls)
	}
}

// resolveVarComponent finds the component a variable resolves to from pr.ComponentRefs.
func (pr *ParseResult) resolveVarComponent(varName string) string {
	// Check ComponentRefs — these are component-wide, always valid
	for i := range pr.ComponentRefs {
		ref := &pr.ComponentRefs[i]

		if strings.EqualFold(ref.Variable, varName) {
			return ref.Component
		}
	}

	return ""
}
