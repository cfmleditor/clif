package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

// registryFixture is cut down from Preside's PageTypesService: getItem returns
// an element of what _getItems() returns, and _calculate types a local from
// getItem(). Typing that local asks getItem's return, which reads _getItems'
// body from the file's parse, whose typing asks getItem's return again.
const registryFixture = `component {
	public any function getItem( required string id ) {
		var items = _getItems();
		if ( StructKeyExists( items, arguments.id ) ) {
			return items[ arguments.id ];
		}
		throw( type="Registry.missing", message="missing" );
	}
	private void function _calculate() {
		var items         = _getItems();
		var visible = listVisible();
		for( var itemId in items ){
			var item = items[ itemId ];
			if ( !item.isVisible() ) {
				if ( Len( Trim( item.getParents() ) ) && item.getParents() != "*" ) {
					for( var parentId in ListToArray( item.getParents() ) ) {
						var parent = getItem( parentId );
						var managed = parent.getManaged();
						var allowed = parent.getChildren();
						parent.setManaged( ListAppend( managed, item.getId() ) );
						if ( allowed != "*" ) {
							allowed = ListToArray( allowed );
							allowed.delete( item.getId() );
							parent.setChildren( allowed.len() ? allowed.toList() : "none" );
						}
					}
				}
			}
			if ( item.getChildren() == "*" ) {
				item.setChildren( visible.toList() );
			}
		}
	}
	private struct function _getItems() {
		return _items;
	}
}
`

// A file whose parse asks what its own functions return is parsed a bounded
// number of times. The parse was cached only once it finished, so each ask
// inside it parsed the file again: this fixture took ten seconds, and
// Preside's checkout ran until the stack overflowed.
func TestAFileAskingAboutItselfIsNotParsedAgainWhileItIsParsed(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Registry.cfc": registryFixture})

	file := filepath.Join(dir, "Registry.cfc")
	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	r.EnsureIndexed(file)

	before := handlerParses.Load()

	if pr := r.handlerParse(file); pr == nil || len(pr.Funcs) != 3 {
		t.Fatal("no parse of the fixture")
	}

	if n := handlerParses.Load() - before; n > 2 {
		t.Fatalf("the fixture was parsed %d times", n)
	}
}

// What handlerParse caches for a file is the same whichever file was asked
// for first: a parse that read another file's unfinished parse is not kept.
func TestABorrowingParseIsNotCached(t *testing.T) {
	frame, reentered := enterHandlerParse("outer")
	if reentered {
		t.Fatal("outer entered twice")
	}

	inner, _ := enterHandlerParse("inner")

	if _, again := enterHandlerParse("outer"); !again {
		t.Fatal("re-entering outer was not seen")
	}

	if !inner.borrowed || frame.borrowed {
		t.Fatalf("inner borrowed %v, outer borrowed %v; want true, false", inner.borrowed, frame.borrowed)
	}

	leaveHandlerParse(inner)
	leaveHandlerParse(frame)

	if again, reentered := enterHandlerParse("outer"); reentered {
		t.Fatal("outer still in progress after it finished")
	} else {
		leaveHandlerParse(again)
	}
}
