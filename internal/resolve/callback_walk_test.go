package resolve

import "testing"

// TestACallbackParameterIsWhatEveryPathInvokesItWith: qb's providers guard
// their callback with isClosure() and replace it with the subquery it
// filled (union), hand it on to another method (unionAll, and withRecursive
// through argumentCollection), and its spec helper invokes it inside a try.
// Each callback parameter was untyped, because only a straight-line body
// that invoked it at the top level was read. A path that invokes it with
// another component, stores it, hands it to code that cannot be followed or
// tests it in a condition nothing reads gives no answer.
func TestACallbackParameterIsWhatEveryPathInvokesItWith(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"q/Other.cfc": `component { function other(){} }`,
		"q/QB.cfc": `component {
public QB function newQuery() { return new q.QB(); }
public QB function union( required any input, boolean all = false ) {
	if ( isClosure( arguments.input ) || isCustomFunction( arguments.input ) ) {
		var subquery = newQuery();
		arguments.input( subquery );
		arguments.input = subquery;
	}
	variables.unions.append( { query: arguments.input, all: arguments.all } );
	return this;
}
public QB function unionAll( required any input ) {
	return union( arguments.input, true );
}
public QB function withRecursive( required string name, required any input ) {
	arguments.recursive = true;
	return with( argumentCollection = arguments );
}
public QB function with( required string name, required any input, boolean recursive = false ) {
	if ( isNull( arguments.input ) ) {
		return this;
	}
	if ( isClosure( arguments.input ) || isCustomFunction( arguments.input ) ) {
		var subquery = newQuery();
		arguments.input( subquery );
		arguments.input = subquery;
	}
	variables.ctes.append( arguments.input );
	return this;
}
public QB function attempt( required callback ) {
	try {
		var q = newQuery();
		local.sql = callback( q );
	} catch ( any e ) {
		rethrow;
	}
	return this;
}
public QB function either( required callback, flag ) {
	if ( flag ) {
		callback( newQuery() );
	} else {
		callback( new q.Other() );
	}
	return this;
}
public QB function unguarded( required any input ) {
	if ( isClosure( arguments.input ) ) {
		arguments.input( newQuery() );
	}
	variables.kept = arguments.input;
	return this;
}
public QB function handedAway( required callback ) {
	elsewhere( callback );
	callback( newQuery() );
	return this;
}
public QB function tested( required callback ) {
	if ( callback( newQuery() ) ) {
		return this;
	}
	return this;
}
function select(){ return this; }
}`,
		"q/Spec.cfc": `component {
function run(){
	var b = new q.QB();
	b.union( function( a ) { a.select(); } );
	b.unionAll( function( c ) { c.select(); } );
	b.withRecursive( "x", function( d ) { d.select(); } );
	b.attempt( function( e ) { e.select(); } );
	b.either( function( f ) { f.select(); }, true );
	b.unguarded( function( g ) { g.select(); } );
	b.handedAway( function( h ) { h.select(); } );
	b.tested( function( i ) { i.select(); } );
}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "q/Spec.cfc"), map[string]string{
		"a.select": "",
		"c.select": "",
		"d.select": "",
		"e.select": "",
		"f.select": "variable 'f' has no component ref",
		"g.select": "variable 'g' has no component ref",
		"h.select": "variable 'h' has no component ref",
		"i.select": "variable 'i' has no component ref",
	})
}
