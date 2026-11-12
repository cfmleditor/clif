// The variables of a parse by scope, per function and for the file.

package parser

import (
	"slices"
	"strings"
)

// GlobalVars returns this.x and variables.x names declared outside any function.
func (pr *ParseResult) GlobalVars() []string {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.scopedVarsLocked()

	return pr.globalVars
}

// VariablesVars returns variables-scoped names from outside functions.
func (pr *ParseResult) VariablesVars() []string {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.scopedVarsLocked()

	return pr.variablesVars
}

// ThisVars returns this-scoped property names from outside functions.
func (pr *ParseResult) ThisVars() []string {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.scopedVarsLocked()

	return pr.thisVars
}

// AllVars returns every variable declaration in the file, with its scope, line,
// and enclosing function range.
//
// Memoised behind the same lock and invalidated by the same edit paths as
// VariablesVars and ThisVars, because it is the same scan: go-to-definition on a
// variable asks for it once per request, and a request is cheap only if the
// answer is not a fresh parse of the whole document every time. The siblings
// return names; this returns the declarations, which is what a caller needing a
// line to jump to requires.
func (pr *ParseResult) AllVars() []VarDef {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if !pr.allVarsDone {
		// A parse that did not keep its declarations (ParseOptions.KeepVars),
		// or whose content an edit has changed since, is read again by the
		// one walk, shallow, as ParseVars reads a file: split as this parse
		// splits it, so a markup template is no script region.
		pr.allVars = ParseWithOptions(pr.URI, pr.Content, &ParseOptions{
			Shallow:            true,
			KeepVars:           true,
			InterpolateAllText: pr.interpolateAll,
		}).allVars
		pr.allVarsDone = true
	}

	return pr.allVars
}

// FuncVars returns the local and arguments names declared on the lines of the
// function at [start, end], nested functions and closures included: the
// parse's walk's declarations there, or, where an edit has changed the body
// since, its walk of that body alone. Results are cached; InvalidateFunc
// marks a body changed.
func (pr *ParseResult) FuncVars(funcStart, funcEnd int) []string {
	key := funcKey(funcStart, funcEnd)

	pr.funcVarsMu.Lock()
	if cached, ok := pr.funcVars[key]; ok {
		pr.funcVarsMu.Unlock()

		return cached
	}
	pr.funcVarsMu.Unlock()

	b := &pr.bodies
	b.mu.Lock()
	pr.readBody(FuncScope{Start: funcStart, End: funcEnd}, false)
	vars := b.funcVarNames(funcStart, funcEnd)
	b.mu.Unlock()

	pr.funcVarsMu.Lock()
	if pr.funcVars == nil {
		pr.funcVars = make(map[fnKey][]string)
	}

	pr.funcVars[key] = vars
	pr.funcVarsMu.Unlock()

	return vars
}

type funcVarSet struct {
	from  *string
	n     int
	names map[string]struct{}
}

// FuncDeclares reports whether FuncVars(funcStart, funcEnd) holds name,
// case-insensitively. Asked for every call, a walk of a function's every
// variable was quadratic in a function that declares as often as it calls.
func (pr *ParseResult) FuncDeclares(funcStart, funcEnd int, name string) bool {
	vars := pr.FuncVars(funcStart, funcEnd)
	if len(vars) < maxScopeScan || !isASCII(name) {
		return slices.ContainsFunc(vars, func(v string) bool { return strings.EqualFold(v, name) })
	}

	key := funcKey(funcStart, funcEnd)

	pr.funcVarsMu.Lock()
	set := pr.funcVarSets[key]

	if set.from != &vars[0] || set.n != len(vars) {
		set = funcVarSet{from: &vars[0], n: len(vars), names: make(map[string]struct{}, len(vars))}

		for _, v := range vars {
			// A name EqualFold matches without lowering alike.
			if !isASCII(v) {
				set.names = nil

				break
			}

			set.names[strings.ToLower(v)] = struct{}{}
		}

		if pr.funcVarSets == nil {
			pr.funcVarSets = map[fnKey]funcVarSet{}
		}

		pr.funcVarSets[key] = set
	}
	pr.funcVarsMu.Unlock()

	if set.names == nil {
		return slices.ContainsFunc(vars, func(v string) bool { return strings.EqualFold(v, name) })
	}

	var buf foldScratch

	_, ok := set.names[string(buf.lowerFold(name))]

	return ok
}

// Memo is build()'s answer for key, computed once per version of this parse.
// It is for a caller that derives something from the parse alone, asked
// again for every call site in it: the answer lives and dies with the parse,
// and an edit forgets it.
func (pr *ParseResult) Memo(key string, build func() any) any {
	pr.memoMu.Lock()
	v, ok := pr.memo[key]
	pr.memoMu.Unlock()

	if ok {
		return v
	}

	v = build()

	pr.memoMu.Lock()
	if pr.memo == nil {
		pr.memo = map[string]any{}
	}

	pr.memo[key] = v
	pr.memoMu.Unlock()

	return v
}

