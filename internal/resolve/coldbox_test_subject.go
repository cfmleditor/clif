package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// ColdBox's unit-test bases build the component under test from the test's
// own attribute: a component that extends coldbox.system.testing.BaseModelTest
// and says model="coldbox.system.core.events.EventPool" runs with
// `variables.model = mockBox.createMock( annotations.model )`, and
// BaseInterceptorTest does the same for interceptor="…". Every call on model
// in such a test was "has no component ref". It is a mock of that class, so
// MockBox's own methods are accepted on it as on any component.

var coldboxTestBases = map[string]string{
	"coldbox.system.testing.basemodeltest":       "model",
	"coldbox.system.testing.baseinterceptortest": "interceptor",
}

// coldboxTestSubject is the class a ColdBox test base mocks into variable, or "".
func (r *Resolver) coldboxTestSubject(variable string, pr *parser.ParseResult) string {
	name := strings.ToLower(strings.TrimPrefix(strings.ToLower(variable), "variables."))
	if name != "model" && name != "interceptor" || !pr.URI.IsFile() {
		return ""
	}

	path := pr.URI.Path()
	if !strings.EqualFold(filepath.Ext(path), ".cfc") {
		return ""
	}

	if !mentions(pr.Content, name) || !mentions(pr.Content, "component") {
		return ""
	}

	subject, end := componentAttribute(fileTokens(pr), name)
	if subject == "" {
		return ""
	}

	// The attribute itself reads as an assignment to setsName; the file
	// assigning the name anywhere else means it is not the base's mock.
	if setsName(pr.Content[end:], name) {
		return ""
	}

	if base := r.coldboxTestBase(path); coldboxTestBases[base] != name {
		return ""
	}

	return r.ComponentPath(subject, filepath.Dir(path))
}

// componentAttribute is the dotted name the component's declaration gives
// attribute, `component model="a.b.C" {` or `<cfcomponent model="a.b.C">`,
// and the offset just past it; "" when it gives none. The declaration is the
// first component or cfcomponent before any brace.
func componentAttribute(tokens []parser.Token, attribute string) (string, int) {
	i := slices.IndexFunc(tokens, func(t parser.Token) bool {
		return t.Kind == parser.TokLBrace || isWord(t, "component") || isWord(t, "cfcomponent")
	})
	if i < 0 || tokens[i].Kind == parser.TokLBrace {
		return "", 0
	}

	for j := i + 1; j+2 < len(tokens) && tokens[j].Kind != parser.TokLBrace && tokens[j].Kind != parser.TokGT; j++ {
		if !isWord(tokens[j], attribute) || !isAssignAt(tokens, j+1) {
			continue
		}

		value := tokens[j+2]
		if lit, ok := stringLiteral(value); ok && lit != "" && strings.IndexFunc(lit, func(r rune) bool { return r != '.' && !isWordRune(r) }) < 0 {
			return lit, value.Offset + len(value.Value)
		}
	}

	return "", 0
}

// coldboxTestBase is the ColdBox test base path's extends chain reaches,
// lowercased, or "".
func (r *Resolver) coldboxTestBase(path string) string {
	seen := map[string]bool{}

	for depth := 0; path != "" && !seen[path] && depth < 12; depth++ {
		seen[path] = true

		ext, ok := r.extendsOf(path, cfpath.ToURI(path))
		if !ok || ext == "" {
			return ""
		}

		if _, known := coldboxTestBases[strings.ToLower(ext)]; known {
			return strings.ToLower(ext)
		}

		path = r.ComponentPath(ext, filepath.Dir(path))
	}

	return ""
}
