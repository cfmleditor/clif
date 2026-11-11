package resolve

import (
	"slices"
	"testing"
)

// Each reader that finds a framework's calls in a file reads the file's
// code: a call commented out, or written inside a string, is not one, and a
// real call beside it still is. Each of these readers was a regular
// expression over the file's text, which read all three.
func TestTheCallReadersReadOnlyCode(t *testing.T) {
	for _, c := range []struct {
		name string
		read func(string) []string
		text string
		want []string
	}{
		{
			"setView, view() and renderView()", readViewRenders,
			"component {\n" +
				"\t// event.setView( \"old/index\" );\n" +
				"\t/* event.setView( view = \"older/index\" ); */\n" +
				"\tfunction index( event, rc, prc ) {\n" +
				"\t\tvar s = \"renderView( 'quoted/view' )\";\n" +
				"\t\tevent.setView( \"comments/index\" );\n" +
				"\t\treturn \"<p>#renderView( 'partials/pager' )#</p>\";\n" +
				"\t}\n}\n",
			[]string{"comments/index", "partials/pager"},
		},
		{
			"a view's renders, through its args", readRenderSites,
			"<cfoutput>\n<!--- #renderView( view = \"old/sidebar\", args = { a = 1 } )# --->\n#renderView( view = \"main/sidebar\", args = { a = 1 } )#\n</cfoutput>\n",
			[]string{"main/sidebar@2"},
		},
		{
			"announce()", readInterceptSites,
			"component {\n\tfunction f() {\n\t\t// announce( \"onOld\", {} );\n\t\tvar s = 'announce( \"onQuoted\" )';\n\t\tannounce( \"onNew\", { a = 1 } );\n\t}\n}\n",
			[]string{"4"},
		},
		{
			"dspObject_include()", readMuraObjectSites,
			"<!--- #dspObject_include( theFile = \"old.cfm\" )# --->\n<cfoutput>#dspObject_include( theFile = \"new.cfm\", item = x )#</cfoutput>\n<cfscript>\n// dspObject_include( theFile = \"old.cfm\" );\ndspObject_include( theFile = \"new.cfm\", item = x );\n</cfscript>\n",
			[]string{"1", "4", "blanked:1"},
		},
		{
			"a write of prc.response, commented out", readPrcResponse,
			"component {\n\t// prc.response = new Response();\n\tvar s = \"prc.response = 1\";\n}\n",
			nil,
		},
		{
			"a write of prc.response", readPrcResponse,
			"component {\n\t// a comment\n\tprc[ \"response\" ] = new Response();\n}\n",
			[]string{"writes"},
		},
		{
			"customErrorTemplate", readCustomErrorTemplates,
			"component {\n\tfunction configure() {\n\t\tcoldbox = {\n\t\t\t// customErrorTemplate : \"/old/error.cfm\",\n\t\t\tcustomErrorTemplate : \"/views/error.cfm\"\n\t\t};\n\t}\n}\n",
			[]string{"/views/error.cfm"},
		},
		{
			"FW/1 setView()", readFW1SetViews,
			"component {\n\tfunction edit( rc ) {\n\t\t// getFW().setView( \"admin:old.detail\" );\n\t\tgetFW().setView( action = \"admin:product.detail\" );\n\t}\n}\n",
			[]string{"admin:product.detail"},
		},
		{
			"FW/1 view()", readFW1ViewCalls,
			"<!--- #view( \"product/old\" )# --->\n<cfoutput>#view( \"product/tabs\" )#</cfoutput>\n",
			[]string{":product/tabs"},
		},
		{
			"an action handing its rc on", readFW1RcCalls,
			"\t\t// old( rc );\n\t\tvar s = \"quoted( rc )\";\n\t\tdetail( rc );\n\t\tthis.list( rc = arguments.rc );\n",
			[]string{"detail", "list"},
		},
		{
			"Wheels associations", readAssociations,
			"component {\n\tfunction config() {\n\t\t// hasMany( \"oldComments\" );\n\t\t/* belongsTo( \"oldAuthor\" ); */\n\t\thasMany( \"comments\" );\n\t\tbelongsTo( name = \"author\", modelName = \"User\", foreignKey = foo( \"x\" ) );\n\t}\n}\n",
			[]string{"hasmany:comments:", "belongsto:author:User"},
		},
		{
			"WireBox map().to()", readBinderMaps,
			"component {\n\tfunction configure() {\n\t\t// map( \"Old\" ).to( \"models.Old\" );\n\t\tmap( \"Svc\" ).to( \"models.Service\" );\n\t}\n}\n",
			[]string{"svc=models.Service"},
		},
		{
			"super.init() with an entity", readSuperInit,
			"component {\n\tfunction init() {\n\t\t// super.init( entityName = \"oldEntity\" );\n\t\tsuper.init( entityName = \"cbAuthor\" );\n\t}\n}\n",
			[]string{"cbAuthor"},
		},
		{
			"super.init() passing an argument on", readSuperInit,
			"component {\n\tfunction init( entityName = \"cbContent\" ) {\n\t\t/* super.init( entityName = \"cbOld\" ); */\n\t\tsuper.init( entityName = arguments.entityName );\n\t}\n}\n",
			[]string{"arguments.entityName"},
		},
		{
			"an application named in a comment", readNamesApplication,
			"component {\n\t// this.name = \"old\";\n\tthis.sessionManagement = true;\n}\n",
			nil,
		},
		{
			"an application named", readNamesApplication,
			"<!--- <cfapplication name=\"old\"> --->\n<cfapplication name=\"blog\" sessionmanagement=\"true\">\n",
			[]string{"named"},
		},
		{
			"this.memento in a comment", readMemento,
			"component {\n\t/* this.memento = { defaultIncludes = [] }; */\n}\n",
			nil,
		},
		{
			"a module's applicationHelper", readModuleHelpers,
			"component {\n\t// this.applicationHelper = [ \"helpers/Old.cfm\" ];\n\tthis.applicationHelper = [ \"helpers/Mixins.cfm\" ];\n}\n",
			[]string{"helpers/Mixins.cfm"},
		},
		{
			"an application's applicationHelper", readAppHelpers,
			"component {\n\tfunction configure() {\n\t\tcoldbox = {\n\t\t\t// applicationHelper : \"includes/Old.cfm\",\n\t\t\tapplicationHelper : \"includes/helpers/ApplicationHelper.cfm\"\n\t\t};\n\t}\n}\n",
			[]string{"includes/helpers/ApplicationHelper.cfm"},
		},
		{
			"includeUDF()", readIncludedUDFs,
			"component {\n\tfunction onLoad() {\n\t\t// renderer.includeUDF( \"#moduleMapping#/helpers/old.cfm\" );\n\t\trenderer.includeUDF( \"#moduleMapping#/helpers/new.cfm\" );\n\t}\n}\n",
			[]string{"helpers/new.cfm"},
		},
		{
			"computed includes", readComputedIncludes,
			"<!--- <cfinclude template=\"#oldPath#\"> --->\n<cfinclude template=\"#templatePath#\">\n<cfscript>\n// include \"#oldPath#\";\ninclude \"#getPath( 'x' )#\";\n</cfscript>\n",
			[]string{"1", "4"},
		},
		{
			"ComponentListPackageAsStruct()", readPackageListings,
			"<!--- <cfset ComponentListPackageAsStruct(\"lucee.admin.old\", oldNames)> --->\n<cfset ComponentListPackageAsStruct(\"lucee.admin.debug\", driverNames)>\n",
			[]string{"lucee.admin.debug>driverNames"},
		},
		{
			"a registry's store", readRegistryContainer,
			"\n<!--- <cfset oldDrivers[name] = createObject(\"component\", name)> --->\n<cfset drivers[name] = createObject(\"component\", name)>\n",
			[]string{"drivers"},
		},
		{
			"a registry's store through a variable", readRegistryContainer,
			"\n<!--- <cfset obj = createObject(\"component\", name)> --->\n<cfset obj = createObject(\"component\", other)>\n<cfset drivers[name] = obj>\n",
			nil,
		},
		{
			"a registry's guards", readRegistrySkips,
			"\n<!--- <cfif name EQ \"OldBase\"><cfcontinue></cfif> --->\n<cfif name EQ \"Base\"><cfcontinue></cfif>\n",
			[]string{"Base"},
		},
		{
			"a factory's walk", readFactoryBodyFills,
			"{\n\tvar sct = structNew();\n\t// loop array=arguments.other item=\"el\" {}\n\tloop array=arguments.arr item=\"local.el\" {\n\t\tsct[el.getClass()] = el;\n\t}\n\treturn sct;\n}",
			[]string{"fills"},
		},
		{
			"a factory whose store is commented out", readFactoryBodyFills,
			"{\n\tvar sct = structNew();\n\tloop array=arguments.arr item=\"local.el\" {\n\t\t// sct[el.getClass()] = el;\n\t}\n\treturn sct;\n}",
			nil,
		},
	} {
		if got := c.read(c.text); !slices.Equal(got, c.want) {
			t.Errorf("%s: read %q, want %q", c.name, got, c.want)
		}
	}
}

