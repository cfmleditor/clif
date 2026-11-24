package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

func (p *scriptParser) resolveCall(expr string) string {
	return resolveCallExpr(p.resolverSet, p.resolvers, expr)
}

// resolveCallExpr is the component a call expression returns: what the
// configured resolvers name, else whatever dynamicCall makes of it. Both parsers
// ask it.
func resolveCallExpr(set *ResolverSet, resolvers []Resolver, expr string) string {
	if set != nil {
		if comp := set.Resolve(expr); comp != "" {
			return comp
		}

		return dynamicCall(expr)
	}

	if comp := ResolveFromCall(expr, resolvers); comp != "" {
		return comp
	}

	return dynamicCall(expr)
}

// dynamicCall is the component a call returns when no componentResolver
// said and the value is known to be one no component describes: "$any",
// or "" for any other call. A configured resolver is asked first, so a
// project can still type either.
//
//   - A MockBox mock — createMock, createEmptyMock, prepareMock and
//     createStub, whatever they are called on. Its methods are added at
//     runtime ($(), $property()), so typing it as the mocked component would
//     report those missing; left untyped, every call on one was "no
//     component ref", about 2,600 over the six-project corpus.
//   - A Java object with no stub to check it against, createObject("java",
//     …): a javaStubsPath resolver types it when one is configured.
func dynamicCall(expr string) string {
	name, args := finalCall(expr)

	var buf foldScratch
	switch string(buf.lowerFold(name)) {
	case "createmock", "createemptymock":
		return mockOf(mockClassArg(args))
	case "preparemock", "createstub":
		return "$any"
	case "createobject":
		if args = strings.TrimLeft(args, " \t"); args != "" && (args[0] == '"' || args[0] == '\'') &&
			lex.HasPrefixFold(args[1:], "java") && len(args) > 5 && args[5] == args[0] {
			sc := NewScanner(args)

			first := sc.NextSkipComments()
			if first.Kind == TokString && sc.NextSkipComments().Kind == TokComma {
				class := sc.NextSkipComments()
				if class.Kind == TokString && (sc.PeekSkipComments().Kind == TokComma || sc.PeekSkipComments().Kind == TokRParen || sc.PeekSkipComments().Kind == TokEOF) {
					if native := NativeJavaComponent(unquote(class.Value)); native != "" {
						return native
					}
				}
			}

			return "$any"
		}
	}

	return ""
}

// MockCall is what a call expression makes when it is a MockBox mock: the
// mocked class behind MockPrefix, "$any" for a mock of no class it names, or
// "" for any other call.
func MockCall(expr string) string {
	name, _ := finalCall(expr)

	var buf foldScratch
	switch string(buf.lowerFold(name)) {
	case "createmock", "createemptymock", "preparemock", "createstub":
		return dynamicCall(expr)
	}

	return ""
}

// docReturn is the dotted component a doc comment's @return names, or "".
// A bare word is far more often a type than a component, and is left out.
func docReturn(comment string) string {
	_, after, ok := strings.Cut(comment, "@return")
	if !ok {
		return ""
	}

	rest := after
	if strings.HasPrefix(rest, "s ") || strings.HasPrefix(rest, "s\t") {
		rest = rest[1:] // @returns
	}

	if rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return ""
	}

	rest = strings.TrimLeft(rest, " \t")

	end := 0
	for end < len(rest) && (isIdentPart(rest[end]) || rest[end] == '.') {
		end++
	}

	word := strings.Trim(rest[:end], ".")
	if !strings.Contains(word, ".") {
		line, _, _ := strings.Cut(rest, "\n")
		found := ""

		for field := range strings.FieldsSeq(line) {
			candidate := strings.Trim(field, "`.,;()")
			if NativeJavaComponent(candidate) != "" {
				if found != "" && found != candidate {
					return ""
				}

				found = candidate
			}
		}

		return found
	}

	return word
}

