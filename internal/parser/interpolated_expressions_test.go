package parser

import (
	"reflect"
	"testing"
)

func TestInterpolatedExpressionsShareParserBoundaries(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   []string
	}{
		{`"plain"`, nil},
		{`"##mutate(x)##"`, nil},
		{`"#mutate(x)# #x.go()#"`, []string{`mutate(x)`, `x.go()`}},
		{`"#wrap('#mutate(x)#')#"`, []string{`wrap('#mutate(x)#')`}},
	} {
		if got := InterpolatedExpressions(tc.source); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s: %v want %v", tc.source, got, tc.want)
		}
	}
}
