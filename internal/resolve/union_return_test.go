package resolve

import "testing"

// TestAUnionReturnTypesTheChainedCall pins the end-to-end shape of a union
// return through a chained call, ColdBox's Util.getClassMappingHelper(): three
// components with no common base, one returned per branch, each declaring
// `shared()`. It fails unless the fixtures it is built from are so arranged.
func TestAUnionReturnTypesTheChainedCall(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc": `component { function shared(){} function onlyA(){} }`,
		"B.cfc": `component { function shared(){} }`,
		"C.cfc": `component { function shared(){} }`,
		"Model.cfc": `component {
function helper( k ){
	if ( k == 1 ) {
		var h = new A();
	} else if ( k == 2 ) {
		h = new B();
	} else {
		h = new C();
	}
	return h;
}
function run(){
	helper( 1 ).shared();
	helper( 2 ).onlyA();
	helper( 3 ).nothingInAny();
}
}`,
	})

	expectReasons(t, reasonsIn(t, dir, "Model.cfc"), map[string]string{
		// Every candidate declares shared().
		"helper.shared": "",
		// Only A does: accepted, the declared-by-some call follows the
		// interface precedent, an implementation may add the method.
		"helper.onlyA": "",
		// Declared by none: reported once, against the joined display.
		"helper.nothingInAny": "method 'nothingInAny' not found in A|B|C",
	})
}
