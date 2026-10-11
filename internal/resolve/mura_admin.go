package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// muraAdminBase assumes standard admin views/layouts are rendered by their
// governing Application's source-backed embedded FW/1 instance. It introduces
// no generic FW/1 resolver rules or fallback stubs into Mura's factory domain.
func (r *Resolver) muraAdminBase(path string) string {
	root := r.FindApplicationRoot(filepath.Dir(path))
	r.trackMuraApplicationCandidates(path, root)

	if root == "" {
		return ""
	}

	key := "$mura-admin:" + pathKey(root)

	r.mu.RLock()
	app, cached := r.implicitCache[key]
	r.mu.RUnlock()

	if cached {
		return app
	}

	app = r.muraAdminOwner(root)
	r.mu.Lock()
	if r.implicitCache == nil {
		r.implicitCache = map[string]string{}
	}

	r.implicitCache[key] = app
	r.mu.Unlock()

	return app
}

func (r *Resolver) muraAdminOwner(root string) string {
	tracked := TrackDiscoveryFS(r.fs(), r.Discovery)

	mapping := cfpath.AssumedProjectMappings(root, tracked.ReadFile)["mura"]
	if mapping == "" || pathKey(root) != pathKey(filepath.Join(filepath.Dir(filepath.Dir(mapping)), "admin")) {
		return ""
	}

	app := filepath.Join(root, "Application.cfc")
	for namespace, target := range parser.MappingSources(app, nil, tracked.ReadFile) {
		if strings.EqualFold(strings.TrimPrefix(namespace, "/"), "mura") && pathKey(target) != pathKey(mapping) {
			return ""
		}
	}

	for namespace, target := range r.Mappings {
		if strings.EqualFold(strings.TrimPrefix(namespace, "/"), "mura") && pathKey(target) != pathKey(mapping) {
			return ""
		}
	}

	framework := filepath.Join(root, "framework.cfc")

	source, err := tracked.ReadFile(framework)
	if err != nil {
		return ""
	}

	for _, method := range []string{"view", "layout", "buildURL", "internalView", "internalLayout"} {
		fd := r.ResolveFunc(app, method, root)
		if fd == nil || pathKey(fd.URI.Path()) != pathKey(framework) {
			return ""
		}
	}

	code := muraRendererSource(string(source))
	for _, method := range []string{"view", "layout"} {
		body := functionBodyNamed(code, method)
		pr := parser.ParseWithOptions(cfpath.ToURI(framework), body, &parser.ParseOptions{ExtractCalls: true})
		calls := pr.AllCalls()
		found := false

		for i := range calls {
			call := &calls[i]
			if call.Variable == "" && strings.EqualFold(call.FuncName, "internal"+method) {
				found = true
			}
		}

		if !found {
			return ""
		}
	}

	if !muraParameterInclude(code, "internalView", "viewPath") || !muraParameterInclude(code, "internalLayout", "layoutPath") {
		return ""
	}

	return app
}

func muraParameterInclude(source, method, parameter string) bool {
	body := functionBodyNamed(source, method)
	found := 0

	for _, tag := range producerTags(body) {
		switch tag.name {
		case "cfinclude":
			template, ok := producerAttr(tag.body, "template")
			if !ok || !strings.EqualFold(template, "#"+parameter+"#") && !strings.EqualFold(template, "#arguments."+parameter+"#") {
				return false
			}

			found++
		case "cfset":
			assignment := producerAssignment(producerTokens(tag.body))
			if strings.EqualFold(assignment.target, parameter) || strings.EqualFold(assignment.target, "arguments."+parameter) {
				return false
			}
		default:
		}
	}

	return found == 1
}

// Remove lexical comments before matching the embedded renderer signature.
// Quoted strings are retained; nested CFML comments cannot supply fake includes.
func muraRendererSource(source string) string {
	sc := parser.NewScanner(source)

	var code strings.Builder

	cursor := 0

	for tok := sc.Next(); tok.Kind != parser.TokEOF; tok = sc.Next() {
		switch tok.Kind {
		case parser.TokCFComment, parser.TokBlockComment, parser.TokLineComment:
			code.WriteString(source[cursor:tok.Offset])
			code.WriteByte(' ')

			cursor = tok.Offset + len(tok.Value)
		default:
		}
	}

	code.WriteString(source[cursor:])

	return code.String()
}

// A nearer Application created later changes template ownership. Track failed
// candidates as well as the selected owner's existing source.
func (r *Resolver) trackMuraApplicationCandidates(path, root string) {
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		r.Discovery.add(filepath.Join(dir, "Application.cfc"))
		r.Discovery.add(filepath.Join(dir, "Application.cfm"))

		if dir == root || filepath.Dir(dir) == dir {
			return
		}
	}
}
