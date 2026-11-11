package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// Preside's WireBox config gives a service PresideSuperClass's methods
// ($getPresideObject(), $translateResource() and the rest) by virtual
// inheritance, not by extends: config/WireBox.cfc's _injectPresideSuperClass
// calls mapping.virtualInheritance( "presideSuperClass" ) for a component
// whose metadata, or an ancestor's, holds the key presideService. The key's
// value is not read (StructKeyExists), so `@presideService false` counts too.
// A component without it does not get them, whatever directory it is in.

// presideSuperClassFunc is name as PresideSuperClass declares it, when a
// component of chain is a Preside service. Every method it adds but init()
// starts with $, which is what keeps this off every other lookup.
func (r *Resolver) presideSuperClassFunc(chain []string, name string, depth int) *parser.FunctionDef {
	if len(chain) == 0 || r.Index == nil || depth > maxDelegateDepth || !strings.HasPrefix(name, "$") {
		return nil
	}

	super := r.presideSuperClass(chain[0])
	if super == "" {
		return nil
	}

	for _, p := range chain {
		if cfpath.SamePath(p, super) {
			return nil
		}

		if wantsPresideInjection(r.sourceInfo(p).content) {
			return r.lookupFunc(super, name, depth+1)
		}
	}

	return nil
}

// presideSuperClass is the PresideSuperClass.cfc a component's service is
// built with: the checkout the component is part of, or the workspace's only
// one, for a site whose Preside is installed beside it.
func (r *Resolver) presideSuperClass(path string) string {
	roots := r.presideRoots()

	for _, system := range roots {
		if under(path, filepath.Dir(system)) {
			return filepath.Join(system, "services", "PresideSuperClass.cfc")
		}
	}

	if len(roots) == 1 {
		return filepath.Join(roots[0], "services", "PresideSuperClass.cfc")
	}

	return ""
}

// wantsPresideInjection reports whether a component's own header holds the
// presideService key: an attribute of its <cfcomponent> tag, or, in script,
// an attribute before the body's brace or an @presideService in a doc comment
// (`/** … */`) there. Those are where the engine reads a component's metadata
// from, so a comment is read on purpose, but only a doc comment in the
// header: the key in a string, a line or block comment, or the body is none.
func wantsPresideInjection(content string) bool {
	if !mentions(content, "presideService") {
		return false
	}

	if !parser.IsScriptSource(content) {
		for _, tag := range parser.CFTags(content, "cfcomponent") {
			if tag.Close {
				continue
			}

			_, ok := tag.Attr("presideService")

			return ok
		}

		return false
	}

	sc := parser.NewScriptScanner(content)

	var previous parser.Token

	for t := sc.Next(); t.Kind != parser.TokEOF && t.Kind != parser.TokLBrace; t = sc.Next() {
		switch {
		case t.Kind == parser.TokBlockComment && strings.HasPrefix(t.Value, "/**") && docTagged(t.Value, "@presideService"):
			return true
		case t.Kind == parser.TokEquals && isWord(previous, "presideService"):
			return true
		}

		if t.Kind != parser.TokNewline && t.Kind != parser.TokBlockComment && t.Kind != parser.TokLineComment {
			previous = t
		}
	}

	return false
}

// docTagged reports whether the doc comment holds the annotation tag, a word
// on its own, ignoring case.
func docTagged(comment, tag string) bool {
	for rest := comment; ; {
		i := parser.IndexFold(rest, tag)
		if i < 0 {
			return false
		}

		rest = rest[i+len(tag):]
		if rest == "" || !lex.IsWord(rest[0]) {
			return true
		}
	}
}

// under reports whether path lies beneath dir, ignoring case.
func under(path, dir string) bool {
	prefix := dir + string(filepath.Separator)

	return len(path) > len(prefix) && strings.EqualFold(path[:len(prefix)], prefix)
}

// extendsChainOf is path and the components it extends, as far as they
// resolve: the chain presideSuperClassFunc reads a bare call's file by.
func (r *Resolver) extendsChainOf(path string) []string {
	chain := []string{path}
	seen := map[string]bool{path: true}

	for len(chain) < 16 {
		ext, ok := r.extendsOf(path, cfpath.ToURI(path))
		if !ok || ext == "" {
			break
		}

		next := r.ComponentPath(ext, filepath.Dir(path))
		if next == "" || seen[next] {
			break
		}

		seen[next] = true
		chain = append(chain, next)
		path = next
	}

	return chain
}
