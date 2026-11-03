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
