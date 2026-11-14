package config

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestTagParser_JavaStubResolver_ResolvesCreateObjectJava is the parser-level
// integration test for config.JavaStubResolver: confirms the synthesized
// resolver actually resolves through the parser's own regex/simple-match
// heuristics and $1 substitution, not just via a raw regexp.MatchString check.
func TestTagParser_JavaStubResolver_ResolvesCreateObjectJava(t *testing.T) {
	cr := JavaStubResolver("tassweb.packages.tass.javastubs")
	resolvers := []parser.Resolver{cr.Parser()}

	content := `<cfcomponent>
<cffunction name="work">
	<cfset variables.jss = createObject('java', 'java.security.Signature') />
</cffunction>
</cfcomponent>`

	pr := parser.ParseWithOptions("file:///test.cfc", content, &parser.ParseOptions{Resolvers: resolvers})

	found := ""

	for _, ref := range pr.ComponentRefs {
		if ref.Variable == "jss" {
			found = ref.Component
		}
	}

	want := "tassweb.packages.tass.javastubs.java.security.Signature"
	if found != want {
		t.Errorf("expected jss -> %s, got %q", want, found)
	}
}

// TestParserCarriesEveryResolverField: Parser is the one conversion from a
// configured resolver to the parser's, so a field it forgets is a setting that
// parses and does nothing. Every field is set to a non-zero value and must
// arrive under the same name.
func TestParserCarriesEveryResolverField(t *testing.T) {
	var in Resolver

	v := reflect.ValueOf(&in).Elem()
	for i := range v.NumField() {
		switch f := v.Field(i); f.Kind() {
		case reflect.String:
			f.SetString("x" + v.Type().Field(i).Name)
		case reflect.Bool:
			f.SetBool(true)
		case reflect.Pointer:
			// Call: the call a rule recognises, which becomes its Matcher.
			f.Set(reflect.ValueOf(&CallMatch{Function: "getBean", NamedArgument: true}))
		case reflect.Interface:
			// A matcher: any one, as long as it arrives as the same one.
			f.Set(reflect.ValueOf(parser.NameSet("x" + v.Type().Field(i).Name)))
		default:
			t.Fatalf("field %s: add its kind to this test", v.Type().Field(i).Name)
		}
	}

	out := in.Parser()
	ov := reflect.ValueOf(&out).Elem()

	for i := range v.NumField() {
		name := v.Type().Field(i).Name
		if name == "Call" {
			continue // converted to the Matcher, checked in TestACallMatchBecomesAMatcher
		}

		got := ov.FieldByName(name)
		if !got.IsValid() {
			t.Errorf("parser.Resolver has no field %s", name)

			continue
		}

		if got.Interface() != v.Field(i).Interface() {
			t.Errorf("%s: Parser() gives %v, want %v", name, got.Interface(), v.Field(i).Interface())
		}
	}
}

// TestACallMatchBecomesAMatcher: a rule written as a call is read by a matcher
// that needs the call to end the expression, and takes the function's name as
// its prefix when it names none.
func TestACallMatchBecomesAMatcher(t *testing.T) {
	rule := Resolver{Call: &CallMatch{Function: "getBean", NamedArgument: true, MoreArguments: true}, Resolve: "$1"}
	got := rule.Parser()

	if got.Prefix != "getBean" {
		t.Errorf("prefix %q, want the function's name", got.Prefix)
	}

	if got.Matcher == nil {
		t.Fatal("no matcher")
	}

	for expr, want := range map[string]string{
		`getBean("userService")`:        "userService",
		`getBean( name = "a.b", true )`: "a.b",
		`x.getBean('c')`:                "c",
		`getBean("a").run()`:            "",
		`getBean(computed)`:             "",
		`getBean("a" & b)`:              "",
		`getUserBean("a")`:              "",
	} {
		captures, ok := got.Matcher.Match(expr)
		if ok != (want != "") || ok && captures[0] != want {
			t.Errorf("%s: %q %v, want %q", expr, captures, ok, want)
		}
	}

	if (&Resolver{Call: &CallMatch{Function: "getBean"}, Prefix: "x", Resolve: "$1"}).Parser().Prefix != "x" {
		t.Error("a prefix the rule names was replaced")
	}
}

// TestACallRuleSurvivesAConfigFile: a rule written as `call` in a configuration
// file is kept, as a rule with a pattern is, and one that names no function, or
// no `resolve`, is dropped as an empty one always was.
func TestACallRuleSurvivesAConfigFile(t *testing.T) {
	var cfg JSON

	text := `{"componentResolvers": [
		{"call": {"function": "getBean", "namedArgument": true, "moreArguments": true}, "resolve": "$1"},
		{"call": {"function": ""}, "resolve": "$1"},
		{"call": {"function": "getBean"}},
		{"match": "getService(\"$1\")", "resolve": "s.$1", "prefix": "getService"}
	]}`
	if err := json.Unmarshal([]byte(text), &cfg); err != nil {
		t.Fatal(err)
	}

	rules := ParserResolvers(Resolve(&cfg, "/proj").ComponentResolvers)
	if len(rules) != 2 {
		t.Fatalf("want the call rule and the pattern rule kept, got %d: %+v", len(rules), rules)
	}

	if rules[0].Matcher == nil || rules[0].Prefix != "getBean" || rules[1].Matcher != nil || rules[1].Match == "" {
		t.Errorf("rules read wrong: %+v", rules)
	}
}
