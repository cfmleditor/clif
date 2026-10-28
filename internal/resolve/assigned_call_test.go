package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestAVariableAssignedFromASubclassHeldReceiverIsTyped: an abstract handler
// writes `var o = variables.svc.get( 1 )` and `prc.item = variables.svc.get( 1 )`
// where only its subclasses hold svc. The variable, and prc.item in the view
// the action renders, hold what each subclass's service returns, as
// alternatives; one service returning no component leaves them untyped.
func TestAVariableAssignedFromASubclassHeldReceiverIsTyped(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/Item.cfc":  `component { function itemOnly(){} }`,
		"models/Other.cfc": `component { function otherOnly(){} }`,
		"svc/AService.cfc": `component { models.Item function get( id ){} }`,
		"svc/BService.cfc": `component { models.Other function get( id ){} }`,
		"svc/CService.cfc": `component { function get( id ){} }`,
		"handlers/base.cfc": `component {
function run(){
	var o = variables.svc.get( 1 );
	o.itemOnly();
	o.otherOnly();
	o.nope();
}
function quickLook( event, rc, prc ){
	prc.item = variables.svc.get( event.getValue( "id", 0 ) );
	event.setView( "content/quickLook" );
}
}`,
		"handlers/a.cfc":              `component extends="base" { function init(){ variables.svc = new svc.AService(); return this; } }`,
		"handlers/b.cfc":              `component extends="base" { function init(){ variables.svc = new svc.BService(); return this; } }`,
		"views/content/quickLook.cfm": `<cfoutput>#prc.item.itemOnly()# #prc.item.otherOnly()# #prc.item.nope()#</cfoutput>`,

		"mixed/mixedBase.cfc": `component { function run(){
	var o = variables.svc.get( 1 );
	o.itemOnly();
} }`,
		"mixed/mixedA.cfc": `component extends="mixedBase" { function init(){ variables.svc = new svc.AService(); return this; } }`,
		"mixed/mixedC.cfc": `component extends="mixedBase" { function init(){ variables.svc = new svc.CService(); return this; } }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "handlers/base.cfc"), map[string]string{
		"o.itemOnly":  "",
		"o.otherOnly": "",
		"o.nope":      "method 'nope' not found in models.Item|models.Other",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "views/content/quickLook.cfm"), map[string]string{
		"prc.item.itemOnly":  "",
		"prc.item.otherOnly": "",
		"prc.item.nope":      "method 'nope' not found in Item|Other",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "mixed/mixedBase.cfc"), map[string]string{
		"o.itemOnly": "variable 'o' has no component ref",
	})
}

// TestAVariableAssignedADynamicReturnIsDynamic: `paragraph =
// document.createParagraph()` on a receiver only a componentResolver types
// is typed at lookup, and a method returning a dynamic value makes the
// variable dynamic, as the parse makes it when it can see the receiver.
// tassweb's createParagraph returns a string or one of several paragraphs,
// and every call on what it returned was "has no component ref".
func TestAVariableAssignedADynamicReturnIsDynamic(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Para.cfc": `component { function add(){} }`,
		"doc.cfc": `component {
function createParagraph( k, text ){
	var r = "";
	if ( k ) {
		r = new Para();
	} else {
		r = arguments.text;
	}
	return r;
}
}`,
		"page.cfm": `<cfscript>
	paragraph = document.createParagraph( 1, "x" );
	paragraph.add();
	paragraph.nope();
</cfscript>`,
	})

	r := &Resolver{Resolvers: []parser.Resolver{{Match: `^document(?:\(\))?$`, Resolve: "doc", Prefix: "document"}}}
	expectReasons(t, reasonsWith(t, r, dir, "page.cfm"), map[string]string{
		"paragraph.add":  "",
		"paragraph.nope": "",
	})
}

