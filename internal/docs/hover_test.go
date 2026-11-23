package docs

import (
	"slices"
	"testing"
)

func TestFunctionPreviewsShowDocumentedOverloads(t *testing.T) {
	entry := Entry{
		Name:    "choose",
		Returns: "models.Choice",
		Syntax:  "choose(string key)\nor\nchoose(string key, [models.Choice fallback]);",
		Params: []Param{
			{Name: "fallback", Type: "models.Choice"},
			{Name: "key", Type: "string", Required: true},
		},
	}
	want := []string{
		"models.Choice function choose(required string key)",
		"models.Choice function choose(required string key, models.Choice fallback)",
	}

	if got := entry.FunctionPreviews(); !slices.Equal(got, want) {
		t.Fatalf("previews = %q, want %q", got, want)
	}
}
