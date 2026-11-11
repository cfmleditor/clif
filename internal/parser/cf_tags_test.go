package parser

import (
	"slices"
	"testing"
)

// CFTags reads tags as the parse does: a commented-out tag is none, nested
// comment and all; a '>' inside a quoted value does not end the tag; names
// are matched whole and without case; attributes are read by the CF-mode
// scanner, an unquoted value and a doubled quote included.
func TestCFTagsReadTagsAsTheParseDoes(t *testing.T) {
	src := `<!--- <cfloop array="#no#" index="x"> <!--- n ---> <cfloop array="#no2#"> --->` +
		`<CFLOOP array="#items#" index="it" hint="a > b">` +
		`<cfloopx array="#nope#">` +
		`<cfloop list="a,b" item=x></cfloop>` +
		`</CFLoop>` +
		`<cfargument name="q" default="say ""hi""">`

	tags := CFTags(src, "cfloop")

	var got []string

	for i := range tags {
		if tags[i].Close {
			got = append(got, "/")

			continue
		}

		attrs := tags[i].Attrs()
		got = append(got, attrs["array"]+"|"+attrs["index"]+"|"+attrs["item"]+"|"+attrs["hint"])
	}

	if want := []string{"#items#|it||a > b", "||x|", "/", "/"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}

	arg := CFTags(src, "cfargument")
	if len(arg) != 1 {
		t.Fatalf("cfargument tags: %d", len(arg))
	}

	if d, _ := arg[0].Attr("default"); d != `say ""hi""` {
		t.Errorf("default = %q", d)
	}

	if body := CFTags(`<cfreturn items[1] />`, "cfreturn")[0].Body(); body != "items[1]" {
		t.Errorf("Body = %q", body)
	}

	if got := WithoutCFTags(`a<cfargument name="x">b<!--- <cfargument name="y"> --->c`, "cfargument"); got != `ab<!--- <cfargument name="y"> --->c` {
		t.Errorf("WithoutCFTags = %q", got)
	}
}
