package index

import (
	"fmt"
	"slices"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

func TestIndexFile_LookupRoundTrip(t *testing.T) {
	idx := New()
	idx.IndexFile(uri.URI("file:///a.cfc"), `component {
	function getUser() {}
}`)

	defs := idx.Lookup("getUser")
	if len(defs) != 1 {
		t.Fatalf("expected 1 def, got %d", len(defs))
	}

	// Lookup is case-insensitive.
	if len(idx.Lookup("GETUSER")) != 1 {
		t.Error("expected case-insensitive lookup to find the function")
	}

	if len(idx.Lookup("missing")) != 0 {
		t.Error("expected no defs for unknown name")
	}
}

func TestIndexFile_ReindexReplacesNotDuplicates(t *testing.T) {
	idx := New()
	fileURI := uri.URI("file:///a.cfc")

	idx.IndexFile(fileURI, `component { function getUser() {} }`)
	idx.IndexFile(fileURI, `component { function getUser() {} function getOrder() {} }`)

	if got := len(idx.Lookup("getUser")); got != 1 {
		t.Errorf("expected re-indexing to replace, not duplicate: got %d getUser defs, want 1", got)
	}

	if got := len(idx.FunctionsForFile(fileURI)); got != 2 {
		t.Errorf("expected 2 functions for file after reindex, got %d", got)
	}
}

func TestIndexFile_MultipleFilesIsolated(t *testing.T) {
	idx := New()
	idx.IndexFile(uri.URI("file:///a.cfc"), `component { function foo() {} }`)
	idx.IndexFile(uri.URI("file:///b.cfc"), `component { function bar() {} }`)

	if got := len(idx.FunctionsForFile(uri.URI("file:///a.cfc"))); got != 1 {
		t.Errorf("expected 1 function in a.cfc, got %d", got)
	}

	if got := len(idx.FunctionsForFile(uri.URI("file:///b.cfc"))); got != 1 {
		t.Errorf("expected 1 function in b.cfc, got %d", got)
	}

	// AllFunctions sees both files.
	if got := len(idx.AllFunctions()); got != 2 {
		t.Errorf("expected 2 total functions, got %d", got)
	}
}

func TestShiftLines_OnlyAffectsMatchingFileAndLinesAfter(t *testing.T) {
	idx := New()

	target := uri.URI("file:///target.cfc")
	other := uri.URI("file:///other.cfc")

	funcs := []parser.FunctionDef{
		{Name: "before", URI: target, Line: 5},
		{Name: "after", URI: target, Line: 15},
		{Name: "exact", URI: target, Line: 10},
	}
	refs := []parser.ComponentRef{
		{Variable: "beforeRef", URI: target, Line: 5},
		{Variable: "afterRef", URI: target, Line: 15},
	}

	idx.IndexFileFromResult(target, funcs, refs)
	idx.IndexFileFromResult(other, []parser.FunctionDef{{Name: "otherFunc", URI: other, Line: 15}}, nil)

	idx.ShiftLines(target, 10, 3)

	get := func(name string) uint32 {
		defs := idx.Lookup(name)
		if len(defs) == 0 {
			t.Fatalf("expected a def named %q", name)
		}

		return defs[0].Line
	}

	if got := get("before"); got != 5 {
		t.Errorf("line at or before afterLine=10 should be untouched: got %d, want 5", got)
	}

	if got := get("exact"); got != 10 {
		t.Errorf("line exactly equal to afterLine should be untouched (only Line > afterLine shifts): got %d, want 10", got)
	}

	if got := get("after"); got != 18 {
		t.Errorf("line after afterLine should shift by delta: got %d, want 18", got)
	}

	if got := get("otherFunc"); got != 15 {
		t.Errorf("ShiftLines must not affect a different file: got %d, want 15 (unshifted)", got)
	}

	beforeRefs := idx.LookupComponentRef("beforeRef")
	if len(beforeRefs) != 1 || beforeRefs[0].Line != 5 {
		t.Errorf("beforeRef line should be untouched, got %+v", beforeRefs)
	}

	afterRefs := idx.LookupComponentRef("afterRef")
	if len(afterRefs) != 1 || afterRefs[0].Line != 18 {
		t.Errorf("afterRef line should shift to 18, got %+v", afterRefs)
	}
}

func TestRemoveFilesUnder_OnlyRemovesMatchingPrefix(t *testing.T) {
	idx := New()
	idx.IndexFileFromResult(uri.URI("file:///proj/a.cfc"),
		[]parser.FunctionDef{{Name: "fooA", URI: uri.URI("file:///proj/a.cfc"), Line: 1}}, nil)
	idx.IndexFileFromResult(uri.URI("file:///proj/sub/b.cfc"),
		[]parser.FunctionDef{{Name: "fooB", URI: uri.URI("file:///proj/sub/b.cfc"), Line: 1}}, nil)
	idx.IndexFileFromResult(uri.URI("file:///other/c.cfc"),
		[]parser.FunctionDef{{Name: "fooC", URI: uri.URI("file:///other/c.cfc"), Line: 1}}, nil)

	idx.RemoveFilesUnder("file:///proj/")

	if len(idx.Lookup("fooA")) != 0 {
		t.Error("expected fooA (under removed prefix) to be gone")
	}

	if len(idx.Lookup("fooB")) != 0 {
		t.Error("expected fooB (under removed prefix, nested) to be gone")
	}

	if len(idx.Lookup("fooC")) != 1 {
		t.Error("expected fooC (outside removed prefix) to remain")
	}
}

func TestLookupComponentRefInFile_PicksClosestPrecedingLine(t *testing.T) {
	idx := New()
	fileURI := uri.URI("file:///a.cfc")
	otherURI := uri.URI("file:///b.cfc")

	idx.SetFuncRefs(fileURI, "1:30", []parser.ComponentRef{
		{Variable: "svc", Component: "early", URI: fileURI, Line: 5},
		{Variable: "svc", Component: "late", URI: fileURI, Line: 20},
		{Variable: "svc", Component: "wrongFile", URI: otherURI, Line: 8},
	})

	// Querying at line 10 should return the ref at line 5 (closest preceding), not line 20.
	ref := idx.LookupComponentRefInFile("svc", fileURI, 10)
	if ref == nil || ref.Component != "early" {
		t.Errorf("expected closest preceding ref 'early' at line <=10, got %+v", ref)
	}

	// Querying at line 25 should now pick up the line-20 ref.
	ref = idx.LookupComponentRefInFile("svc", fileURI, 25)
	if ref == nil || ref.Component != "late" {
		t.Errorf("expected 'late' ref at line <=25, got %+v", ref)
	}

	// Querying before any ref exists should return nil.
	if ref := idx.LookupComponentRefInFile("svc", fileURI, 3); ref != nil {
		t.Errorf("expected nil when no ref precedes the queried line, got %+v", ref)
	}

	// A ref in a different file must never be returned.
	if ref := idx.LookupComponentRefInFile("svc", otherURI, 10); ref == nil || ref.Component != "wrongFile" {
		t.Errorf("expected the other file's own ref, got %+v", ref)
	}
}

func TestFindFilesByBasename(t *testing.T) {
	idx := New()
	idx.IndexFileFromResult(uri.URI("file:///proj/models/User.cfc"),
		[]parser.FunctionDef{{Name: "init", URI: uri.URI("file:///proj/models/User.cfc"), Line: 1}}, nil)

	// A file with no functions at all still needs to be findable via the fallback path.
	idx.IndexFileFromResult(uri.URI("file:///proj/models/Empty.cfc"), nil, nil)

	paths := idx.FindFilesByBasename("user")
	if len(paths) != 1 || paths[0] != "/proj/models/User.cfc" {
		t.Errorf("expected case-insensitive basename match with real casing preserved, got %v", paths)
	}

	paths = idx.FindFilesByBasename("empty")
	if len(paths) != 1 {
		t.Errorf("expected fallback match for a file with no functions, got %v", paths)
	}

	if len(idx.FindFilesByBasename("nonexistent")) != 0 {
		t.Error("expected no matches for a name that isn't indexed")
	}
}

func TestIndex_BeansAndEntities(t *testing.T) {
	idx := New()
	idx.SetBeans(map[string]string{"userservice": "app.services.UserService"})

	if got := idx.LookupBean("UserService"); got != "app.services.UserService" {
		t.Errorf("expected case-insensitive bean lookup, got %q", got)
	}

	if got := idx.LookupBean("unknown"); got != "" {
		t.Errorf("expected empty string for unknown bean, got %q", got)
	}

	user := uri.URI("file:///models/User.cfc")
	idx.IndexEntity(user, parser.Parse(user, `component persistent="true" entityName="User" {}`))

	if got := idx.NamedEntities("user"); len(got) != 1 || got[0] != uri.URI("file:///models/User.cfc") {
		t.Errorf("expected case-insensitive entity lookup, got %q", got)
	}
}

func TestIndex_ConcurrentAccess(_ *testing.T) {
	idx := New()

	done := make(chan struct{})

	go func() {
		for i := range 100 {
			idx.IndexFileFromResult(uri.URI("file:///a.cfc"),
				[]parser.FunctionDef{{Name: "foo", URI: uri.URI("file:///a.cfc"), Line: uint32(i)}}, nil)
		}

		done <- struct{}{}
	}()

	for range 100 {
		idx.Lookup("foo")
		idx.AllFunctions()
	}

	<-done
}

// A function's refs are indexed lazily, on the first hover or definition lookup
// that lands inside it, and re-indexed after every edit to that function. The
// plain append this replaced had no way to tell those apart, so each lookup
// left another copy behind and comprefs grew for as long as the session ran.
func TestSetFuncRefs_ReplacesRatherThanAccumulates(t *testing.T) {
	idx := New()
	fileURI := uri.URI("file:///svc.cfc")

	refs := []parser.ComponentRef{
		{Variable: "svc", Component: "models.User", URI: fileURI, Line: 7},
	}

	for range 5 {
		idx.SetFuncRefs(fileURI, "3:12", refs)
	}

	if got := len(idx.LookupComponentRef("svc")); got != 1 {
		t.Errorf("re-indexing one scope five times left %d refs, want 1", got)
	}

	if got := len(idx.RefsForFile(fileURI)); got != 1 {
		t.Errorf("file refs grew to %d, want 1", got)
	}

	// A second scope in the same file is additive, not a replacement.
	idx.SetFuncRefs(fileURI, "20:30", []parser.ComponentRef{
		{Variable: "dao", Component: "models.UserDAO", URI: fileURI, Line: 22},
	})

	if got := len(idx.RefsForFile(fileURI)); got != 2 {
		t.Errorf("second scope gave %d file refs, want 2", got)
	}

	// Replacing a scope drops the refs it contributed and nothing else.
	idx.SetFuncRefs(fileURI, "3:12", []parser.ComponentRef{
		{Variable: "svc", Component: "models.Account", URI: fileURI, Line: 7},
	})

	got := idx.LookupComponentRef("svc")
	if len(got) != 1 || got[0].Component != "models.Account" {
		t.Errorf("expected the scope's single ref to be replaced, got %+v", got)
	}

	if n := len(idx.LookupComponentRef("dao")); n != 1 {
		t.Errorf("replacing one scope disturbed another: dao has %d refs, want 1", n)
	}
}

// The files come out of a map, so without a sort their order changed from call
// to call, and a caller choosing among equals took whichever came first. Ten
// matches make an accidental pass one in 3.6 million.
func TestFindFilesByBasenameIsSorted(t *testing.T) {
	idx := New()

	for i := 9; i >= 0; i-- {
		u := uri.URI(fmt.Sprintf("file:///proj/patches/d%d/Widget.cfc", i))
		idx.IndexFileFromResult(u, []parser.FunctionDef{{Name: "run", URI: u, Line: 1}}, nil)
	}

	paths := idx.FindFilesByBasename("widget")
	if len(paths) != 10 || !slices.IsSorted(paths) {
		t.Errorf("want 10 sorted paths, got %v", paths)
	}
}

// Every file declaring an entity name is kept, sorted, and a re-index or a
// removal drops only that file's own names.
func TestEveryDeclarationOfAnEntityNameIsKept(t *testing.T) {
	idx := New()
	a, b := uri.URI("file:///b/Writer.cfc"), uri.URI("file:///a/Writer.cfc")

	idx.IndexFile(a, `component persistent="true" entityName="Author" {}`)
	idx.IndexFile(b, `component persistent="true" entityName="Author" {}`)

	if got := idx.NamedEntities("author"); !slices.Equal(got, []uri.URI{b, a}) {
		t.Fatalf("Author is declared by %v, want both, sorted", got)
	}

	idx.IndexFile(a, `component persistent="true" entityName="Writer" {}`)

	if got := idx.NamedEntities("author"); !slices.Equal(got, []uri.URI{b}) {
		t.Fatalf("after a re-index renamed one, Author is declared by %v", got)
	}

	if got := idx.NamedEntities("writer"); !slices.Equal(got, []uri.URI{a}) {
		t.Fatalf("Writer is declared by %v", got)
	}

	idx.RemoveFile(b)

	if got := idx.NamedEntities("author"); len(got) != 0 {
		t.Fatalf("after its removal, Author is declared by %v", got)
	}
}

// TestAReindexReplacesAFilesEntity: each re-index of a persistent component
// named after its file appended it again, and one that went on to declare an
// entityname, or stopped being persistent, stayed a candidate under its file
// name. The editor's IndexEntity must replace in the same way, and must not
// file the file's name as a declared entityname.
func TestAReindexReplacesAFilesEntity(t *testing.T) {
	idx := New()
	author := uri.URI("file:///app/Author.cfc")

	for range 3 {
		idx.IndexFile(author, `component persistent="true" {}`)
	}

	if got := idx.EntityCandidates("author"); !slices.Equal(got, []uri.URI{author}) {
		t.Fatalf("after three re-indexes, Author's candidates are %v", got)
	}

	idx.IndexFile(author, `component persistent="true" entityName="cbAuthor" {}`)

	if got := idx.EntityCandidates("author"); len(got) != 0 {
		t.Fatalf("Author.cfc declares cbAuthor, yet Author's candidates are %v", got)
	}

	idx.IndexFile(author, `component {}`)

	if got := idx.EntityCandidates("cbauthor"); len(got) != 0 {
		t.Fatalf("Author.cfc is no longer persistent, yet cbAuthor's candidates are %v", got)
	}

	idx.IndexEntity(author, parser.Parse(author, `component persistent="true" entityName="cbAuthor" {}`))
	idx.IndexEntity(author, parser.Parse(author, `component persistent="true" entityName="cbAuthor" {}`))

	if got := idx.NamedEntities("author"); len(got) != 0 {
		t.Fatalf("the file's name was filed as a declared entity name: %v", got)
	}

	idx.IndexEntity(author, parser.Parse(author, `component persistent="true" entityName="cbWriter" {}`))

	if got := idx.NamedEntities("cbauthor"); len(got) != 0 {
		t.Fatalf("after renaming its entity, cbAuthor is still declared by %v", got)
	}

	if got := idx.NamedEntities("cbwriter"); !slices.Equal(got, []uri.URI{author}) {
		t.Fatalf("cbWriter is declared by %v", got)
	}

	idx.IndexEntity(author, parser.Parse(author, `component entityName="cbWriter" {}`))

	if got := idx.EntityCandidates("cbwriter"); len(got) != 0 {
		t.Fatalf("outside ORM scope, cbWriter is still declared by %v", got)
	}
}

// TestADeclaredEntityDoesNotHideANearerOneByFileName: EntityCandidates
// offered the declared entities alone whenever there were any, so an Author
// declared by entityname in one application hid a plain persistent
// Author.cfc in another, however near the caller.
func TestADeclaredEntityDoesNotHideANearerOneByFileName(t *testing.T) {
	idx := New()
	declared, byFile := uri.URI("file:///a/Writer.cfc"), uri.URI("file:///b/Author.cfc")

	idx.IndexFile(declared, `component persistent="true" entityName="Author" {}`)
	idx.IndexFile(byFile, `component persistent="true" {}`)

	if got := idx.EntityCandidates("author"); !slices.Equal(got, []uri.URI{declared, byFile}) {
		t.Fatalf("Author's candidates are %v, want the declared one then the one by file name", got)
	}
}

// TestRemovingAFolderDropsItsEntities: RemoveFilesUnder cleared a folder's
// functions and refs but left its entities as candidates.
func TestRemovingAFolderDropsItsEntities(t *testing.T) {
	idx := New()
	kept, gone, plain := uri.URI("file:///keep/Writer.cfc"), uri.URI("file:///gone/Writer.cfc"), uri.URI("file:///gone/Poet.cfc")

	idx.IndexFile(kept, `component persistent="true" entityName="Author" {}`)
	idx.IndexFile(gone, `component persistent="true" entityName="Author" {}`)
	idx.IndexFile(plain, `component persistent="true" {}`)

	idx.RemoveFilesUnder("file:///gone/")

	if got := idx.EntityCandidates("author"); !slices.Equal(got, []uri.URI{kept}) {
		t.Errorf("after removing /gone, Author's candidates are %v", got)
	}

	if got := idx.EntityCandidates("poet"); len(got) != 0 {
		t.Errorf("after removing /gone, Poet's candidates are %v", got)
	}
}

// TestRemovingAFolderLeavesItsSibling: removing /work/app must not take
// /work/app2 with it.
func TestRemovingAFolderLeavesItsSibling(t *testing.T) {
	idx := New()
	sibling := uri.URI("file:///work/app2/Writer.cfc")

	idx.IndexFile(uri.URI("file:///work/app/Writer.cfc"), `component persistent="true" entityName="Author" { function f() {} }`)
	idx.IndexFile(sibling, `component persistent="true" entityName="Author" { function f() {} }`)

	idx.RemoveFilesUnder("file:///work/app")

	if got := idx.EntityCandidates("author"); !slices.Equal(got, []uri.URI{sibling}) {
		t.Errorf("after removing /work/app, Author's candidates are %v", got)
	}

	if !idx.HasFile(sibling) {
		t.Error("removing /work/app removed /work/app2's file")
	}
}

// TestRefreshEntityFollowsOnlyAFileItHolds: RefreshEntity adds nothing for a
// file the index holds no entity for, and replaces one it does.
func TestRefreshEntityFollowsOnlyAFileItHolds(t *testing.T) {
	idx := New()
	held, stray := uri.URI("file:///a/Writer.cfc"), uri.URI("file:///b/Writer.cfc")

	idx.IndexFile(held, `component persistent="true" entityName="Author" {}`)

	idx.RefreshEntity(stray, parser.Parse(stray, `component persistent="true" entityName="Poet" {}`))

	if got := idx.EntityCandidates("poet"); len(got) != 0 {
		t.Fatalf("a file the index held no entity for registered Poet: %v", got)
	}

	idx.RefreshEntity(held, parser.Parse(held, `component persistent="true" entityName="Poet" {}`))

	if got := idx.EntityCandidates("author"); len(got) != 0 {
		t.Errorf("after the rename, Author is declared by %v", got)
	}

	if got := idx.EntityCandidates("poet"); !slices.Equal(got, []uri.URI{held}) {
		t.Errorf("after the rename, Poet is declared by %v", got)
	}
}

// TestRefreshEntityFollowsAFileThroughAnEditThatDropsIt: an edit that leaves
// a component no entity, then one that restores it, registers it again.
func TestRefreshEntityFollowsAFileThroughAnEditThatDropsIt(t *testing.T) {
	idx := New()
	held := uri.URI("file:///a/Writer.cfc")

	idx.IndexFile(held, `component persistent="true" entityName="Author" {}`)
	idx.RefreshEntity(held, parser.Parse(held, `component entityName="Author" {}`))

	if got := idx.EntityCandidates("author"); len(got) != 0 {
		t.Fatalf("no longer persistent, yet Author is declared by %v", got)
	}

	idx.RefreshEntity(held, parser.Parse(held, `component persistent="true" entityName="Author" {}`))

	if got := idx.EntityCandidates("author"); !slices.Equal(got, []uri.URI{held}) {
		t.Errorf("persistent again, Author is declared by %v", got)
	}

	idx.RemoveFile(held)
	idx.RefreshEntity(held, parser.Parse(held, `component persistent="true" entityName="Author" {}`))

	if got := idx.EntityCandidates("author"); len(got) != 0 {
		t.Errorf("after the file's removal a refresh registered Author: %v", got)
	}
}

// TestOnlyComponentsAreRecordedAsRead: entityRead exists for RefreshEntity,
// which only asks about components, so a page is never recorded.
func TestOnlyComponentsAreRecordedAsRead(t *testing.T) {
	idx := New()

	idx.IndexFile(uri.URI("file:///a/page.cfm"), `<cfoutput>hi</cfoutput>`)
	idx.IndexFile(uri.URI("file:///a/Plain.cfc"), `component {}`)

	if _, ok := idx.entityRead[uriKey(uri.URI("file:///a/page.cfm"))]; ok {
		t.Error("a page was recorded as an entity read")
	}

	if _, ok := idx.entityRead[uriKey(uri.URI("file:///a/Plain.cfc"))]; !ok {
		t.Error("a component was not recorded as read")
	}
}

// TestEntityChoicesSplitsTheDeclaredOnes: declared is exactly the files
// declaring the name, sorted, and the prefix of all.
func TestEntityChoicesSplitsTheDeclaredOnes(t *testing.T) {
	idx := New()
	b, a, plain := uri.URI("file:///b/Writer.cfc"), uri.URI("file:///a/Writer.cfc"), uri.URI("file:///c/Author.cfc")

	idx.IndexFile(b, `component persistent="true" entityName="Author" {}`)
	idx.IndexFile(plain, `component persistent="true" {}`)
	idx.IndexFile(a, `component persistent="true" entityName="Author" {}`)

	declared, all := idx.EntityChoices("author")

	if !slices.Equal(declared, []uri.URI{a, b}) {
		t.Errorf("declared is %v", declared)
	}

	if !slices.Equal(all, []uri.URI{a, b, plain}) {
		t.Errorf("all is %v", all)
	}
}
