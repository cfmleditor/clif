package parser

import (
	"slices"
	"testing"
)

func TestParseRecordsImplements(t *testing.T) {
	cases := map[string]string{
		"script quoted":   "component implements=\"IShape, models.IDrawable\" {\n}\n",
		"script unquoted": "component extends=\"Base\" implements=IShape {\n}\n",
		"tag":             "<cfcomponent implements=\"IShape,models.IDrawable\">\n</cfcomponent>\n",
	}

	want := map[string][]string{
		"script quoted":   {"IShape", "models.IDrawable"},
		"script unquoted": {"IShape"},
		"tag":             {"IShape", "models.IDrawable"},
	}

	for name, src := range cases {
		pr := Parse("file:///x/A.cfc", src)
		if !slices.Equal(pr.Implements, want[name]) {
			t.Errorf("%s: Implements = %q, want %q", name, pr.Implements, want[name])
		}
	}

	if pr := Parse("file:///x/A.cfc", "component {\n}\n"); len(pr.Implements) != 0 {
		t.Errorf("no attribute: Implements = %q, want none", pr.Implements)
	}
}
