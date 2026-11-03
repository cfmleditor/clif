package resolve

import "testing"

// TestAnEntityNameIsTheEntityNotTheFileBesideTheCaller: FW/1's qBall writes
// entityNew( "question" ) in services/question.cfc, whose own file a path
// lookup finds; the name is the persistent beans/question.cfc's. With an
// application of each beside the other, each takes its own nearest entity.
func TestAnEntityNameIsTheEntityNotTheFileBesideTheCaller(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{}

	for _, app := range []string{"appA", "appB"} {
		files[app+"/model/beans/question.cfc"] = `component persistent="true" accessors="true" { property name="` + app + `Field"; }`
		files[app+"/model/services/question.cfc"] = `component {
	function post(){
		var q = entityNew( "question" );
		q.set` + app + `Field( 1 );
		q.nope();
		return q;
	}
}`
	}

	writeFiles(t, dir, files)

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "appA/model/services/question.cfc"), map[string]string{
		"q.setappAField": "",
		"q.nope":         "method 'nope' not found in question",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "appB/model/services/question.cfc"), map[string]string{
		"q.setappBField": "",
	})
}

// TestADeclaredEntityNameIsTheNearestDeclaration: Lucee's tests declare
// entityName="Author" in three directories. The index kept one file per name,
// the last indexed, so which one entityNew( "Author" ) named depended on the
// order a parallel scan finished in, and one application read another's
// entity whichever won.
func TestADeclaredEntityNameIsTheNearestDeclaration(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{}

	for _, app := range []string{"appA", "appB"} {
		files[app+"/model/beans/Writer.cfc"] = `component persistent="true" entityName="Author" accessors="true" { property name="` + app + `Field"; }`
		files[app+"/model/services/books.cfc"] = `component {
	function post(){
		var a = entityNew( "Author" );
		a.set` + app + `Field( 1 );
		return a;
	}
}`
	}

	writeFiles(t, dir, files)

	for _, app := range []string{"appA", "appB"} {
		expectReasons(t, reasonsWith(t, &Resolver{}, dir, app+"/model/services/books.cfc"), map[string]string{
			"a.set" + app + "Field": "",
		})
	}
}

// TestAnEntityByFileNameIsNotHiddenByOneDeclaredElsewhere: appA declares
// entityName="Author" on Writer.cfc, and appB has a plain persistent
// Author.cfc. appB's entityNew( "Author" ) is its own Author.cfc, and was
// appA's Writer.cfc while a declared name hid every file-named candidate.
func TestAnEntityByFileNameIsNotHiddenByOneDeclaredElsewhere(t *testing.T) {
	dir := t.TempDir()

	writeFiles(t, dir, map[string]string{
		"appA/model/Writer.cfc": `component persistent="true" entityName="Author" accessors="true" { property name="pen"; }`,
		"appB/model/Author.cfc": `component persistent="true" accessors="true" { property name="quill"; }`,
		"appB/model/books.cfc": `component {
	function post(){
		var a = entityNew( "Author" );
		a.setQuill( 1 );
		return a;
	}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "appB/model/books.cfc"), map[string]string{
		"a.setQuill": "",
	})
}

// TestADeclaredEntityInTheCallersApplicationOutranksANearerFile: within one
// application the entity is the component declaring the name, not a nearer
// persistent component that merely shares it.
func TestADeclaredEntityInTheCallersApplicationOutranksANearerFile(t *testing.T) {
	dir := t.TempDir()

	writeFiles(t, dir, map[string]string{
		"app/Application.cfc":       `component { this.ormEnabled = true; }`,
		"app/models/orm/Writer.cfc": `component persistent="true" entityName="Author" accessors="true" { property name="pen"; }`,
		"app/legacy/Author.cfc":     `component persistent="true" accessors="true" { property name="quill"; }`,
		"app/legacy/books.cfc": `component {
	function post(){
		var a = entityNew( "Author" );
		a.setPen( 1 );
		return a;
	}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "app/legacy/books.cfc"), map[string]string{
		"a.setPen": "",
	})
}

// TestADeclaredEntityWinsATieWithOneByFileName: with no application to
// decide, a declared entity and a component called after the name, equally
// near the caller, resolve to the declared one.
func TestADeclaredEntityWinsATieWithOneByFileName(t *testing.T) {
	dir := t.TempDir()

	writeFiles(t, dir, map[string]string{
		"x/Writer.cfc": `component persistent="true" entityName="Author" accessors="true" { property name="pen"; }`,
		"y/Author.cfc": `component persistent="true" accessors="true" { property name="quill"; }`,
		"z/books.cfc": `component {
	function post(){
		var a = entityNew( "Author" );
		a.setPen( 1 );
		return a;
	}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "z/books.cfc"), map[string]string{
		"a.setPen": "",
	})
}

// TestANestedApplicationsEntityIsNotTheCallers: a test application nested
// under the caller's declares entityName="Author". It is under the caller's
// application folder but has an Application.cfc of its own, so from the
// outer application entityNew( "Author" ) is the outer one's Author.cfc, and
// from inside the nested one it is the nested one's Writer.cfc.
func TestANestedApplicationsEntityIsNotTheCallers(t *testing.T) {
	dir := t.TempDir()
	books := func(setter string) string {
		return `component {
	function post(){
		var a = entityNew( "Author" );
		a.` + setter + `( 1 );
		return a;
	}
}`
	}

	writeFiles(t, dir, map[string]string{
		"root/Application.cfc":                `component { this.ormEnabled = true; }`,
		"root/models/Author.cfc":              `component persistent="true" accessors="true" { property name="quill"; }`,
		"root/models/books.cfc":               books("setQuill"),
		"root/tests/orm/Application.cfc":      `component { this.ormEnabled = true; }`,
		"root/tests/orm/model/Writer.cfc":     `component persistent="true" entityName="Author" accessors="true" { property name="pen"; }`,
		"root/tests/orm/specs/deep/books.cfc": books("setPen"),
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "root/models/books.cfc"), map[string]string{
		"a.setQuill": "",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "root/tests/orm/specs/deep/books.cfc"), map[string]string{
		"a.setPen": "",
	})
}