// TestAUnionIsDisplayedPerAlternative: a union may mix a file and a dot-path
// (a function returning `this` in one branch and a named component in
// another), and each alternative is shown as it would be alone.
func TestAUnionIsDisplayedPerAlternative(t *testing.T) {
	if got := displayComponent("/w/core/mura/event.cfc|mura.servletEvent"); got != "event|mura.servletEvent" {
		t.Errorf("displayComponent = %q, want %q", got, "event|mura.servletEvent")
	}
}

// TestAWrapperReturnsWhatItsHelperReturns: tassweb's itext.createParagraph
// returns `result = VARIABLES._itextUtils.createParagraph( … )`, the helper
// built in init() and copied out of a struct, and the helper returns a
// string or a paragraph by its arguments. The index's parse cannot look the
// helper's method up (ReturnVia), and the helper's body guards on its
// arguments, so only the parse's proof that one branch returns no component
// (ReturnsUntyped) says every call gets a dynamic value.
func TestAWrapperReturnsWhatItsHelperReturns(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"helpers/Para.cfc": `component { function add(){} }`,
		"helpers/utils.cfc": `<cfcomponent>
<cffunction name="init"><cfreturn this></cffunction>
<cffunction name="make" returntype="any">
	<cfargument name="text" default="">
	<cfset var result = "" />
	<cfif StructKeyExists(ARGUMENTS,"object")>
		<cfset result = createObject("component","Para")>
	<cfelse>
		<cfset result = ARGUMENTS.text />
	</cfif>
	<cfreturn result />
</cffunction>
</cfcomponent>`,
		"doc.cfc": `<cfcomponent>
<cffunction name="init">
	<cfset var temp = 0 />
	<cfset temp = StructNew()>
	<cfset temp.u = CreateObject("component","helpers.utils").init()>
	<cfset VARIABLES._u = temp.u>
	<cfset temp = 1>
	<cfreturn this>
</cffunction>
<cffunction name="createParagraph" returntype="any">
	<cfset var result = "">
	<cfset var args = ARGUMENTS>
	<cfset result = VARIABLES._u.make(argumentCollection=args) />
	<cfreturn result />
</cffunction>
</cfcomponent>`,
		"page.cfm": `<cfscript>
	paragraph = document.createParagraph( text = "x" );
	paragraph.add();
	paragraph.anything();
</cfscript>`,
	})

	r := &Resolver{Resolvers: []parser.Resolver{{Match: `^document(?:\(\))?$`, Resolve: "doc", Prefix: "document"}}}
	expectReasons(t, reasonsWith(t, r, dir, "page.cfm"), map[string]string{
		"paragraph.add":      "",
		"paragraph.anything": "",
	})
}

// TestAVariableAssignedAnUntypedReturnIsDynamic: `o = variables.svc.get( 1 )`
// on a receiver only the lookup types, calling a method that declares no
// component, is dynamic, as the parse makes it when it sees the receiver
// (walkChainRest). MasaCMS's `var comment = event.getValue( "commentBean" )`
// in a handler was "has no component ref" on every call. A method one
// alternative lacks, or one returning a component the other does not, is no
// answer.
func TestAVariableAssignedAnUntypedReturnIsDynamic(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/Item.cfc":  `component { function itemOnly(){} }`,
		"svc/AService.cfc": `component { function get( id ){} function onlyA(){} }`,
		"svc/BService.cfc": `component { function get( id ){} }`,
		"handlers/base.cfc": `component {
function run(){
	var o = variables.svc.get( 1 );
	o.anything();
	var m = variables.svc.onlyA();
	m.anything();
}
}`,
		"handlers/a.cfc": `component extends="base" { function init(){ variables.svc = new svc.AService(); return this; } }`,
		"handlers/b.cfc": `component extends="base" { function init(){ variables.svc = new svc.BService(); return this; } }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "handlers/base.cfc"), map[string]string{
		"o.anything": "",
		"m.anything": "variable 'm' has no component ref",
	})
}
