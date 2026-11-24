package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// missingBasePrefix starts MissingBaseReason's answer, which MissingBaseOf
// reads back.
const missingBasePrefix = "extends chain breaks at "

// missingBaseSuffix closes MissingBaseReason's answer; MissingBaseOf requires
// the same text.
const missingBaseSuffix = "', which does not resolve"

// MissingBaseReason is canResolveCall's answer for an inherited call when a
// component in the file's extends chain does not resolve. The method may well
// be declared there; what is missing is the base, and a report that blames the
// call names the wrong problem once per call — 56,000 times over the six-project
// corpus, for a dozen missing bases.
func MissingBaseReason(base string) string {
	return missingBasePrefix + "'" + base + missingBaseSuffix
}

// MissingBaseOf reports the base a MissingBaseReason names.
func MissingBaseOf(reason string) (string, bool) {
	rest, ok := strings.CutPrefix(reason, missingBasePrefix+"'")
	if !ok {
		return "", false
	}

	if !strings.HasSuffix(rest, missingBaseSuffix) {
		return "", false
	}

	base := strings.TrimSuffix(rest, missingBaseSuffix)
	// Every producer of MissingBaseReason guards base != "", so ok means a
	// base was found, never none.
	return base, base != ""
}

// inheritedFromMissingBase is the base a receiver with no component is taken
// to come from: the missing link of the file's extends chain, when the
// receiver is an unscoped or variables./this. name the file never declares.
// `print` in a CommandBox command and `$assert` in a TestBox spec are the
// base's, injected or declared there, and with the base missing they are as
// unchecked as its methods. A name in any other scope, or one the file
// declares, is not the base's to explain.
func (r *Resolver) inheritedFromMissingBase(variable string, line uint32, pr *parser.ParseResult, baseDir string) (base string, implied bool) {
	if r.fileExtends(pr) == "" || variable == "" || strings.ContainsAny(variable, "[(") {
		return "", false
	}

	root, rest, _ := strings.Cut(variable, ".")
	if strings.EqualFold(root, "variables") || strings.EqualFold(root, "this") {
		root, _, _ = strings.Cut(rest, ".")
	} else if isScopeName(root) {
		return "", false
	}

	if root == "" || pr.Declares(root) {
		return "", false
	}

	// A closure's locals and parameters and a catch variable are the
	// enclosing function's to the body scan, and not in the file-wide
	// declarations.
	if scope := pr.FuncScopeAt(int(line)); scope.Start != -1 {
		if pr.FuncDeclares(scope.Start, scope.End, root) {
			return "", false
		}
	}

	return r.fileMissingBase(pr, baseDir)
}

// isScopeName reports whether name is a CFML scope other than variables and
// this, whose members a base component does not supply.
func isScopeName(name string) bool {
	switch strings.ToLower(name) {
	case "arguments", "local", "request", "session", "application", "server",
		"url", "form", "cgi", "cookie", "client", "super", "attributes", "caller", "thread":
		return true
	default:
		return false
	}
}

// MissingBase returns the first component in the extends chain starting at
// extends, from a file in baseDir, that does not resolve to a file; "" when
// every link resolves, or the chain loops.
func (r *Resolver) MissingBase(extends, baseDir string) string {
	base, _ := r.chainBreak(extends, baseDir, false)

	return base
}

// chainBreak walks the extends chain from extends and names the first link
// that does not resolve, and whether any link up to it was one a framework
// preset implied rather than one a file wrote — implied says so of the first.
// A list of alternatives is a component a file cannot declare, so it answers
// nothing, unless it was implied: a Wheels view's base is its controller and
// every mixin, and any of them missing is the framework's source missing.
func (r *Resolver) chainBreak(extends, baseDir string, implied bool) (string, bool) {
	seen := make(map[string]bool)

	for extends != "" {
		if strings.Contains(extends, "|") {
			if !implied {
				return "", false
			}

			for alt := range strings.SplitSeq(extends, "|") {
				if base, _ := r.chainBreak(alt, baseDir, true); base != "" {
					return base, true
				}
			}

			return "", false
		}

		p := r.ComponentPath(extends, baseDir)
		if p == "" {
			return extends, implied
		}

		if seen[p] {
			return "", false
		}

		seen[p] = true

		// A chain that reaches a framework's stubs and breaks beyond them is
		// the framework's source not being here — BaseTestCase's TestBox base
		// in a workspace naming ColdBox but not TestBox — not a finding.
		implied = implied || frameworkapi.IsStub(p)

		declared, ok := r.declaredExtendsOf(p, cfpath.ToURI(p))
		if !ok {
			return "", false
		}

		extends, baseDir = declared, filepath.Dir(p)
		if declared == "" {
			extends = r.extendsFor("", p)
			implied = implied || extends != ""
		}
	}

	return "", false
}

