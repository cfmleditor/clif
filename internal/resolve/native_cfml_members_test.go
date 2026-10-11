package resolve

import "testing"

func TestNativeCFMLMemberResultsAndReceiverOwnership(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Custom.cfc": `component {function listMap(callback){} function columnData(name){} }`,
		"Use.cfc": `component {function run(){
 var text="a,b,a";
 text.hash().length();text.listGetDuplicates().length();
 text.listMap(function(value){return value;}).length();
 text.listFilter(function(value){return true;}).length();
 text.map(function(value){return value;}).length();
 text.filter(function(value){return true;}).length();
 text.listEvery(function(value){return true;}).toString();
 text.listSome(function(value){return true;}).toString();
 text.every(function(value){return true;}).toString();
 text.some(function(value){return true;}).toString();
 text.each(function(value){});text.listEach(function(value){});
 text.each(function(value){}).length();text.listEach(function(value){}).length();
 text.reduce(function(result,value){return result;},0);
 text.listReduce(function(result,value){return result;},0);
 var data=structNew();
 data.toSorted().keyArray().len();data.toSorted().toJson().length();data.toSorted().absent();
 var rows=queryNew("id");
 rows.columnData("id").len();rows.columnData("id").absent();
 text.columnData("id");rows.hash();
 var custom=new Custom();custom.listMap(function(value){return value;}).length();custom.columnData("id").len();
 }}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	for _, name := range []string{"text.hash.length", "text.listGetDuplicates.length", "text.listMap.length", "text.listFilter.length", "text.map.length", "text.filter.length", "text.listEvery.toString", "text.listSome.toString", "text.every.toString", "text.some.toString", "text.each", "text.listEach", "text.reduce", "text.listReduce", "data.toSorted.keyArray.len", "data.toSorted.toJson.length", "rows.columnData.len"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"text.each.length", "text.listEach.length", "data.toSorted.absent", "rows.columnData.absent", "text.columnData", "rows.hash", "custom.listMap.length", "custom.columnData.len"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must remain diagnosed: %q present=%v", name, reason, ok)
		}
	}
}
