package resolve

import (
	"slices"
	"strconv"
	"strings"
)

// The readers code_calls_test.go holds to its cases, each reduced to the
// strings it read.

func readViewRenders(text string) []string {
	var out []string

	for _, r := range viewRenders(text, "app") {
		if r.view != "" {
			out = append(out, r.view)
		} else {
			out = append(out, "#"+r.variable+"#"+r.rest)
		}
	}

	return out
}

func readRenderSites(text string) []string {
	var out []string

	for view, sites := range renderSites(text, "f.cfm", "app") {
		for _, s := range sites {
			out = append(out, view+"@"+strconv.Itoa(s.line))
		}
	}

	slices.Sort(out)

	return out
}

func readInterceptSites(text string) []string {
	var out []string

	for _, s := range interceptSitesIn(text, "f.cfc") {
		out = append(out, strconv.Itoa(s.line))
	}

	return out
}

func readMuraObjectSites(text string) []string {
	var out []string

	for _, s := range muraObjectSitesIn(text, "f.cfm") {
		out = append(out, strconv.Itoa(s.line))
	}

	if blanked := withoutMuraObjectArgs(text); blanked != text {
		out = append(out, "blanked:"+strconv.Itoa(strings.Count(blanked, "theFile")))
	}

	return out
}

func readPrcResponse(text string) []string {
	if writesPrcResponse(tokenize(text)) {
		return []string{"writes"}
	}

	return nil
}

func readCustomErrorTemplates(text string) []string { return customErrorTemplateValues(text) }

func readFW1SetViews(text string) []string {
	var out []string

	for _, sv := range fw1SetViewsIn(text) {
		out = append(out, sv.subsystem+":"+sv.target)
	}

	return out
}

func readFW1ViewCalls(text string) []string {
	var out []string

	for _, c := range fw1ViewCalls(text) {
		out = append(out, c.subsystem+":"+c.view)
	}

	return out
}

func readFW1RcCalls(text string) []string { return fw1RcCalls(text) }

func readAssociations(text string) []string {
	var out []string

	for _, a := range wheelsAssociationCalls(text) {
		out = append(out, strings.ToLower(a.kind)+":"+a.name+":"+a.model)
	}

	return out
}

func readBinderMaps(text string) []string {
	var out []string

	for _, m := range binderMaps(text) {
		out = append(out, strings.ToLower(m[0])+"="+m[1])
	}

	return out
}

func readSuperInit(text string) []string {
	entity, arg := superInitEntity(text)

	switch {
	case entity != "":
		return []string{entity}
	case arg != "":
		return []string{"arguments." + arg}
	default:
		return nil
	}
}

func readNamesApplication(text string) []string {
	if namesAnApplication(text) {
		return []string{"named"}
	}

	return nil
}

func readMemento(text string) []string {
	if declaresMemento(text) {
		return []string{"memento"}
	}

	return nil
}

func readModuleHelpers(text string) []string { return helperSetting(text, true) }

func readAppHelpers(text string) []string { return helperSetting(text, false) }

func readIncludedUDFs(text string) []string { return includedUDFPaths(text) }

func readComputedIncludes(text string) []string {
	var out []string

	for _, at := range computedIncludeOffsets(text) {
		out = append(out, strconv.Itoa(strings.Count(text[:at], "\n")))
	}

	return out
}

func readPackageListings(text string) []string {
	var out []string

	for _, l := range packageListings(text) {
		out = append(out, l.pkg+">"+l.target)
	}

	return out
}

func readRegistryContainer(text string) []string {
	if c, ok := registryContainer(text, "name"); ok {
		return []string{c}
	}

	return nil
}

func readRegistrySkips(text string) []string { return registrySkips(text, "name") }

func readFactoryBodyFills(text string) []string {
	if factoryBodyFills(text, "arr") {
		return []string{"fills"}
	}

	return nil
}
