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

// superInitEntity reads the entity src's code binds its service to: the
// first super.init() whose first argument, positionally or as entityName =,
// is a literal with no expression in it; else the first passing on an
// argument of the service's own init, `super.init( entityName =
// arguments.x )`, whose name is arg.
func superInitEntity(src string) (entity, arg string) {
	var argument string

	for _, call := range codeCalls(textCode(src), "init") {
		if call.before(1).Kind != parser.TokDot || !isWord(call.before(2), "super") {
			continue
		}

		value := firstArgValue(call.pieces(), "entityName")

		if lit, ok := soleString(value); ok && strings.TrimSpace(lit) != "" && !strings.ContainsAny(lit, "#\"'") {
			return strings.TrimSpace(lit), ""
		}

		name, next := dottedAt(value, 0)
		if argument != "" || name == "" || next != len(value) {
			continue
		}

		if rest, ok := strings.CutPrefix(strings.ToLower(name), "arguments."); ok {
			name = name[len(name)-len(rest):]
		}

		if !strings.Contains(name, ".") && !strings.Contains(name, "$") {
			argument = name
		}
	}

	return "", argument
}

// scriptInitRe and tagInitRe find the service's own init(): its parameter
// list in script syntax, its body in tag syntax, where the <cfargument>s are.
var (
	scriptInitRe = regexp.MustCompile(`(?is)\bfunction\s+init\s*\(([^)]*)\)`)
	tagInitRe    = regexp.MustCompile(`(?is)<cffunction\s[^>]*\bname\s*=\s*["']init["'][^>]*>(.*?)</cffunction>`)
	cfargumentRe = regexp.MustCompile(`(?is)<cfargument\s[^>]*>`)

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

		// Both forms call super.init(), which most components never mention.
		if text := string(data); mentions(text, "super") && mentions(text, "init") {
			entity, arg := superInitEntity(text)
			if entity != "" {
				return entity
			}

			if arg != "" {
				// The service binds whatever its own init() was handed, so the
				// answer is the argument's default, and nothing without one.
				return initArgumentDefault(data, arg)
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

	m := tagInitRe.FindSubmatch(src)
	if m == nil {
		return ""
	}

	for _, tag := range cfargumentRe.FindAll(m[1], -1) {
		if !strings.EqualFold(tagAttr(tag, "name"), name) {
			continue
		}

		if d := tagAttr(tag, "default"); d != "" && !strings.Contains(d, "#") {
			return strings.TrimSpace(d)
		}

		return ""
	}

	return ""
}

// tagAttr is the value of attribute attr in tag, or "", read as the parse
// reads a CF tag's attributes (parser.TagAttribute). Its own pattern took
// letters alone for a name and quoted values alone, so `show3d` and
// `default=0` were never read.
func tagAttr(tag []byte, attr string) string {
	return parser.TagAttribute(string(tag), attr)
}