// codeLists reads the code inside a string's expressions, and places it in
// the text: a call there has the offset and line it is written at.
func TestCodeListsReadTheExpressionsInsideStrings(t *testing.T) {
	text := "x = 1;\ny = \"a\n#renderView( 'b' )#\";\n"

	calls := textCalls(text, "renderView")
	if len(calls) != 1 {
		t.Fatalf("found %d calls, want 1", len(calls))
	}

	name := calls[0].name()
	if text[name.Offset:name.Offset+len(name.Value)] != "renderView" || name.Line != 2 {
		t.Errorf("the call is at offset %d, line %d: %q", name.Offset, name.Line, text[name.Offset:])
	}
}

// textCalls answers from the text's call index what codeCalls answers by
// walking its code, for any names, in the same order.
func TestTextCallsAreCodeCalls(t *testing.T) {
	texts := []string{
		`setView("a"); x = view( "b" ); renderView(view="c"); // view("d")` + "\n" + `y = "#view('e')# #renderView('f')#";`,
		"<cfscript>\nmap(\"a\").to(\"b\"); hasMany(\"x\"); belongsTo(name=\"y\");\n</cfscript><cfoutput>#view('z')#</cfoutput>",
		`super.init( entityName = "e" ); init(); x.init("y");`,
	}

	nameSets := [][]string{{"view"}, {"setView", "view", "renderView"}, {"map"}, {"hasMany", "belongsTo", "hasOne"}, {"init"}, {"nothing"}}

	for _, text := range texts {
		for _, names := range nameSets {
			want, got := codeCalls(textCodeOf(text).lists, names...), textCalls(text, names...)
			if len(want) != len(got) {
				t.Fatalf("%q %v: %d calls, codeCalls finds %d", text, names, len(got), len(want))
			}

			for i := range want {
				if want[i].name().Offset != got[i].name().Offset || want[i].closing != got[i].closing {
					t.Fatalf("%q %v: call %d at %d, codeCalls has %d", text, names, i, got[i].name().Offset, want[i].name().Offset)
				}
			}
		}
	}
}
