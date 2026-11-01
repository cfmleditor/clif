package resolve

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// An FW/1 view reads the rc its controller filled: views/<section>/<item>.cfm
// is rendered after controllers/<section>.cfc's <item>( rc ), which runs after
// its before( rc ), and Mura's admin views read rc.contentBean, rc.siteBean and
// the rest without a word about their type. So rc.X in such a view is what the
// item action assigns arguments.rc.X by its end, or, when it assigns nothing,
// what before() does; a controller action that calls setView("section.item")
// renders the view too, and every action must agree. An action that hands its
// rc to another (`detail( rc )`) leaves what that one assigns, and a view no
// action renders, a partial, holds the rc of every view rendering it with
// view( "section/sub/item" ). A subsystem's setView( "admin:section.item" )
// names the views under admin/.
//
// The convention applies only where a framework preset gives views a base
// (ImplicitExtends), which is how the fw1 preset says views run inside the
// framework.

// fw1SetViewRe is setView( "section.item" ), the action named or not, with a
// subsystem or not: Slatwall's controllers write
// `getFW().setView( action = "admin:product.detail" )`.
var fw1SetViewRe = regexp.MustCompile(`(?i)\bsetView\s*\(\s*(?:action\s*=\s*)?["'](?:([\w-]+):)?([\w.]+)["']\s*\)`)

// fw1ViewCallRe is a view rendering another inside itself, `#view(
// "product/producttabs/settings" )#`, which FW/1 renders with the same rc.
var fw1ViewCallRe = regexp.MustCompile(`(?i)\bview\s*\(\s*(?:path\s*=\s*)?["'](?:([\w-]+):)?([\w/]+)["']\s*[,)]`)

// maxFW1ViewDepth bounds how many views rendering views rc is followed through.
const maxFW1ViewDepth = 4

// fw1ViewRc is the component rc.X holds in the FW/1 view pr, or "".
func (r *Resolver) fw1ViewRc(variable, funcName string, pr *parser.ParseResult, tr *callTrace) string {
	scope, name, ok := strings.Cut(variable, ".")
	if !ok || !strings.EqualFold(scope, "rc") || name == "" || strings.ContainsAny(name, ".[") || !pr.URI.IsFile() {
		return ""
	}

	path := pr.URI.Path()
	if r.ImplicitExtends == nil || r.ImplicitExtends(path) == "" {
		return ""
	}

	answer := r.fw1ViewRcAt(path, name, funcName, 0)
	if answer != "" {
		tr.addf("%q is what the FW/1 controller action(s) rendering this view assign it: %q", variable, answer)
	}

	return answer
}

// fw1ViewRcAt is what rc.name holds in the view at path: what the actions
// rendering it assign, or, when no action renders it, what rc.name holds in
// every view rendering it with view(). Every one must agree.
func (r *Resolver) fw1ViewRcAt(path, name, funcName string, depth int) string {
	app, view, ok := fw1ViewOf(path)
	if !ok || depth > maxFW1ViewDepth {
		return ""
	}

	answer := ""
	agree := func(comp string) bool {
		if comp == "" || answer != "" && !cfpath.SamePath(answer, comp) {
			return false
		}

		answer = comp

		return true
	}

	if section, item, _, ok := fw1ViewName(path); ok {
		actions := r.fw1Actions(filepath.Join(app, "controllers"), section, item)
		for _, a := range actions {
			if !agree(r.fw1ActionRc(a.file, a.action, name, funcName)) {
				return ""
			}
		}

		if len(actions) > 0 {
			return answer
		}
	}

	includers := r.fw1Includers(app)[strings.ToLower(view)]
	for _, inc := range includers {
		if cfpath.SamePath(inc, path) || !agree(r.fw1ViewRcAt(inc, name, funcName, depth+1)) {
			return ""
		}
	}

	return answer
}

