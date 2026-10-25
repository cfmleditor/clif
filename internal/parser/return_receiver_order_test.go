package parser

import (
	"strings"
	"testing"
)

// TestAReturnWaitsForAReceiverTypedLaterInTheFile: Lucee's MongoDB tests
// return a collection from a helper written above the function that connects:
//
//	private any function resetTestCollection() {
//		var coll = db.getCollection( "test" ); …; return coll;
//	}
//	public function beforeTests() { db = MongoDBConnect( … ); }
//
// The calls are typed in file order, so coll's was looked at before db had a
// type, and the untyped calls were looked at again only once some function's
// return had settled. None had, so coll stayed untyped, the helper returned
// nothing, and every call in every test on what it returned was reported: 76
// over the corpus. A pass that types a call now earns the untyped ones
// another look.
func TestAReturnWaitsForAReceiverTypedLaterInTheFile(t *testing.T) {
	const (
		connection = "$builtin.native.mongodb"
		collection = "$builtin.native.mongodbcollection"
	)

	src := `component {
	private any function resetTestCollection() {
		var coll = db.getCollection( "test" );
		coll.drop();
		return coll;
	}
	public function beforeTests() {
		db = MongoDBConnect( "a", "b" );
	}
	function testUpdate() {
		var updated = resetTestCollection();
		updated.update( {}, {} );
	}
}`

	pr := ParseWithOptions("file:///w/M.cfc", src, &ParseOptions{
		ExtractCalls: true,
		BuiltinReturnLookup: func(name string) string {
			if strings.EqualFold(name, "MongoDBConnect") {
				return connection
			}

			return ""
		},
		FuncLookup: func(comp, name string) string {
			if strings.EqualFold(comp, connection) && strings.EqualFold(name, "getCollection") {
				return collection
			}

			return ""
		},
	})

	for i := range pr.Funcs {
		if f := &pr.Funcs[i]; f.Name == "resetTestCollection" && f.ReturnComponent != collection {
			t.Errorf("resetTestCollection returns %q, want %q", f.ReturnComponent, collection)
		}
	}

	scope := pr.Scopes[len(pr.Scopes)-1]

	refs, _ := pr.FuncRefs(scope.Start, scope.End)
	for i := range refs {
		if refs[i].Variable == "updated" && refs[i].Component != collection {
			t.Errorf("updated holds %q, want %q", refs[i].Component, collection)
		}
	}
}
