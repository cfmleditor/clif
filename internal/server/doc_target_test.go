package server

import (
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The expressions doc_target.go replaced, kept as the oracle each reader is
// held to.
var (
	oracleDocFunction  = regexp.MustCompile(`(?i)^(\s*)(?:\b(?:private|package|public|remote|static|final|abstract|default)\s+)*(?:\b(?:[A-Za-z0-9_.$]+)\s+)?function\s+([_$a-zA-Z][$\w]*)\s*(?:\((?:=\s*\{|[^{])*)[{;]`)
	oracleDocProperty  = regexp.MustCompile(`(?i)^(\s*property)\s+`)
	oracleDocComponent = regexp.MustCompile(`(?i)^(\s*(component|interface))\b[^{]*\{`)
	oracleDocCFTag     = regexp.MustCompile(`(?i)^\s*<cf(function|property|component|interface)\b`)
	oracleDocTagWord   = regexp.MustCompile(`@([\w$]*)(?:\.([\w$]*))?$`)
)

var docTargetCases = []string{
	"function f() {}", "public function f( a, b ) {", "public string function f( required string a = \"x\" ) {",
	"private static final function f();", "remote any function getX() output=false {", "package function f(){",
	"public function f( a = {} ) {", "function f( a = { b: 1 }, c ) {", "function f( a = {\n} ) {\n}", "function f( a={} , b = {x:1}) ;",
	"function f( a = { ) ;", "function f( a = ", "function f( a ={", "function f( a == { b ) {", "function f( a\t=\n{}\n) {",
	"function f( a ) ; x; y", "function f( a ) x y z", "function f(", "function f", "function f ( ", "function", "function ", "function  f ( ) {",
	"function function f() {", "function function(){", "function function (){", "public public function f(){", "public function function f(){",
	"string function f(){", "pkg.models.User function f(){", "$x function f(){", ".x function f(){", "a.b.c function f(){", "9 function f(){",
	"public string static function f(){", "string public function f(){", "a b function f(){", "public a b function f(){",
	"public a function f(){", "Public Static FUNCTION F(){", "PUBLIC STRING FUNCTION F(){", "publicfunction f(){", "public\tfunction\nf(){",
	"public\vfunction f(){", "\n\t  function f(){", "xfunction f(){", "function f(){", "function 1f(){", "function _f(){", "function $f(){",
	"function f$1_(){", "function f-g(){", "function f.g(){", "function f (){", "function f\n(){", "function f x(){", "function\tf\t(\t)\t{",
	"property name=\"x\";", "property\tname", "property", "property ", "Property\nx", "  property x", "propertyx", "properties x",
	"component {", "component extends=\"a\" {", "interface {", "component", "component ", "component\n{", "component x", "components {",
	"Component{", "  COMPONENT hint=\"x\"\n{", "interface extends=\"I\"\n{", "componentx {", "component {", "component x\n\n y {",
	"<cffunction name=\"f\">", "<CFFUNCTION", "<cfproperty name=\"p\">", "  <cfcomponent>", "<cfinterface>", "<cffunctions>", "<cf function>",
	"< cffunction>", "<cfcomponentx>", "<cffunction\tname=x>", "<cfFunction", "<cfset x=1>", "<cffunction", "<cf",
	"@", "@hint", "@arg.", "@arg.hint", "x @a.b", "@a @b", "@a@b", "@a.b.c", "@a b", "@a.b c", "@$a.$b", "@é", "@a.é", "@@", "@.", "@a.", "text",
	"", " ", "{", ";", "(", "=", "\xff", "é function f(){",
}

var docTargetFragments = []string{
	"function", "FUNCTION", "public", "private", "static", "final", "package", "remote", "abstract", "default", "string", "any", "a.b.C",
	"$", ".", "_", "x", "f", "1", " ", "  ", "\t", "\n", "\r", "\f", "\v", "(", ")", "{", "}", ";", "=", "= {", "={", ", ", ",",
	"\"", "'", "component", "interface", "property", "<cf", "<cffunction", "<cfproperty", "@", "@a", ".k", "é", "\xff", "output=false",
}

func randomDocText(rng *rand.Rand) string {
	if rng.IntN(2) == 0 {
		s := docTargetCases[rng.IntN(len(docTargetCases))]

		for range rng.IntN(4) {
			i := rng.IntN(len(s) + 1)
			j := min(len(s), i+rng.IntN(3))

			switch rng.IntN(3) {
			case 0:
				s = s[:i] + docTargetFragments[rng.IntN(len(docTargetFragments))] + s[i:]
			case 1:
				s = s[:i] + s[j:]
			default:
				s = s[:i] + docTargetFragments[rng.IntN(len(docTargetFragments))] + s[j:]
			}
		}

		return s
	}

	var b strings.Builder

	for range 1 + rng.IntN(14) {
		b.WriteString(docTargetFragments[rng.IntN(len(docTargetFragments))])
	}

	return b.String()
}

func docDisagreement(s string) []string {
	var bad []string

	check := func(name, got, want string) {
		if got != want {
			bad = append(bad, fmt.Sprintf("%s(%q) = %s, want %s", name, s, got, want))
		}
	}

	end, ok := docFunctionEnd(s)
	loc := oracleDocFunction.FindStringIndex(s)
	wantEnd := -1

	if loc != nil {
		wantEnd = loc[1]
	}

	gotEnd := -1
	if ok {
		gotEnd = end
	}

	check("docFunctionEnd", strconv.Itoa(gotEnd), strconv.Itoa(wantEnd))
	check("docPropertyStart", strconv.FormatBool(docPropertyStart(s)), strconv.FormatBool(oracleDocProperty.MatchString(s)))

	word, ok := docComponentStart(s)
	m := oracleDocComponent.FindStringSubmatch(s)
	wantWord := ""

	if m != nil {
		wantWord = m[2]
	}

	check("docComponentStart", fmt.Sprint(word, ok), fmt.Sprint(wantWord, m != nil))

	name, ok := docCFTagStart(s)
	m = oracleDocCFTag.FindStringSubmatch(s)
	wantName := ""

	if m != nil {
		wantName = m[1]
	}

	check("docCFTagStart", fmt.Sprint(name, ok), fmt.Sprint(wantName, m != nil))
	check("docTagWord", fmt.Sprint(docTagWord(s)), fmt.Sprint(oracleDocTagWord.FindStringSubmatchIndex(s)))

	return bad
}

// unfoldedDocRune is a character a regular expression's (?i) folds to ASCII and
// the readers, folding in ASCII alone, do not: the Kelvin sign and the long s.
func unfoldedDocRune(s string) bool { return strings.ContainsAny(s, "Kſ") }

// Every reader answers what its expression answered, the end of a function's
// header and the groups included, on the forms the extension's patterns read,
// their near misses, and texts built from fragments of them.
func TestDocTargetReadersMatchTheirExpressions(t *testing.T) {
	n := 300_000
	if testing.Short() {
		n = 30_000
	}

	texts := append([]string(nil), docTargetCases...)

	rng := rand.New(rand.NewPCG(1, 7))
	for range n {
		texts = append(texts, randomDocText(rng))
	}

	failures, functions := 0, 0

	for _, s := range texts {
		if unfoldedDocRune(s) {
			continue
		}

		for _, msg := range docDisagreement(s) {
			t.Error(msg)

			failures++

			if failures == 10 {
				return
			}
		}

		if oracleDocFunction.MatchString(s) {
			functions++
		}
	}

	if functions < len(texts)/50 {
		t.Errorf("%d of %d texts are a function header; the generated texts do not reach the reader", functions, len(texts))
	}
}

// FuzzDocTargetReaders holds each reader to its expression on any text.
func FuzzDocTargetReaders(f *testing.F) {
	for _, s := range docTargetCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if unfoldedDocRune(s) {
			t.Skip()
		}

		for _, msg := range docDisagreement(s) {
			t.Error(msg)
		}
	})
}