// matchResolver tries the componentResolvers against text. It reports the
// component, the resolver's noFollow, and whether that resolver is
// dynamicIfMissing; describe is told about a match, for the trace.
func (r *Resolver) matchResolver(text string, tr *callTrace, describe func(comp, desc string, noFollow bool)) (comp string, noFollow, soft bool) {
	comp, noFollow, idx := parser.ResolveFromCallMatch(text, r.Resolvers)
	soft = idx >= 0 && r.Resolvers[idx].DynamicIfMissing

	if comp != "" && tr != nil {
		describe(comp, r.describeResolver(idx), noFollow)
	}

	return comp, noFollow, soft
}

// missingChainHop is canResolveCall's answer when a hop of a chained call is
// not a method of the component the chain has reached.
func (r *Resolver) missingChainHop(comp, softComp, hop, funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	if !r.componentExists(comp, baseDir) {
		if softMissing(comp, softComp, pr) {
			tr.addf("%q names no file and came from a dynamicIfMissing resolver — the rest of the chain is dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		if r.uninstalledModule(comp) {
			tr.addf("%q is a module's model and no such module is in the workspace — the rest of the chain is dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		if _, ok := engineComponent(comp); ok {
			tr.addf("%q is a component the engine provides and its source is not in the workspace — the rest of the chain is dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		if frameworkapi.NamespaceOf(comp) != "" {
			tr.addf("%q is in the %s namespace and the bundled stubs do not hold it — the rest of the chain is dynamic", comp, frameworkapi.NamespaceOf(comp))
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		return "component '" + displayComponent(comp) + "' does not exist (chain hop '" + hop + "' to '" + funcName + "')"
	}

	if mockDecoration(hop) {
		tr.addf("%q is a method MockBox adds to a mock — chain hop accepted, the rest of the chain is dynamic", hop)
		tr.hit(TargetDynamic, comp, nil)

		return ""
	}

	// A hop onMissingMethod answers is as valid as a last call it answers, and
	// what it returns is whatever the dispatcher decides. Lucee's own Http and
	// Query build their setters this way: new Http().setUrl(u).send().
	if r.ResolveFunc(comp, "onMissingMethod", baseDir) != nil {
		tr.addf("%q defines onMissingMethod — chain hop %q accepted, the rest of the chain is dynamic", comp, hop)
		tr.hit(TargetDynamic, comp, nil)

		return ""
	}

	if r.isInterface(r.ComponentPath(comp, baseDir)) {
		tr.addf("%q is an interface, and what implements it may have %q — the rest of the chain is dynamic", comp, hop)
		tr.hit(TargetDynamic, comp, nil)

		return ""
	}

	if base, implied := r.componentMissingBase(comp, baseDir); base != "" {
		if implied {
			tr.addf("%q's framework base does not resolve (%s) — chain hop %q accepted, the rest of the chain is dynamic", comp, base, hop)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		return MissingBaseReason(base)
	}

	return "method '" + hop + "' not found in " + displayComponent(comp) + " (chain to '" + funcName + "')"
}

// componentMissingBase is MissingBase for the chain comp extends: a method
// comp does not declare may be its base's, and when a link of that chain
// names no file the method was never looked for. ContentBox's services extend
// cborm's VirtualEntityService, and without cborm on disk every findWhere,
// save and list was "not found in" the service.
func (r *Resolver) componentMissingBase(comp, baseDir string) (base string, implied bool) {
	if strings.Contains(comp, "|") {
		return "", false
	}

	p := comp
	if !filepath.IsAbs(p) {
		p = r.ComponentPath(comp, baseDir)
	}

	if p == "" {
		return "", false
	}

	declared, ok := r.declaredExtendsOf(p, cfpath.ToURI(p))
	if !ok {
		return "", false
	}

	ext := declared
	if ext == "" {
		ext = r.extendsFor("", p)
	}

	return r.chainBreak(ext, filepath.Dir(p), declared == "" && ext != "")
}
