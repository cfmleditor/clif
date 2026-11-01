package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// An abstract base calls what its subclasses supply, methods as well as
// variables (subclassComponent): qb's AbstractQueryBuilderSpec calls
// selectAllColumns(), which each grammar's spec defines and the base never
// does, and its getBuilder() only throws "Must be implemented in a subclass".
// A call is looked up in the file and up its extends chain, never down, so
// every expected-value method was "not found in extends chain" and every
// builder getBuilder() made had no type.
//
// subclassFuncs is the definition each concrete subclass (a leaf of the tree
// under the file) gives a name, its own or that of a component between it and
// the file, and nothing unless every leaf has one: a leaf that does not leaves
// the call to fail at runtime there, or to a mixin, and the others' answer
// would be a guess.

func (r *Resolver) subclassFuncs(base, name string) []*parser.FunctionDef {
	if r.Index == nil || base == "" || name == "" {
		return nil
	}

	leaves := r.leafSubclasses(base)
	if len(leaves) == 0 {
		return nil
	}

	defs := make([]*parser.FunctionDef, 0, len(leaves))

	for _, leaf := range leaves {
		def := r.subclassFunc(leaf, base, name)
		if def == nil {
			return nil
		}

		defs = append(defs, def)
	}

	return defs
}

// subclassFunc is the definition of name nearest to the file at path, walking
// up its extends chain and stopping before base.
func (r *Resolver) subclassFunc(path, base, name string) *parser.FunctionDef {
	for range 16 {
		if path == "" || cfpath.SamePath(path, base) {
			return nil
		}

		if def := r.Index.FunctionInFile(cfpath.ToURI(path), name); def != nil {
			return def
		}

		ext, ok := r.extendsOf(path, cfpath.ToURI(path))
		if !ok || ext == "" {
			return nil
		}

		path = r.ComponentPath(ext, filepath.Dir(path))
	}

	return nil
}

// subclassFuncComponents names the components declaring defs, sorted, as
// alternatives in the `a|b` spelling.
func subclassFuncComponents(defs []*parser.FunctionDef) string {
	var names []string

	for _, def := range defs {
		name := strings.TrimSuffix(filepath.Base(def.URI.Path()), filepath.Ext(def.URI.Path()))
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}

	slices.Sort(names)

	return strings.Join(names, "|")
}

// abstractOverridesReturn is what a method the file declares only to throw
// returns: what every leaf subclass's override returns, as alternatives. A
// call to it on the file, or on anything typed as the file, runs one of the
// overrides, since the declaration itself never returns. Every override must
// return a component; `$any` from one makes the answer `$any`, as a branch
// union does.
func (r *Resolver) abstractOverridesReturn(fd *parser.FunctionDef, depth int, budget *int) string {
	if fd == nil || !fd.URI.IsFile() {
		return ""
	}

	// The overrides first: they come from a cached walk of the index, and
	// most methods have none, while abstractBody makes the method's plan. It
	// was asked of every method whose return was looked up, and the plans it
	// made for methods nothing overrides were 160MB of a tassweb scan's heap.
	defs := r.subclassFuncs(fd.URI.Path(), fd.Name)
	if len(defs) == 0 || !r.abstractBody(fd) {
		return ""
	}

	var comps []string

	for _, def := range defs {
		if r.abstractBody(def) {
			return ""
		}

		comp := r.returnComponentOf(def, depth+1, budget)
		if comp == "" {
			return ""
		}

		if comp == "$any" {
			return "$any"
		}

		if p := r.ComponentPath(comp, filepath.Dir(def.URI.Path())); p != "" {
			comp = p
		}

		if !slices.ContainsFunc(comps, func(c string) bool { return strings.EqualFold(c, comp) }) {
			comps = append(comps, comp)
		}
	}

	slices.Sort(comps)

	return strings.Join(comps, "|")
}

// abstractBody reports whether fd's body does nothing but throw: the CFML
// spelling of an abstract method.
func (r *Resolver) abstractBody(fd *parser.FunctionDef) bool {
	method := r.producerFor(fd)
	if method == nil {
		return false
	}

	stops := false

	for i := range method.body {
		switch method.body[i].kind {
		case "noop":
		case "stop":
			stops = true
		default:
			return false
		}
	}

	return stops
}

// subclassesDeclare names the leaf subclasses under the file when every one
// declares a method the file does not, or "": a bare or `this.` call in an
// abstract base runs on an instance of one of them.
func (r *Resolver) subclassesDeclare(pr *parser.ParseResult, funcName string, tr *callTrace) string {
	if !pr.URI.IsFile() {
		return ""
	}

	defs := r.subclassFuncs(pr.URI.Path(), funcName)
	if defs == nil {
		return ""
	}

	comps := subclassFuncComponents(defs)
	tr.addf("%q is not declared here, and every subclass of this component declares it (%s)", funcName, comps)

	return comps
}