// TestDocTargetReadersAgreeOnACorpus is the same comparison on what follows
// every `*/` of every CFML file under CLIF_DOCBLOCK_CORPUS (directories, as PATH
// is), which is the text a doc block's target is read from. Skipped without it,
// so it never runs in CI.
func TestDocTargetReadersAgreeOnACorpus(t *testing.T) {
	corpus := os.Getenv("CLIF_DOCBLOCK_CORPUS")
	if corpus == "" {
		t.Skip("CLIF_DOCBLOCK_CORPUS not set")
	}

	files, texts, functions := 0, 0, 0

	for _, dir := range filepath.SplitList(corpus) {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}

			switch strings.ToLower(filepath.Ext(path)) {
			case ".cfc", ".cfm", ".cfml":
			default:
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil //nolint:nilerr // a file that cannot be read is not compared
			}

			content := string(data)
			files++

			for off := 0; ; {
				i := strings.Index(content[off:], "*/")
				if i < 0 {
					break
				}

				off += i + 2
				after := strings.TrimLeft(content[off:], " \t")

				texts++

				for _, msg := range docDisagreement(after) {
					t.Errorf("%s: %s", path, msg[:min(len(msg), 200)])
				}

				if oracleDocFunction.MatchString(after) {
					functions++
				}
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Logf("%d files, %d texts compared, %d of them a function header", files, texts, functions)
}
