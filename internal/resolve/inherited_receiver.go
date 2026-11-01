package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// inheritedReceiver is what a receiver the file never types holds through its
// extends chain: a ColdBox test base's mock of the class the test names,
// asked first because the base's own ref is the $any that
// createMock( annotations.model ) gives, then a ComponentRef in a parent
// (variables.$assert assigned in a base spec).
func (r *Resolver) inheritedReceiver(variable, lookupVar string, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	if comp := r.coldboxTestSubject(variable, pr); comp != "" {
		tr.addf("resolved %q to %q: the class the test's ColdBox base mocks from its attribute", variable, comp)

		return comp
	}

	if comp := r.coldboxPrcResponse(variable, pr); comp != "" {
		tr.addf("resolved %q to %q: ColdBox keeps the request's Response in prc.response", variable, comp)

		return comp
	}

	comp := ""

	if r.fileExtends(pr) != "" {
		tr.addf("no ref found in this file — checking extends chain (%s) for a ComponentRef", r.fileExtends(pr))

		r.walkExtendsRefs(r.fileExtends(pr), baseDir, lookupVar, func(ref *parser.ComponentRef, parent string) bool {
			comp = ref.Component

			tr.addf("resolved %q to %q via ComponentRef in parent %s", variable, comp, parent)

			return comp != ""
		})
	}

	if comp == "" {
		if comp = r.presideSuperClassField(lookupVar, pr); comp != "" {
			tr.addf("resolved %q to %q: a field of PresideSuperClass, which Preside's WireBox gives a presideService component", variable, comp)
		}
	}

	return comp
}

// presideSuperClassField is what PresideSuperClass holds in the field name,
// for a presideService component: its methods are the component's by virtual
// inheritance (presideSuperClassFunc), and so are the fields they read, such
// as this.$helpers. Every one of them starts with $, which is what keeps this
// off every other receiver.
func (r *Resolver) presideSuperClassField(name string, pr *parser.ParseResult) string {
	if !strings.HasPrefix(name, "$") || pr == nil || !pr.URI.IsFile() || r.Index == nil {
		return ""
	}

	chain := r.extendsChainOf(pr.URI.Path())
	if len(chain) == 0 {
		return ""
	}

	super := r.presideSuperClass(chain[0])
	if super == "" || !slices.ContainsFunc(chain, func(p string) bool { return wantsPresideInjection(r.sourceInfo(p).content) }) {
		return ""
	}

	r.EnsureIndexed(super)

	for _, ref := range r.Index.RefsForFile(cfpath.ToURI(super)) {
		if strings.EqualFold(ref.Variable, name) && ref.Component != "" {
			if p := r.ComponentPath(ref.Component, filepath.Dir(super)); p != "" {
				return p
			}

			return ref.Component
		}
	}

	return ""
}
