package resolve

import (
	"testing"
)

// The Lucee MongoDB extension's shell-like API, documented in the extension's
// own ReadMe (pinned master):
//
//	db = MongoDBConnect("mydb", "mongodb://localhost:27017");
//	coll = db.getCollection("myCollection");
//	cursor = coll.find(filter, projection).sort(["score": -1]).limit(10);
//	while (cursor.hasNext()) { row = cursor.next(); }
//	result = coll.update(filter, update);
//	writeOutput(result.getN());
//	id = MongoDBID();
//	writeOutput(id.toString() & id.getDate());
//
// The database handle, the collection, the cursor, the aggregation output,
// the write result, the ObjectId and the write concern are public runtime
// interfaces: each method's result type is the surface the ReadMe states,
// and a method it states with an erased result stays dynamic.
func TestTheMongoExtensionShellAPITypesItsHandles(t *testing.T) {
	dir := t.TempDir()

	// The extension is a Lucee one: the contract is served under the
	// engine's identity, which the scan's settings detect and this fixture
	// states.
	r := &Resolver{}
	r.NativeEngine = "lucee"

	writeFiles(t, dir, map[string]string{
		"core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestCase.cfc": `component {}`,
		"core/src/main/cfml/context/admin/services.update.functions.cfm":              `<cfadmin action="x">`,

		// The extension's test shape: a component using the shell API.
		"core/src/main/cfml/test/mongo.cfc": `component {
	function findDocs(uri) {
		db = MongoDBConnect("mydb", uri);
		coll = db.getCollection("myCollection");
		cursor = coll.find({"score": {"$gt": 10}}, {"name": 1}).sort(["score": -1]).limit(10);
		while (cursor.hasNext()) {
			row = cursor.next();
			echo(row.name);
		}
		result = coll.update({"_id": 1}, {"$set": {"active": true}});
		echo(result.getN());
		echo(result.isOk());
		echo(result.isUpdateOfExisting());
		id = MongoDBID();
		echo(id.toString());
		echo(id.getDate());
		coll.drop();
		echo(coll.count());
		echo(coll.getIndexes());
		results = coll.aggregate([{"$group": {"_id": "$x"}}]);
		while (results.hasNext()) {
			r = results.next();
			echo(r);
		}
		echo(coll.findOne({"name": "Alice"}));
		echo(coll.distinct("name"));
		missing = coll.getMissingShellMethod();
		return "";
	}
}`,
	})

	got := reasonsWith(t, r, dir, "core/src/main/cfml/test/mongo.cfc")
	expectReasons(t, got, map[string]string{
		// The extension's functions are ones the workspace does not define:
		// a call to one reads as it did, a call to a function nobody declared.
		"MongoDBConnect":            "no qualifier, not in file",
		"MongoDBID":                 "no qualifier, not in file",
		"db.getCollection":          "",
		"coll.find":                 "",
		"coll.find.sort":            "",
		"coll.find.sort.limit":      "",
		"cursor.hasNext":            "",
		"cursor.next":               "",
		"coll.update":               "",
		"result.getN":               "",
		"result.isOk":               "",
		"result.isUpdateOfExisting": "",
		"id.toString":               "",
		"id.getDate":                "",
		"coll.drop":                 "",
		"coll.count":                "",
		"coll.getIndexes":           "",
		"coll.aggregate":            "",
		"results.hasNext":           "",
		"results.next":              "",
		"coll.findOne":              "",
		"coll.distinct":             "",
		// A method the ReadMe does not state is a finding, as it was.
		"coll.getMissingShellMethod": "method 'getMissingShellMethod' not found on builtin native.mongodbcollection",
	})
}
