package resolve

import (
	"strings"
	"testing"
)

func TestBuiltinStringProducers(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(value){
 var pathInfo=right(value,10);
 pathInfo.split("/");pathInfo.listToArray("/").len();
 pathInfo.hasPrefix("a").toString();pathInfo.hasSuffix("b").toString();
 pathInfo.hasPrefix("a").append("x");
 var replaced=reReplace(value,"a","b");
 replaced.len();replaced.absent();
 var tail=listRest(value);
 tail.trim().len();
 var directory=getDirectoryFromPath(value);
 directory.replace("a","b").toString();
 var encoded=charsetEncode(value,"UTF-8");
 encoded.len();
 var decoded=charsetDecode(value,"UTF-8");
 decoded.keyArray();
 var binary=toBinary(value);
 binary.keyArray();
 var json=serializeJSON({});
 json.trim().len();
 }}`})

	got := builtinProducerReasons(t, dir)
	for _, name := range []string{"pathInfo.split", "pathInfo.listToArray.len", "replaced.len", "tail.trim.len", "directory.replace.toString", "encoded.len", "json.trim.len", "pathInfo.hasPrefix.toString", "pathInfo.hasSuffix.toString"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"replaced.absent", "decoded.keyArray", "binary.keyArray", "pathInfo.hasPrefix.append"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must stay diagnosed: %q present=%v", name, reason, ok)
		}
	}
}

func TestBuiltinStringProducerOwnership(t *testing.T) {
	for _, source := range []string{
		`component {function right(){return new Item();} function run(){
 var result=right();result.ready();result.absent();
 }}`,
		`component {function run(right){
 var result=right();result.len().toString();
 }}`,
		`component {function run(){
 var right=unknown;
 var result=right();result.len().toString();
 }}`,
		`component {function run(){arrayEach([],function(right){
 var result=right();result.len().toString();
 });}}`,
	} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{"Use.cfc": source, "Item.cfc": `component {function ready(){}}`})

		got := builtinProducerReasons(t, dir)
		if strings.Contains(source, "function right()") {
			if reason, ok := got["result.ready"]; !ok || reason != "" {
				t.Errorf("source return lost: %v", got)
			}

			if !strings.Contains(got["result.absent"], "not found") {
				t.Errorf("source identity lost: %v", got)
			}
		} else if reason, ok := got["result.len.toString"]; !ok || reason == "" {
			t.Errorf("shadow borrowed builtin type: %v", got)
		}
	}
}

func TestBuiltinNativeUnknownReassignment(t *testing.T) {
	for _, init := range []string{`right("abc",1)`, `listToArray("a,b")`} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var x=` + init + `;
 x.len().toString();
 x=unknown();
 unknown(
 x=x,
 other=1
 );
 x.absent();
 }}`})

		got := builtinProducerReasons(t, dir)
		if reason, ok := got["x.len.toString"]; !ok || reason != "" {
			t.Errorf("initial interface lost: %v", got)
		}

		if reason, ok := got["x.absent"]; !ok || !strings.Contains(reason, "has no component ref") {
			t.Errorf("unknown store borrowed prior type: %v", got)
		}
	}
}

func TestBuiltinNativeNamedArgumentsDoNotRebind(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var buffer=createObject("java","java.lang.StringBuilder");
 unknown(
 buffer=buffer,
 other="x"
 );
 buffer.append("x").toString();
 var data={
 buffer=unknown(),
 other="x"
 };
 buffer.append("x").toString();
 var d=createDate(2018,1,1);
 d=dateAdd("d",1,d);
 d.dateTimeFormat("yyyy");
 }}`})

	got := builtinProducerReasons(t, dir)
	for _, name := range []string{"buffer.append.toString", "d.dateTimeFormat"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("valid native lifetime lost: %v", got)
		}
	}
}