// fw1ViewOf is the application directory holding the view at path, and the
// view's name as view() spells it: its path under views/, without .cfm.
func fw1ViewOf(path string) (app, view string, ok bool) {
	slash := filepath.ToSlash(path)

	before, after, found := strings.CutLast(slash, "/views/")
	if !found || !strings.EqualFold(filepath.Ext(path), ".cfm") || !strings.Contains(after, "/") {
		return "", "", false
	}

	return filepath.FromSlash(before), strings.TrimSuffix(after, filepath.Ext(after)), true
}

// fw1Includers maps each view of the application at app, by lowercased
// name, to the views that render it with view(). A call naming another
// subsystem is not this application's. Cached per application.
func (r *Resolver) fw1Includers(app string) map[string][]string {
	key := pathKey(app)

	r.mu.RLock()
	cached, ok := r.fw1IncluderCache[key]
	r.mu.RUnlock()

	if ok {
		return cached
	}

	out := map[string][]string{}

	for _, file := range cfpath.CollectCFML(r.fs(), []string{filepath.Join(app, "views")}) {
		data, err := r.fs().ReadFile(file)
		if err != nil {
			continue
		}

		seen := map[string]bool{}

		for _, m := range fw1ViewCallRe.FindAllSubmatch(data, -1) {
			if len(m[1]) > 0 && !strings.EqualFold(string(m[1]), filepath.Base(app)) {
				continue
			}

			view := strings.ToLower(strings.Trim(string(m[2]), "/"))
			if !seen[view] {
				seen[view] = true
				out[view] = append(out[view], file)
			}
		}
	}

	r.mu.Lock()
	if r.fw1IncluderCache == nil {
		r.fw1IncluderCache = map[string]map[string][]string{}
	}

	r.fw1IncluderCache[key] = out
	r.mu.Unlock()

	return out
}

type fw1Action struct{ file, action string }

// fw1ViewName splits a view's path into its section and item and the
// application directory holding views/.
func fw1ViewName(path string) (section, item, app string, ok bool) {
	slash := filepath.ToSlash(path)

	before, after, found := strings.CutLast(slash, "/views/")
	if !found || !strings.EqualFold(filepath.Ext(path), ".cfm") {
		return "", "", "", false
	}

	section, file, found := strings.Cut(after, "/")
	if !found || strings.Contains(file, "/") || section == "" {
		return "", "", "", false
	}

	return section, strings.TrimSuffix(file, filepath.Ext(file)), filepath.FromSlash(before), true
}

// fw1Actions are the controller actions that render section.item: the
// section controller's item action, and every action whose body calls
// setView("section.item").
//
// Cached per resolver: a view asks once per rc call, and each answer scans
// every controller's source (Mura's editsite.cfm, 160 rc.siteBean calls, took
// 3s of a scan doing so).
func (r *Resolver) fw1Actions(controllers, section, item string) []fw1Action {
	key := pathKey(controllers) + "\x00" + strings.ToLower(section+"."+item)

	r.mu.RLock()
	cached, ok := r.fw1ActionCache[key]
	r.mu.RUnlock()

	if ok {
		return cached
	}

	out := r.fw1ActionsUncached(controllers, section, item)

	r.mu.Lock()
	if r.fw1ActionCache == nil {
		r.fw1ActionCache = map[string][]fw1Action{}
	}

	r.fw1ActionCache[key] = out
	r.mu.Unlock()

	return out
}

func (r *Resolver) fw1ActionsUncached(controllers, section, item string) []fw1Action {
	var out []fw1Action

	own := filepath.Join(controllers, section+".cfc")
	if hpr := r.handlerParse(own); hpr != nil && hasFunc(hpr, item) {
		out = append(out, fw1Action{own, item})
	}

	entries, err := r.fs().ReadDir(controllers)
	if err != nil {
		return out
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".cfc") {
			continue
		}

		file := filepath.Join(controllers, entry.Name())

		hpr := r.handlerParse(file)
		if hpr == nil {
			continue
		}

		for _, m := range fw1SetViewRe.FindAllStringSubmatchIndex(hpr.Content, -1) {
			if !strings.EqualFold(hpr.Content[m[4]:m[5]], section+"."+item) {
				continue
			}

			// A subsystem is a directory of its own: admin:product.detail
			// is admin/views/product/detail.cfm.
			if m[2] >= 0 && !strings.EqualFold(hpr.Content[m[2]:m[3]], filepath.Base(filepath.Dir(controllers))) {
				continue
			}

			line := strings.Count(hpr.Content[:m[0]], "\n")

			scope := hpr.FuncScopeAt(line)
			if scope.Start == -1 {
				return nil
			}

			out = append(out, fw1Action{file, scope.Name})
		}
	}

	return out
}

