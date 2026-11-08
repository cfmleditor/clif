// The component an injected or generated property holds: inject ids, the injection DSL, accessors and relationships.

package parser

import (
	"slices"
	"strings"
	"sync"
)

// extractBeanName strips framework namespace prefixes from an inject value.
// Handles: "model:UserService" → "UserService", "UserDAO@model" → "UserDAO",
// "coldbox:setting:appName" → "appName", "userService" → "userService".
func extractBeanName(inject string) string {
	inject = strings.TrimSpace(inject)
	// WireBox @-style: "BeanName@namespace"
	if at := strings.IndexByte(inject, '@'); at > 0 {
		return inject[:at]
	}
	// Colon-namespaced: take the last segment after ':'
	if colon := strings.LastIndexByte(inject, ':'); colon >= 0 {
		return inject[colon+1:]
	}

	return inject
}

// normalizeBeanKey converts an inject value to the bean map key format.
// "UserDAO@model" → "userdao@model", "model:UserService" → "userservice@model",
// "userService" → "userservice".
func normalizeBeanKey(inject string) string {
	inject = strings.TrimSpace(inject)
	// Already in @-style: "BeanName@namespace"
	if at := strings.IndexByte(inject, '@'); at > 0 {
		return strings.ToLower(inject[:at]) + "@" + strings.ToLower(inject[at+1:])
	}
	// Colon-namespaced: "namespace:BeanName" → "beanname@namespace"
	if colon := strings.LastIndexByte(inject, ':'); colon >= 0 {
		ns := inject[:colon]
		name := inject[colon+1:]
		// If namespace itself has colons (e.g. "coldbox:setting"), use last segment as ns
		if innerColon := strings.LastIndexByte(ns, ':'); innerColon >= 0 {
			ns = ns[innerColon+1:]
		}

		return strings.ToLower(name) + "@" + strings.ToLower(ns)
	}

	return strings.ToLower(inject)
}

// ucFirst capitalizes the first character of a string.
func ucFirst(s string) string {
	if s == "" {
		return s
	}

	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}

	return s
}

// generatePropertyAccessors creates synthetic get/set FunctionDefs and ComponentRefs
// for properties, skipping any where an explicit or already-generated function exists.
func (pr *ParseResult) generatePropertyAccessors() {
	if len(pr.Properties) == 0 {
		return
	}
	// Build set of existing function names (explicit + previously generated)
	existing := make(map[string]bool, len(pr.Funcs)+len(pr.Properties)*2)
	for i := range pr.Funcs {
		f := &pr.Funcs[i]

		existing[strings.ToLower(f.Name)] = true
	}

	u := pr.URI

	// A generated setter returns the component, for chaining — Lucee's and
	// Adobe's alike — so it is typed as `return this;` is: by the file's path.
	self, _ := strings.CutPrefix(string(u), "file://")

	fieldComponents := pr.propertyFieldComponents()

	for _, prop := range pr.Properties {
		capName := ucFirst(prop.name)

		getterIdx := -1

		getter := "get" + strings.ToLower(prop.name)
		if !existing[getter] {
			existing[getter] = true
			getterIdx = len(pr.Funcs)

			getterDef := FunctionDef{
				Name: "get" + capName, URI: u, Line: prop.line,
				ReturnType: prop.typeName,
				Generated:  true,
			}
			if pr.Persistent {
				getterDef.ElementComponent = prop.collectionEntity()
			}

			pr.Funcs = append(pr.Funcs, getterDef)
		} else if pr.Persistent {
			pr.overriddenRelationshipGetter(getter, &prop)
		}

		setter := "set" + strings.ToLower(prop.name)
		if !existing[setter] {
			existing[setter] = true

			pr.Funcs = append(pr.Funcs, FunctionDef{
				Name: "set" + capName, URI: u, Line: prop.line,
				Generated:       true,
				Arguments:       []Argument{{Name: prop.name, Type: prop.typeName}},
				ReturnComponent: self,
			})
		}

		for _, m := range relationshipMethods(&prop) {
			if !existing[strings.ToLower(m)] {
				existing[strings.ToLower(m)] = true

				pr.Funcs = append(pr.Funcs, FunctionDef{Name: m, URI: u, Line: prop.line, Generated: true})
			}
		}

		comp := pr.propertyComponent(&prop)

		// Bean-ORM collection relationships use the same cfc bean id as their
		// single-valued counterparts, but the generated getter returns the
		// collection rather than that component. Keep the proven component as
		// the collection's element contract instead of a receiver contract.
		pr.setRelationshipElement(getterIdx, &prop)

		if comp != "" {
			pr.ComponentRefs = append(pr.ComponentRefs, ComponentRef{
				Variable: prop.name, Component: comp, URI: u, Line: prop.line,
			})

			// A generated getter returns the property, so it returns what
			// the property holds: cborm's getWireBox() is the injector.
			if getterIdx >= 0 {
				pr.Funcs[getterIdx].ReturnComponent = comp
			}
		} else if getterIdx >= 0 && (prop.typeName == "" || strings.EqualFold(prop.typeName, "any")) {
			pr.Funcs[getterIdx].ReturnComponent = fieldComponents[strings.ToLower(prop.name)]
		}
	}
}

