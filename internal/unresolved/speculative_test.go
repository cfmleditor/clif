package unresolved

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

func TestSpeculativeCandidatesRequireOptInAndExposeWrongOrigin(t *testing.T) {
	dir := t.TempDir()
	sources := map[string]string{
		"A.cfc":       `component { function ready(){} }`,
		"Service.cfc": `component { function use(bean){arguments.bean.ready();} }`,
		"Use.cfm":     `<cfscript>new Service().use(42);</cfscript>`,
	}

	var files []string

	for name, src := range sources {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, p)
	}

	opt := &Options{WorkspaceFolders: []string{dir}, InferArgs: true}
	strict := Scan(vfs.OS{}, files, nil, opt)
	opt.Speculative = true

	guessed := Scan(vfs.OS{}, files, nil, opt)
	if len(strict.Calls) != 1 || len(guessed.Calls) != 0 || len(guessed.SpeculativeConnections) != 1 {
		t.Fatalf("strict=%+v guessed=%+v", strict, guessed)
	}
	// This acceptance is deliberately unreliable: the actual caller passed 42.
	if guessed.SpeculativeConnections[0].Component != filepath.Join(dir, "A.cfc") {
		t.Fatal(guessed.SpeculativeConnections)
	}

	opt.Speculative = false

	again := Scan(vfs.OS{}, files, nil, opt)
	if len(again.Calls) != len(strict.Calls) || len(again.SpeculativeConnections) != 0 {
		t.Fatal("off did not restore strict behavior")
	}
	// A second matching component must prevent promotion.
	other := filepath.Join(dir, "B.cfc")
	if err := os.WriteFile(other, []byte(sources["A.cfc"]), 0o600); err != nil {
		t.Fatal(err)
	}

	files = append(files, other)
	opt.Speculative = true

	ambiguous := Scan(vfs.OS{}, files, nil, opt)
	if len(ambiguous.Calls) != 1 || len(ambiguous.SpeculativeConnections) != 0 {
		t.Fatal(ambiguous)
	}
}

func TestSpeculativeCandidatesPreserveKnownReceiversAndIncompleteGroups(t *testing.T) {
	dir := t.TempDir()
	sources := map[string]string{
		"A.cfc": `component { function ready(){} }`,
		"B.cfc": `component { function other(){} }`,
		"Use.cfc": `component {
 function known(){var bean=new B();bean.ready();}
 function incomplete(bean){bean.ready();bean.absent();}
 function unique(bean){bean.ready();}
 }`,
	}

	var files []string

	for name, src := range sources {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, p)
	}

	rep := Scan(vfs.OS{}, files, nil, &Options{WorkspaceFolders: []string{dir}, Speculative: true})
	if len(rep.SpeculativeConnections) != 1 || rep.SpeculativeConnections[0].Caller != "unique" {
		t.Fatalf("connections=%+v", rep.SpeculativeConnections)
	}

	if len(rep.Calls) != 3 {
		t.Fatalf("known receiver and incomplete group must remain: %+v", rep.Calls)
	}
}

func TestSpeculativeNativeFamiliesAreSeparateAndReversible(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "Use.cfc")
	src := `component {
 function use(value){value.entrySet();value.containsKey("x");}
 function ambiguous(value){value.toString();}
 function incomplete(value){value.entrySet();value.absent();}
 }`

	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	opt := &Options{WorkspaceFolders: []string{dir}}
	strict := Scan(vfs.OS{}, []string{file}, nil, opt)
	opt.SpeculativeNative = true

	guessed := Scan(vfs.OS{}, []string{file}, nil, opt)
	if len(guessed.SpeculativeConnections) != 1 || guessed.SpeculativeConnections[0].Policy != "sole-native-family" || guessed.SpeculativeConnections[0].Caller != "use" {
		t.Fatalf("%+v", guessed)
	}

	if len(guessed.Calls) != len(strict.Calls)-2 {
		t.Fatalf("strict=%+v guessed=%+v", strict.Calls, guessed.Calls)
	}

	opt.SpeculativeNative = false

	again := Scan(vfs.OS{}, []string{file}, nil, opt)
	if len(again.Calls) != len(strict.Calls) || len(again.SpeculativeConnections) != 0 {
		t.Fatal("native off did not restore strict scan")
	}
}

func TestSpeculativePrimitiveFallbackIsOptInAndExposesWrongOrigin(t *testing.T) {
	dir := t.TempDir()
	sources := map[string]string{
		"Service.cfc": `component { function use(value){arguments.value.ensureCapacity(20);} }`,
		"Use.cfm":     `<cfscript>new Service().use(42);</cfscript>`,
	}

	var files []string

	for name, source := range sources {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, file)
	}

	opt := &Options{WorkspaceFolders: []string{dir}, InferArgs: true}
	strict := Scan(vfs.OS{}, files, nil, opt)
	opt.SpeculativeNative = true

	guessed := Scan(vfs.OS{}, files, nil, opt)
	if len(strict.Calls) != 1 || len(guessed.Calls) != 0 || len(guessed.SpeculativeConnections) != 1 {
		t.Fatalf("strict=%+v guessed=%+v", strict, guessed)
	}

	connection := guessed.SpeculativeConnections[0]
	if connection.Policy != "sole-primitive-family" || connection.Component != "$builtin.native.stringbuilder" {
		t.Fatalf("unlabelled primitive fallback: %+v", connection)
	}
	// The caller supplied 42: this tests an explicitly unreliable connection.
	opt.SpeculativeNative = false

	again := Scan(vfs.OS{}, files, nil, opt)
	if len(again.Calls) != len(strict.Calls) || len(again.SpeculativeConnections) != 0 {
		t.Fatal("off did not restore strict behavior")
	}
}

func TestSpeculativePrimitiveFallbackPreservesCFCAndGroupBoundaries(t *testing.T) {
	dir := t.TempDir()
	sources := map[string]string{
		"A.cfc": `component { function ensureCapacity(size){} }`,
		"B.cfc": `component { function other(){} }`,
		"Use.cfc": `component {
 function known(){var value=new B();value.ensureCapacity(20);}
 function incomplete(value){value.ensureCapacity(20);value.absent();}
 function unique(value){value.ensureCapacity(20);}
 }`,
	}

	var files []string

	for name, source := range sources {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, file)
	}

	for _, cfc := range []bool{false, true} {
		opt := &Options{WorkspaceFolders: []string{dir}, SpeculativeNative: true, Speculative: cfc}

		rep := Scan(vfs.OS{}, files, nil, opt)
		if len(rep.SpeculativeConnections) != 1 || rep.SpeculativeConnections[0].Caller != "unique" {
			t.Fatalf("cfc=%v: %+v", cfc, rep)
		}

		connection := rep.SpeculativeConnections[0]
		if cfc {
			if connection.Policy != "sole-cfc-candidate" || connection.Component != filepath.Join(dir, "A.cfc") {
				t.Fatal(connection)
			}
		} else if connection.Policy != "sole-primitive-family" {
			t.Fatal(connection)
		}

		callers := map[string]int{}
		for _, call := range rep.Calls {
			callers[call.Caller]++
		}

		if callers["known"] != 1 || callers["incomplete"] != 2 || callers["unique"] != 0 {
			t.Fatalf("cfc=%v: %+v", cfc, rep.Calls)
		}
	}
}
