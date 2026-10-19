package parser

import "testing"

// An edit inside a function that adds no line resets no name index, and the
// memo was cleared only with those: typing one character left every answer
// derived from the old text in place until a line was added or removed.
func TestAnEditInsideAFunctionForgetsTheMemo(t *testing.T) {
	pr := Parse("file:///x.cfc", "component {\n\tfunction f() {\n\t\tvar a = 1;\n\t}\n}\n")

	builds := 0
	get := func() {
		pr.Memo("k", func() any {
			builds++

			return builds
		})
	}

	get()
	get()

	if builds != 1 {
		t.Fatalf("built %d times before the edit, want 1", builds)
	}

	if kind := pr.ApplyEdit(2, 10, 2, 11, "2"); kind != EditInFunc {
		t.Fatalf("edit kind %v, want EditInFunc", kind)
	}

	get()

	if builds != 2 {
		t.Errorf("built %d times after an edit, want 2: the memo outlived the text", builds)
	}
}