// overriddenRelationshipGetter gives an entity's own getter for a
// relationship collection the element the generated one would have had: it
// stands in for the accessor, so it returns the relationship's collection.
// Slatwall's Product declares `array function getSkus( sorted = false )`,
// which hands back the skus or a sorted copy of them. Only a getter whose
// declared type is the collection's (array, or none) is read so.
func (pr *ParseResult) overriddenRelationshipGetter(getter string, prop *propertyDef) {
	elem := prop.collectionEntity()
	if elem == "" {
		return
	}

	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		if f.Generated || f.ElementComponent != "" || !strings.EqualFold(f.Name, getter) {
			continue
		}

		switch strings.ToLower(f.ReturnType) {
		case "", "any", "array":
			f.ElementComponent = elem
		}
	}
}

func (pr *ParseResult) propertyComponent(prop *propertyDef) string {
	if len(pr.PropertyResolvers) > 0 && len(prop.attrs) > 0 {
		if comp := ResolveProperty(prop.attrs, pr.PropertyResolvers); comp != "" {
			return comp
		}
	}

	if prop.typeName != "" && looksLikeCFCType(prop.typeName) {
		return prop.typeName
	}

	if comp := prop.documentedComponent(); comp != "" {
		return comp
	}

	related := prop.relatedEntity()
	if pr.Persistent && related != "" {
		return related
	}

	if related != "" && pr.BeanLookup != nil {
		if comp := pr.BeanLookup(related); comp != "" {
			return comp
		}
	}

	if related != "" {
		resolved := ResolveFromCall(`getBean("`+related+`")`, pr.Resolvers)
		if resolved != "" && !strings.EqualFold(resolved, related) {
			return resolved
		}
	}

	return pr.propertyBeanComponent(prop)
}

func (pr *ParseResult) setRelationshipElement(getterIdx int, prop *propertyDef) {
	if getterIdx < 0 || pr.Persistent || prop.collectionEntity() == "" {
		return
	}

	pr.Funcs[getterIdx].ElementComponent = pr.relationshipBeanComponent(prop)
}

func (pr *ParseResult) relationshipBeanComponent(prop *propertyDef) string {
	related := prop.collectionEntity()
	if related == "" {
		return ""
	}

	if pr.BeanLookup != nil {
		if comp := pr.BeanLookup(related); comp != "" {
			return comp
		}
	}

	resolved := ResolveFromCall(`getBean("`+related+`")`, pr.Resolvers)
	if strings.EqualFold(resolved, related) {
		return ""
	}

	return resolved
}

// collectionEntity is the entity each element of a collection ORM
// relationship holds: a one-to-many or many-to-many property names it in its
// cfc attribute. ContentBox loops over them: `for ( var c in getComments() )`.
func (prop *propertyDef) collectionEntity() string {
	switch strings.ToLower(prop.attrs["fieldtype"]) {
	case "one-to-many", "many-to-many":
	default:
		return ""
	}

	cfc := strings.TrimSpace(prop.attrs["cfc"])
	if cfc == "" || strings.Contains(cfc, "#") {
		return ""
	}

	return cfc
}

// relatedEntity is the entity a single-valued ORM relationship holds: a
// many-to-one or one-to-one property names it in its cfc attribute, so the
// generated getter returns one. ContentBox's content items reach their site
// through `property name="site" fieldtype="many-to-one" cfc="…system.Site"`.
// A collection relationship holds an array or struct of them, not one.
func (prop *propertyDef) relatedEntity() string {
	switch strings.ToLower(prop.attrs["fieldtype"]) {
	case "many-to-one", "one-to-one":
	default:
		return ""
	}

	if prop.typeName != "" && !strings.EqualFold(prop.typeName, "any") {
		return ""
	}

	cfc := strings.TrimSpace(prop.attrs["cfc"])
	if cfc == "" || strings.Contains(cfc, "#") {
		return ""
	}

	return cfc
}

