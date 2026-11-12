package parser

import (
	"strings"
	"testing"
)

func TestApplicationSectionsMaskDeclarationHeaders(t *testing.T) {
	source := `component {
 variables.public="kept";
 public any function init(){return this;}
 private types.Result function hidden(){application.unused=new Service();}
 public static function helper(){return "value";}
}`

	constructor, methods := ApplicationSections(source)
	if len(constructor) != len(source) || !strings.Contains(constructor, `variables.public="kept";`) {
		t.Fatal("constructor changed shape or lost executable field", constructor)
	}

	for _, header := range []string{"public any", "private types.Result", "public static", "application.unused"} {
		if strings.Contains(constructor, header) {
			t.Fatal("declaration leaked into constructor", header, constructor)
		}
	}

	for name, header := range map[string]string{"init": "public any", "hidden": "private types.Result", "helper": "public static"} {
		projection := methods[name]
		if len(projection) != len(source) || !strings.Contains(projection, header) {
			t.Fatal("lost header or origin", name, projection)
		}

		for i := range source {
			if source[i] == '\n' && (constructor[i] != '\n' || projection[i] != '\n') {
				t.Fatal("newline moved", i)
			}
		}
	}
}

// Lookups asking for one text's sections share one scan of it.
func TestApplicationScanIsSharedPerText(t *testing.T) {
	source := "component { function onRequest() { include 'x.cfm'; } function onApplicationStart() {} }"

	_, first := applicationScan(source)
	_, second := applicationScan(string([]byte(source)))

	if len(first) == 0 || &first[0] != &second[0] {
		t.Fatal("a second lookup of the same text scanned it again")
	}
}