// FuncScopeAt is FindFuncScopeAt(line, pr.Scopes), through an index of the
// scopes built once: asked for every call, a walk of every scope was
// quadratic in a file of thousands of functions.
func (pr *ParseResult) FuncScopeAt(line int) FuncScope {
	scopes := pr.Scopes
	if len(scopes) <= maxScopeScan {
		return findFuncScope(line, scopes)
	}

	pr.scopeAtMu.Lock()
	if pr.scopeAt == nil || pr.scopeAtFrom != &scopes[0] || pr.scopeAtCount != len(scopes) {
		pr.scopeAt, pr.scopeAtFrom, pr.scopeAtCount = scopeIndex(scopes, true), &scopes[0], len(scopes)
	}

	at := pr.scopeAt
	pr.scopeAtMu.Unlock()

	if i := at(line); i >= 0 {
		return scopes[i]
	}

	return FuncScope{Start: -1, End: -1}
}

// AssignsMember reports whether the function holding line assigns
// variable.member on a line before it: `a.getVariables = getVariables;` then
// `a.getVariables()` calls what was stored there, which is no method of a's
// component. Compared case-insensitively, as CFML names are. A member stored
// as `variables.a.m = …` counts from any function and any line.
func (pr *ParseResult) AssignsMember(variable, member string, line uint32) bool {
	// Asked for every call the method check looks at, it scanned every scope
	// for the call's function and every member assignment in the file each
	// time: quadratic in a file with many of both. The assignments are looked
	// up by name, and the scope found only once one names the member.
	key, keyed := noFn, false

	for _, i := range pr.memberSetsNamed(member) {
		m := &pr.memberSets[i]

		if !keyed {
			if s := findFuncScope(int(line), pr.Scopes); s.Start >= 0 {
				key = funcKey(s.Start, s.End)
			}

			keyed = true
		}

		if m.funcKey == key && m.line <= line && strings.EqualFold(m.varName, variable) {
			return true
		}

		// One stored through variables. is on the component's own variable,
		// which every function sees: FW/1's tests set it in setup() and call
		// it, spelled with or without the scope, from each spec.
		if held, ok := cutVariablesScope(m.varName); ok {
			if v, _ := cutVariablesScope(variable); strings.EqualFold(held, v) {
				return true
			}
		}
	}

	return false
}

// memberSetsNamed is the indexes of memberSets assigning member, in order.
func (pr *ParseResult) memberSetsNamed(member string) []int {
	pr.memberSetsMu.Lock()
	defer pr.memberSetsMu.Unlock()

	if pr.memberSetsByName == nil || pr.memberSetsCount != len(pr.memberSets) {
		pr.memberSetsByName = make(map[string][]int, len(pr.memberSets))

		for i := range pr.memberSets {
			name := strings.ToLower(pr.memberSets[i].funcName)
			pr.memberSetsByName[name] = append(pr.memberSetsByName[name], i)
		}

		pr.memberSetsCount = len(pr.memberSets)
	}

	if !isASCII(member) {
		return pr.memberSetsByName[strings.ToLower(member)]
	}

	var buf foldScratch

	return pr.memberSetsByName[string(buf.lowerFold(member))]
}

// RefsNamed is the indexes of the ComponentRefs whose Variable is name,
// case-insensitively, in order. A resolver asking for one variable's refs on
// every call in a file walked them all each time, which is quadratic in a
// file that assigns as often as it calls.
func (pr *ParseResult) RefsNamed(name string) []int {
	pr.refsMu.Lock()
	defer pr.refsMu.Unlock()

	var from *ComponentRef
	if len(pr.ComponentRefs) > 0 {
		from = &pr.ComponentRefs[0]
	}

	if pr.refsByName == nil || pr.refsBuiltFrom != from || pr.refsBuiltLen != len(pr.ComponentRefs) {
		pr.refsByName = make(map[string][]int, len(pr.ComponentRefs))

		for i := range pr.ComponentRefs {
			n := strings.ToLower(pr.ComponentRefs[i].Variable)
			pr.refsByName[n] = append(pr.refsByName[n], i)
		}

		pr.refsBuiltFrom, pr.refsBuiltLen = from, len(pr.ComponentRefs)
	}

	if !isASCII(name) {
		return pr.refsByName[strings.ToLower(name)]
	}

	var buf foldScratch

	return pr.refsByName[string(buf.lowerFold(name))]
}

// cutVariablesScope is name without a leading variables., and whether it had one.
func cutVariablesScope(name string) (string, bool) {
	const scope = "variables."
	if len(name) > len(scope) && strings.EqualFold(name[:len(scope)], scope) {
		return name[len(scope):], true
	}

	return name, false
}

