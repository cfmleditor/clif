package config

import (
	"encoding/json"
	"testing"
)

func TestCallerInferenceLimitsPolicyMerge(t *testing.T) {
	var base JSON
	if err := json.Unmarshal([]byte(`{"callerInferenceLimits":{"maxCandidateFiles":512,"maxCallerFiles":60,"maxCallSites":40,"maxConstructorFiles":60}}`), &base); err != nil {
		t.Fatal(err)
	}

	if base.CallerInferenceLimits == nil || *base.CallerInferenceLimits != (CallerInferenceLimits{512, 60, 40, 60}) {
		t.Fatalf("decoded limits: %+v", base.CallerInferenceLimits)
	}

	if got := Merge(&base, &JSON{}).CallerInferenceLimits; got != base.CallerInferenceLimits {
		t.Fatal("omitted block lost inherited policy")
	}

	var reset JSON
	if err := json.Unmarshal([]byte(`{"callerInferenceLimits":{}}`), &reset); err != nil {
		t.Fatal(err)
	}

	if got := Merge(&base, &reset).CallerInferenceLimits; got == nil || *got != (CallerInferenceLimits{}) {
		t.Fatalf("empty block did not reset limits: %+v", got)
	}

	override := &JSON{CallerInferenceLimits: &CallerInferenceLimits{MaxCallSites: 2}}
	if got := Merge(&base, override).CallerInferenceLimits; *got != *override.CallerInferenceLimits {
		t.Fatalf("block did not replace inherited policy: %+v", got)
	}

	if base.CallerInferenceLimits.MaxCallSites != 40 {
		t.Fatal("merge modified original policy")
	}
}
