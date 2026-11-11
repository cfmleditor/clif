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

// scriptInitRe finds the service's own init() in script syntax, its parameter
// list; in tag syntax its <cfargument>s are read with parser.CFTags.
var (
	scriptInitRe = regexp.MustCompile(`(?is)\bfunction\s+init\s*\(([^)]*)\)`)

	// paramDefaultRe is a script parameter with a literal default.
	paramDefaultRe = regexp.MustCompile(`(?is)(?:^|[,\s])([a-z_]\w*)\s*=\s*["']([^"'#]+)["']`)
)

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
				return initArgumentDefault(data, string(m[1]))
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
// init() declared in src, or "": `init( entityName = "cbContent" )`, or a
// `<cfargument name="entityName" default="cbContent">` inside
// `<cffunction name="init">`.
func initArgumentDefault(src []byte, name string) string {
	if m := scriptInitRe.FindSubmatch(src); m != nil {
		for _, d := range paramDefaultRe.FindAllSubmatch(m[1], -1) {
			if strings.EqualFold(string(d[1]), name) {
				return strings.TrimSpace(string(d[2]))
			}
		}
	}

	// The <cfargument>s between <cffunction name="init"> and its close.
	inInit := false

	tags := parser.CFTags(string(src), "cffunction", "cfargument")
	for i := range tags {
		tag := &tags[i]

		if tag.Name == "cffunction" {
			if inInit {
				return "" // its close, or a function after it
			}

			fn, _ := tag.Attr("name")
			inInit = !tag.Close && strings.EqualFold(fn, "init")

			continue
		}

		if !inInit || tag.Close {
			continue
		}

		if arg, _ := tag.Attr("name"); !strings.EqualFold(arg, name) {
			continue
		}

		if d, _ := tag.Attr("default"); d != "" && !strings.Contains(d, "#") {
			return strings.TrimSpace(d)
		}

		return ""
	}

	return ""
}
