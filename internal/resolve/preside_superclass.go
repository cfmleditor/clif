package resolve

import (
	"path/filepath"
	"regexp"
	"strings"

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

var (
	// presideServiceRe is the key in a component's doc comment or among its
	// attributes.
	presideServiceRe = regexp.MustCompile(`(?i)@presideService\b|\bpresideService\s*=`)
	cfcomponentRe    = regexp.MustCompile(`(?i)<cfcomponent\b[^>]*>`)
)

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

// wantsPresideInjection reports whether a component's own header, the doc
// comment and attributes before its body, holds the presideService key.
func wantsPresideInjection(content string) bool {
	header := content
	if loc := cfcomponentRe.FindStringIndex(content); loc != nil {
		header = content[:loc[1]]
	} else if before, _, found := strings.Cut(content, "{"); found {
		header = before
	}

	return presideServiceRe.MatchString(header)
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
