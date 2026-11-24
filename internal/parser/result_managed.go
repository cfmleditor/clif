package parser

import (
	"strings"
	"sync"
)

// prepareManagedSetterLookup reads properties before signatures so declaration
// order cannot change DI/1 eligibility. The metadata parse has no injection
// callbacks, and therefore does not recurse. Global edits rebuild this snapshot.
func (pr *ParseResult) prepareManagedSetterLookup() {
	pr.managedSetterLookup = pr.SetterLookup
	if pr.SetterLookup == nil || pr.PropertyBeanLookup == nil {
		return
	}

	metadata := pr.managedPropertyMetadata()

	properties := make(map[string]map[string]string, len(metadata.Properties))
	for _, prop := range metadata.Properties {
		if !metadata.Accessors && !metadata.Persistent {
			continue
		}

		properties[strings.ToLower(prop.name)] = prop.attrs
	}

	pr.managedSetterLookup = func(name string) string {
		attrs, found := properties[strings.ToLower(name)]
		// setter=false omits the implicit property setter, allowing an explicit one.
		if found && !strings.EqualFold(attrs["setter"], "false") && pr.PropertyBeanLookup(name, attrs) == "" {
			return ""
		}

		return pr.SetterLookup(name)
	}
}

func (pr *ParseResult) propertyBeanComponent(prop *propertyDef) string {
	if pr.PropertyBeanLookup != nil && !pr.Accessors && !pr.Persistent {
		return ""
	}

	if pr.BeanLookup != nil {
		lookup := pr.BeanLookup
		if pr.PropertyBeanLookup != nil {
			lookup = func(name string) string { return pr.PropertyBeanLookup(name, prop.attrs) }
		}

		if inject := prop.attrs["inject"]; inject != "" {
			if comp := lookup(normalizeBeanKey(inject)); comp != "" {
				return comp
			}

			if comp := lookup(extractBeanName(inject)); comp != "" {
				return comp
			}
		}

		if comp := lookup(prop.name); comp != "" {
			return comp
		}
	}

	if pr.PropertyBeanLookup != nil {
		return ""
	}

	inject, ok := prop.attrs["inject"]
	if ok && inject == "" {
		inject = prop.name
	}

	return injectedComponent(inject)
}

func (pr *ParseResult) applyManagedArgumentTypes(name, access string, args []Argument) {
	applySetterArgumentTypes(name, access, args, pr.managedSetterLookup)
	applyConstructorArgumentTypes(name, access, args, pr.ConstructorLookup)
}

func (pr *ParseResult) argumentTypeLookup() func(string, string, []Argument) {
	if pr.SetterLookup == nil && pr.ConstructorLookup == nil {
		return nil
	}

	return pr.applyManagedArgumentTypes
}

func callHopName(hop string) string {
	if expression, ok := CallExpression(hop); ok {
		name, _, _ := strings.Cut(expression, "(")
		_, name, _ = strings.CutLast("."+name, ".")

		return strings.TrimSpace(name)
	}

	return hop
}

// memoFuncLookup answers each (component, method) pair once for the length of
// one signature pass. A component's methods are asked about once per call site,
// and a large file names the same few thousands of times: on a 65,000-line
// component the repeats were most of what a reparse cost. The answer depends on
// the index and the filesystem, which the pass does not change, so it is held
// only while the pass runs and never across an edit.
func memoFuncLookup(lookup func(component, funcName string) string) func(component, funcName string) string {
	type key struct{ component, funcName string }

	var mu sync.Mutex

	seen := make(map[key]string)

	return func(component, funcName string) string {
		k := key{component, funcName}

		mu.Lock()
		answer, ok := seen[k]
		mu.Unlock()

		if ok {
			return answer
		}

		answer = lookup(component, funcName)

		mu.Lock()
		seen[k] = answer
		mu.Unlock()

		return answer
	}
}

// aliasComponent is what `varName = baseVar` copies: what baseVar holds at
// the assignment, read from the function's refs and then the component's, as
// a receiver is. It waits for the member pass, so a member is answered too:
// tassweb's itext.cfc builds its helpers into `temp` in init() and keeps them
// with `VARIABLES._itextUtils = temp.itextUtils`, and every method calling
// one returned an untyped value.
func (pr *ParseResult) aliasComponent(c *pendingCall, globals []ComponentRef) string {
	if globals == nil {
		globals = pr.ComponentRefs
	}

	scope, name := c.baseScope, c.baseVar
	if lower := strings.ToLower(name); strings.HasPrefix(lower, "variables.") || strings.HasPrefix(lower, "this.") || strings.HasPrefix(lower, "local.") {
		_, name, _ = strings.Cut(name, ".")
	}

	var ref *ComponentRef

	if scope == RefAny && c.funcKey.in {
		ref = refReaching(pr.funcRefsMap[c.funcKey], name, c.line, func(r *ComponentRef) bool { return r.VisibleTo == 0 && !r.HasClosureSpan })
	}

	if ref == nil && !c.baseLocal {
		ref = refReaching(globals, name, c.line, scope.Admits)
	}

	if ref == nil {
		return ""
	}

	return pr.settledComponent(ref)
}

// guessedMethod is the method a ref typed by its receiver's own type was not
// looked up for, which the resolver may look up (FunctionDef.ReturnVia).
// ChainBase stays empty, so applyChainedReturnLookup leaves the ref alone.
func guessedMethod(c *pendingCall) string {
	if !c.baseGuess {
		return ""
	}

	return c.funcName
}
