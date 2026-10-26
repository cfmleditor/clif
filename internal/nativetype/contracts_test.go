package nativetype

import "testing"

func TestNativeContractsPreserveDifferentResultTypes(t *testing.T) {
	for _, tc := range []struct{ family, method, result string }{
		{"iterator", "next", "$any"},
		{"resultset", "next", Type("boolean")},
		{"map", "entrySet", Type("set.entry")},
		{"iterator.entry", "next", Type("entry")},
		{"hibernatesession", "getStatistics", Type("sessionstatistics")},
		{"sessionfactory", "getStatistics", Type("statistics")},
		{"object", "toString", Type("string")},
		{"duration", "dividedBy", "$any"},
		{"array", "addAll", Type("boolean")},
		{"array", "add", "$any"},
		{"array", "get", "$any"},
		{"array", "remove", "$any"},
		{"array", "size", Type("numeric")},
		{"array", "iterator", Type("iterator")},
		{"array", "containsAll", Type("boolean")},
		{"array", "removeAll", Type("boolean")},
		{"array", "retainAll", Type("boolean")},
		{"array", "toJson", Type("string")},
		{"query", "valueList", Type("string")},
		{"query", "toJson", Type("string")},
		{"string", "hash", Type("string")},
		{"string", "listGetDuplicates", Type("string")},
		{"string", "listMap", Type("string")},
		{"string", "listFilter", Type("string")},
		{"string", "map", Type("string")},
		{"string", "filter", Type("string")},
		{"string", "listSome", Type("boolean")},
		{"string", "listEvery", Type("boolean")},
		{"string", "some", Type("boolean")},
		{"string", "every", Type("boolean")},
		{"string", "each", Type("void")},
		{"string", "listEach", Type("void")},
		{"struct", "toSorted", Type("struct")},
		{"struct", "toJson", Type("string")},
		{"query", "columnData", Type("array")},
		{"query", "some", Type("boolean")},
		{"query", "every", Type("boolean")},
		{"query", "reduce", "$any"},
		{"string", "reduce", "$any"},
		{"string", "deserializeJSON", "$any"},
		{"string", "listReduce", "$any"},
		{"struct", "containsValue", Type("boolean")},
		{"struct", "keySet", Type("set")},
		{"mongodbcollection", "stats", Type("struct")},
		{"mongodbcollection", "storageSize", Type("numeric")},
		{"mongodbcollection", "totalIndexSize", Type("numeric")},
		{"mongodbcollection", "getIndexInfo", Type("array")},
		{"mongodbcollection", "rename", Type("void")},
		{"mongodbcollection", "createSearchIndex", Type("string")},
	} {
		got, ok := Method(Type(tc.family), tc.method)
		if !ok || got != tc.result {
			t.Errorf("%s.%s = %q,%v; want %q", tc.family, tc.method, got, ok, tc.result)
		}
	}

	for _, tc := range []struct{ family, method string }{
		{"mongodbcollection", "mapReduce"},
		{"object", "clone"},
		{"map", "clone"},
		{"object", "finalize"},
		{"instant", "getDayOfWeek"},
		{"localtime", "plusWeeks"},
		{"sessionstatistics", "setStatisticsEnabled"},
	} {
		if _, ok := Method(Type(tc.family), tc.method); ok {
			t.Errorf("invalid public contract %s.%s", tc.family, tc.method)
		}
	}
}

func TestNativeGuessRequiresOneCompleteFamily(t *testing.T) {
	if got := Guess([]string{"entrySet", "containsKey"}); got != Type("map") {
		t.Fatal(got)
	}

	for _, observed := range [][]string{nil, {"toString"}, {"next"}, {"entrySet", "absent"}} {
		if got := Guess(observed); got != "" {
			t.Fatalf("%v unexpectedly selected %s", observed, got)
		}
	}

	if got := EngineProducer("getPageContext", "generic"); got != "" {
		t.Fatal(got)
	}

	if got := EngineProducer("getPageContext", "lucee"); got != Type("lucee.pagecontext") {
		t.Fatal(got)
	}
}

