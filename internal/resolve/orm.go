package resolve

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"

	cfpath "github.com/cfmleditor/clif/internal/path"
)

// cborm's VirtualEntityService is a service bound to one entity: ContentBox's
// AuthorService calls `super.init( entityName = "cbAuthor" )`, and from then
// its new(), get(), getOrFail() and findWhere() hand back cbAuthor entities.
// The methods are the base's, declared `any`, so the answer is the entity
// the service was bound to — which the service's own source states.

// entityMethods are the VirtualEntityService methods that return one entity
// of the service's own.
var entityMethods = map[string]bool{
	"new": true, "get": true, "getorfail": true, "findwhere": true,
}

var superInitEntityRe = regexp.MustCompile(`(?is)\bsuper\.init\(\s*(?:entityName\s*=\s*)?["']([^"'#]+)["']`)

// superInitArgRe is `super.init( entityName = arguments.x )`: the service
// passes on an argument of its own init, bound to that argument's default.
var superInitArgRe = regexp.MustCompile(`(?is)\bsuper\.init\(\s*(?:entityName\s*=\s*)?(?:arguments\.)?([a-z_][\w]*)\s*[,)]`)

// entityReturn is the entity a call to method on the service at path
// returns, or "".
func (r *Resolver) entityReturn(path, method string) string {
	if path == "" || !entityMethods[strings.ToLower(method)] {
		return ""
	}

	return r.boundEntity(path)
}

// boundEntity is the entity the service at path, or a service it extends,
// binds itself to with super.init( entityName = … ), or "". Cached per path:
// it is asked for every call on a cborm service, and each answer read the
// service and its bases and ran two regular expressions over them.
func (r *Resolver) boundEntity(path string) string {
	key := pathKey(path)

	r.mu.RLock()
	entity, ok := r.boundEntityCache[key]
	r.mu.RUnlock()

	if ok {
		return entity
	}

	entity = r.boundEntityUncached(path)

	r.mu.Lock()
	if r.boundEntityCache == nil {
		r.boundEntityCache = map[string]string{}
	}

	r.boundEntityCache[key] = entity
	r.mu.Unlock()

	return entity
}

func (r *Resolver) boundEntityUncached(path string) string {
	seen := map[string]bool{}

	for depth := 0; path != "" && !seen[path] && depth < 8; depth++ {
		seen[path] = true

		data, err := r.fs().ReadFile(path)
		if err != nil {
			return ""
		}

		// Both expressions need `super.init(`, which most components lack.
		if mentions(string(data), "super.init(") {
			if m := superInitEntityRe.FindSubmatch(data); m != nil {
				return strings.TrimSpace(string(m[1]))
			}

			if m := superInitArgRe.FindSubmatch(data); m != nil {
				// The service binds whatever its own init() was handed, so the
				// answer is the argument's default, and nothing without one.
				content := string(data)

				return initArgumentDefault(content, r.functionSpansOf(path, content), string(m[1]))
			}
		}

		ext, ok := r.extendsOf(path, cfpath.ToURI(path))
		if !ok || ext == "" {
			return ""
		}

		path = r.ComponentPath(ext, filepath.Dir(path))
	}

	return ""
}

// initArgumentDefault is the literal default of the argument name of the
// first init() the parse found in content (spans, its FunctionSpans), or "":
// a script init's `entityName = "cbContent"`, or the default of the
// `<cfargument name="entityName">` in a `<cffunction name="init">`. A
// default that is an expression (`#…#`) is none.
func initArgumentDefault(content string, spans []parser.FunctionSpan, name string) string {
	literal := func(d string) string {
		if strings.Contains(d, "#") {
			return ""
		}

		return strings.TrimSpace(d)
	}

	for i := range spans {
		s := &spans[i]
		if !strings.EqualFold(s.Name, "init") {
			continue
		}

		if s.Kind != parser.FunctionTag {
			return literal(s.Defaults[strings.ToLower(name)])
		}

		tags := parser.CFTags(s.BodyText(content), "cfargument")
		for j := range tags {
			tag := &tags[j]
			if arg, _ := tag.Attr("name"); tag.Close || !strings.EqualFold(arg, name) {
				continue
			}

			d, _ := tag.Attr("default")

			return literal(d)
		}

		return ""
	}

	return ""
}