// CFML's doc_generic property metadata names the value's component, just as
// argument.doc_generic does for a generic argument. Preserve explicit types
// and do not treat an array's element type as its receiver.
func (prop *propertyDef) documentedComponent() string {
	if prop.typeName != "" && !strings.EqualFold(prop.typeName, "any") && !strings.EqualFold(prop.typeName, "struct") {
		return ""
	}

	if documented := prop.attrs["doc_generic"]; isComponentType(documented) {
		return documented
	}

	return ""
}

// A generated getter reads variables.name. Constructor assignments
// already type that field, even when the property has no metadata.
// Locals and this.name are separate stores; unresolved call chains
// cannot supply a getter type. Conflicting field types stay dynamic.
func (pr *ParseResult) propertyFieldComponents() map[string]string {
	fieldComponents := make(map[string]string, len(pr.ComponentRefs))
	for i := range pr.ComponentRefs {
		ref := &pr.ComponentRefs[i]
		if ref.This {
			continue
		}

		candidate := ref.Component
		if candidate == "" || ref.ChainBase != "" || chainPending(ref) {
			candidate = "$any"
		}

		key := strings.ToLower(ref.Variable)
		if previous := fieldComponents[key]; previous != "" && !strings.EqualFold(previous, candidate) {
			candidate = "$any"
		}

		fieldComponents[key] = candidate
	}

	return fieldComponents
}

// PresideObject is what Preside's presidecms:object:x DSL injects: the
// object x, decorated by PresideObjectDecorator with the data API its
// onMissingMethod hands to the PresideObjectService. The decorator stands
// for it, as the resolver's answer for getObject() does; without Preside's
// source a call on it is dynamic (IsSoftComponent).
const PresideObject = "preside.system.services.presideObjects.PresideObjectDecorator"

// injectedComponent is the component a WireBox injection names, when no
// beanPaths entry or propertyResolver said.
//
// `inject="id:settingService@contentbox"`, `inject="model:UserService"` and
// `inject="provider:UserService"` name the component UserService, which the
// resolver then finds by path and, failing that, by file name — WireBox's own
// convention is that a model's id is its file's. A provider stands in for the
// object it provides, so its methods are that object's. A dotted id is a path
// already.
//
// The DSL's namespaces name the frameworks' own objects, and are answered by
// their ColdBox dot-paths: `coldbox` the controller, `coldbox:requestService`
// a service, `wirebox:populator` the object populator, `logbox:logger:{this}`
// a logger, `cachebox:template` a cache — the provider ColdBox configures its
// caches with, since the interface they share declares less than each has. So
// are the models ColdBox registers as `Name@coldbox`, which a file-name search
// would find whatever the workspace calls. Without ColdBox's source or its
// stubs, a call on any of them is dynamic, not a component that is missing
// (IsSoftComponent). Whatever else the DSL names is not a component — `coldbox:setting:x`
// is a setting, `coldbox:moduleSettings:x` a struct — and resolving it by
// name would find an unrelated file.
func injectedComponent(inject string) string {
	id := strings.TrimSpace(inject)

	// A provider stands in for what it provides, DSL included:
	// `provider:cachebox` is the CacheFactory.
	if p := "provider:"; len(id) > len(p) && strings.EqualFold(id[:len(p)], p) {
		id = id[len(p):]
	}

	// So does Preside's delayedInjector:x (config/WireBox.cfc maps the DSL):
	// it injects a DelayedInjector, whose onMissingMethod hands every call
	// to the instance x names once it is first used. Only a model id: what
	// a DSL name like `coldbox` is depends on the application (Preside runs
	// its own controller), so it is left to the resolvers, as it was.
	if p := "delayedInjector:"; len(id) > len(p) && strings.EqualFold(id[:len(p)], p) {
		id = id[len(p):]
		if dslComponent(id) != "" || strings.Contains(id, ":") {
			return ""
		}
	}

	if c := dslComponent(id); c != "" {
		return c
	}

	if p := "presidecms:object:"; len(id) > len(p) && strings.EqualFold(id[:len(p)], p) {
		return PresideObject
	}

	for _, prefix := range []string{"id:", "model:"} {
		if len(id) > len(prefix) && strings.EqualFold(id[:len(prefix)], prefix) {
			id = id[len(prefix):]

			break
		}
	}

	name, module, qualified := strings.Cut(id, "@")
	if qualified {
		if c, found := coldboxModels[strings.ToLower(name)]; found && strings.EqualFold(module, "coldbox") {
			return coldboxSystem + c
		}
	}

	if name == "" || strings.ContainsAny(name, ":{}$#/\\ ") {
		return ""
	}

	var buf foldScratch
	switch string(buf.lowerFold(name)) {
	case "box", "executor", "java", "entityservice":
		return ""
	}

	// The module is kept: `X@cbstorages` is X as that module registers it,
	// which the resolver finds in the module's own models and, with no such
	// module in the workspace, takes as the module not being installed
	// (resolve.wireboxID).
	if qualified && module != "" && !strings.ContainsAny(module, ":{}$#/\\ ") {
		return name + "@" + module
	}

	return name
}

