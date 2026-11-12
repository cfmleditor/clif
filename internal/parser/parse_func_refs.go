// The component refs and links of a function body, and the return types its calls settle.

package parser

import (
	"slices"
)

// FuncComponentRefs returns cached component refs for a function scope.
//
// A ref a closure in the function declared is left out: it exists only inside
// the closure. FuncComponentRefsAt includes the ones in force at a line.
func (pr *ParseResult) FuncComponentRefs(funcStart, funcEnd int) []ComponentRef {
	refs, _ := pr.cachedFuncRefs(funcStart, funcEnd)

	scoped := 0

	for i := range refs {
		if refs[i].VisibleTo != 0 || refs[i].HasClosureSpan {
			scoped++
		}
	}

	if scoped == 0 {
		return refs
	}

	wide := make([]ComponentRef, 0, len(refs)-scoped)

	for i := range refs {
		if refs[i].VisibleTo == 0 && !refs[i].HasClosureSpan {
			wide = append(wide, refs[i])
		}
	}

	return wide
}

// FuncComponentRefsAt returns the function's refs in force at line: the ones
// the whole function sees, and the ones declared by a closure holding line.
// The slice is the parse's own storage when no closure declared anything, so
// callers must not write to it.
func (pr *ParseResult) FuncComponentRefsAt(funcStart, funcEnd int, line uint32) []ComponentRef {
	refs, _ := pr.cachedFuncRefs(funcStart, funcEnd)

	for i := range refs {
		if !refs[i].VisibleAt(line) {
			out := slices.Clone(refs[:i])

			for j := i + 1; j < len(refs); j++ {
				if refs[j].VisibleAt(line) {
					out = append(out, refs[j])
				}
			}

			return out
		}
	}

	return refs
}

// FuncRefs returns cached component refs and document links for a function body.
func (pr *ParseResult) FuncRefs(funcStart, funcEnd int) ([]ComponentRef, []DocumentLink) {
	return pr.cachedFuncRefs(funcStart, funcEnd)
}

// TopLevelClosureRefs returns the full parse's anonymous-function bindings.
// They are keyed separately from named functions and never become file globals.
func (pr *ParseResult) TopLevelClosureRefs() []ComponentRef {
	pr.funcRefsMu.Lock()
	defer pr.funcRefsMu.Unlock()

	return slices.Clone(pr.funcRefsMap[""])
}

// FuncLinks returns document links for a function body from the parse-time cache.
func (pr *ParseResult) FuncLinks(funcStart, funcEnd int) []DocumentLink {
	key := funcKey(funcStart, funcEnd)

	pr.funcRefsMu.Lock()
	defer pr.funcRefsMu.Unlock()

	if pr.funcLinksMap != nil {
		return pr.funcLinksMap[key]
	}

	return nil
}

// cachedFuncRefs is the refs and links the parse filed under the function at
// [start, end]: those of its walk, typed by its passes, or, where it did not
// type them (Shallow) or an edit has changed the body since, those of its walk
// of that body alone. A function with none has none: an assignment that is
// not the function's own (`y = …` without var) is filed for the file, and a
// function nested in this one has its own.
func (pr *ParseResult) cachedFuncRefs(funcStart, funcEnd int) ([]ComponentRef, []DocumentLink) {
	b := &pr.bodies
	b.mu.Lock()
	pr.readBody(FuncScope{Start: funcStart, End: funcEnd}, true)
	b.mu.Unlock()

	key := funcKey(funcStart, funcEnd)

	pr.funcRefsMu.Lock()
	defer pr.funcRefsMu.Unlock()

	refs := pr.funcRefsMap[key]

	// A closure written outside every function, on the function's lines (a
	// one-line template), is the function's by line, as FuncVars counts its
	// declarations: the resolver reads the function's refs for any line in
	// it. Each carries its closure's span, so it is in force only inside it.
	if closures := pr.funcRefsMap[""]; len(closures) > 0 {
		refs = slices.Clip(refs)

		for i := range closures {
			if line := int(closures[i].Line); line >= funcStart && line <= funcEnd {
				refs = append(refs, closures[i])
			}
		}
	}

	return refs, pr.funcLinksMap[key]
}

func isIdentifier(s string) bool {
	for i, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '$' {
			continue
		}

		if i > 0 && c >= '0' && c <= '9' {
			continue
		}

		return false
	}

	return s != ""
}

// isValidVarChain returns true if s looks like a valid CFML variable chain
// (e.g. "VARIABLES.prs", "obj", "result[1].data"). Rejects strings containing
// operators, quotes, hash signs, or whitespace.
func isValidVarChain(s string) bool {
	if s == "" {
		return false
	}

	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '$' || c >= '0' && c <= '9' || c == '.' || c == '[' || c == ']' {
			continue
		}

		return false
	}

	return true
}
