package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfiguredCallerInferenceLimitsReachScan(t *testing.T) {
	for _, tc := range []struct {
		name, policy string
		limited      bool
	}{
		{"unlimited", `{}`, false},
		{"limited", `{"maxCallSites":1}`, true},
		{"at boundary", `{"maxCallSites":2}`, false},
	} {
		name, policy, limited := tc.name, tc.policy, tc.limited
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()

			files := map[string]string{
				".clif.json": `{"callerInferenceLimits":` + policy + `}`,
				"Item.cfc":   `component { function ready(){} }`,
				"DAO.cfc":    `component { function update(bean){ arguments.bean.ready(); arguments.bean.missing(); } }`,
				"Caller.cfc": `component { function run(){ new DAO().update(new Item()); new DAO().update(new Item()); } }`,
			}
			for file, src := range files {
				if err := os.WriteFile(filepath.Join(dir, file), []byte(src), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			report, _, _ := scanUnresolved([]string{dir}, &unresolvedFlags{})

			var ready, missing string

			for _, call := range report.Calls {
				if call.Variable != "arguments.bean" {
					continue
				}

				if call.Function == "ready" {
					ready = call.Reason
				}

				if call.Function == "missing" {
					missing = call.Reason
				}
			}

			if limited {
				if !strings.Contains(ready, "no component ref") {
					t.Fatalf("config limit did not reach scan: %q", ready)
				}
			} else if ready != "" || !strings.Contains(missing, "not found") {
				t.Fatalf("unlimited scan did not check Item methods: ready=%q missing=%q", ready, missing)
			}
		})
	}
}
