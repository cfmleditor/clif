package resolve

import (
	"strings"
	"testing"
)

// The resolver's tag readers read tags as the parse does (parser.CFTags): a
// commented-out tag is none, and a '>' inside a quoted value does not end a
// tag. Each read its tags with a regular expression of its own, which took
// both.
func TestTheResolversTagReadersSkipCommentsAndQuotedGT(t *testing.T) {
	const commented = "<!--- <cfloop array=\"#ghosts#\" index=\"g\"></cfloop> --->\n"

	t.Run("cfloop", func(t *testing.T) {
		src := commented + `<cfloop array="#items#" index="it" hint="a > b">` + "\n<cfset x = it>\n</cfloop>"

		loops := tagLoops(src)
		if len(loops) != 1 || loops[0].collection != "items" || loops[0].name != "it" {
			t.Fatalf("tagLoops = %+v", loops)
		}
	})

	t.Run("init cfargument", func(t *testing.T) {
		src := `<!--- <cffunction name="init"><cfargument name="entityName" default="Ghost"></cffunction> --->` +
			`<cffunction name="init" hint="a > b"><cfargument name="entityName" default="Real"></cffunction>`

		if got := initArgumentDefault([]byte(src), "entityName"); got != "Real" {
			t.Fatalf("initArgumentDefault = %q", got)
		}
	})

	t.Run("cffunction bodies", func(t *testing.T) {
		src := `<!--- <cffunction name="ghost">x</cffunction> --->` +
			`<cffunction name="real" hint="a > b">body</cffunction>`

		bodies := tagFunctionBodies(src)
		if _, ghost := bodies["ghost"]; ghost || bodies["real"] != "body" {
			t.Fatalf("tagFunctionBodies = %q", bodies)
		}
	})

	t.Run("cfreturn", func(t *testing.T) {
		src := "<!--- <cfreturn other> --->\n<cfreturn items[1]>"

		facts := tagReturns(src)
		if facts.all != 1 || len(facts.reads) != 1 || facts.reads[0].name != "items" {
			t.Fatalf("tagReturns = %+v", facts)
		}
	})

	t.Run("shared cfset", func(t *testing.T) {
		src := `<!--- <cfset request.ghost = createObject("component", "a.Ghost")> --->` +
			`<cfset request.real = createObject("component", "a.Real")>`

		var keys []string
		for _, a := range sharedAssignments(src, "") {
			keys = append(keys, a.key)
		}

		if strings.Join(keys, ",") != "request.real" {
			t.Fatalf("sharedAssignments keys = %q", keys)
		}
	})
}