const coldboxSystem = "coldbox.system."

// injectionDSL maps a WireBox DSL string, lowercased, to the class it
// injects, under coldbox.system. Measured over the corpus: the namespaces a
// property uses and that hold a component.
var injectionDSL = map[string]string{
	"coldbox":                    "web.Controller",
	"coldbox:flash":              "web.flash.AbstractFlashScope",
	"coldbox:renderer":           "web.Renderer",
	"coldbox:requestcontext":     "web.context.RequestContext",
	"coldbox:router":             "web.routing.Router",
	"coldbox:requestservice":     "web.services.RequestService",
	"coldbox:interceptorservice": "web.services.InterceptorService",
	"coldbox:moduleservice":      "web.services.ModuleService",
	"coldbox:routingservice":     "web.services.RoutingService",
	"coldbox:handlerservice":     "web.services.HandlerService",
	"coldbox:loaderservice":      "web.services.LoaderService",
	"coldbox:schedulerservice":   "web.services.SchedulerService",
	"coldbox:asyncmanager":       "async.AsyncManager",
	"wirebox":                    "ioc.Injector",
	"wirebox:root":               "ioc.Injector",
	"wirebox:binder":             "ioc.config.Binder",
	"wirebox:populator":          "core.dynamic.ObjectPopulator",
	"wirebox:asyncmanager":       "async.AsyncManager",
	"cachebox":                   "cache.CacheFactory",
	"logbox":                     "logging.LogBox",
	"logbox:root":                "logging.Logger",
}

// coldboxModels are the models ColdBox registers under its own module, as
// `Name@coldbox`: a file-name search for them would find whatever the
// workspace holds by that name.
var coldboxModels = map[string]string{
	"htmlhelper":     "modules.HTMLHelper.models.HTMLHelper",
	"renderer":       "web.Renderer",
	"datamarshaller": "core.conversion.DataMarshaller",
	"xmlconverter":   "core.conversion.XMLConverter",
}

// isInjectedFrameworkComponent reports whether comp is one of
// InjectedFrameworkComponents.
func isInjectedFrameworkComponent(comp string) bool {
	if strings.EqualFold(comp, PresideObject) {
		return true
	}

	if len(comp) <= len(coldboxSystem) || !strings.EqualFold(comp[:len(coldboxSystem)], coldboxSystem) {
		return false
	}

	injectedOnce.Do(func() {
		injected = map[string]bool{}
		for _, c := range InjectedFrameworkComponents() {
			injected[strings.ToLower(c)] = true
		}
	})

	return injected[strings.ToLower(comp)]
}

var (
	injectedOnce sync.Once
	injected     map[string]bool
)

// InjectedFrameworkComponents lists every framework class an injection can
// name, sorted: the bundled framework API must hold each of them.
func InjectedFrameworkComponents() []string {
	out := make([]string, 0, 2+len(injectionDSL)+len(coldboxModels))

	out = append(out, coldboxSystem+"logging.Logger", coldboxSystem+"cache.providers.CacheBoxColdBoxProvider")
	for _, c := range injectionDSL {
		out = append(out, coldboxSystem+c)
	}

	for _, c := range coldboxModels {
		out = append(out, coldboxSystem+c)
	}

	slices.Sort(out)

	return slices.Compact(out)
}

// InjectionDSL maps each WireBox DSL string that names one of ColdBox's
// classes, lowercased, to that class: what `getInstance( "logbox:root" )`
// returns as much as what `inject="logbox:root"` injects.
func InjectionDSL() map[string]string {
	out := make(map[string]string, len(injectionDSL))
	for k, c := range injectionDSL {
		out[k] = coldboxSystem + c
	}

	return out
}

// dslComponent is the framework class a DSL injection names, or "".
func dslComponent(id string) string {
	lower := strings.ToLower(id)
	if c, ok := injectionDSL[lower]; ok {
		return coldboxSystem + c
	}

	switch {
	case strings.HasPrefix(lower, "logbox:logger:"):
		return coldboxSystem + "logging.Logger"
	case strings.HasPrefix(lower, "cachebox:") && !strings.ContainsAny(lower[len("cachebox:"):], ":{}#"):
		return coldboxSystem + "cache.providers.CacheBoxColdBoxProvider"
	}

	return ""
}
