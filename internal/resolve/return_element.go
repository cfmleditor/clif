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
	tagElementReturnRe    = regexp.MustCompile(`(?i)<cfreturn\s+([A-Za-z_]\w*)\s*\[[^\]\n]*\]\s*/?>`)
	anyReturnRe           = regexp.MustCompile(`(?i)\breturn\b|<cfreturn\b`)
	bareReturnRe          = regexp.MustCompile(`(?i)\breturn\s*;|<cfreturn\s*/?>`)
)

func (r *Resolver) elementReturn(fd *parser.FunctionDef) string {
	if !fd.URI.IsFile() {
		return ""
	}

	// Not handlerParse: that parse asks this resolver what functions return,
	// and the parse of this very file is what is asking.
	source := r.sourceInfo(fd.URI.Path())
	if !scriptElementReturnRe.MatchString(source.content) && !tagElementReturnRe.MatchString(source.content) {
		return ""
	}

	pr := parser.Parse(cfpath.ToURI(fd.URI.Path()), source.content)

	scope := pr.FuncScopeAt(int(fd.Line))
	if scope.Start == -1 || !strings.EqualFold(scope.Name, fd.Name) {
		return ""
	}

	text := sourceText(pr.Content, scope.Start, scope.End+1)
	if !mentions(text, "return") {
		return ""
	}

	reads := scriptElementReturnRe.FindAllStringSubmatchIndex(text, -1)
	reads = append(reads, tagElementReturnRe.FindAllStringSubmatchIndex(text, -1)...)

	if len(reads) == 0 || len(reads)+len(bareReturnRe.FindAllStringIndex(text, -1)) != len(anyReturnRe.FindAllStringIndex(text, -1)) {
		return ""
	}

	dir := filepath.Dir(fd.URI.Path())
	component := ""

	for _, m := range reads {
		line := scope.Start + strings.Count(text[:m[0]], "\n")

		element := r.elementOf(text[m[2]:m[3]], line, scope.Start, fd.Name, pr, dir, 0, "")
		if element == "" || strings.HasPrefix(element, "$") || component != "" && element != component {
			return ""
		}

		component = element
	}

	return component
}
