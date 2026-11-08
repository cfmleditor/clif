// The calls of a parse: per function, by name, in file order, and the callers they are filled with.

package parser

import (
	"slices"
	"sort"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// FuncCalls returns all variable.method() calls recorded for a function scope.
// Requires ExtractCalls: true in parse options.
//
// A function that calls nothing has no entry, and that is not the same question
// as "is this a function scope at all". This used to answer both by falling
// back to every call in the file, so a leaf method was handed its siblings'
// calls — `deps` drew an edge out of an empty function, labelled with another
// function's line. Callers wanting the whole file ask AllCalls for it.
func (pr *ParseResult) FuncCalls(funcStart, funcEnd int) []CallSite {
	if pr.extractCalls {
		return pr.funcCallsMap[funcKey(funcStart, funcEnd)]
	}

	return pr.funcCallsUncached(funcStart, funcEnd)
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

// funcCallsUncached parses a function body on-demand to extract call sites.
func (pr *ParseResult) funcCallsUncached(funcStart, funcEnd int) []CallSite {
	start, end := pr.lineOffsets(funcStart, funcEnd)
	if start < 0 {
		return nil
	}

	body := pr.Content[start:end]

	regionKind := RegionScript

	for _, r := range pr.Regions {
		if r.StartLine <= funcStart {
			regionKind = r.Kind
		}
	}

	var calls []CallSite

	if regionKind == RegionScript {
		sp := newScriptParser(body, string(pr.URI), funcStart, pr.Resolvers)
		sp.baseOffset = start
		sp.resolverSet = pr.resolverSet
		sp.extractCalls = true
		sp.parse()
		calls = append(calls, sp.calls...)

		for _, c := range sp.funcCalls {
			calls = append(calls, c...)
		}
	} else {
		tp := newTagParser(body, string(pr.URI))
		tp.srcOffset = start
		tp.resolvers = pr.Resolvers
		tp.resolverSet = pr.resolverSet
		tp.extractCalls = true
		tp.parse()
		calls = append(calls, tp.calls...)

		for _, c := range tp.funcCalls {
			calls = append(calls, c...)
		}

		for i := range calls {
			calls[i].Line += conv.Uint32(funcStart)
		}
	}

	return calls
}

// callerAtLine returns the enclosing function name for a given line number.
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

func (pr *ParseResult) callerAtLine(lineNum int) string {
	for _, sc := range pr.Scopes {
		if lineNum > sc.Start && lineNum < sc.End {
			return sc.Name
		}
	}

	return ""
}

// scanLineForCalls checks a line for calls to any of pr.findCalls targets.
func (pr *ParseResult) scanLineForCalls(line string, lineNum int, caller string) {
	lower := strings.ToLower(line)
	trimmed := strings.TrimSpace(lower)
	// Skip function definition lines
	if strings.HasPrefix(trimmed, "function ") || strings.Contains(trimmed, " function ") ||
		strings.HasPrefix(trimmed, "<cffunction") {
		return
	}

	for _, target := range pr.findCalls {
		t := strings.ToLower(target)
		if idx := strings.Index(lower, "."+t+"("); idx >= 0 {
			varName, comp := pr.qualifiedTarget(line, idx)

			pr.Calls = append(pr.Calls, CallSite{
				FuncName: target, Component: comp, Variable: varName, Line: conv.Uint32(lineNum), Caller: caller,
				Resolved: comp != "", Text: strings.TrimSpace(line),
			})
		} else if strings.Contains(lower, " "+t+"(") || strings.Contains(lower, "="+t+"(") || strings.HasPrefix(lower, t+"(") {
			pr.Calls = append(pr.Calls, CallSite{
				FuncName: target, Line: conv.Uint32(lineNum), Caller: caller,
				Resolved: false, Text: strings.TrimSpace(line),
			})
		}
	}
}

// qualifiedTarget reads the receiver of a call whose dot is at dot: the
// variable before it and its component, or, when the receiver is itself a
// call, the component a resolver gives that call.
func (pr *ParseResult) qualifiedTarget(line string, dot int) (varName, comp string) {
	varStart := dot - 1
	for varStart >= 0 && isLineIdentByte(line[varStart]) {
		varStart--
	}

	varName = line[varStart+1 : dot]
	comp = pr.resolveVarComponent(varName)

	// If no variable match, try resolving call expression before the dot
	if comp != "" || dot == 0 || line[dot-1] != ')' || len(pr.Resolvers) == 0 {
		return varName, comp
	}

	// Find matching open paren
	depth := 0

	for j := dot - 1; j >= 0; j-- {
		switch line[j] {
		case ')':
			depth++
		case '(':
			depth--
			if depth != 0 {
				continue
			}

			fnStart := j - 1
			for fnStart >= 0 && isLineIdentByte(line[fnStart]) {
				fnStart--
			}

			return varName, ResolveFromCall(line[fnStart+1:dot], pr.Resolvers)
		}
	}

	return varName, comp
}

// isLineIdentByte is the identifier test scanLineForCalls has always used:
// ASCII letters, digits, '_' and '$'.
func isLineIdentByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '$'
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
