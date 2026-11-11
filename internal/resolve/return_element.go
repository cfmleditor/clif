package resolve

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// A function that returns one element of a collection it holds, as Galaxie
// Blog's loadOrder does,
//
//	var found = ormExecuteQuery( "FROM ShopOrder WHERE …", { … } );
//	if ( arrayLen( found ) ) { return found[ 1 ]; }
//
// returns what the collection's elements are. The parse types `return x` by
// what x holds and has nothing for `return x[ n ]`, and the producer plans
// read an element only of an array the function filled itself.
//
// Only a function every one of whose returns is such a read, of collections
// whose elements are all the same component, answers. A bare `return;` returns
// nothing and does not count against it, since a caller of a function that may
// find nothing tests for that (isNull, arrayLen).
var (
	scriptElementReturnRe = regexp.MustCompile(`(?i)\breturn\s+([A-Za-z_]\w*)\s*\[[^\]\n]*\]\s*;`)
	anyReturnRe           = regexp.MustCompile(`(?i)\breturn\b`)
	bareReturnRe          = regexp.MustCompile(`(?i)\breturn\s*;`)
	// elementReadRe is a <cfreturn>'s expression when it reads an element,
	// the tags themselves read with parser.CFTags.
	elementReadRe = regexp.MustCompile(`^([A-Za-z_]\w*)\s*\[[^\]\n]*\]$`)
)

// holdsElementReturn reports whether the source has a return that reads an
// element of a collection anywhere in it. It is asked for every function of the
// file whose return type is looked up, and two expressions over the whole file
// were a quarter of a MasaCMS scan, so the answer is read once per source.
func (s *sourceInfo) holdsElementReturn() bool {
	if s.derived == nil {
		return scriptElementReturnRe.MatchString(s.content) || len(tagReturns(s.content).reads) > 0
	}

	s.derived.elementOnce.Do(func() {
		s.derived.elementReads = scriptElementReturnRe.MatchString(s.content) || len(tagReturns(s.content).reads) > 0
	})

	return s.derived.elementReads
}

// plainParse is the source parsed with no resolver, lookup or option: a
// function of its text alone, so it is the same for every caller and is kept
// with the source. A parse is safe to share (its lazy parts are locked), and
// every function of the file whose return is looked up asked for a fresh one.
func (s *sourceInfo) plainParse(path string) *parser.ParseResult {
	if s.derived == nil {
		return parser.Parse(cfpath.ToURI(path), s.content)
	}

	s.derived.plainOnce.Do(func() { s.derived.plain = parser.Parse(cfpath.ToURI(path), s.content) })

	return s.derived.plain
}

// plainParseOf is file's plain parse (sourceInfo.plainParse) when content is
// the text its source holds, as it is for a file read just now; a fresh one
// otherwise. A caller must not change what it is given: the parse is shared.
func (r *Resolver) plainParseOf(file, content string) *parser.ParseResult {
	if source := r.sourceInfo(file); source.derived != nil && source.content == content {
		return source.plainParse(file)
	}

	return parser.Parse(cfpath.ToURI(file), content)
}

func (r *Resolver) elementReturn(fd *parser.FunctionDef) string {
	if !fd.URI.IsFile() {
		return ""
	}

	// Not handlerParse: that parse asks this resolver what functions return,
	// and the parse of this very file is what is asking.
	source := r.sourceInfo(fd.URI.Path())
	if !source.holdsElementReturn() {
		return ""
	}

	pr := source.plainParse(fd.URI.Path())

	scope := pr.FuncScopeAt(int(fd.Line))
	if scope.Start == -1 || !strings.EqualFold(scope.Name, fd.Name) {
		return ""
	}

	text := sourceText(pr.Content, scope.Start, scope.End+1)
	if !mentions(text, "return") {
		return ""
	}

	tags := tagReturns(text)

	reads := tags.reads
	for _, m := range scriptElementReturnRe.FindAllStringSubmatchIndex(text, -1) {
		reads = append(reads, elementRead{at: m[0], name: text[m[2]:m[3]]})
	}

	bare := len(bareReturnRe.FindAllStringIndex(text, -1)) + tags.bare
	if len(reads) == 0 || len(reads)+bare != len(anyReturnRe.FindAllStringIndex(text, -1))+tags.all {
		return ""
	}

	dir := filepath.Dir(fd.URI.Path())
	component := ""
	lines := parser.NewLineTable(text)

	for _, m := range reads {
		line := scope.Start + lines.LineOf(m.at)

		element := r.elementOf(m.name, line, scope.Start, fd.Name, pr, dir, 0, "")
		if element == "" || strings.HasPrefix(element, "$") || component != "" && element != component {
			return ""
		}

		component = element
	}

	return component
}

// elementRead is a return that reads an element: where it starts, and the
// collection it reads.
type elementRead struct {
	at   int
	name string
}

// tagReturnFacts is what the <cfreturn> tags of a text return: the element
// reads, how many return nothing, and how many there are.
type tagReturnFacts struct {
	reads     []elementRead
	bare, all int
}

// tagReturns reads text's <cfreturn> tags with parser.CFTags: a tag in a
// comment is none, and a '>' inside the expression's quotes does not end it.
func tagReturns(text string) tagReturnFacts {
	var facts tagReturnFacts

	if !mentions(text, "<cfreturn") {
		return facts
	}

	tags := parser.CFTags(text, "cfreturn")
	for i := range tags {
		if tags[i].Close {
			continue
		}

		facts.all++

		body := tags[i].Body()
		if body == "" {
			facts.bare++

			continue
		}

		if m := elementReadRe.FindStringSubmatch(body); m != nil {
			facts.reads = append(facts.reads, elementRead{at: tags[i].Start, name: m[1]})
		}
	}

	return facts
}
