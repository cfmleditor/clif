package parser

import "testing"

func TestKeyTableInternsNamesOnce(t *testing.T) {
	kt := newKeyTable()

	a, b := kt.local("12:34", "userDAO"), kt.local("12:34", "userdao")
	if a != b {
		t.Errorf("one name in two spellings gave two keys")
	}

	if kt.name(a) != "userdao" {
		t.Errorf("name %q, want userdao", kt.name(a))
	}

	if !a.local() || kt.field(idVariables, "userDAO").local() {
		t.Errorf("kind lost: local %v", a.local())
	}

	// The same name as a local of another function, as a field of each scope,
	// and the keys standing for a whole scope are all different keys.
	keys := map[collKey]string{}

	for name, k := range map[string]collKey{
		"local 12:34":  a,
		"local 56:78":  kt.local("56:78", "userDAO"),
		"variables.":   kt.field(idVariables, "userDAO"),
		"this.":        kt.field(idThis, "userDAO"),
		"scope of fn":  scopeLevel(a),
		"scope of var": kt.fieldScope(idVariables),
		"scope of ths": kt.fieldScope(idThis),
	} {
		if prev, dup := keys[k]; dup {
			t.Errorf("%s and %s are the same key", prev, name)
		}

		keys[k] = name
	}

	if scopeLevel(a) != kt.localScope("12:34") {
		t.Errorf("a local's scope-level key is not its function's")
	}

	if got := kt.text(a); got != "12:34\tuserdao" {
		t.Errorf("local text %q", got)
	}

	if got := kt.text(kt.field(idThis, "Foo.Bar")); got != "this.foo.bar" {
		t.Errorf("field text %q", got)
	}

	if parent := kt.withName(a, "user"); parent.local() != a.local() || parent.scopeID() != a.scopeID() || kt.name(parent) != "user" {
		t.Errorf("withName changed the scope or kind")
	}
}

// A lookup must not add what it asks about: a name nothing has used cannot be
// in any map, and the table is not the place to remember that it was asked.
func TestLocalSetLookupAddsNothing(t *testing.T) {
	kt := newKeyTable()
	set := &localSet{kt: kt, m: map[collKey]bool{}}
	set.add("12:34", "Result")

	before := len(kt.strs)

	for _, c := range []struct {
		function, name string
		want           bool
	}{
		{"12:34", "result", true},
		{"12:34", "RESULT", true},
		{"12:34", "other", false},
		{"56:78", "result", false},
		{"never", "never", false},
	} {
		if got := set.has(c.function, c.name); got != c.want {
			t.Errorf("has(%q, %q) = %v, want %v", c.function, c.name, got, c.want)
		}
	}

	if len(kt.strs) != before {
		t.Errorf("lookups added %d strings to the table", len(kt.strs)-before)
	}
}
