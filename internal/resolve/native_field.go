package resolve

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// A component often keeps a registry in a field it only ever empties:
// ColdBox's FrameworkSupertype sets `variables.cbInjectedHelpers = {}` in
// init(), then asks `variables.cbInjectedHelpers.keyExists( h )` and writes
// its keys. A struct literal types a pseudo-constructor assignment, and the
// field read in another function was typed nowhere.
//
// emptyCollectionField types such a field from the text of every component
// that shares its variables scope: the file, the classes above it and the
// classes below it. Every direct write to the field there must be the same
// empty collection (`{}`, `[:]`, `structNew()`, `[]`, `arrayNew( 1 )`); a key or
// element write and a member call do not change what it is. A write of
// anything else, two kinds of collection, the variables scope indexed by a
// computed key or used whole, a file that includes a template, which runs in
// the same scope, or a family too large or not followed to its root, leaves it
// untyped. A struct can hold a closure under a key, so a call of a method a
// struct does not have (`variables.scopes.alpha()`) may be a key's function,
// and says nothing about the field.

// emptyCollectionField is the native collection variable (`variables.x`, or a
// bare `x` the calling function does not hold itself) always is in the
// component pr, for a call of method on it, or "".
func (r *Resolver) emptyCollectionField(variable string, line uint32, caller, method string, pr *parser.ParseResult, tr *callTrace) string {
	if pr == nil || !pr.URI.IsFile() || r.Index == nil || !strings.EqualFold(filepath.Ext(pr.URI.Path()), ".cfc") {
		return ""
	}

	name, scoped := variable, false
	if rest, ok := strings.CutPrefix(strings.ToLower(variable), "variables."); ok {
		name, scoped = variable[len(variable)-len(rest):], true
	}

	if name == "" || strings.ContainsAny(name, ".[(") || !scoped && (isCFMLScope(name) || argumentOf(pr, caller, name) != nil || declaresInFunction(pr, line, name)) {
		return ""
	}

	kind := r.collectionFieldKind(pr.URI.Path(), name)
	if kind == "" {
		return ""
	}

	if kind == "struct" && method != "" && nativeDefaultOverwritten(nativetype.Type(kind), method) {
		if _, mapped := nativetype.Method(nativetype.Type(kind), method); !mapped {
			return ""
		}
	}

	tr.addf("%q is only ever written an empty %s, in its component and every class above and below it", variable, kind)

	return nativetype.Type(kind)
}

// collectionFieldKind is the kind of empty collection ("struct" or "array")
// every direct write to the field name assigns across the variables family of
// the component at path, or "", cached per index generation.
func (r *Resolver) collectionFieldKind(path, name string) string {
	o := r.owner()
	key := pathKey(path) + "\x00" + strings.ToLower(name) + "\x00" + strconv.FormatUint(r.Index.Generation(), 10)

	o.mu.RLock()
	kind, ok := o.collectionFields[key]
	o.mu.RUnlock()

	if ok {
		return kind
	}

	kind = r.collectionFieldKindUncached(path, name)

	o.mu.Lock()
	if o.collectionFields == nil || len(o.collectionFields) >= 8192 {
		o.collectionFields = map[string]string{}
	}

	o.collectionFields[key] = kind
	o.mu.Unlock()

	return kind
}

func (r *Resolver) collectionFieldKindUncached(path, name string) string {
	family, ok := r.variablesFamily(path)
	if !ok {
		return ""
	}

	kind := ""

	for _, file := range family {
		if len(r.Index.IncludesForFile(cfpath.ToURI(file))) > 0 {
			return ""
		}

		text, err := r.readText(file)
		if err != nil {
			return ""
		}

		tokens := allTokens(text)
		if countVariablesReferences(tokens).escapes {
			return ""
		}

		k, read := fieldWrites(tokens, name)
		if !read || k != "" && kind != "" && k != kind {
			return ""
		}

		if k != "" {
			kind = k
		}
	}

	return kind
}

