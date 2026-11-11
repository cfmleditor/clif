package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
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
// The returns are read from the scanner's tokens (scriptReturns) and the
// parse's tags (tagReturns): a return in a comment or a string, or the word in
// a hint, is none.

// holdsElementReturn reports whether the source has a return that reads an
// element of a collection anywhere in it. It is asked for every function of the
// file whose return type is looked up, and two expressions over the whole file
// were a quarter of a MasaCMS scan, so the answer is read once per source.
func (s *sourceInfo) holdsElementReturn() bool {
	if s.derived == nil {
		return holdsElementRead(s.content)
	}

	s.derived.elementOnce.Do(func() {
		s.derived.elementReads = holdsElementRead(s.content)
	})

	return s.derived.elementReads
}

// holdsElementRead reports whether content has a script or a tag return that
// reads an element.
func holdsElementRead(content string) bool {
	if !mentions(content, "return") {
		return false
	}

	return len(scriptReturns(textCode(content)[0]).reads) > 0 || len(tagReturns(content).reads) > 0
}

// plainParse is the source parsed with no resolver, lookup or option but the
// framework readers' calls (plainParseText): a function of its text alone, so
// it is the same for every caller and is kept with the source. A parse is
// safe to share (its lazy parts are locked), and every function of the file
// whose return is looked up asked for a fresh one.
func (s *sourceInfo) plainParse(path string) *parser.ParseResult {
	if s.derived == nil {
		return plainParseText(path, s.content)
	}

	s.derived.plainOnce.Do(func() { s.derived.plain = plainParseText(path, s.content) })

	return s.derived.plain
}

// plainParseOf is file's plain parse (sourceInfo.plainParse) when content is
// the text its source holds, as it is for a file read just now; a fresh one
// otherwise. A caller must not change what it is given: the parse is shared.
func (r *Resolver) plainParseOf(file, content string) *parser.ParseResult {
	if source := r.sourceInfo(file); source.derived != nil && source.content == content {
		return source.plainParse(file)
	}

	return plainParseText(file, content)
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
	script := scriptReturns(significantTokens(text))

	reads := slices.Concat(tags.reads, script.reads)
	if len(reads) == 0 || len(reads)+tags.bare+script.bare != tags.all+script.all {
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

// returnFacts is what the returns of a text return: the element reads, how
// many return nothing, and how many there are.
type returnFacts struct {
	reads     []elementRead
	bare, all int
}

// tagReturns reads text's <cfreturn> tags with parser.CFTags: a tag in a
// comment is none, and a '>' inside the expression's quotes does not end it.
func tagReturns(text string) returnFacts {
	var facts returnFacts

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

		tokens := significantTokens(body)
		if name, next, ok := elementReadAt(tokens, 0); ok && next == len(tokens) {
			facts.reads = append(facts.reads, elementRead{at: tags[i].Start, name: name})
		}
	}

	return facts
}

// scriptReturns reads the return statements in tokens, a text's code with no
// newlines: `return x[ i ];` is an element read, `return;` returns nothing.
// A return written in a comment or a string is none, and neither is a member
// named return (`x.return`). A <cfreturn> tag is the identifier cfreturn, so
// it is not counted here.
func scriptReturns(tokens []parser.Token) returnFacts {
	var facts returnFacts

	for i, t := range tokens {
		if !isWord(t, "return") || i > 0 && tokens[i-1].Kind == parser.TokDot {
			continue
		}

		facts.all++

		if i+1 < len(tokens) && tokens[i+1].Kind == parser.TokSemicolon {
			facts.bare++

			continue
		}

		if name, next, ok := elementReadAt(tokens, i+1); ok && next < len(tokens) && tokens[next].Kind == parser.TokSemicolon {
			facts.reads = append(facts.reads, elementRead{at: t.Offset, name: name})
		}
	}

	return facts
}

// elementReadAt reads `name[ … ]` at tokens[i]: a plain name and one index,
// and the position after the index's closing bracket.
func elementReadAt(tokens []parser.Token, i int) (string, int, bool) {
	if i+1 >= len(tokens) || tokens[i].Kind != parser.TokIdent || tokens[i+1].Kind != parser.TokLBracket {
		return "", 0, false
	}

	closing := producerGroupEnd(tokens, i+1, parser.TokLBracket, parser.TokRBracket)
	if closing < 0 {
		return "", 0, false
	}

	return tokens[i].Value, closing + 1, true
}
