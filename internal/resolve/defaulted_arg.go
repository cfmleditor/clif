package resolve

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// defaultedArgumentComponent is the component an argument holds when the
// body guards its absence and constructs what it holds then:
//
//	<cfargument name="keyFactory" type="any">
//	<cfif not structKeyExists(arguments, "keyFactory")>
//		<cfset arguments.keyFactory=createObject("component","mura.publisherKeys").init(...)>
//	</cfif>
//	<cfset arguments.keyFactory.setMode("publish")>
//
// The construction is the argument's component — proven in the body's own
// text, whatever the caller passes.
func (r *Resolver) defaultedArgumentComponent(name string, pr *parser.ParseResult, caller string) string {
	fi := slices.IndexFunc(pr.Funcs, func(f parser.FunctionDef) bool { return strings.EqualFold(f.Name, caller) })
	if fi < 0 {
		return ""
	}

	fd := &pr.Funcs[fi]

	// Asked for every call on an untyped argument, so each function's
	// guarded arguments are found once, for all of its arguments: going
	// through the body per call site tokenised it again each time, which was
	// a fifth of a tassweb scan's CPU and 9GB of its allocation.
	answers, _ := pr.Memo("defaulted\x00"+strconv.Itoa(int(fd.Line))+"\x00"+fd.Name, func() any {
		return r.defaultedArguments(fd, pr)
	}).(map[string]string)

	return answers[strings.ToLower(name)]
}

// defaultedArguments is defaultedFromGuard's answer for each of fd's
// arguments the body guards, by lowercased name.
func (r *Resolver) defaultedArguments(fd *parser.FunctionDef, pr *parser.ParseResult) map[string]string {
	out := map[string]string{}

	// Most functions guard no argument: look for the guard's words before
	// tokenising the body. It is read from pr, the parse at hand; going by
	// the file's path reparsed it whole.
	text, ok := funcScopeText(pr, fd)
	if !ok || parser.IndexFoldFrom(text, "structKeyExists", 0) < 0 {
		return out
	}

	var tokens []parser.Token

	for _, a := range fd.Arguments {
		if !quotesName(text, a.Name) {
			continue
		}

		if tokens == nil {
			tokens = significantTokens(text)
		}

		if comp := r.defaultedFromGuard(tokens, a.Name, pr); comp != "" {
			out[strings.ToLower(a.Name)] = comp
		}
	}

	return out
}

// defaultedFromGuard is the construction the guard on name's absence makes:
// a structKeyExists (arguments, "name") branch that then assigns
// arguments.name a construction.
func (r *Resolver) defaultedFromGuard(tokens []parser.Token, name string, pr *parser.ParseResult) string {
	dir := filepath.Dir(pr.URI.Path())

	for i := 0; i+5 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokIdent || !strings.EqualFold(tokens[i].Value, "structKeyExists") ||
			tokens[i+1].Kind != parser.TokLParen {
			continue
		}

		args, ok := callArgs(tokens, i+1)
		if !ok || len(args) != 2 {
			continue
		}

		if !strings.EqualFold(producerText(args[0]), "arguments") || argLiteral(args[1]) != name {
			continue
		}

		if comp := r.defaultedConstruction(tokens[i+1:], name, dir); comp != "" {
			return comp
		}
	}

	return ""
}

// argLiteral is the bare string literal's unquoted text, or "".
func argLiteral(tokens []parser.Token) string {
	if len(tokens) != 1 || tokens[0].Kind != parser.TokString {
		return ""
	}

	return strings.Trim(tokens[0].Value, "\"'")
}

// defaultedConstruction is the component the body assigns arguments.name
// after the guard: arguments.name = createObject("component",
// literal).init(...), or arguments.name = new X(...).
func (r *Resolver) defaultedConstruction(tokens []parser.Token, name, dir string) string {
	for i := 0; i+3 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokEquals {
			continue
		}

		if recv, ok := forwardedArgsName(tokens, i); !ok || !strings.EqualFold(recv, name) {
			continue
		}

		end := i + 1
		for end < len(tokens) && tokens[end].Kind != parser.TokGT && tokens[end].Kind != parser.TokLT && tokens[end].Kind != parser.TokSemicolon {
			end++
		}

		text := strings.TrimSpace(producerText(tokens[i+1 : end]))

		lower := strings.ToLower(text)

		switch {
		case strings.HasPrefix(lower, "createobject"):
			m := createdComponent.FindStringSubmatch(text)
			if m == nil {
				continue
			}

			if comp := r.componentFromConstruction(m[1], dir); comp != "" {
				return comp
			}
		case strings.HasPrefix(lower, "new "):
			dotted := strings.TrimSpace(strings.SplitN(text[4:], "(", 2)[0])
			if p := r.ComponentPath(dotted, dir); p != "" {
				return p
			}
		}
	}

	return ""
}

// forwardedArgsName is the member of the arguments scope an assignment
// before the equals at i writes: arguments.name — the name, or "".
func forwardedArgsName(tokens []parser.Token, i int) (string, bool) {
	if i < 3 {
		return "", false
	}

	if tokens[i-3].Kind == parser.TokIdent && strings.EqualFold(tokens[i-3].Value, "arguments") &&
		tokens[i-2].Kind == parser.TokDot && tokens[i-1].Kind == parser.TokIdent {
		return tokens[i-1].Value, true
	}

	return "", false
}

// quotesName reports whether text holds name as a string literal, as the
// guard's structKeyExists( arguments, "name" ) must.
func quotesName(text, name string) bool {
	return parser.IndexFoldFrom(text, `"`+name+`"`, 0) >= 0 || parser.IndexFoldFrom(text, "'"+name+"'", 0) >= 0
}
