package resolve

import "testing"

// A call on arguments.x is typed from the enclosing function's argument list:
// a dotted type is the component, and a primitive type calling a member
// method of its own needs no component at all.
func TestAnArgumentQualifiedReceiverIsTypedFromTheArgumentList(t *testing.T) {
	files := map[string]string{
		"app/Svc.cfc": `component { function save() {} }`,
		"A.cfc": `component {
	function run( required app.Svc svc, required string name, required any loose ) {
		arguments.svc.save();
		arguments.svc.absent();
		arguments.name.toCharArray();
		arguments.loose.anything();
	}
}`,
	}

	reasons := reasonsFor(t, files, nil)

	if reasons["arguments.svc.save"] != "" {
		t.Errorf("arguments.svc.save: %q, want resolved", reasons["arguments.svc.save"])
	}

	if reasons["arguments.name.toCharArray"] != "" {
		t.Errorf("arguments.name.toCharArray: %q, want accepted as a member method", reasons["arguments.name.toCharArray"])
	}

	t.Logf("%v", reasons)
}
