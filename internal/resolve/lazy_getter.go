package resolve

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// A lazy getter loads its field the first time it is asked:
//
//	<cfset variables.instance.extensionManager = ""/>
//	…
//	<cffunction name="getClassExtensionManager">
//		<cfif not isObject(variables.instance.extensionManager)>
//			<cfset loadClassExtensionManager()/>
//		</cfif>
//		<cfreturn variables.instance.extensionManager />
//	</cffunction>
//
// The field is written twice, a "" placeholder and a component, so it has no
// single type, and the getter none either; Mura's configBean hands out its
// class extension manager this way, and 64 admin calls on it were untyped. The
// guard is what settles it: what the getter returns is never the placeholder.
//
// lazyGetterReturn answers for a function whose whole body is that guard and
// that return, in script or in tags: the component every other write of the
// field in the file creates in place. A write of anything else withholds the
// answer.

// lazyGetterReturn is the component fd returns when it is a lazy getter, or "".
// It reads the file's text, not a parse of it: a parse asks for return types,
// which is how this was reached.
func (r *Resolver) lazyGetterReturn(fd *parser.FunctionDef) string {
	path := fd.URI.Path()
	o := r.owner()
	key := pathKey(path) + "\x00" + strings.ToLower(fd.Name)

	o.mu.RLock()
	answer, ok := o.lazyGetters[key]
	guarded, scanned := o.lazyFiles[pathKey(path)]
	o.mu.RUnlock()

	if ok {
		return answer
	}

	// Asked about every untyped function, so a file with no isObject()
	// guard at all is remembered as such and not read again.
	if scanned && !guarded {
		return ""
	}

	text, err := r.readText(path)
	if err != nil {
		return ""
	}

	content := text
	guarded = mentions(content, "isobject(")

	if guarded {
		if field := lazyField(functionBodyNamed(content, r.functionSpansOf(path, content), fd.Name)); field != "" {
			answer = r.fieldWrites(content, field, filepath.Dir(path))
		}
	}

	o.mu.Lock()
	if o.lazyGetters == nil {
		o.lazyGetters, o.lazyFiles = map[string]string{}, map[string]bool{}
	}

	o.lazyFiles[pathKey(path)] = guarded
	o.lazyGetters[key] = answer
	o.mu.Unlock()

	return answer
}

// functionBodyNamed is the body of the function name the parse found in
// content (spans, its FunctionSpans), trimmed: the text between a script
// function's braces, or between its <cffunction> tag and </cffunction>
// with its <cfargument> tags removed; "" when content declares not exactly
// one function of that name, or one with no body.
func functionBodyNamed(content string, spans []parser.FunctionSpan, name string) string {
	s, ok := parser.FunctionSpanNamed(spans, name)
	if !ok || s.Body < 0 {
		return ""
	}

	body := s.BodyText(content)
	if s.Kind == parser.FunctionTag {
		body = parser.WithoutCFTags(body, "cfargument")
	}

	return strings.TrimSpace(body)
}

// lazyField is the field body guards and returns, without a variables.
// prefix, when body is only that guard and that return. The body is read as
// the parse reads it, its tags by parser.CFTags and its script by the
// scanner's tokens, so a comment is nothing and a string is one token: a
// commented-out return or <cfif> is not one, nor the word inside a string.
func lazyField(body string) string {
	guarded, returned, ok := lazyTagShape(body)
	if !ok {
		guarded, returned, ok = lazyScriptShape(body)
	}

	if !ok {
		return ""
	}

	guarded, returned = stripVariables(guarded), stripVariables(returned)
	if guarded == "" || !strings.EqualFold(guarded, returned) {
		return ""
	}

	return guarded
}

// lazyTagShape reads body as `<cfif not isObject( X )> … </cfif>
// <cfreturn Y>` and nothing else but comments, answering X and Y. The guarded
// block only loads the field: one that returns, or opens another <cfif>, is
// not this shape.
func lazyTagShape(body string) (guarded, returned string, ok bool) {
	tags := parser.CFTags(body)
	if len(tags) < 3 {
		return "", "", false
	}

	open, closing, ret := tags[0], tags[len(tags)-2], tags[len(tags)-1]
	if open.Name != "cfif" || open.Close || closing.Name != "cfif" || !closing.Close || ret.Name != "cfreturn" || ret.Close {
		return "", "", false
	}

	if !onlyComments(body[:open.Start]) || !onlyComments(body[closing.End:ret.Start]) || !onlyComments(body[ret.End:]) {
		return "", "", false
	}

	for _, t := range tags[1 : len(tags)-2] {
		if t.Name == "cfif" || t.Name == "cfreturn" {
			return "", "", false
		}
	}

	if holdsWord(significantTokens(body[open.End:closing.Start]), "return") {
		return "", "", false
	}

	guarded, ok = notIsObject(significantTokens(open.Body()))
	if !ok {
		return "", "", false
	}

	returned, ok = soleName(significantTokens(ret.Body()))

	return guarded, returned, ok
}

