package nativetype

import "testing"

func TestRuntimeFollowupContracts(t *testing.T) {
	for _, tc := range []struct{ family, method, result string }{
		{"struct", "some", "boolean"},
		{"struct", "every", "boolean"},
		{"string", "repeatString", "string"},
		{"string", "numberFormat", "string"},
		{"numeric", "numberFormat", "string"},
		{"date", "setDay", "date"},
		{"date", "setHour", "date"},
		{"date", "setMilliSecond", "date"},
		{"date", "setMinute", "date"},
		{"date", "setMonth", "date"},
		{"date", "setSecond", "date"},
		{"date", "setYear", "date"},
		{"array", "removeDuplicates", "array"},
		{"array", "getMetadata", "struct"},
		{"lucee.pagecontext", "getCurrentPageSource", "lucee.pagesource"},
		{"lucee.pagesource", "getMapping", "lucee.mapping"},
		{"lucee.mapping", "getVirtual", "string"},
		{"lucee.mapping", "getPageSource", "lucee.pagesource"},
		{"pool", "getFactory", "lucee.datasourcefactory"},
		{"lucee.datasourcefactory", "getDatasource", "lucee.datasource"},
		{"lucee.datasource", "getName", "string"},
		{"thread", "getThreadGroup", "threadgroup"},
		{"threadgroup", "getParent", "threadgroup"},
		{"threadgroup", "getName", "string"},
	} {
		got, ok := Method(Type(tc.family), tc.method)
		if !ok || got != Type(tc.result) {
			t.Errorf("%s.%s = %q,%v; want %q", tc.family, tc.method, got, ok, Type(tc.result))
		}
	}

	for _, tc := range []struct{ family, method string }{
		{"object", "numberFormat"},
		{"struct", "repeatString"},
		{"string", "removeDuplicates"},
		{"numeric", "setYear"},
		{"array", "getMapping"},
		{"object", "getCurrentPageSource"},
		{"sessionfactory", "getDialect"},
		{"datasource", "getName"},
	} {
		if _, ok := Method(Type(tc.family), tc.method); ok {
			t.Errorf("wrong owner accepted: %s.%s", tc.family, tc.method)
		}
	}

	if got := Guess([]string{"some", "every", "toJson"}); got != "" {
		t.Fatalf("overlapping struct/array APIs selected %s", got)
	}

	for _, observed := range [][]string{{"getMetadata"}, {"getMetadata", "toString", "getClass"}, {"numberFormat"}, {"numberFormat", "toString"}} {
		if got := Guess(observed); got != "" {
			t.Fatalf("ambiguous runtime receiver %v selected %s", observed, got)
		}
	}
}