// MockPrefix marks a component a MockBox mock was made from:
// `$mock:models.User`. The resolver checks calls against the class when it
// resolves and takes them as dynamic when it does not — a spec mocking a
// class of a module the workspace lacks is not a finding — and the methods
// MockBox adds are accepted on any component.
const MockPrefix = "$mock:"

// ApplicationBase marks an implicit base that is the governing Application.cfc
// when that file is the framework instance, and otherwise the component after
// the marker: FW/1 includes a view inside the framework object, and an
// Application.cfc extending the framework is that object. config's FW/1 preset
// writes it and the resolver reads it; the parser only holds the spelling.
const ApplicationBase = "$application:"

// MuraAdminBase requests the recognized embedded admin renderer owner only.
const MuraAdminBase = ApplicationBase + "$mura-admin"

// mockDecorations are the methods MockBox's decorateMock adds to an object it
// mocks (TestBox system/MockBox.cfc). Each returns the mock it is called on,
// except the ones that report on it ($count, $callLog, …), whose result no
// chain goes on from. Every one starts with $, which no component's own
// method conventionally does.
var mockDecorations = map[string]bool{
	"$": true, "$spy": true, "$property": true, "$getproperty": true,
	"$results": true, "$throws": true, "$callback": true, "$args": true,
	"$calllog": true, "$count": true, "$times": true, "$never": true,
	"$verifycallcount": true, "$atleast": true, "$once": true, "$atmost": true,
	"$debug": true, "$reset": true,
}

// mockReports are the decorations that report on a mock rather than return
// it: $callLog() is a struct of calls by method name, $count() a number.
var mockReports = map[string]bool{
	"$getproperty": true, "$calllog": true, "$count": true, "$times": true, "$never": true,
	"$verifycallcount": true, "$atleast": true, "$once": true, "$atmost": true, "$debug": true,
}

// IsMockReport reports whether name is a decoration that returns something
// other than the mock, so a chain going on from it is on a value no
// component describes: `mock.$callLog().deleteVar.len()`.
func IsMockReport(name string) bool {
	var buf foldScratch

	return mockReports[string(buf.lowerFold(name))]
}

// IsMockDecoration reports whether name is a method MockBox adds to a mock.
func IsMockDecoration(name string) bool {
	if !strings.HasPrefix(name, "$") {
		return false
	}

	var buf foldScratch

	return mockDecorations[string(buf.lowerFold(name))]
}

// mockOf is the component a mock of class is, or $any without one.
func mockOf(class string) string {
	if class == "" || strings.ContainsAny(class, "#$ ") {
		return "$any"
	}

	return MockPrefix + class
}

// mockClassArg is the class a createMock or createEmptyMock argument list
// names: its first argument when that is a string, or className="…".
func mockClassArg(args string) string {
	args = strings.TrimSpace(args)
	if len(args) > 1 && (args[0] == '"' || args[0] == '\'') {
		if end := strings.IndexByte(args[1:], args[0]); end > 0 {
			return args[1 : 1+end]
		}
	}

	return mockClassName(args)
}

// finalCall is the name of the last call in expr and the text after its
// `(`: `getMockBox().createEmptyMock("x")` gives createEmptyMock and `"x")`.
// A `(` inside another call's arguments is not the final call's.
func finalCall(expr string) (name, args string) {
	depth, open := 0, -1

	for i := 0; i < len(expr); i++ {
		switch expr[i] {
		case '(':
			if depth == 0 {
				open = i
			}

			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '"', '\'':
			// Skip a string, whose parentheses are text.
			if end := strings.IndexByte(expr[i+1:], expr[i]); end >= 0 {
				i += end + 1
			}
		}
	}

	if open < 0 {
		return "", ""
	}

	start := open
	for start > 0 && isIdentPart(expr[start-1]) {
		start--
	}

	return expr[start:open], expr[open+1:]
}
