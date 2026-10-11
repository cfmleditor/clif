package nativetype

import "testing"

func TestBuiltinCollectionErasedContracts(t *testing.T) {
	for _, tc := range []struct {
		family, method, want string
		ok                   bool
	}{
		{"struct", "$property:mappings", "$any", true},
		{"array", "$property:mappings", "", false},
		{"array", "toArray", "$any", true},
		{"array", "subList", Type("list"), true},
		{"list", "get", "$any", true},
		{"array", "getDimension", Type("numeric"), true},
		{"array", "hasNext", "", false},
	} {
		got, ok := Method(Type(tc.family), tc.method)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s.%s: %q %v want %q %v", tc.family, tc.method, got, ok, tc.want, tc.ok)
		}
	}
}
