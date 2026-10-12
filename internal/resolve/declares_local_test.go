package resolve

import (
	"regexp"
	"testing"
)

func declaresLocalReference(text, name string) bool {
	return regexp.MustCompile(`(?i)(?:\bvar\s+|\blocal\.|<cfargument\s[^>]*name\s*=\s*["'])` + regexp.QuoteMeta(name) + `\b`).MatchString(text)
}

// textDeclaresLocal replaced a regular expression compiled per name; it must
// answer as the expression did.
func TestTextDeclaresLocalMatchesTheExpression(t *testing.T) {
	for _, tc := range []struct{ text, name string }{
		{"var svc = new A();", "svc"},
		{"VAR\tSvc = 1", "svc"},
		{"myvar svc = 1", "svc"},
		{"var svcs = 1", "svc"},
		{"local.svc = 1", "svc"},
		{"xlocal.svc = 1", "svc"},
		{"local.svc2 = 1", "svc"},
		{`<cfargument name="svc" type="any">`, "svc"},
		{`<CFARGUMENT required="true" NAME = 'svc'>`, "svc"},
		{`<cfargument type="x"> name="svc"`, "svc"},
		{`<cfargument typename="svc">`, "svc"},
		{`<cfargumentx name="svc">`, "svc"},
		{"svc = 1; variables.svc = 2", "svc"},
		{"var svc", "svc"},
		{"local.svc", "svc"},
		{"", "svc"},
	} {
		if got, want := textDeclaresLocal(tc.text, tc.name), declaresLocalReference(tc.text, tc.name); got != want {
			t.Errorf("textDeclaresLocal(%q, %q) = %v, want %v", tc.text, tc.name, got, want)
		}
	}
}