// HasScopedAssignment reports whether name was ever assigned in the given scope
// (ScopeVariables or ScopeThis) anywhere in the file — globally, inside init(), or
// inside any other function body. VARIABLES./THIS.-scoped values assigned inside one
// function remain visible from every other function in the file (unlike a var-scoped
// local), so telling "VARIABLES.someName(...) calls a property holding a function
// reference" apart from "no such property exists" requires checking every function
// body, not just VariablesVars/ThisVars (which only cover outside-function code and
// init()). Results are cached per scope on first call.
func (pr *ParseResult) HasScopedAssignment(scope Scope, name string) bool {
	names := pr.VariablesVars()
	if scope == ScopeThis {
		names = pr.ThisVars()
	}

	for _, n := range names {
		if strings.EqualFold(n, name) {
			return true
		}
	}

	for _, n := range pr.anyScopedNames(scope) {
		if strings.EqualFold(n, name) {
			return true
		}
	}

	return false
}

// anyScopedNames is every name the walk declared in scope (ScopeVariables or
// ScopeThis) inside any function body, in the order it met them, each once,
// with every body an edit changed walked again first.
func (pr *ParseResult) anyScopedNames(scope Scope) []string {
	b := &pr.bodies
	b.mu.Lock()
	defer b.mu.Unlock()

	if names, ok := b.anyScoped[scope]; ok {
		return names
	}

	scopes := slices.Clone(pr.Scopes)
	for _, s := range scopes {
		pr.readBody(s, false)
	}

	inFunc := scopeIndex(scopes, true)
	seen := make(map[string]bool)

	var names []string

	for i := range b.vars {
		v := &b.vars[i]
		if v.Scope == scope && !seen[v.Name] && inFunc(int(v.Line)) >= 0 {
			seen[v.Name] = true

			names = append(names, v.Name)
		}
	}

	if b.anyScoped == nil {
		b.anyScoped = make(map[Scope][]string)
	}

	b.anyScoped[scope] = names

	return names
}

// InvalidateFunc marks the body of the function at [start, end] changed: the
// next question about it, or about a function nested in it, walks it again.
func (pr *ParseResult) InvalidateFunc(funcStart, funcEnd int) {
	pr.staleBody(funcStart, funcEnd)
}

// scopedVarsLocked fills variablesVars, thisVars and globalVars from one scan
// of the file. Each used to scan it for itself, keeping only its own scope, and
// globalVars scanned it twice more: on a 65,000-line component that was 17ms a
// scan, and the completion cache built after every open and save asked for
// two of them. Callers hold pr.mu.
func (pr *ParseResult) scopedVarsLocked() {
	if pr.scopedDone {
		return
	}

	pr.variablesVars, pr.thisVars = pr.computeScopedVars()
	pr.globalVars = append(slices.Clip(pr.variablesVars), pr.thisVars...)
	pr.scopedDone = true
}

// computeScopedVars extracts the variables- and this-scoped names declared
// outside functions and in the init() function body, each in declaration
// order and without repeats. Properties come first among the variables, since
// they default to that scope.
func (pr *ParseResult) computeScopedVars() (variables, this []string) {
	vars := scopeNames{seen: map[string]bool{}}
	thisNames := scopeNames{seen: map[string]bool{}}

	for _, prop := range pr.Properties {
		vars.add(prop.name)
	}

	note := func(v *VarDef) {
		switch v.Scope { //nolint:exhaustive // only these two scopes are collected
		case ScopeVariables:
			vars.add(v.Name)
		case ScopeThis:
			thisNames.add(v.Name)
		}
	}

	b := &pr.bodies
	b.mu.Lock()
	defer b.mu.Unlock()

	initScope := pr.initFuncScope()
	if initScope.Start != -1 {
		pr.readBody(initScope, false)
	}

	inFunc := scopeIndex(pr.Scopes, true)

	// The walk's declarations outside every function. A var or local there
	// is a variables one in a script region; a tag region's <cfset var> there
	// declares nothing.
	for i := range b.vars {
		v := b.vars[i].VarDef

		switch {
		case v.Scope != ScopeVariables && v.Scope != ScopeThis && v.Scope != ScopeLocal,
			v.Scope == ScopeLocal && b.vars[i].tag,
			inFunc(int(v.Line)) >= 0:
			continue
		case v.Scope == ScopeLocal:
			v.Scope = ScopeVariables
		}

		note(&v)
	}

	if initScope.Start != -1 {
		for _, v := range b.varsOn(initScope.Start, initScope.End) {
			note(&v.VarDef)
		}
	}

	return vars.names, thisNames.names
}

// scopeNames collects names in the order first seen.
type scopeNames struct {
	seen  map[string]bool
	names []string
}

func (n *scopeNames) add(name string) {
	if !n.seen[name] {
		n.seen[name] = true
		n.names = append(n.names, name)
	}
}

// initFuncScope returns the FuncScope for the init() function, or {-1,-1} if not found.
func (pr *ParseResult) initFuncScope() FuncScope {
	for i := range pr.Funcs {
		f := &pr.Funcs[i]

		if strings.EqualFold(f.Name, "init") {
			return findFuncScope(int(f.Line), pr.Scopes)
		}
	}

	return FuncScope{Start: -1, End: -1}
}
