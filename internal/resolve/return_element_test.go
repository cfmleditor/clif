package resolve

import "testing"

// A function whose every return reads one element of a collection returns what
// the collection's elements are: Galaxie Blog's loadOrder returns found[ 1 ] of
// an HQL query, and its callers' `o.getCurrency()` were 100 "no component ref".
func TestAFunctionReturningAnElementOfAnEntityArrayReturnsTheEntity(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Item.cfc":  `component persistent="true" entityname="LoadedItem" { function ready(){} }`,
		"Other.cfc": `component persistent="true" entityname="OtherItem" { function other(){} }`,
		"Svc.cfc": `component {
	private any function load( required numeric id ) {
		var found = ormExecuteQuery( "FROM LoadedItem WHERE id = :id", { id = arguments.id } );
		if ( arrayLen( found ) ) {
			return found[ 1 ];
		}
	}
	private any function loadBoth( required numeric id ) {
		var a = entityLoad( "LoadedItem" );
		var b = entityLoad( "OtherItem" );
		if ( id ) return a[ 1 ];
		return b[ 1 ];
	}
	private any function loadOrString( required numeric id ) {
		var found = entityLoad( "LoadedItem" );
		if ( id ) return found[ 1 ];
		return "none";
	}
	private any function loadUnknown( required numeric id ) {
		var found = unknownList();
		return found[ 1 ];
	}
	// A return in a comment or a string is not one of its returns.
	private any function loadQuietly( required numeric id ) {
		var found = entityLoad( "LoadedItem" );
		// return legacy[ 1 ];
		if ( !arrayLen( found ) ) throw( "nothing to return" );
		return found[ 1 ];
	}
	function run() {
		var quiet = loadQuietly( 1 );
		quiet.ready();
		quiet.missingQuiet();
		var one = load( 1 );
		one.ready();
		one.missingOne();
		var both = loadBoth( 1 );
		both.ready();
		var str = loadOrString( 1 );
		str.stringReady();
		var unk = loadUnknown( 1 );
		unk.unknownReady();
	}
}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Svc.cfc")

	if got["quiet.ready"] != "" || got["quiet.missingQuiet"] == "" || got["quiet.missingQuiet"] == "variable 'quiet' has no component ref" {
		t.Errorf("quiet: ready %q, missingQuiet %q; want the function typed past the commented-out and quoted returns",
			got["quiet.ready"], got["quiet.missingQuiet"])
	}

	if got["one.ready"] != "" {
		t.Errorf("one.ready: %q, want it resolved", got["one.ready"])
	}

	if got["one.missingOne"] == "" || got["one.missingOne"] == "variable 'one' has no component ref" {
		t.Errorf("one.missingOne: %q, want a method missing from LoadedItem's component", got["one.missingOne"])
	}

	// Returns of two components, of a string beside an element, and of an
	// element of nothing typed prove no component.
	for _, k := range []string{"both.ready", "str.stringReady", "unk.unknownReady"} {
		if got[k] == "" {
			t.Errorf("%s was typed from returns that disagree or prove nothing", k)
		}
	}
}