// lazyScriptShape reads body as `if ( !isObject( X ) ) { … } return Y;`,
// answering X and Y. The guarded block holds no braces and no return.
func lazyScriptShape(body string) (guarded, returned string, ok bool) {
	tokens := significantTokens(body)
	if len(tokens) < 4 || !isWord(tokens[0], "if") || tokens[1].Kind != parser.TokLParen {
		return "", "", false
	}

	condEnd := producerGroupEnd(tokens, 1, parser.TokLParen, parser.TokRParen)
	if condEnd < 0 || condEnd+1 >= len(tokens) || tokens[condEnd+1].Kind != parser.TokLBrace {
		return "", "", false
	}

	guarded, ok = notIsObject(tokens[2:condEnd])
	if !ok {
		return "", "", false
	}

	blockEnd := -1

	for i := condEnd + 2; i < len(tokens); i++ {
		if tokens[i].Kind == parser.TokRBrace {
			blockEnd = i

			break
		}

		if tokens[i].Kind == parser.TokLBrace || isWord(tokens[i], "return") {
			return "", "", false
		}
	}

	rest := tokens[blockEnd+1:]
	if blockEnd < 0 || len(rest) < 3 || !isWord(rest[0], "return") || rest[len(rest)-1].Kind != parser.TokSemicolon {
		return "", "", false
	}

	returned, ok = soleName(rest[1 : len(rest)-1])

	return guarded, returned, ok
}

// notIsObject reads a condition `not isObject( X )` or `!isObject( X )`,
// answering the dotted name X.
func notIsObject(tokens []parser.Token) (string, bool) {
	if len(tokens) == 0 || !isWord(tokens[0], "not") && tokens[0].Kind != parser.TokBang {
		return "", false
	}

	tokens = tokens[1:]
	if len(tokens) < 3 || !isWord(tokens[0], "isObject") || tokens[1].Kind != parser.TokLParen || tokens[len(tokens)-1].Kind != parser.TokRParen {
		return "", false
	}

	return soleName(tokens[2 : len(tokens)-1])
}

// soleName is the dotted name tokens are, when they are nothing else.
func soleName(tokens []parser.Token) (string, bool) {
	name, next := dottedAt(tokens, 0)

	return name, name != "" && next == len(tokens)
}

// onlyComments reports whether text holds no code: whitespace and comments.
func onlyComments(text string) bool {
	return strings.TrimSpace(text) == "" || len(significantTokens(text)) == 0
}

// holdsWord reports whether any of tokens is the identifier word.
func holdsWord(tokens []parser.Token, word string) bool {
	return slices.ContainsFunc(tokens, func(t parser.Token) bool { return isWord(t, word) })
}

func stripVariables(name string) string {
	if len(name) > len("variables.") && strings.EqualFold(name[:len("variables.")], "variables.") {
		return name[len("variables."):]
	}

	return name
}

// fieldWrites is the component every write of field in content agrees on, the ""
// placeholder aside, or "".
func (r *Resolver) fieldWrites(content, field, dir string) string {
	// Anywhere, not only at a statement's start: a write inside a one-line if
	// is a write all the same, and one missed would be one not checked.
	answer := ""

	for _, w := range codeWrites(codeOf(content), field, writeRule{scopes: []string{"variables"}}) {
		rhs := tokensText(content, w.value())
		if rhs == "" || isEmptyLiteral(rhs) {
			continue
		}

		comp := r.writtenComponent(rhs, dir)
		if comp == "" || strings.HasPrefix(comp, "$") || answer != "" && !strings.EqualFold(answer, comp) {
			return ""
		}

		answer = comp
	}

	return answer
}

// writtenComponent types rhs, written at line of pr: a component created in
// place, with an init() on it.
func (r *Resolver) writtenComponent(rhs string, dir string) string {
	if path := createdOnly(rhs); path != "" {
		return r.ComponentPath(path, dir)
	}

	// Anything else is not read: typing a call re-enters return inference,
	// which is what asked.
	return ""
}

// createdOnly is the component rhs creates, when that is all it does: `new
// a.b.C( … )` or `createObject( "component", "a.b.C" )`, alone or with an
// init() on it, which returns the instance by the convention every CFML
// constructor follows.
func createdOnly(rhs string) string {
	tokens := significantTokens(rhs)

	if n := len(tokens); n > 4 && tokens[n-1].Kind == parser.TokRParen {
		if open := lastCallOpen(tokens); open > 2 && tokens[open-2].Kind == parser.TokDot && strings.EqualFold(tokens[open-1].Value, "init") {
			tokens = tokens[:open-2]
		}
	}

	return instantiated(tokens)
}
