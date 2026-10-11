package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

func TestBuiltinCollectionProducers(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var parts=listToArray("a,b");
 parts.len();parts.each(function(value,index,collection){index.toString();collection.len();value.ready();});
 var sorted=structToSorted({a:1});
 sorted.keyArray().len();sorted.filter(function(key,value,collection){key.len();collection.keyList();value.ready();});
 var settings=getApplicationSettings();
 settings.keyExists("name");settings.s3.keyArray();
 var matches=reMatch("a","a");
 matches.len();matches.absent();
 var sliced=arraySlice(parts,1,1);
 sliced.len();
 var unknown=deserializeJSON("{}");
 unknown.keyArray();
 var listing=directoryList(".");
 listing.len();
 }}`})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	for _, name := range []string{"parts.len", "index.toString", "collection.len", "sorted.keyArray.len", "key.len", "collection.keyList", "settings.keyExists", "matches.len", "sliced.len"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v all=%v", name, reason, ok, got)
		}
	}

	for _, name := range []string{"value.ready", "settings.s3.keyArray", "matches.absent", "unknown.keyArray", "listing.len"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must remain diagnosed: %q present=%v", name, reason, ok)
		}
	}
}

func TestBuiltinCollectionProducerShadows(t *testing.T) {
	for _, source := range []string{
		`component {function listToArray(){return new Item();} function run(){
 var parts=listToArray();
 parts.ready();parts.len().toString();
 }}`,
		`component {function run(listToArray){
 var parts=listToArray();
 parts.len().toString();
 }}`,
		`component {function run(){
 var listToArray=unknown;
 var parts=listToArray();
 parts.len().toString();
 }}`,
		`component {function run(){arrayEach([],function(listToArray){
 var parts=listToArray();
 parts.len().toString();
 });}}`,
	} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{"Item.cfc": `component {function ready(){}}`, "Use.cfc": source})

		got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
		if reason, ok := got["parts.len.toString"]; !ok || reason == "" {
			t.Errorf("shadowed producer borrowed builtin type: %q present=%v", reason, ok)
		}
	}
}

func TestBuiltinCollectionProducerForwardAssignment(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(collection){
 arguments.collection.hasNext();
 arguments.collection=listToArray(arguments.collection);
 arguments.collection.len();
 }}`})

	got := builtinProducerReasons(t, dir)
	if reason, ok := got["arguments.collection.hasNext"]; !ok || reason == "" || strings.Contains(reason, "native.array") {
		t.Errorf("earlier incoming argument borrowed future array: %q present=%v", reason, ok)
	}

	if reason, ok := got["arguments.collection.len"]; !ok || reason != "" {
		t.Errorf("assigned array: %q present=%v", reason, ok)
	}
}

func TestBuiltinArrayPublicInterfaces(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var parts=listToArray("a,b");
 parts.getDimension().toString();
 parts.subList(0,1).size().toString();
 parts.toArray();
 parts.subList(0,1).get(0).ready();
 parts.getDimension().append("x");
 var settings=getApplicationSettings();
 application action="update" mappings=getApplicationSettings().mappings.append({});
 settings.absent();
 }}`})

	got := builtinProducerReasons(t, dir)
	for _, name := range []string{"parts.getDimension.toString", "parts.subList.size.toString", "parts.toArray", "getApplicationSettings.$property:mappings.append"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v all=%v", name, reason, ok, got)
		}
	}

	for _, name := range []string{"parts.getDimension.append", "settings.absent"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s remains diagnosed: %q present=%v", name, reason, ok)
		}
	}
}

// Exercise the scan's builtin parse callback as well as resolver inference.
func builtinProducerReasons(t *testing.T, dir string) map[string]string {
	t.Helper()

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")
	file := filepath.Join(dir, "Use.cfc")

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	pr := parser.ParseWithOptions(cfpath.ToURI(file), string(content), &parser.ParseOptions{ExtractCalls: true, FuncLookup: r.FuncLookup(dir), BuiltinReturnLookup: docs.LookupBuiltinReturnComponent})
	got := map[string]string{}

	calls := pr.AllCalls()
	for i := range calls {
		c := &calls[i]
		key := strings.TrimPrefix(strings.Join(append(append([]string{c.Variable}, hopNames(c.Chain)...), c.FuncName), "."), ".")
		got[key] = r.CanResolveCall(c, pr, dir)
	}

	return got
}
