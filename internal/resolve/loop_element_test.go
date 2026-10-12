package resolve

import "testing"

// TestALoopVariableHoldsTheCollectionsElement: a loop over a persistent
// entity's collection relationship, read through its getter or its property,
// or over a cborm service's getAll(), gives its variable the entity. A
// subclass of the entity is an element too. Outside the loop body, over
// properties-only results and over a collection nothing types, the variable
// stays untyped.
func TestALoopVariableHoldsTheCollectionsElement(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"orm/VirtualEntityService.cfc": `component { function init( entityName ){ return this; } function getAll( id, sortOrder, properties ){} }`,
		"models/Comment.cfc":           `component persistent="true" entityname="cbComment" { function getText(){} }`,
		"models/ReplyComment.cfc":      `component persistent="true" extends="Comment" { function getParentText(){} }`,
		"models/CommentService.cfc":    `component extends="orm.VirtualEntityService" { function init(){ super.init( entityName = "cbComment" ); return this; } }`,
		"models/Post.cfc": `component persistent="true" {
property name="comments" fieldtype="one-to-many" cfc="Comment";
function scriptLoops( svc ){
	for ( var c in getComments() ) {
		c.getText();
		c.nope();
		c.getParentText();
	}
	for ( var p in variables.comments ) {
		p.getText();
	}
	var all = getComments();
	for ( var a in all ) {
		a.getText();
	}
	var service = new CommentService();
	for ( var s in service.getAll() ) {
		s.getText();
	}
	for ( var q in service.getAll( properties = "text" ) ) {
		q.getText();
	}
	for ( var u in unknownThings ) {
		u.getText();
	}
	after.getText();
}
}`,
		"views/list.cfm": `<cfset post = new models.Post()>
<cfloop array="#post.getComments()#" index="row">
	<cfoutput>#row.getText()#</cfoutput>
</cfloop>
<cfset row.later()>`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "models/Post.cfc"), map[string]string{
		"c.getText":       "",
		"c.nope":          "method 'nope' not found in ReplyComment|Comment",
		"c.getParentText": "",
		"p.getText":       "",
		"a.getText":       "",
		"s.getText":       "",
		"q.getText":       "variable 'q' has no component ref",
		"u.getText":       "variable 'u' has no component ref",
		"after.getText":   "variable 'after' has no component ref",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "views/list.cfm"), map[string]string{
		"row.getText": "",
		"row.later":   "variable 'row' has no component ref",
	})
}

func TestLoadedEntityCollectionsTypeLoopsAndEachCallbacks(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/Item.cfc":   `component persistent="true" entityname="LoadedItem" { function ready(){} }`,
		"models/Holder.cfc": `component persistent="true" { property name="items" fieldtype="one-to-many" cfc="Item"; }`,
		"Use.cfc": `component { function run(){
   var rows=entityLoad("LoadedItem");
   for(var row in rows){ row.ready(); row.missing(); }
   rows.each(function(entry){entry.ready();entry.missing();});
   var filtered=entityLoad("LoadedItem", {enabled=true});
   for(var f in filtered){f.ready();}
   var singleton=entityLoad("LoadedItem", {}, true);
   for(var single in singleton){single.ready();}
   var uncertain=entityLoad("LoadedItem", {}, uniqueFlag);
   for(var unsure in uncertain){unsure.ready();}
   var holder=new models.Holder();
   var related=holder.getItems();
   related.each(function(item,index){ item.ready(); item.missing(); index.ready(); });
   related.each(function(changed){ changed=unknownValue; changed.ready(); });
   var unknown=unknownList();
   unknown.each(function(untyped){untyped.ready();});
  } }`,
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Use.cfc"), map[string]string{
		"row.ready": "", "row.missing": "method 'missing' not found in Item", "entry.ready": "", "entry.missing": "method 'missing' not found in Item", "f.ready": "", "item.ready": "", "item.missing": "method 'missing' not found in Item", "changed.ready": "variable 'changed' has no component ref",
		"single.ready": "variable 'single' has no component ref", "unsure.ready": "variable 'unsure' has no component ref", "index.ready": "variable 'index' has no component ref", "untyped.ready": "variable 'untyped' has no component ref",
	})
}

func TestEntityLoadCollectionEvidenceRejectsUnknownsAndShadows(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/Item.cfc":  `component persistent="true" entityname="LoadedItem" { function ready(){} }`,
		"models/Plain.cfc": `component { function ready(){} }`,
		"Unknown.cfc": `component { function run(){
   for(var dynamic in entityLoad(name)){dynamic.ready();}
   for(var missing in entityLoad("NotRegistered")){missing.ready();}
   for(var plain in entityLoad("Plain")){plain.ready();}
   for(var unique in entityLoad("LoadedItem", {}, true)){unique.ready();}
   for(var known in entityLoad("LoadedItem", {}, false)){known.ready();}
  } }`,
		"Shadow.cfc": `component { function entityLoad(name){return [];} function run(){for(var shadow in entityLoad("LoadedItem")){shadow.ready();}} }`,
		"Use.cfm": `<cfset rows=entityLoad("LoadedItem")>
<cfloop array="#rows#" index="tagRow"><cfoutput>#tagRow.ready()#</cfoutput></cfloop>`,
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Unknown.cfc"), map[string]string{
		"dynamic.ready": "variable 'dynamic' has no component ref", "missing.ready": "variable 'missing' has no component ref", "plain.ready": "variable 'plain' has no component ref", "unique.ready": "variable 'unique' has no component ref", "known.ready": "",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Shadow.cfc"), map[string]string{"shadow.ready": "variable 'shadow' has no component ref"})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Use.cfm"), map[string]string{"tagRow.ready": ""})
}

// TestALoopOverADynamicReceiversResultIsDynamic: a call on a model of a module
// the workspace lacks is accepted as dynamic, and so must be a call on an item
// of the collection it returns. ContentBox's 1.0.4 patch loops over
// `pageService.getAll()` from `pageService@cb`, a namespace the current modules
// no longer declare, and `thisPage.getContentID()` was reported as untyped.
func TestALoopOverADynamicReceiversResultIsDynamic(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Update.cfc": `component {
	property name="pageService" inject="pageService@cb";
	function run() {
		var allPages = pageService.getAll( sortOrder = "parent" );
		for ( var thisPage in allPages ) {
			thisPage.getContentID();
		}
	}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Update.cfc"), map[string]string{
		"pageService.getAll":    "",
		"thisPage.getContentID": "",
	})
}
