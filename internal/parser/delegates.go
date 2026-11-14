package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// Delegate is a WireBox delegation (ColdBox 7): the component holding it has
// every public method of Target, under the name Prefix + method + Suffix.
// `property name="memory" inject delegate delegatePrefix;` gives the
// component memoryRead() and memoryWrite() from Memory, and
// `component delegates=">Memory, Worker=vacation"` does the same at
// component level. Nothing declares those methods anywhere a parse can see:
// the injector mixes them in when it builds the object.
type Delegate struct {
	Target string // the id delegated to, as injectedComponent reads one
	Prefix string
	Suffix string
	// Includes, when set, are the only methods delegated; Excludes are
	// never delegated.
	Includes []string
	Excludes []string
	Line     uint32
}

// DelegateTarget is the component a delegation's id names, as an injection's
// id is read: `Memory@module` is Memory.
func DelegateTarget(id string) string {
	return injectedComponent(id)
}

// DelegatedMethod is the method of d's target that name reaches, or "" when
// name is not one d delegates.
func (d *Delegate) DelegatedMethod(name string) string {
	if len(name) <= len(d.Prefix)+len(d.Suffix) ||
		!strings.EqualFold(name[:len(d.Prefix)], d.Prefix) ||
		!strings.EqualFold(name[len(name)-len(d.Suffix):], d.Suffix) {
		return ""
	}

	inner := name[len(d.Prefix) : len(name)-len(d.Suffix)]

	if coreDelegateExclusions[strings.ToLower(inner)] || lex.ContainsFold(d.Excludes, inner) {
		return ""
	}

	if len(d.Includes) > 0 && !lex.ContainsFold(d.Includes, inner) {
		return ""
	}

	return inner
}

// coreDelegateExclusions are the methods WireBox never delegates
// (Mapping.cfc's CORE_DELEGATE_EXCLUSIONS, less its $wb internals).
var coreDelegateExclusions = map[string]bool{
	"init": true, "$init": true, "ondicomplete": true, "setinjector": true,
	"setbeanfactory": true, "setcoldbox": true,
}

// propertyFlags are the property attributes WireBox reads by their presence
// alone: `property name="m" inject delegate delegatePrefix;`. Both parsers
// record one written without a value as an attribute whose value is "".
var propertyFlags = map[string]bool{
	"inject": true, "delegate": true, "delegateprefix": true, "delegatesuffix": true,
}

// collectDelegates fills pr.Delegates from the properties marked `delegate`
// and the component's `delegates` attribute.
func (pr *ParseResult) collectDelegates() {
	pr.Delegates = nil

	for i := range pr.Properties {
		prop := &pr.Properties[i]
		if _, ok := prop.attrs["delegate"]; !ok {
			continue
		}

		d := Delegate{
			Target:   prop.attrs["inject"],
			Includes: splitList(prop.attrs["delegate"]),
			Excludes: splitList(prop.attrs["delegateexcludes"]),
			Line:     prop.line,
		}

		if d.Target == "" {
			d.Target = prop.name
		}

		d.Prefix = flagOrName(prop, "delegateprefix")
		d.Suffix = flagOrName(prop, "delegatesuffix")

		pr.Delegates = append(pr.Delegates, d)
	}

	if u := string(pr.URI); len(u) < 4 || !strings.EqualFold(u[len(u)-4:], ".cfc") {
		return
	}

	for _, item := range splitList(componentAttr(pr.Content, "delegates")) {
		pr.Delegates = append(pr.Delegates, parseDelegateItem(item))
	}
}

// DeclaresTable reports whether the component names a database table with
// `table="y"`: what a component not marked persistent has when an ORM other
// than the engine's, Mura's bean ORM, maps it.
func (pr *ParseResult) DeclaresTable() bool {
	if u := string(pr.URI); len(u) < 4 || !strings.EqualFold(u[len(u)-4:], ".cfc") {
		return false
	}

	return strings.TrimSpace(componentAttr(pr.Content, "table")) != ""
}

