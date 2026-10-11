package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/cfmleditor/clif/internal/unresolved"
)

func TestMethodShortcutFlagsAndAuditJSON(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		enabled bool
	}{
		{[]string{"--speculative-methods", "project"}, true},
		{[]string{"--speculative-methods", "--no-speculative-methods", "project"}, false},
		{[]string{"--no-speculative-methods", "--speculative-methods", "project"}, true},
	} {
		fl, paths := parseUnresolvedFlags(tc.args)
		if fl.speculativeMethods != tc.enabled || len(paths) != 1 || paths[0] != "project" {
			t.Fatal(fl, paths)
		}
	}

	original := unresolved.Call{Function: "ready", Reason: "unproven"}
	rep := unresolved.Report{Resolved: 3, SpeculativeMethodConnections: []unresolved.SpeculativeMethodConnection{{Call: original, Target: unresolved.Candidate{File: "A.cfc"}, Policy: "unique-global-method", Ownership: "unproven"}}}

	var out bytes.Buffer
	if err := writeOpenWorldJSON(&out, &rep); err != nil {
		t.Fatal(err)
	}

	var decoded struct {
		Methods  []unresolved.SpeculativeMethodConnection `json:"speculativeMethodConnections"`
		Resolved int                                      `json:"resolved"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}

	if len(decoded.Methods) != 1 || decoded.Methods[0].Reason != "unproven" || decoded.Resolved != 3 {
		t.Fatal(out.String())
	}
}
