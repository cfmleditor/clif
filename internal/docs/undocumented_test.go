package docs

import "testing"

// TestLuceeUndocumentedFunctionsAreBuiltins: struct() is a Lucee function its
// published docs omit (<status>hidden</status>), and its administrator calls
// it 137 times, each reported as a call to a function nobody declared.
func TestLuceeUndocumentedFunctionsAreBuiltins(t *testing.T) {
	for name, want := range map[string]bool{
		"struct":         true,
		"Struct":         true,
		"sessionTouch":   true,
		"dumpStruct":     true,
		"MongoDBConnect": true, "LuceeAIHas": true,
		"arrayLen":  true,  // documented
		"cfhttp":    true,  // a tag called as a function
		"structure": false, // not a function at all
	} {
		if got := IsBuiltinFunction(name); got != want {
			t.Errorf("IsBuiltinFunction(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestUndocumentedFunctionsAreNotDocumented: the list is for what the docs
// cannot see. When a regeneration picks one up, it belongs to the docs and
// should leave the list, so the list never grows into a second copy of them.
func TestUndocumentedFunctionsAreNotDocumented(t *testing.T) {
	for name := range undocumentedFunctions {
		if _, ok := LookupFunction(name); ok {
			t.Errorf("%s is documented now; remove it from undocumentedFunctions", name)
		}
	}
}

func TestBuiltinReturnObjectsKeepTheirJavaMembers(t *testing.T) {
	for _, tc := range [][2]string{
		{"xmlparse", "createCDATASection"},
		{"xmlnew", "getDocumentURI"},
		{"fileopen", "getStatus"},
		{"imagenew", "drawBeveledRect"},
		{"imageread", "writeBase64"},
	} {
		if !LookupBuiltinMethod(tc[0], tc[1]) {
			t.Errorf("%s.%s not accepted", tc[0], tc[1])
		}
	}

	if LookupBuiltinMethod("xmlparse", "noSuchMethod") {
		t.Error("an unknown xml member was accepted")
	}
}

func TestABuiltinDocumentedToReturnAnyIsDynamic(t *testing.T) {
	if got := LookupBuiltinReturnComponent("cacheGetSession"); got != "$any" {
		t.Errorf("cacheGetSession: %q, want $any", got)
	}

	if got := LookupBuiltinReturnComponent("arrayLen"); got == "$any" {
		t.Errorf("arrayLen is documented to return a number, not any")
	}
}