// fw1ActionRc is what action in the controller file leaves in rc.name: its
// own last assignment, else what the actions it hands its rc to leave there
// (Slatwall's edit() is `detail( rc ); setView( "admin:product.detail" )`),
// else before()'s.
func (r *Resolver) fw1ActionRc(file, action, name, funcName string) string {
	return r.fw1ActionRcD(file, action, name, funcName, 0)
}

// fw1RcCallRe is an action handing its rc to another of the controller's:
// `detail( rc )`, `this.detail( arguments.rc )`.
var fw1RcCallRe = regexp.MustCompile(`(?i)(?:^|[^\w.]|\bthis\.)(\w+)\s*\(\s*(?:rc\s*=\s*)?(?:arguments\.)?rc\s*\)`)

func (r *Resolver) fw1ActionRcD(file, action, name, funcName string, depth int) string {
	hpr := r.handlerParse(file)
	if hpr == nil || depth > maxFW1ViewDepth {
		return ""
	}

	for _, fn := range []string{action, "before"} {
		scope, ok := funcScope(hpr, fn)
		if !ok {
			continue
		}

		if !fw1Assigns(hpr.Content, scope, name) {
			if fn == action {
				if comp, handed := r.fw1HandedRc(file, hpr, scope, name, funcName, depth); handed {
					return comp
				}
			}

			continue
		}

		for _, receiver := range []string{"arguments.rc." + name, "rc." + name} {
			comp, _ := r.receiverComponentD(receiver, conv.Uint32(scope.End), scope.Name, funcName, hpr, filepath.Dir(file), nil, lookupCtx{})
			if comp != "" && !strings.HasPrefix(comp, "$") {
				return r.pathsOf(comp, filepath.Dir(file))
			}
		}

		return ""
	}

	return ""
}

// fw1HandedRc is what rc.name holds after the actions the action at scope
// hands its rc to, and whether it hands it to any that assign rc.name. Every
// such action must agree.
func (r *Resolver) fw1HandedRc(file string, hpr *parser.ParseResult, scope parser.FuncScope, name, funcName string, depth int) (string, bool) {
	answer, handed := "", false

	for _, m := range fw1RcCallRe.FindAllStringSubmatch(sourceText(hpr.Content, scope.Start, scope.End+1), -1) {
		callee, ok := funcScope(hpr, m[1])
		if !ok || strings.EqualFold(callee.Name, scope.Name) || !fw1Assigns(hpr.Content, callee, name) {
			continue
		}

		comp := r.fw1ActionRcD(file, callee.Name, name, funcName, depth+1)
		if comp == "" || answer != "" && !cfpath.SamePath(answer, comp) {
			return "", true
		}

		answer, handed = comp, true
	}

	return answer, handed
}

func hasFunc(pr *parser.ParseResult, name string) bool {
	_, ok := funcScope(pr, name)

	return ok
}

func funcScope(pr *parser.ParseResult, name string) (parser.FuncScope, bool) {
	for _, s := range pr.Scopes {
		if strings.EqualFold(s.Name, name) {
			return s, true
		}
	}

	return parser.FuncScope{}, false
}

// fw1Assigns reports whether the function at scope assigns rc.name.
func fw1Assigns(content string, scope parser.FuncScope, name string) bool {
	re := compiledPattern(`(?i)\b(?:arguments\.)?rc\.` + regexp.QuoteMeta(name) + `\s*=[^=]`)

	for line := range strings.Lines(sourceText(content, scope.Start, scope.End+1)) {
		if mentions(line, "rc."+name) && re.MatchString(strings.TrimSuffix(line, "\n")) {
			return true
		}
	}

	return false
}
