package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/cfmleditor/clif/internal/unresolved"
)

func TestOpenWorldJSONPreservesOriginalDiagnostics(t *testing.T) {
	call := unresolved.Call{File: "Use.cfc", Line: 12, Variable: "value", Function: "ready", Reason: "variable 'value' has no component ref"}
	rep := unresolved.Report{Resolved: 3, Indexed: 2, Scanned: 2, AssumedCalls: []unresolved.AssumedCall{{Call: call, Policy: "runtime-receiver"}}}

	var out bytes.Buffer
	if err := writeOpenWorldJSON(&out, &rep); err != nil {
		t.Fatal(err)
	}

	var got struct {
		AssumedCalls []unresolved.AssumedCall `json:"assumedCalls"`
		Resolved     int                      `json:"resolved"`
		Indexed      int                      `json:"indexed"`
		Scanned      int                      `json:"scanned"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}

	if len(got.AssumedCalls) != 1 || !reflect.DeepEqual(got.AssumedCalls[0].Call, call) || got.AssumedCalls[0].Policy != "runtime-receiver" || got.Resolved != 3 || got.Indexed != 2 || got.Scanned != 2 {
		t.Fatalf("%s", out.String())
	}
}

func TestUnresolvedLiberalPolicyFlags(t *testing.T) {
	for _, args := range [][]string{
		{"--speculative-ranked", "--assume-open-world", "project"},
		{"--speculative-ranked", "--no-speculative-ranked", "--assume-open-world", "--no-assume-open-world", "project"},
	} {
		flags, paths := parseUnresolvedFlags(args)
		opt := unresolvedOptions(nil, paths, &flags)

		want := len(args) == 3
		if opt.SpeculativeRanked != want || opt.AssumeOpenWorld != want || opt.Speculative || opt.SpeculativeNative || len(paths) != 1 {
			t.Fatalf("%+v", opt)
		}
	}
}

func TestUnresolvedSpeculativeFlagsAreExplicitAndReversible(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"project"}, false},
		{[]string{"--speculative", "project"}, true},
		{[]string{"--speculative", "--no-speculative", "project"}, false},
		{[]string{"--no-speculative", "--speculative", "project"}, true},
	} {
		flags, paths := parseUnresolvedFlags(tc.args)
		if flags.speculative != tc.want || len(paths) != 1 || paths[0] != "project" {
			t.Fatalf("%v: %+v %v", tc.args, flags, paths)
		}

		options := unresolvedOptions(nil, paths, &flags)
		if options.Speculative != tc.want {
			t.Fatal("flag did not reach scan options")
		}
	}
}

func TestUnresolvedNativePolicyFlags(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		enabled bool
		engine  string
	}{
		{[]string{"--speculative-native", "--native-engine=lucee", "project"}, true, "lucee"},
		{[]string{"--speculative-native", "--no-speculative-native", "--native-engine", "generic", "project"}, false, "generic"},
	} {
		flags, paths := parseUnresolvedFlags(tc.args)

		opt := unresolvedOptions(nil, paths, &flags)
		if opt.SpeculativeNative != tc.enabled || opt.NativeEngine != tc.engine || opt.Speculative || len(paths) != 1 {
			t.Fatalf("%+v %v", opt, paths)
		}
	}
}
