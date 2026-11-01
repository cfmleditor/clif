package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestSlatwallServicesReturnTheEntityTheirMissingMethodNames: Slatwall's
// BaseService answers newProduct(), getProduct( id ), getProductByCode( c ),
// saveProduct( p ) and getProductSmartList( rc ) with onMissingMethod, and
// its DAO loads the entity "Slatwall" & name. Every variable assigned one was
// untyped, and every call on it reported. Slatwall calls them through
// `this.`, which is what CFML hands onMissingMethod. A service with an onMissingMethod
// of its own, and a name no entity has, get nothing.
func TestSlatwallServicesReturnTheEntityTheirMissingMethodNames(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"com/service/BaseService.cfc": `component { function onMissingMethod( missingMethodName, missingMethodArguments ) {} }`,
		"com/service/ProductService.cfc": `component extends="BaseService" {
	function show( id ) {
		var a = this.newProduct();
		a.getProductName();
		a.nowhere();
		var b = this.getProduct( arguments.id, true );
		b.getProductName();
		var c = this.getProductByProductCode( "x" );
		c.getProductName();
		var s = this.getProductSmartList( {} );
		s.getPageRecords();
		this.newProduct().getProductName();
		var w = this.newWidget();
		w.anything();
	}
}`,
		"com/service/OtherService.cfc": `component { function onMissingMethod( missingMethodName, missingMethodArguments ) {}
	function show() {
		var a = this.newProduct();
		a.getProductName();
	}
}`,
		"com/entity/Product.cfc":            `component entityname="SlatwallProduct" persistent="true" { function getProductName() {} }`,
		"org/entitySmartList/SmartList.cfc": `component { function getPageRecords() {} }`,
	})

	r := &Resolver{Mappings: map[string]string{"Slatwall": dir}}

	expectReasons(t, reasonsWith(t, r, dir, "com/service/ProductService.cfc"), map[string]string{
		"a.getProductName":          "",
		"a.nowhere":                 "method 'nowhere' not found in Product",
		"b.getProductName":          "",
		"c.getProductName":          "",
		"s.getPageRecords":          "",
		"newProduct.getProductName": "",
		"w.anything":                "variable 'w' has no component ref",
	})

	r = &Resolver{Mappings: map[string]string{"Slatwall": dir}}

	expectReasons(t, reasonsWith(t, r, dir, "com/service/OtherService.cfc"), map[string]string{
		"a.getProductName": "variable 'a' has no component ref",
	})

	if got := r.slatwallGatewayReturn(filepath.Join(dir, "com", "service", "ProductService.cfc"), parser.CallHop(`saveProduct( p )`), dir); got != filepath.Join(dir, "com", "entity", "Product.cfc") {
		t.Errorf("saveProduct( p ): %q", got)
	}
}
