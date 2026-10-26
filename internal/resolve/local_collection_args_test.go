package resolve

import (
	"strings"
	"testing"
)

// TestAnArgumentPassedInALocalStructIsTypedByItsKey: Mura hands its publisher
// its arguments in a struct built a key at a time,
//
//	<cfset var sArgs = structNew()>
//	<cfset sArgs.Bundle = Bundle />
//	<cfset publisher.getToWork( argumentCollection = sArgs )>
//
// and an argumentCollection other than the caller's own arguments was opaque,
// so getToWork's arguments.Bundle was untyped. When every reference to the
// struct between its declaration and the call writes or reads a key, the call
// passes Bundle what the key was written. A struct passed elsewhere or
// deleted from, a key written twice (even with the same value, since the text
// cannot say which write reached the call), and a value reassigned before the
// call still give nothing; a struct with no such key does not pass it.
func TestAnArgumentPassedInALocalStructIsTypedByItsKey(t *testing.T) {
	for _, tc := range []struct{ name, caller, want string }{
		{"tag syntax", `<cfcomponent><cffunction name="run">
	<cfset var sArgs = structNew()>
	<cfset var Bundle = new models.Bundle() />
	<cfset sArgs.fromDSN = "x" />
	<cfset sArgs.Bundle = Bundle />
	<cfset var note = sArgs.fromDSN />
	<cfset new svc.Publisher().getToWork( argumentCollection = sArgs )>
</cffunction></cfcomponent>`, ""},
		{"script syntax with a literal and a bracket key", `component { function run() {
	var sArgs = { fromDSN: "x" };
	sArgs[ "Bundle" ] = new models.Bundle();
	new svc.Publisher().getToWork( argumentCollection = sArgs );
} }`, ""},
		{"struct passed elsewhere first", `component { function run() {
	var sArgs = {};
	sArgs.Bundle = new models.Bundle();
	mutate( sArgs );
	new svc.Publisher().getToWork( argumentCollection = sArgs );
} }`, "no component ref"},
		{"key deleted", `component { function run() {
	var sArgs = {};
	sArgs.Bundle = new models.Bundle();
	structDelete( sArgs, "Bundle" );
	new svc.Publisher().getToWork( argumentCollection = sArgs );
} }`, "no component ref"},
		{"key written twice", `component { function run( flag ) {
	var sArgs = {};
	if ( flag ) { sArgs.Bundle = new models.Bundle(); } else { sArgs.Bundle = new models.Bundle(); }
	new svc.Publisher().getToWork( argumentCollection = sArgs );
} }`, "no component ref"},
		{"value reassigned before the call", `component { function run() {
	var sArgs = {};
	var b = new models.Bundle();
	sArgs.Bundle = b;
	b = new models.Other();
	new svc.Publisher().getToWork( argumentCollection = sArgs );
} }`, "no component ref"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"models/Bundle.cfc": `component { function getValue( key ) {} }`,
				"models/Other.cfc":  `component {}`,
				"svc/Publisher.cfc": `component { function getToWork( fromDSN, Bundle ) { arguments.Bundle.getValue( "x" ); } }`,
				"app/Caller.cfc":    tc.caller,
			})

			got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "svc/Publisher.cfc")

			reason, ok := got["arguments.Bundle.getValue"]
			switch {
			case !ok:
				t.Fatalf("call not recorded: %v", got)
			case tc.want == "" && reason != "":
				t.Errorf("got %q, want resolved", reason)
			case tc.want != "" && !strings.Contains(reason, tc.want):
				t.Errorf("got %q, want %q", reason, tc.want)
			}
		})
	}
}
