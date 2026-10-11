package resolve

import (
	"strings"
	"testing"
)

func TestInvokeTransfersLiteralArguments(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		typed      bool
	}{
		{"array", `invoke(new Svc(),"use",[new Item()]);`, true},
		{"struct", `invoke(new Svc(),"use",{thing:new Item()});`, true},
		{"quoted struct key", `invoke("Svc","use",{"thing":new Item()});`, true},
		{"named", `invoke(name="use",arguments={thing:new Item()},object=new Svc());`, true},
		{"named aliases", `invoke(cfcInstance=new Svc(),methodName="use",args=[new Item()]);`, true},
		{"duplicate target alias", `invoke(object=new Svc(),cfcInstance=new Other(),name="use");`, false},
		{"mixed", `invoke(new Svc(),name="use",arguments=[new Item()]);`, false},
		{"unknown named", `invoke(object=new Svc(),name="use",unsupported=[new Item()]);`, false},
		{"known plus ambiguous named", `invoke(new Svc(),"use",[new Item()]);invoke(object=new Svc(),cfcInstance=new Other(),name="use");`, false},
		{"unknown argument", `invoke(new Svc(),"use",[unknown]);`, false},
		{"unknown target", `invoke(unknown,"use",[new Item()]);`, false},
		{"unknown bag", `invoke(new Svc(),"use",bag);`, false},
		{"computed selector", `invoke(new Svc(),direction,[new Item()]);`, false},
		{"different target", `invoke(new Other(),"use",[new Item()]);`, false},
		{"conflicting known and unknown", `invoke(new Svc(),"use",[new Item()]);invoke(new Svc(),"use",[unknown]);`, false},
		{"duplicate named value", `invoke(new Svc(),"use",{thing:new Item(),thing:unknown});`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"Item.cfc":   `component {function ready(){}}`,
				"Svc.cfc":    `component {function use(thing){arguments.thing.ready();}}`,
				"Other.cfc":  `component {function use(thing){}}`,
				"Caller.cfc": `component {function run(){` + tc.body + `}}`,
			})

			got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Svc.cfc")
			if reason, exists := got["arguments.thing.ready"]; !exists || (reason == "") != tc.typed {
				t.Fatalf("typed %v: %v", tc.typed, got)
			}
		})
	}
}

func TestSameLineCallArgumentsStaySeparate(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc": `component {function a(){}}`,
		"B.cfc": `component {function b(){}}`,
		"Svc.cfc": `component {
 function use(thing){arguments.thing.a();arguments.thing.b();}
 function run(){use(new A());use(new B());}
 }`,
	})

	got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Svc.cfc")
	for _, key := range []string{"arguments.thing.a", "arguments.thing.b"} {
		if reason, exists := got[key]; !exists || reason != "" {
			t.Fatalf("%s: %v", key, got)
		}
	}

	writeFiles(t, dir, map[string]string{"Svc.cfc": `component {
 function use(thing){arguments.thing.a();}
 function run(){use(new A());use(unknown);}
 }`})

	got = reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Svc.cfc")
	if !strings.Contains(got["arguments.thing.a"], "no component ref") {
		t.Fatal("unknown second call ignored", got)
	}
}

func TestAmbiguousTagInvokeDoesNotBorrowKnownArguments(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Item.cfc":   `component {function ready(){}}`,
		"Svc.cfc":    `component {function use(thing){arguments.thing.ready();}}`,
		"Caller.cfm": `<cfset invoke(new Svc(),"use",[new Item()])+invoke(new Svc(),"use",[unknown])>`,
	})

	got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Svc.cfc")
	if reason, exists := got["arguments.thing.ready"]; !exists || reason == "" {
		t.Fatalf("ambiguous invoke borrowed known argument: %v", got)
	}
}