// declaresInFunction reports whether the function holding line declares name
// as a local of its own.
func declaresInFunction(pr *parser.ParseResult, line uint32, name string) bool {
	scope := parser.FindFuncScopeAt(int(line), pr.Scopes)
	if scope.Start == -1 {
		return false
	}

	return slices.ContainsFunc(pr.FuncVars(scope.Start, scope.End), func(v string) bool { return strings.EqualFold(v, name) })
}

// variablesFamily is the file at path, the components above it and the
// components below it, which share one variables scope at run time; ok false
// when a link above cannot be followed or the family is too large to read.
func (r *Resolver) variablesFamily(path string) (family []string, ok bool) {
	family = []string{path}

	for current := path; ; {
		ext, found := r.extendsOf(current, cfpath.ToURI(current))
		if !found {
			return nil, false
		}

		if ext == "" {
			break
		}

		next := r.ComponentPath(ext, filepath.Dir(current))
		if next == "" || slices.ContainsFunc(family, func(f string) bool { return cfpath.SamePath(f, next) }) || len(family) > maxSubclasses {
			return nil, false
		}

		family = append(family, next)
		current = next
	}

	for i := 0; i < len(family); i++ {
		parent := family[i]
		base := strings.TrimSuffix(filepath.Base(parent), filepath.Ext(parent))

		for _, u := range r.Index.FilesExtendingName(base) {
			sub := cfpath.FromURI(u)
			if slices.ContainsFunc(family, func(f string) bool { return cfpath.SamePath(f, sub) }) || !r.descendsFrom(sub, parent) {
				continue
			}

			if len(family) >= 2*maxSubclasses {
				return nil, false
			}

			family = append(family, sub)
		}
	}

	return family, true
}

// fieldWrites reads the direct writes to the variable name in tokens: the
// kind of empty collection they all assign ("struct" or "array"), "" when
// there are none, and read false when one assigns something else, or two
// kinds. A local declaration of the name (`var name =`) is not a write to it.
func fieldWrites(tokens []parser.Token, name string) (kind string, read bool) {
	for i := range tokens {
		t := tokens[i]
		if t.Kind != parser.TokIdent || !strings.EqualFold(t.Value, name) || !variablesReference(tokens, i) {
			continue
		}

		if i > 0 && tokens[i-1].Kind == parser.TokIdent && strings.EqualFold(tokens[i-1].Value, "var") {
			continue
		}

		if i+1 >= len(tokens) || tokens[i+1].Kind != parser.TokEquals || i+2 < len(tokens) && tokens[i+2].Kind == parser.TokEquals {
			continue
		}

		k := emptyCollectionAt(tokens, i+2)
		if k == "" || kind != "" && k != kind {
			return "", false
		}

		kind = k
	}

	return kind, true
}

// emptyCollectionAt is the kind of empty collection the expression at
// tokens[i] builds, or "".
func emptyCollectionAt(tokens []parser.Token, i int) string {
	at := func(j int, kind parser.TokenKind) bool { return j < len(tokens) && tokens[j].Kind == kind }
	call := func(name string) bool {
		return i < len(tokens) && tokens[i].Kind == parser.TokIdent && strings.EqualFold(tokens[i].Value, name) && at(i+1, parser.TokLParen)
	}

	switch {
	case at(i, parser.TokLBrace) && at(i+1, parser.TokRBrace):
		return "struct"
	case at(i, parser.TokLBracket) && at(i+1, parser.TokColon) && at(i+2, parser.TokRBracket):
		return "struct"
	case at(i, parser.TokLBracket) && at(i+1, parser.TokRBracket):
		return "array"
	case call("structNew") && at(i+2, parser.TokRParen):
		return "struct"
	case call("arrayNew") && (at(i+2, parser.TokRParen) || at(i+2, parser.TokNumber) && tokens[i+2].Value == "1" && at(i+3, parser.TokRParen)):
		return "array"
	default:
		return ""
	}
}
