package resolve

import (
	"regexp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// The logging admin's create page reads its registry containers through a
// name the page itself never spells:
//
//	<cfloop list="appender,layout" item="_name">
//		<cfset drivers=variables[_name&"s"]>
//		<cfloop collection="#drivers#" index="driverClass" item="driver">
//
// _name is the list-loop item, bound to the literals appender and layout;
// each turn's concatenation names one container the recipes fill. The loop's
// item is then a member of whichever turn's family holds the receiver — and
// the turns' families the receiver cannot be told apart statically, so it is
// a comma-joined pair of family markers, and a call on it is answered when
// every one of them answers it, since each holds the receiver on its turn.
// A turn whose container is not one of the registries declines the pickup.

// concatPickupRe is the read of the variables scope under a name an
// identifier concatenated with a literal spells.
var concatPickupRe = regexp.MustCompile(`(?i)^variables\.?\[\s*([A-Za-z_]\w*)\s*&\s*["']([^"']*)["']\s*\]$`)

// indirectRegistryPickup is the families the variables read of rhs holds,
// when its name is a bound list-loop item's concatenation with a literal and
// the turns' containers are the template's own registries — line is the loop
// header the read fills the collection of.
func (r *Resolver) indirectRegistryPickup(rhs string, line int, pr *parser.ParseResult) string {
	m := concatPickupRe.FindStringSubmatch(strings.TrimSpace(rhs))
	if m == nil {
		return ""
	}

	var (
		markers  []string
		registry map[string]bool
	)

	for _, literal := range listLoopItems(pr.Content, m[1], line) {
		// A turn whose container is not a registry holds something this
		// cannot prove on that turn, so the receiver is unproven.
		fact, ok := r.registryFamily(strings.ToLower(literal+m[2]), pr)
		if !ok {
			return ""
		}

		if registry == nil {
			registry = map[string]bool{}
		}

		if registry[fact.container] {
			continue
		}

		registry[fact.container] = true
		markers = append(markers, familyComp(fact))
	}

	return strings.Join(markers, ",")
}

// listLoopItems is the literal items of the list the cfloop whose item= is
// name binds, when the loop's span holds line: `<cfloop list="a,b" item="x">`
// binds x to a and b over its body. A list the header spells dynamically
// binds nothing this can read.
func listLoopItems(content, name string, line int) []string {
	lines := newLineIndex(content)

	for _, m := range loopTagPartsRe.FindAllStringSubmatchIndex(content, -1) {
		if content[m[2]:m[3]] != "" { // a close tag
			continue
		}

		attrs := loopAttrs(content[m[4]:m[5]])

		list, ok := attrs["list"]
		if !ok || !strings.EqualFold(attrs["item"], name) {
			continue
		}

		end, ok := tagLoopEnd(content, m[1])
		if !ok {
			continue
		}

		if line < lines.lineOf(m[1]) || line > lines.lineOf(end) {
			continue
		}

		var items []string

		for item := range strings.SplitSeq(list, ",") {
			item = strings.TrimSpace(item)

			if item == "" || strings.ContainsAny(item, "#$") {
				return nil
			}

			if !slices.Contains(items, item) {
				items = append(items, item)
			}
		}

		return items
	}

	return nil
}
