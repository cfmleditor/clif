package parser

import (
	"strings"
)

// mayDeclare is mayDeclareVariable for this file, used only while AllVars has
// not been built, and for at most maxDeclareScans names: each answer is a
// search of the whole file, and the resolver asks Declares of many names in
// one file, so past a few names building AllVars once is cheaper.
func (pr *ParseResult) mayDeclare(name string) bool {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.allVarsDone || len(pr.declareScans) >= maxDeclareScans {
		return true
	}

	key := strings.ToLower(name)
	if may, ok := pr.declareScans[key]; ok {
		return may
	}

	may := mayDeclareVariable(pr.Content, name)
	if pr.declareScans == nil {
		pr.declareScans = map[string]bool{}
	}

	pr.declareScans[key] = may

	return may
}

// maxDeclareScans bounds mayDeclare's whole-file searches per file.
const maxDeclareScans = 4

// mayDeclareVariable reports whether content could declare a variable called
// name: whether the name appears anywhere, as a whole word, other than called.
// Every declaration (an assignment, var, a for-in or catch binding, a tag's
// name="…") has something other than "(" after the name, so a name only ever
// called is declared by nothing, and AllVars, a second whole-file parse, is
// not run to find that out: Declares( "structNew" ) asked it of every file
// with a collection seeded by structNew(), 750 MB over a tassweb scan.
func mayDeclareVariable(content, name string) bool {
	if name == "" {
		return true
	}

	for at := 0; ; {
		i := indexFoldFrom(content, name, at)
		if i < 0 {
			return false
		}

		at = i + len(name)

		if i > 0 && isIdentPart(content[i-1]) || at < len(content) && isIdentPart(content[at]) {
			continue
		}

		j := at
		for j < len(content) && (content[j] == ' ' || content[j] == '\t') {
			j++
		}

		if j >= len(content) || content[j] != '(' {
			return true
		}
	}
}

// IsShallow reports whether pr is a shallow parse (ParseOptions.Shallow):
// signatures and extends, but no properties, refs or arguments.
func (pr *ParseResult) IsShallow() bool { return pr.shallow }

// Declares reports whether the file declares name anywhere, case-insensitively:
// a variable in any scope, a property, a function or a function's argument.
// A name it does not declare is one it expects from elsewhere — its base
// component, an injector, the framework — which is what the resolver needs to
// know before blaming a missing base for a receiver with no component.
func (pr *ParseResult) Declares(name string) bool {
	for i := range pr.Funcs {
		if strings.EqualFold(pr.Funcs[i].Name, name) {
			return true
		}

		for j := range pr.Funcs[i].Arguments {
			if strings.EqualFold(pr.Funcs[i].Arguments[j].Name, name) {
				return true
			}
		}
	}

	for i := range pr.Properties {
		if strings.EqualFold(pr.Properties[i].name, name) {
			return true
		}
	}

	if !pr.mayDeclare(name) {
		return false
	}

	for _, v := range pr.AllVars() {
		if strings.EqualFold(v.Name, name) {
			return true
		}
	}

	return false
}

// A script island or tag continuation has no opening function declaration of
// its own. Carry the enclosing typed arguments without publishing a duplicate
// definition; whole-argument assignments must see the same signature throughout.
func (pr *ParseResult) seedRegionFunction(start int, name string) (int, []FunctionDef) {
	for i := range pr.Funcs {
		fn := &pr.Funcs[i]
		if int(fn.Line) != start || !strings.EqualFold(fn.Name, name) {
			continue
		}

		for _ai := range fn.Arguments {
			arg := &fn.Arguments[_ai]
			if argumentComponentType(arg) != "" {
				return i, []FunctionDef{*fn}
			}
		}

		break
	}

	return -1, nil
}

// An island can begin on the opening tag's own line. The declaration must
// already have been parsed: a first tag region on that line is not a continuation.
func (pr *ParseResult) regionTagScope(scopes []FuncScope, line int) (FuncScope, bool) {
	if scope, ok := enclosingTagScope(scopes, line); ok {
		return scope, true
	}

	for _, scope := range scopes {
		if scope.Start != line {
			continue
		}

		for i := range pr.Funcs {
			fn := &pr.Funcs[i]
			if int(fn.Line) == line && strings.EqualFold(fn.Name, scope.Name) {
				return scope, true
			}
		}
	}

	return FuncScope{}, false
}
