package parser

import "strings"

// applyParameterDocs recognizes both JSDoc and CFML's argument metadata.
// doc_generic describes a component only when its value is a dotted path
// and the declaration is generic; array element types are not receivers.
func applyParameterDocs(comment string, args []Argument) {
	applyJSDocParams(comment, args)

	for line := range strings.SplitSeq(comment, "\n") {
		_, annotation, ok := strings.Cut(line, "@")
		if !ok {
			continue
		}

		end := strings.IndexAny(annotation, " \t\r")
		if end < 0 {
			continue
		}

		name, kind, ok := strings.Cut(annotation[:end], ".")
		if ok && strings.EqualFold(kind, "inject") {
			applyInjectDoc(name, strings.TrimSpace(annotation[end:]), args)

			continue
		}

		if !ok || !strings.EqualFold(kind, "doc_generic") {
			continue
		}

		value := strings.TrimSpace(annotation[end:])
		if end := strings.IndexAny(value, " \t\r"); end >= 0 {
			value = value[:end]
		}

		if !isComponentType(value) {
			continue
		}

		for i := range args {
			a := &args[i]
			if strings.EqualFold(a.Name, name) && (a.Type == "" || strings.EqualFold(a.Type, "any") || strings.EqualFold(a.Type, "struct")) {
				a.Type = value
			}
		}
	}
}

// applyInjectDoc types an argument WireBox injects by its doc comment,
// `@presideObjectService.inject PresideObjectService`, as an injected property
// is typed (injectedComponent): WireBox reads an argument's inject metadata
// from the comment, and Preside's services take every dependency so. A
// declared component type outranks it, and a DSL injectedComponent does not
// read (`coldbox:setting:x`, Preside's `delayedInjector:x`) types nothing.
func applyInjectDoc(name, value string, args []Argument) {
	if end := strings.IndexAny(value, " \t\r"); end >= 0 {
		value = value[:end]
	}

	comp := injectedComponent(value)
	if comp == "" {
		return
	}

	for i := range args {
		a := &args[i]
		if strings.EqualFold(a.Name, name) && a.Component == "" && !isComponentType(a.Type) {
			a.Component = comp
		}
	}
}
