package resolve

import "testing"

// TestIsScopeNameCoversKnownScopes holds isScopeName to the full list of CFML
// scope names it is meant to recognise, case-insensitively.
func TestIsScopeNameCoversKnownScopes(t *testing.T) {
	for _, name := range []string{
		"arguments", "local", "request", "session", "application", "server",
		"url", "form", "cgi", "cookie", "client", "super", "attributes",
		"caller", "thread",
		"Arguments", "LOCAL", "Super",
	} {
		if !isScopeName(name) {
			t.Errorf("isScopeName(%q) = false, want true", name)
		}
	}

	if isScopeName("foo") || isScopeName("") {
		t.Fatal("non-scope name reported as a scope")
	}
}

// TestContainsFoldIsCaseInsensitive covers the helper.
func TestContainsFoldIsCaseInsensitive(t *testing.T) {
	if !containsFold([]string{"Foo", "Bar"}, "foo") {
		t.Fatal("case-insensitive match missed")
	}

	if containsFold([]string{"Foo"}, "baz") {
		t.Fatal("non-member reported as present")
	}

	if containsFold(nil, "x") {
		t.Fatal("nil slice reported a match")
	}
}

// TestAllWordBytesAcceptsIdentifiers ensures only identifier-shaped names pass.
func TestAllWordBytesAcceptsIdentifiers(t *testing.T) {
	for _, name := range []string{"foo", "Foo_Bar", "_x", "a1"} {
		if !allWordBytes(name) {
			t.Errorf("allWordBytes(%q) = false, want true", name)
		}
	}

	for _, name := range []string{"", "a b", "a-b", "a.b", "1a?"} {
		if allWordBytes(name) {
			t.Errorf("allWordBytes(%q) = true, want false", name)
		}
	}
}

// TestExceedsCallerLimitHonoursZeroAsUnlimited documents that a non-positive
// limit disables the check.
func TestExceedsCallerLimitHonoursZeroAsUnlimited(t *testing.T) {
	if exceedsCallerLimit(100, 0) {
		t.Fatal("zero limit reported as exceeded")
	}

	if !exceedsCallerLimit(10, 5) {
		t.Fatal("over-limit count not reported")
	}

	if exceedsCallerLimit(5, 5) {
		t.Fatal("equal count reported as exceeded")
	}
}
