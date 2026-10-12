package parser

import "testing"

func TestNativeFactoryContract(t *testing.T) {
	for _, src := range []string{`createObject("java","java.util.HashMap")`, `createObject("java","java.sql.DriverManager")`} {
		if got := dynamicCall(src); got == "$any" || got == "" {
			t.Fatalf("%s -> %q", src, got)
		}
	}
}
