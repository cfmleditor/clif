package parser

import (
	"slices"
	"testing"

	"go.lsp.dev/uri"
)

func TestCFParserParityFixes(t *testing.T) {
	t.Parallel()

	t.Run("cfset lambda arrow continues past -> without breaking decrement", func(t *testing.T) {
		t.Parallel()

		src := "<cfoutput>prefix</cfoutput>\n" +
			"<cfset closure = t => t.b()>\n" +
			"<cfset lambda = t -> t.b()>\n" +
			"<cfset nested = getURLs().map( target -> target.toString() )>\n" +
			"<cfset countdown = i-->\n" +
			"<cfset comparison = total gt 1>\n"

		pr := ParseWithOptions(uri.File("/test/lambda.cfm"), src, &ParseOptions{
			ExtractCalls: true,
			KeepVars:     true,
		})

		var names []string
		for _, c := range pr.Calls {
			names = append(names, c.FuncName)
		}

		for _, want := range []string{"b", "getURLs", "map", "toString"} {
			if !slices.Contains(names, want) {
				t.Errorf("calls = %v, missing %q", names, want)
			}
		}

		var vars []string
		for _, v := range pr.AllVars() {
			vars = append(vars, v.Name)
		}

		for _, want := range []string{"closure", "lambda", "nested", "countdown", "comparison"} {
			if !slices.Contains(vars, want) {
				t.Errorf("vars = %v, missing %q", vars, want)
			}
		}
	})

	t.Run("static and final variable modifiers at top level and in function body", func(t *testing.T) {
		t.Parallel()

		src := "component {\n" +
			"\tstatic myStatic = \"v\";\n" +
			"\tfinal myConst = \"c\";\n" +
			"\tstatic numeric function helper() { return 1; }\n" +
			"\tfunction f() {\n" +
			"\t\tfinal immutable = 2;\n" +
			"\t\tstatic shared = 3;\n" +
			"\t\tfinal var finalLocal = 4;\n" +
			"\t\tstatic var staticLocal = 5;\n" +
			"\t\tvar final = getValue();\n" +
			"\t\treturn myStatic;\n" +
			"\t}\n" +
			"}\n"

		pr := ParseWithOptions(uri.File("/test/mods.cfc"), src, &ParseOptions{
			KeepVars: true,
		})

		if len(pr.Funcs) != 2 {
			t.Fatalf("Funcs = %v, want 2 (helper, f)", pr.Funcs)
		}

		var vars []string
		for _, v := range pr.AllVars() {
			vars = append(vars, v.Name)
		}

		for _, want := range []string{"myStatic", "myConst", "immutable", "shared", "finalLocal", "staticLocal", "final"} {
			if !slices.Contains(vars, want) {
				t.Errorf("vars = %v, missing %q", vars, want)
			}
		}
	})

	t.Run("namespaced function metadata attributes do not truncate function body", func(t *testing.T) {
		t.Parallel()

		src := "component {\n" +
			"\tfunction setStatus( rc ) my:role=\"support:admin\" {\n" +
			"\t\tvar ok = true;\n" +
			"\t\treturn ok;\n" +
			"\t}\n" +
			"\tfunction setStatus2( rc ) role:support {\n" +
			"\t\tvar ok2 = true;\n" +
			"\t\treturn ok2;\n" +
			"\t}\n" +
			"}\n"

		pr := ParseWithOptions(uri.File("/test/meta.cfc"), src, &ParseOptions{
			KeepVars: true,
		})

		if len(pr.Funcs) != 2 {
			t.Fatalf("Funcs = %v, want 2", pr.Funcs)
		}

		for _, s := range pr.Scopes {
			if s.End <= s.Start {
				t.Errorf("scope %s has empty body range [%d,%d]", s.Name, s.Start, s.End)
			}
		}

		var vars []string
		for _, v := range pr.AllVars() {
			vars = append(vars, v.Name)
		}

		for _, want := range []string{"ok", "ok2"} {
			if !slices.Contains(vars, want) {
				t.Errorf("vars = %v, missing %q", vars, want)
			}
		}
	})

	t.Run("dotted property types and scoped property names", func(t *testing.T) {
		t.Parallel()

		src := "component {\n" +
			"\tproperty models.User user;\n" +
			"\tproperty struct variables.displays;\n" +
			"}\n"

		pr := ParseWithOptions(uri.File("/test/props.cfc"), src, &ParseOptions{})
		if len(pr.Properties) != 2 {
			t.Fatalf("Properties = %d, want 2", len(pr.Properties))
		}

		if pr.Properties[0].typeName != "models.User" || pr.Properties[0].name != "user" {
			t.Errorf("prop[0] = (%q, %q), want (models.User, user)", pr.Properties[0].typeName, pr.Properties[0].name)
		}

		if pr.Properties[1].typeName != "struct" || pr.Properties[1].name != "variables.displays" {
			t.Errorf("prop[1] = (%q, %q), want (struct, variables.displays)", pr.Properties[1].typeName, pr.Properties[1].name)
		}
	})
}