func TestNativeGuessFallsBackToPrimitiveFamilies(t *testing.T) {
	for _, tc := range []struct {
		name    string
		methods []string
		want    string
	}{
		{"string", []string{"strip", "listToArray"}, Type("string")},
		{"array", []string{"resize", "indexExists"}, Type("array")},
		{"struct", []string{"isOrdered", "keyArray"}, Type("struct")},
		{"builder", []string{"ensureCapacity", "setLength"}, Type("stringbuilder")},
		{"existing family priority", []string{"indexOf", "length"}, Type("list")},
		{"query-only API", []string{"columnData", "reduce"}, Type("query")},
		{"ambiguous query reducer", []string{"reduce"}, ""},
		{"ambiguous query predicate", []string{"some"}, ""},
		{"ambiguous query every", []string{"every"}, ""},
		{"ambiguous query transform", []string{"filter", "map"}, ""},
		{"ambiguous primitive get", []string{"get"}, ""},
		{"ambiguous primitive", []string{"reverse"}, ""},
		{"mixed incompatible methods", []string{"strip", "resize"}, ""},
		{"unsupported method", []string{"ensureCapacity", "absent"}, ""},
		{"generic object", []string{"toString", "getClass"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Guess(tc.methods); got != tc.want {
				t.Fatalf("Guess(%v) = %q; want %q", tc.methods, got, tc.want)
			}
		})
	}
}

// TestHibernateContractsCoverWhatORMCodeCalls: cborm and the applications
// built on it call Session.beginTransaction()/createCriteria(), Lucee's
// getActualSession(), and ClassMetadata.getIdentifierType()/getPropertyType()/
// hasIdentifierProperty()/getTableName(). Missing from the contracts, each was
// reported as a method the object lacks, where v0.5.2 accepted them.
func TestHibernateContractsCoverWhatORMCodeCalls(t *testing.T) {
	for _, tc := range []struct{ typ, method, want string }{
		{"hibernatesession", "beginTransaction", "$any"},
		{"hibernatesession", "createCriteria", "$any"},
		{"hibernatesession", "getActualSession", "$any"},
		{"classmetadata", "getIdentifierType", "$any"},
		{"classmetadata", "getPropertyType", "$any"},
		{"classmetadata", "hasIdentifierProperty", Type("boolean")},
		{"classmetadata", "getTableName", Type("string")},
	} {
		if got, ok := Method(Type(tc.typ), tc.method); !ok || got != tc.want {
			t.Errorf("%s.%s = %q, %v; want %q", tc.typ, tc.method, got, ok, tc.want)
		}
	}
}

// TestNumericContractsCoverMemberFunctionSyntax: Lucee (and CF2016+) support
// member-function syntax on primitive numeric receivers — the lucee corpus
// test functions call a.sgn(), y.abs(), a.aCos(), and similar directly on
// numbers — natives were declared without members, and each call was
// reported as "not found on builtin native.numeric".
func TestNumericContractsCoverMemberFunctionSyntax(t *testing.T) {
	numeric := []string{"abs", "aCos", "aSin", "atn", "bitAnd", "bitMaskClear", "bitMaskRead", "bitMaskSet", "bitNot", "bitOr", "bitSHLN", "bitSHRN", "bitXor", "ceiling", "cos", "fix", "floor", "int", "len", "log", "log10", "max", "min", "round", "sgn", "sin", "sqr", "tan"}
	for _, name := range numeric {
		if got, ok := Method(Type("numeric"), name); !ok || got != Type("numeric") {
			t.Errorf("numeric.%s = %q, %v; want %q", name, got, ok, Type("numeric"))
		}
	}

	for _, name := range []string{"booleanFormat", "formatBaseN"} {
		if got, ok := Method(Type("numeric"), name); !ok || got != Type("string") {
			t.Errorf("numeric.%s = %q, %v; want %q", name, got, ok, Type("string"))
		}
	}

	for _, name := range []string{"nope", "reReplace", "append"} {
		if _, ok := Method(Type("numeric"), name); ok {
			t.Errorf("numeric.%s unexpectedly supported", name)
		}
	}
}
