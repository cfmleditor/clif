package parser

import (
	"testing"
	"unsafe"

	"go.lsp.dev/uri"
)

func TestFunctionDefNameRanges(t *testing.T) {
	cases := []struct {
		name           string
		file           string
		src            string
		wantFunc       string
		wantLine       uint32
		wantNameLine   uint32
		wantNameCol    uint32
		wantNameEndCol uint32
	}{
		{
			name:           "single-line script method",
			file:           "file:///Script.cfc",
			src:            "component {\n\tpublic string function render() {}\n}\n",
			wantFunc:       "render",
			wantLine:       1,
			wantNameLine:   1,
			wantNameCol:    24,
			wantNameEndCol: 30,
		},
		{
			name:           "multiline script header",
			file:           "file:///MultiScript.cfc",
			src:            "component {\n\tpublic\n\tstring\n\tfunction\n\trender() {\n\t\treturn \"ok\";\n\t}\n}\n",
			wantFunc:       "render",
			wantLine:       1,
			wantNameLine:   4,
			wantNameCol:    1,
			wantNameEndCol: 7,
		},
		{
			name:           "multiline function return type",
			file:           "file:///ReturnFunc.cfc",
			src:            "component {\n\tpublic function\n\tfunction\n\tstartWorker() {}\n}\n",
			wantFunc:       "startWorker",
			wantLine:       1,
			wantNameLine:   3,
			wantNameCol:    1,
			wantNameEndCol: 12,
		},
		{
			name:           "single-line tag function",
			file:           "file:///Tag.cfc",
			src:            "<cfcomponent>\n\t<cffunction name=\"render\" access=\"public\"></cffunction>\n</cfcomponent>\n",
			wantFunc:       "render",
			wantLine:       1,
			wantNameLine:   1,
			wantNameCol:    19,
			wantNameEndCol: 25,
		},
		{
			name:           "multiline tag header",
			file:           "file:///MultiTag.cfc",
			src:            "<cfcomponent>\n\t<cffunction\n\t\taccess=\"public\"\n\t\tname=\"render\"\n\t\treturntype=\"string\">\n\t</cffunction>\n</cfcomponent>\n",
			wantFunc:       "render",
			wantLine:       1,
			wantNameLine:   3,
			wantNameCol:    8,
			wantNameEndCol: 14,
		},
		{
			name:           "single-quoted tag function name",
			file:           "file:///SingleQuoteTag.cfc",
			src:            "<cfcomponent><cffunction name='render'></cffunction></cfcomponent>\n",
			wantFunc:       "render",
			wantLine:       0,
			wantNameLine:   0,
			wantNameCol:    31,
			wantNameEndCol: 37,
		},
		{
			name:           "unquoted tag function name",
			file:           "file:///UnquotedTag.cfc",
			src:            "<cfcomponent><cffunction name=render></cffunction></cfcomponent>\n",
			wantFunc:       "render",
			wantLine:       0,
			wantNameLine:   0,
			wantNameCol:    30,
			wantNameEndCol: 36,
		},
		{
			name:           "scoped function value",
			file:           "file:///Value.cfc",
			src:            "component {\n\tthis.helper = function() {};\n}\n",
			wantFunc:       "helper",
			wantLine:       1,
			wantNameLine:   1,
			wantNameCol:    6,
			wantNameEndCol: 12,
		},
		{
			name:           "multiline scoped function value",
			file:           "file:///MultiValue.cfc",
			src:            "component {\n\tthis.\n\thelper = function() {};\n}\n",
			wantFunc:       "helper",
			wantLine:       1,
			wantNameLine:   2,
			wantNameCol:    1,
			wantNameEndCol: 7,
		},
		{
			name:           "arrow function value",
			file:           "file:///Arrow.cfc",
			src:            "component {\n\tvariables.arrow = (x) => x;\n}\n",
			wantFunc:       "arrow",
			wantLine:       1,
			wantNameLine:   1,
			wantNameCol:    11,
			wantNameEndCol: 16,
		},
		{
			name:           "unscoped function value",
			file:           "file:///TopValue.cfc",
			src:            "component {\n\ttopHelper = function() {};\n}\n",
			wantFunc:       "topHelper",
			wantLine:       1,
			wantNameLine:   1,
			wantNameCol:    1,
			wantNameEndCol: 10,
		},
		{
			name:           "named nested script function with multiline header",
			file:           "file:///Nested.cfc",
			src:            "component {\n\tfunction outer() {\n\t\tpublic string\n\t\tfunction inner() {}\n\t}\n}\n",
			wantFunc:       "inner",
			wantLine:       2,
			wantNameLine:   3,
			wantNameCol:    11,
			wantNameEndCol: 16,
		},
		{
			name:           "inline cfscript mid-line",
			file:           "file:///InlineScript.cfm",
			src:            "<cfif true><cfscript>function inlineScript() {}</cfscript></cfif>\n",
			wantFunc:       "inlineScript",
			wantLine:       0,
			wantNameLine:   0,
			wantNameCol:    30,
			wantNameEndCol: 42,
		},
		{
			name:           "tag function after closing cfscript on same line",
			file:           "file:///AfterScript.cfm",
			src:            "<cfscript>x = 1;</cfscript><cffunction name=\"afterScript\"></cffunction>\n",
			wantFunc:       "afterScript",
			wantLine:       0,
			wantNameLine:   0,
			wantNameCol:    45,
			wantNameEndCol: 56,
		},
		{
			name:           "UTF-16 prefix in script",
			file:           "file:///UTF16Script.cfc",
			src:            "component {\n\t/* 😀 */ public function greet() {}\n}\n",
			wantFunc:       "greet",
			wantLine:       1,
			wantNameLine:   1,
			wantNameCol:    26,
			wantNameEndCol: 31,
		},
		{
			name:           "UTF-16 prefix in tag",
			file:           "file:///UTF16Tag.cfm",
			src:            "<!-- 😀 --><cffunction name=\"greet\"></cffunction>\n",
			wantFunc:       "greet",
			wantLine:       0,
			wantNameLine:   0,
			wantNameCol:    29,
			wantNameEndCol: 34,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pr := Parse(uri.URI(tc.file), tc.src)

			var got *FunctionDef

			for i := range pr.Funcs {
				if pr.Funcs[i].Name == tc.wantFunc {
					got = &pr.Funcs[i]

					break
				}
			}

			if got == nil {
				t.Fatalf("function %q not found in %+v", tc.wantFunc, pr.Funcs)
			}

			if !got.HasNameRange() {
				t.Fatalf("HasNameRange() = false for %+v", *got)
			}

			if got.Line != tc.wantLine || got.NameLine != tc.wantNameLine || got.NameCol != tc.wantNameCol || got.NameEndCol != tc.wantNameEndCol {
				t.Errorf("got Line=%d NameRange=(%d:%d..%d), want Line=%d NameRange=(%d:%d..%d)",
					got.Line, got.NameLine, got.NameCol, got.NameEndCol,
					tc.wantLine, tc.wantNameLine, tc.wantNameCol, tc.wantNameEndCol)
			}
		})
	}
}

