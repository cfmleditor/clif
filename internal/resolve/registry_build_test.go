package resolve

import (
	"strings"
	"testing"
)

// A service getter that builds the array it returns — the body starts a
// local empty, appends values to it, and returns it (or a variables field
// assigned from it) — proves the appended values' common component as the
// collection's element:
//
//	<cffunction name="getBeans" returntype="array">
//		<cfset var built=arrayNew(1)>
//		<cfloop from="1" to="#rs.recordcount#" index="s">
//			<cfset bean=getManagerBean()>
//			<cfset arrayAppend(built, bean)>
//		</cfloop>
//		<cfreturn built>
//	</cffunction>
//
// The factory's own return — createObject("component", literal).init(...) —
// is the element, and the consumer's keyed read types against it: a method
// the element declares resolves, one it does not stays a finding.
func TestTheCollectionAFunctionBuildsProvesItsElement(t *testing.T) {
	dir := t.TempDir()

	writeFiles(t, dir, map[string]string{
		// The service: a factory method and two array builders. The cached
		// builder returns a variables field assigned from the built array —
		// the alias the real extendManager's getter spells. The mixed builder
		// appends a value nobody proves: an open world, so no answer.
		"pkg/service.cfc": `<cfcomponent>
	<cffunction name="getManagerBean" access="public" returntype="any">
		<cfset var managerBean=createObject("component","pkg.widget").init(variables.configBean)>
		<cfset managerBean.setSiteID(getSiteID())>
		<cfreturn managerBean>
	</cffunction>

	<cffunction name="getBeans" access="public" returntype="array">
		<cfset var built=arrayNew(1)>
		<cfset var bean="">
		<cfloop from="1" to="3" index="i">
			<cfset bean=getManagerBean()>
			<cfset arrayAppend(built, bean)>
		</cfloop>
		<cfreturn built>
	</cffunction>

	<cffunction name="getCachedBeans" access="public" returntype="array">
		<cfset var built=arrayNew(1)>
		<cfset arrayAppend(built, getManagerBean())>
		<cfset variables.cached=built>
		<cfreturn variables.cached>
	</cffunction>

	<cffunction name="getMixedBeans" access="public" returntype="array">
		<cfset var built=arrayNew(1)>
		<cfset arrayAppend(built, getManagerBean())>
		<cfset arrayAppend(built, somethingUnbuilt())>
		<cfreturn built>
	</cffunction>

	<cffunction name="getSiteID" access="public" returntype="string">
		<cfreturn "">
	</cffunction>
</cfcomponent>`,
		"pkg/widget.cfc": `<cfcomponent>
	<cffunction name="init" access="public" returntype="any">
		<cfreturn this>
	</cffunction>

	<cffunction name="getSiteID" access="public" returntype="string">
		<cfreturn "">
	</cffunction>

	<cffunction name="getLabel" access="public" returntype="string">
		<cfreturn "">
	</cffunction>
</cfcomponent>`,
		// The lazy field getter: the function returns a field the file's own
		// loader fills — a local assigned an empty array, appended with the
		// constructions, assigned to the field.
		"pkg/lazy.cfc": `<cfcomponent>
	<cffunction name="getWidgets">
		<cfif not isArray(variables.instance.widgets)>
			<cfset loadWidgets()>
		</cfif>
		<cfreturn variables.instance.widgets />
	</cffunction>

	<cffunction name="loadWidgets">
		<cfset variables.instance.widgets=arrayNew(1)/>
		<cfset var w="">
		<cfloop from="1" to="2" index="i">
			<cfset w=createObject("component","pkg.widget").init()>
			<cfset arrayAppend(variables.instance.widgets, w)/>
		</cfloop>
		<cfreturn this>
	</cffunction>
</cfcomponent>`,
		"pkg/page.cfm": `<cfset service = new pkg.service()>
<cfset sets = service.getBeans()>
<cfset cached = service.getCachedBeans()>
<cfset mixed = service.getMixedBeans()>
<cfset b = sets[1]>
<cfset x = b.getSiteID()>
<cfset y = cached[1].getLabel()>
<cfset z = b.getUndeclared()>
<cfset lazy = new pkg.lazy()>
<cfset ws = lazy.getWidgets()>
<cfset lw = ws[1].getLabel()>`,
	})

	got := reasonsIn(t, dir, "pkg/page.cfm")
	expectReasons(t, got, map[string]string{
		"service.getBeans":       "",
		"service.getCachedBeans": "",
		// The mixed builder's own call is fine — the method is declared; it
		// is its element proof that the unprovable append withholds.
		"service.getMixedBeans": "",
		"b.getSiteID":           "",
		"b.getUndeclared":       "method 'getUndeclared' not found in pkg.widget",
		"cached[1].getLabel":    "",
		"ws[1].getLabel":        "",
	})
}

