package parser

import (
	"strings"
	"testing"
)

func TestWholeStructValueContracts(t *testing.T) {
	for _, tc := range []struct {
		name, writes, returns string
		valid                 bool
	}{
		{"known values", `built[key]=new models.Site();`, `return variables.items;`, true},
		{"producer values", `built[key]=variables.dao.read();`, `return variables.items;`, true},
		{"alias", `var alias=built;alias[key]=new models.Site();`, `return variables.items;`, true},
		{"unknown value", `built[key]=arguments.value;`, `return variables.items;`, false},
		{"conflicting returns", `built[key]=new models.Site();`, `return variables.items;return false;`, false},
		{"unknown mutator", `built[key]=new models.Site();built.append(arguments.value);`, `return variables.items;`, false},
		{"escaped argument", `built[key]=new models.Site();mutate(built);`, `return variables.items;`, false},
		{"escaped named argument", `built[key]=new models.Site();mutate(value=built);`, `return variables.items;`, false},
		{"escaped alias", `var alias=built;built[key]=new models.Site();mutate(alias);`, `return variables.items;`, false},
		{"nested escape", `built[key]=new models.Site();mutate({items:built});`, `return variables.items;`, false},
		{"interpolated escape", `built[key]=new models.Site();var message="#mutate(built)#";`, `return variables.items;`, false},
		{"public exposure", `built[key]=new models.Site();this.exposed=built;`, `return variables.items;`, false},
		{"array contents", `built=[];built[key]=new models.Site();`, `return variables.items;`, false},
		{"closure", `built[key]=new models.Site();var callback=function(){mutate(built);};`, `return variables.items;`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pr := Parse(testURI, collectionFixture("script", tc.writes, tc.returns))

			f := collectionFunction(t, pr, "one")
			if f.ReturnComponent != "" || len(f.ReturnSources) != 0 {
				t.Fatal("container acquired value's component")
			}

			if (f.CollectionValues != nil) != tc.valid {
				t.Fatalf("collection contract=%+v valid=%v", f.CollectionValues, tc.valid)
			}
		})
	}
}

func TestWholeStructValuesInvalidateAfterWriterEdit(t *testing.T) {
	content := collectionFixture("script", `built[key]=new models.Site();`, `return variables.items;`)

	pr := Parse(testURI, content)
	if collectionFunction(t, pr, "all").CollectionValues == nil {
		t.Fatal("initial contract missing")
	}

	for line, text := range strings.Split(content, "\n") {
		if column := strings.Index(text, "new models.Site()"); column >= 0 {
			pr.ApplyEdit(line, column, line, column+len("new models.Site()"), "arguments.unknown")

			if collectionFunction(t, pr, "all").CollectionValues != nil {
				t.Fatal("stale value contract survived a writer edit")
			}

			return
		}
	}

	t.Fatal("write missing")
}
