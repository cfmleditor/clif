package resolve

import "testing"

func TestOrderedStructLiteralMembers(t *testing.T) {
	for _, literal := range []string{`["a":1]`, `[a=1]`, `[:]`, `[=]`, `(["a": [1, 2]])`} {
		t.Run(literal, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Use.cfc": "component {function run(){\nvar record=" + literal + ";\nrecord.keyArray();\nrecord.getDimension();\n}}"})

			got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
			if reason, ok := got["record.keyArray"]; !ok || reason != "" {
				t.Errorf("struct member: %q present=%v", reason, ok)
			}

			if reason, ok := got["record.getDimension"]; !ok || reason == "" {
				t.Errorf("array-only member: %q present=%v", reason, ok)
			}
		})
	}
}

func TestArrayLiteralMembersRemainArrays(t *testing.T) {
	for _, literal := range []string{`[]`, `[1,2]`, `[{a:1}]`, `[["a":1]]`, `[flag ? "a" : "b"]`, `[choose(a=1)]`} {
		t.Run(literal, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Use.cfc": "component {function run(){\nvar values=" + literal + ";\nvalues.getDimension();\nvalues.keyArray();\n}}"})

			got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
			if reason, ok := got["values.getDimension"]; !ok || reason != "" {
				t.Errorf("array member: %q present=%v", reason, ok)
			}

			if reason, ok := got["values.keyArray"]; !ok || reason == "" {
				t.Errorf("struct-only member: %q present=%v", reason, ok)
			}
		})
	}
}
