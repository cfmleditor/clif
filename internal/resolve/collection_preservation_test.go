package resolve

import "testing"

func TestORMCollectionElementsSurvivePreservingOperations(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Item.cfc":   `component persistent="true" entityname="KnownItem" {function ready(){}}`,
		"Holder.cfc": `component persistent="true" {property name="items" fieldtype="one-to-many" cfc="Item";}`,
		"Use.cfc": `component {function run(){
 var holder=new Holder();
 var selected=holder.getItems().filter(function(entry){return true;}).slice(1);
 for(var selectedItem in selected){selectedItem.ready();selectedItem.absent();}
 var rows=entityLoad("KnownItem");
 rows.filter(function(entry){return true;}).each(function(chained){chained.ready();chained.absent();});
 var globalSelected=arrayFilter(rows,function(entry){return true;});
 globalSelected.each(function(globalItem){globalItem.ready();});
 entityLoad("KnownItem").filter(function(entry){return true;}).each(function(direct){direct.ready();direct.absent();});
 var copied=rows.duplicate();
 for(var copiedItem in copied){copiedItem.ready();}
 var mapped=rows.map(function(entry){return runtimeValue;});
 mapped.each(function(changed){changed.ready();});
 var unknown=runtimeList();
 unknown.filter(function(entry){return true;}).each(function(untyped){untyped.ready();});
 var transformed=holder.getItems().map(function(entry){return runtimeValue;});
 for(var transformedItem in transformed){transformedItem.ready();}
 var brokenHop=holder.getItems().missing();
 for(var brokenItem in brokenHop){brokenItem.ready();}
 var broken=holder.missing().slice(1);
 for(var bad in broken){bad.ready();}
 }}`,
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Use.cfc"), map[string]string{
		"selectedItem.ready": "", "selectedItem.absent": "method 'absent' not found in Item",
		"chained.ready": "", "chained.absent": "method 'absent' not found in Item",
		"globalItem.ready": "", "direct.ready": "", "direct.absent": "method 'absent' not found in Item",
		"copiedItem.ready": "",
		"changed.ready":    "variable 'changed' has no component ref", "untyped.ready": "variable 'untyped' has no component ref", "transformedItem.ready": "variable 'transformedItem' has no component ref", "brokenItem.ready": "variable 'brokenItem' has no component ref",
		"bad.ready": "variable 'bad' has no component ref",
	})
}

func TestStructCollectionFieldRejectsUnresolvedReturnChains(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"orm/VirtualEntityService.cfc": `component {function init(entityName){return this;} function newCriteria(){}}`,
		"Item.cfc":                     `component persistent="true" entityname="KnownItem" {function ready(){}}`,
		"Service.cfc": `component extends="orm.VirtualEntityService" {
 function init(){super.init(entityName="KnownItem");return this;}
 struct function search(){var result={"rows":[]};var criteria=newCriteria();result.rows=criteria.list(asQuery=false);return result;}
 }`,
		"Use.cfc": `component {function run(){
 var service=new Service();
 var good=service.search();
 for(var goodItem in good.rows){goodItem.ready();}
 var bad=service.search().missing();
 for(var badItem in bad.rows){badItem.ready();}
 }}`,
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Use.cfc"), map[string]string{"goodItem.ready": "", "badItem.ready": "variable 'badItem' has no component ref"})
}