func TestApplyEditShiftsFunctionNameLine(t *testing.T) {
	src := "component {\n\tfunction first() {\n\t\tvar a = 1;\n\t}\n\tpublic\n\tstring\n\tfunction\n\tsecond() {\n\t\treturn \"ok\";\n\t}\n}\n"
	pr := Parse(uri.URI("file:///Shift.cfc"), src)

	kind := pr.ApplyEdit(2, 12, 2, 12, "\n\t\tvar b = 2;\n\t\tvar c = 3;")
	if kind != EditInFunc {
		t.Fatalf("edit kind = %v, want EditInFunc", kind)
	}

	var second *FunctionDef

	for i := range pr.Funcs {
		if pr.Funcs[i].Name == "second" {
			second = &pr.Funcs[i]

			break
		}
	}

	if second == nil {
		t.Fatalf("second not found after edit")
	}

	if second.Line != 6 || second.NameLine != 9 || second.NameCol != 1 || second.NameEndCol != 7 {
		t.Errorf("after edit: Line=%d NameRange=(%d:%d..%d), want Line=6 NameRange=(9:1..7)",
			second.Line, second.NameLine, second.NameCol, second.NameEndCol)
	}
}

func TestFunctionDefSizeDoesNotGrow(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("sizes are stated for 64-bit words")
	}

	if got := unsafe.Sizeof(FunctionDef{}); got > 208 {
		t.Errorf("FunctionDef is %d bytes, over 208: keep small fields grouped", got)
	}
}