// A write to the collection the analysis does not account for is an open
// world: the built array or the getter's field may hold anything it wrote,
// and the element is withheld. A setter assigning the field an argument, a
// member append, an indexed write, arrayPrepend and arrayInsertAt each hand
// the collection a value nobody proved, so a gadget method read off the
// collection is not a finding against the widget the appends built. An
// append through the getter — arrayAppend(getWidgets(), x) — is one more
// proven value, and joins the element.
func TestAnUnaccountedWriteWithholdsTheBuiltElement(t *testing.T) {
	dir := t.TempDir()

	field := func(write string) string {
		return `<cfcomponent>
	<cffunction name="getWidgets">
		<cfif not isArray(variables.instance.widgets)>
			<cfset loadWidgets()>
		</cfif>
		<cfreturn variables.instance.widgets />
	</cffunction>

	<cffunction name="loadWidgets">
		<cfset variables.instance.widgets=arrayNew(1)/>
		<cfset var w=createObject("component","pkg.widget").init()>
		<cfset arrayAppend(variables.instance.widgets, w)/>
		<cfreturn this>
	</cffunction>

	<cffunction name="other">
		<cfargument name="widgets">
		` + write + `
	</cffunction>
</cfcomponent>`
	}

	builder := func(write string) string {
		return `<cfcomponent>
	<cffunction name="getBeans" returntype="array">
		<cfset var built=arrayNew(1)>
		<cfset var w=createObject("component","pkg.widget").init()>
		<cfset arrayAppend(built, w)>
		` + write + `
		<cfreturn built>
	</cffunction>
</cfcomponent>`
	}

	gadget := `createObject("component","pkg.gadget").init()`

	writeFiles(t, dir, map[string]string{
		"pkg/widget.cfc": `<cfcomponent>
	<cffunction name="init"><cfreturn this></cffunction>
	<cffunction name="getLabel"><cfreturn ""></cffunction>
</cfcomponent>`,
		"pkg/gadget.cfc": `<cfcomponent>
	<cffunction name="init"><cfreturn this></cffunction>
	<cffunction name="spin"><cfreturn ""></cffunction>
</cfcomponent>`,
		"pkg/setter.cfc":  field(`<cfset variables.instance.widgets=arguments.widgets>`),
		"pkg/member.cfc":  field(`<cfset variables.instance.widgets.append(` + gadget + `)>`),
		"pkg/indexed.cfc": field(`<cfset variables.instance.widgets[2]=` + gadget + `>`),
		"pkg/viaget.cfc": field(`<cfset var g=` + gadget + `>
		<cfset arrayAppend(getWidgets(), g)>`),
		"pkg/bplain.cfc":   builder(""),
		"pkg/bindex.cfc":   builder(`<cfset built[2]=` + gadget + `>`),
		"pkg/bmember.cfc":  builder(`<cfset built.append(` + gadget + `)>`),
		"pkg/bprepend.cfc": builder(`<cfset arrayPrepend(built, ` + gadget + `)>`),
		"pkg/binsert.cfc":  builder(`<cfset arrayInsertAt(built, 1, ` + gadget + `)>`),
		"pkg/page.cfm": `<cfset a = new pkg.setter()>
<cfset as = a.getWidgets()>
<cfset as[1].spin()>
<cfset b = new pkg.member()>
<cfset bs = b.getWidgets()>
<cfset bs[1].spin()>
<cfset c = new pkg.indexed()>
<cfset cs = c.getWidgets()>
<cfset cs[1].spin()>
<cfset d = new pkg.viaget()>
<cfset ds = d.getWidgets()>
<cfset ds[1].spin()>
<cfset ds[1].getLabel()>
<cfset ds[1].nope()>
<cfset p = new pkg.bplain()>
<cfset ps = p.getBeans()>
<cfset ps[1].spin()>
<cfset e = new pkg.bindex()>
<cfset es = e.getBeans()>
<cfset es[2].spin()>
<cfset f = new pkg.bmember()>
<cfset fs = f.getBeans()>
<cfset fs[2].spin()>
<cfset g = new pkg.bprepend()>
<cfset gs = g.getBeans()>
<cfset gs[1].spin()>
<cfset h = new pkg.binsert()>
<cfset hs = h.getBeans()>
<cfset hs[1].spin()>`,
	})

	got := reasonsIn(t, dir, "pkg/page.cfm")

	for _, call := range []string{"as[1].spin", "bs[1].spin", "cs[1].spin", "es[2].spin", "fs[2].spin", "gs[1].spin", "hs[1].spin"} {
		if r := got[call]; strings.Contains(r, "pkg.widget") {
			t.Errorf("%s: %q; an unaccounted write withholds the widget element", call, r)
		}
	}

	expectReasons(t, got, map[string]string{
		"ps[1].spin":     "method 'spin' not found in pkg.widget",
		"ds[1].spin":     "",
		"ds[1].getLabel": "",
		"ds[1].nope":     "method 'nope' not found in pkg.gadget|pkg.widget",
	})
}