// EntityName is the ORM entity name the component declares with
// `entityname="cbAuthor"`, or "" when it names none — in which case the
// entity is called after the file. entityNew( "cbAuthor" ) names it.
func (pr *ParseResult) EntityName() string {
	if u := string(pr.URI); len(u) < 4 || !strings.EqualFold(u[len(u)-4:], ".cfc") {
		return ""
	}

	return strings.TrimSpace(componentAttr(pr.Content, "entityname"))
}

// flagOrName is a delegatePrefix or delegateSuffix: its value, the property's
// name when it is written without one, or "" when it is not written.
func flagOrName(prop *propertyDef, key string) string {
	v, ok := prop.attrs[key]

	switch {
	case !ok:
		return ""
	case v == "":
		return prop.name
	default:
		return v
	}
}

// parseDelegateItem reads one entry of a component's `delegates` attribute,
// WireBox's own grammar (Mapping.cfc processComponentDelegates): `Memory`,
// `>Memory` (prefixed by its id), `ram>Memory`, `<Memory`, `ram<Memory`, and
// `Worker=vacation` for the one method named.
func parseDelegateItem(item string) Delegate {
	var d Delegate

	ref, method, _ := strings.Cut(item, "=")
	if method = strings.TrimSpace(method); method != "" {
		d.Includes = []string{method}
	}

	if i := strings.LastIndexAny(ref, "<>"); i >= 0 {
		affix, op := strings.TrimSpace(ref[:i]), ref[i]
		ref = ref[i+1:]

		if affix == "" {
			affix = strings.TrimSpace(ref)
		}

		if op == '>' {
			d.Prefix = affix
		} else {
			d.Suffix = affix
		}
	}

	d.Target = strings.TrimSpace(ref)

	return d
}

// componentAttr is the value of one of the component's attributes,
// `delegates` or `entityname`, script or tag, or "". It is read from the
// declaration alone, by the script scanner: the attributes after `component`
// up to its opening brace, or after <cfcomponent up to the tag's `>`, so a
// parse pays for the few lines of the declaration, not the file. A brace or a
// `>` in a quoted value, and anything in a comment before the declaration,
// is no end of it: a byte search for the first `{` stopped in a doc
// comment's `@return {struct}` and lost the attribute.
func componentAttr(content, attr string) string {
	sc := NewScriptScanner(content)
	inDeclaration := false

	for {
		t := sc.NextSkipComments()

		switch {
		case t.Kind == TokEOF || t.Kind == TokLBrace || inDeclaration && t.Kind == TokGT:
			return ""
		case t.Kind != TokIdent:
		case !inDeclaration:
			inDeclaration = identEq(t.Value, "component") || identEq(t.Value, "cfcomponent")
		case identEq(t.Value, attr) && sc.PeekSkipComments().Kind == TokEquals:
			sc.NextSkipComments()

			if v := sc.NextSkipComments(); v.Kind == TokString && len(v.Value) >= 2 {
				return v.Value[1 : len(v.Value)-1]
			}

			return ""
		}
	}
}

func splitList(s string) []string {
	var out []string

	for v := range strings.SplitSeq(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}

	return out
}

// relationshipMethods are the methods CFML's ORM generates for a
// relationship property, beyond its getter and setter (Adobe's "ORM -
// Generated methods for relationships", which Lucee follows): a collection
// has add<singular>(), remove<singular>(), has<name>() and has<singular>(),
// and a single-valued one has<name>(). singularName defaults to the name.
func relationshipMethods(prop *propertyDef) []string {
	name := ucFirst(prop.name)

	singular := prop.attrs["singularname"]
	if singular == "" {
		singular = prop.name
	}

	singular = ucFirst(singular)

	switch strings.ToLower(prop.attrs["fieldtype"]) {
	case "one-to-many", "many-to-many":
		return []string{"add" + singular, "remove" + singular, "has" + name, "has" + singular}
	case "many-to-one", "one-to-one":
		return []string{"has" + name}
	}

	return nil
}
