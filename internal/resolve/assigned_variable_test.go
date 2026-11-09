package resolve

import "testing"

// splitAssignedVariable says which assignment a lookup of a variable reads:
// the bare name, and the container (prc, rc, arguments) it is a key of, or
// whether it was spelled variables.x.
func TestSplitAssignedVariable(t *testing.T) {
	for _, c := range []struct {
		in        string
		name      string
		container string
		scoped    bool
	}{
		{"x", "x", "", false},
		{"variables.x", "x", "", true},
		{"VARIABLES.Cache", "Cache", "", true},
		{"local.x", "x", "", false},
		{"prc.page", "page", "prc", false},
		{"PRC.page", "page", "prc", false},
		{"rc.id", "id", "rc", false},
		{"arguments.rc.id", "id", "rc", false},
		{"arguments.item", "item", "arguments", false},
		{"this.x", "this.x", "", false},
		{"a.b", "a.b", "", false},
		{"prc.", "", "prc", false},
	} {
		name, container, scoped := splitAssignedVariable(c.in)
		if name != c.name || container != c.container || scoped != c.scoped {
			t.Errorf("splitAssignedVariable(%q) = %q, %q, %v; want %q, %q, %v",
				c.in, name, container, scoped, c.name, c.container, c.scoped)
		}
	}
}
