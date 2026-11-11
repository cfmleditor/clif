package resolve

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The regular expressions expression_shapes.go replaced, kept as the oracle
// each reader is held to.
var (
	oracleStructField       = regexp.MustCompile(`^(?:local\.)?(\w+)\.(\w+)$`)
	oracleKeyedField        = regexp.MustCompile(`(?i)^(?:local\s*\.\s*)?(\w+)\s*\[\s*variables\s*\.\s*(\w+)\s*\]$`)
	oracleKeyedReceiver     = regexp.MustCompile(`^([A-Za-z_]\w*)\[[^[\]]*\]$`)
	oracleNullCast          = regexp.MustCompile(`(?i)^(?:javacast\s*\(\s*["']null["']|nullvalue\s*\(\s*\)$|null$)`)
	oracleConcatPickup      = regexp.MustCompile(`(?i)^variables\.?\[\s*([A-Za-z_]\w*)\s*&\s*["']([^"']*)["']\s*\]$`)
	oracleNullFallback      = regexp.MustCompile(`(?is)^(.+?)\s*\?:\s*(?:nullValue\s*\(\s*\)|javaCast\s*\(\s*["']null["']\s*,\s*["']{2}\s*\))$`)
	oracleInitArgAssign     = regexp.MustCompile(`(?i)^\s*(?:variables\.)?(\w+)\s*=\s*arguments\.(\w+)\s*;?\s*$`)
	oracleInitArgSet        = regexp.MustCompile(`(?i)^\s*(?:this\.|variables\.)?set(\w+)\(\s*arguments\.(\w+)\s*\)\s*;?\s*$`)
	oracleJavaObject        = regexp.MustCompile(`(?is)^(?:createObject\s*\(\s*["']java["']|new\s+java:)`)
	oracleComputedComponent = regexp.MustCompile(`(?is)^createObject\s*\(\s*["']component["']\s*,\s*["'][^"']*#[^"']*\.(\w+)["']\s*\)$`)
	oracleMuraEventSource   = regexp.MustCompile(`(?i)(?:^|\.)(?:getEvent|event)\(\s*\)$|mura\.(?:servlet)?event["']`)
	oracleReturnFormat      = regexp.MustCompile(`(?i)\breturnFormat\s*=`)
	oracleSelfFormat        = regexp.MustCompile(`(?i)\breturnFormat\s*=\s*["']self["']`)
	oraclePositional        = regexp.MustCompile(`^\s*[^\s=]+\s*(?:,|$)`)
	oracleLoopPath          = regexp.MustCompile(`^(?:(local|variables|this|arguments)\.)?(\w+)$`)
	oracleIndexedRead       = regexp.MustCompile(`^((?:local|variables|this|arguments)\.[\w.]+|[A-Za-z_]\w*)\s*\[[^\]]*\]$`)
	oracleExactKeyedRead    = regexp.MustCompile(`^(?:[A-Za-z_]\w*|(?:local|variables|this|arguments)\.[\w.]+)\s*\[[^\]]*\]$`)
	oracleChainKeyedRead    = regexp.MustCompile(`^([A-Za-z_]\w*(?:\.\w+)*?(?:\.\w+\s*\([^()]*\))+)\s*\[[^\]]*\]$`)
	oracleIdent             = regexp.MustCompile(`^[A-Za-z_]\w*$`)
	oracleDottedName        = regexp.MustCompile(`^[A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)+$`)
	oracleListingRead       = regexp.MustCompile(`^((?:variables\.)?[A-Za-z_]\w*)\s*\[`)
	oracleDuplicate         = regexp.MustCompile(`(?i)^duplicate\s*\(\s*([A-Za-z_][\w.]*)\s*(?:,\s*(?:true|false)\s*)?\)$`)
	oracleDuplicatePlain    = regexp.MustCompile(`(?i)^duplicate\s*\(\s*(?:var\s+)?(?:variables\.)?([A-Za-z_]\w*)\s*\)`)
	oracleCreatedComponent  = regexp.MustCompile(`(?i)^createObject\s*\(\s*["']component["']\s*,\s*["']([^"']+)["']`)
	oracleNewComponent      = regexp.MustCompile(`(?i)^new\s+(?:cfml:)?([A-Za-z_$#][\w$.#]*)\s*\(`)
	oraclePlaceholder       = regexp.MustCompile(`^(?:""|'')$`)
	oracleAsQueryFalse      = regexp.MustCompile(`\basquery\s*[=:]\s*(?:false|no|0)\b`)
)

// noiseFragments go into every shape's random texts: whitespace RE2's \s
// takes and \v, which it does not, quotes, brackets, and bytes outside ASCII.
var noiseFragments = []string{
	" ", "  ", "\t", "\n", "\r", "\f", "\v", `"`, `'`, "[", "]", "(", ")", ".", ",", "=", ":", ";",
	"#", "&", "?", "$", "_", "x", "Z", "1", "é", "\xff", "-",
}

// Seeds and fragment sets for the shapes that read alike.
var (
	keyedSeeds = []string{
		"db[ k ]", "local.x", "arguments.y[1]", "this.a.b [ 'k' ]", "rc.order.getOrderDeliveries()[ i ]",
		"a.b().c()[1]", "a.b.c", "$x.y", "x", "variables.listing[ k ]", "a.b(x[1])[2]",
	}
	createSeeds = []string{
		`createObject( "java", "java.util.HashMap" )`, "new java:java.lang.String()",
		`createObject("component", "#root#.model.Thing")`, `createObject('component','a.b').init()`,
		"new cfml:a.b()", "new #x#.y ( 1 )",
	}
	formatSeeds    = []string{`id=1, returnFormat = "self"`, `returnFormat="query"`, "a,b", " a b", "=1", "x, y=2"}
	duplicateSeeds = []string{
		"duplicate( x.y, true )", "duplicate(var variables.x) >", "duplicate( var )", "DUPLICATE(variables.a)",
	}
	keyedFragments = []string{
		"local", "variables", "this", "arguments", "Local", "VARIABLES", "local.", "variables.",
		"a", "b1", "_k", "a.b", "x.y.z", "[", "]", "[i]", "[ 'k' ]", "(", ")", "()", ".get()",
		".items( 1 )", "get", ".", "$", "$x", " ",
	}
	createFragments = []string{
		"createObject", "CREATEOBJECT", "createobject", "new", "NEW", " ", "(", ")", "( ", `"`, `'`,
		"java", "JAVA", "java:", "component", "Component", ",", ", ", "#", ".", "a", "a.b", "#x#",
		"#x#.y", "cfml:", "$", "_", "\n", "Svc",
	}
	formatFragments = []string{
		"returnFormat", "returnformat", "RETURNFORMAT", "xreturnFormat", "_", "x", "=", " ", "\t",
		`"`, `'`, "self", "SELF", ",", "argumentCollection", "1", "\v", "a=1", "id",
	}
	duplicateFragments = []string{
		"duplicate", "DUPLICATE", "Duplicate", "(", ")", " ", "\t", "var", "var ", "variables.",
		"VARIABLES.", "variables", "x", "a.b", "_y", ",", "true", "FALSE", "false", "1", ".",
	}
)

// expressionShape is one reader held to its expression: texts it matches,
// the fragments its random texts are built from, and how the two answers
// compare.
type expressionShape struct {
	seeds     []string
	fragments []string
	compare   func(s string) (got, want string)
}

// submatches compares a reader that stands for FindStringSubmatch.
func submatches(re *regexp.Regexp, read func(string) []string) func(string) (string, string) {
	return func(s string) (string, string) {
		return fmt.Sprintf("%q", read(s)), fmt.Sprintf("%q", re.FindStringSubmatch(s))
	}
}

// matches compares a reader that stands for MatchString.
func matches(re *regexp.Regexp, read func(string) bool) func(string) (string, string) {
	return func(s string) (string, string) {
		return strconv.FormatBool(read(s)), strconv.FormatBool(re.MatchString(s))
	}
}

var expressionShapes = map[string]expressionShape{
	"structField": {
		[]string{"local.results.comments", "local.b", "a.b"},
		[]string{"local.", "Local.", "local", ".", "a", "b1", "_", "results", "comments"},
		submatches(oracleStructField, structField),
	},
	"keyedField": {
		[]string{"local . results [ VARIABLES . key ]", "results[variables.k]", "LOCAL.x[variables.y]"},
		[]string{"local", "LOCAL", " ", "\t", ".", "x", "results", "[", "variables", "VARIABLES", "]", "entityPlural"},
		submatches(oracleKeyedField, keyedField),
	},
	"keyedReceiver":  {keyedSeeds, keyedFragments, submatches(oracleKeyedReceiver, keyedReceiver)},
	"loopPath":       {keyedSeeds, keyedFragments, submatches(oracleLoopPath, loopPath)},
	"indexedRead":    {keyedSeeds, keyedFragments, submatches(oracleIndexedRead, indexedRead)},
	"exactKeyedRead": {keyedSeeds, keyedFragments, matches(oracleExactKeyedRead, isExactKeyedRead)},
	"chainKeyedRead": {keyedSeeds, keyedFragments, submatches(oracleChainKeyedRead, chainKeyedRead)},
	"ident":          {keyedSeeds, keyedFragments, matches(oracleIdent, isIdent)},
	"dottedName":     {keyedSeeds, keyedFragments, matches(oracleDottedName, isDottedName)},
	"listingRead":    {keyedSeeds, keyedFragments, submatches(oracleListingRead, listingRead)},
	"nullCast": {
		[]string{"null", `javacast( "null", "" )`, "nullValue( )"},
		[]string{"javacast", "JavaCast", "nullvalue", "nullValue", "null", "NULL", "(", ")", " ", `"`, `'`, ",", "x"},
		matches(oracleNullCast, isNullCast),
	},
	"concatPickup": {
		[]string{`variables[ name & "Service" ]`, `variables.[n&'']`},
		[]string{"variables", "Variables", ".", "[", "]", "name", "_n1", "&", `"`, `'`, "lit", "Service", " ", "1"},
		submatches(oracleConcatPickup, concatPickup),
	},
	"nullFallback": {
		[]string{`drivers[k] ?: nullValue()`, " ?: nullValue()", `a ?: b ?: javaCast( 'null', "" )`},
		[]string{
			"a", "drivers[k]", " ", "?", ":", "?:", " ?: ", "nullValue", "NULLVALUE", "javaCast", "(", ")",
			`"`, `'`, `""`, "null", ",", "\n", "é", "\xe2",
		},
		submatches(oracleNullFallback, nullFallback),
	},
	"initArgAssign": {
		[]string{"variables.dao = arguments.dao;", " x=arguments.y "},
		[]string{" ", "\t", "variables.", "VARIABLES.", "variables", "x", "dao", "=", "arguments.", "ARGUMENTS.", ";", "."},
		submatches(oracleInitArgAssign, initArgAssign),
	},
	"initArgSet": {
		[]string{"  setDao( arguments.dao ) ;  ", "this.setX(arguments.y)", "variables.setA(arguments.b);"},
		[]string{" ", "\t", "this.", "variables.", "set", "Set", "x", "Dao", "(", ")", "arguments.", "Arguments.", ";", "."},
		submatches(oracleInitArgSet, initArgSet),
	},
	"javaObject":        {createSeeds, createFragments, matches(oracleJavaObject, isJavaObject)},
	"computedComponent": {createSeeds, createFragments, submatches(oracleComputedComponent, computedComponent)},
	"createdComponent":  {createSeeds, createFragments, submatches(oracleCreatedComponent, createdComponent)},
	"newComponent":      {createSeeds, createFragments, submatches(oracleNewComponent, newComponent)},
	"muraEventSource": {
		[]string{"renderer.getEvent()", "event( )", `$.mura.event"`, `mura.servletEvent'`},
		[]string{"getEvent", "GETEVENT", "event", "Event", "(", ")", "()", " ", ".", "mura.", "MURA.", "servlet", `"`, `'`, "$", "x", "renderer"},
		matches(oracleMuraEventSource, isMuraEventSource),
	},
	"returnFormat":   {formatSeeds, formatFragments, matches(oracleReturnFormat, namesReturnFormat)},
	"selfFormat":     {formatSeeds, formatFragments, matches(oracleSelfFormat, namesSelfFormat)},
	"positional":     {formatSeeds, formatFragments, matches(oraclePositional, positionalFirst)},
	"duplicate":      {duplicateSeeds, duplicateFragments, submatches(oracleDuplicate, duplicateCall)},
	"duplicatePlain": {duplicateSeeds, duplicateFragments, submatches(oracleDuplicatePlain, duplicatePlainCall)},
	"placeholder":    {[]string{`""`, `''`}, []string{`"`, `'`, " "}, matches(oraclePlaceholder, isEmptyLiteral)},
	"asQueryFalse": {
		[]string{"asquery = false, x", "asquery: no", "id=1,asquery=0"},
		[]string{"asquery", "asQuery", "xasquery", "_", " ", "=", ":", "false", "no", "0", "01", "falsey", ",", "1", "properties"},
		matches(oracleAsQueryFalse, asQueryFalseIn),
	},
}

// expressionShapeCases are hand-written texts every shape is checked on,
// beside the random ones.
var expressionShapeCases = []string{
	"", "local.results.comments", "local.b", "results[ variables.entityPlural ]",
	"local . results [ VARIABLES . key ]", "db[ k ]", "db[[]", "null", "NULL", "nullValue( )",
	`javacast( "null", "" )`, `javaCast("null",'')`, `variables[ name & "Service" ]`,
	`variables.[n&'']`, `drivers[k] ?: nullValue()`, "?: nullValue()", " ?:nullvalue()", `a ?: b ?: javaCast( 'null', "" )`,
	"variables.dao = arguments.dao;", "  setDao( arguments.dao ) ;  ", "this.setX(arguments.y)",
	`createObject( "java", "java.util.HashMap" )`, "new java:java.lang.String()",
	`createObject("component", "#root#.model.Thing")`, `createObject('component','a.b').init()`,
	"new cfml:a.b()", "new #x#.y ( 1 )", "renderer.getEvent()", "event( )", `$.mura.event"`,
	`mura.servletEvent'`, `id=1, returnFormat = "self"`, `returnFormat="query"`, "a,b", " a b", "=1",
	"local.x", "arguments.y[1]", "this.a.b [ 'k' ]", "rc.order.getOrderDeliveries()[ i ]",
	"a.b().c()[1]", "a.b.c", "$x.y", "x", "variables.listing[ k ]", "duplicate( x.y, true )",
	"duplicate(var variables.x) >", "duplicate( var )", `""`, `''`, "asquery = false, x",
	"asQuery: no", "asquery=0", `[ "a", 'b', "#c#" ]`,
}

// unfoldedRune reports whether s holds a character a regular expression's
// (?i) folds to ASCII and the readers, folding in ASCII alone, do not: the
// Kelvin sign and the long s.
func unfoldedRune(s string) bool { return strings.ContainsAny(s, "Kſ") }

// randomExpression is either a seed with up to three edits, each a
// fragment inserted, a few bytes cut, or a few bytes replaced by a fragment,
// or a text of up to twelve fragments; the fragments are mostly the shape's
// own and some from noiseFragments.
func randomExpression(rng *rand.Rand, shape *expressionShape) string {
	fragment := func() string {
		if rng.IntN(5) == 0 {
			return noiseFragments[rng.IntN(len(noiseFragments))]
		}

		return shape.fragments[rng.IntN(len(shape.fragments))]
	}

	if rng.IntN(2) == 0 {
		s := shape.seeds[rng.IntN(len(shape.seeds))]

		for range rng.IntN(4) {
			i := rng.IntN(len(s) + 1)
			j := min(len(s), i+rng.IntN(3))

			switch rng.IntN(3) {
			case 0:
				s = s[:i] + fragment() + s[i:]
			case 1:
				s = s[:i] + s[j:]
			default:
				s = s[:i] + fragment() + s[j:]
			}
		}

		return s
	}

	var b strings.Builder

	for range 1 + rng.IntN(12) {
		b.WriteString(fragment())
	}

	return b.String()
}

// Every reader answers what its regular expression answered, captures
// included, on the hand-written cases and on 200,000 texts built from its
// own seeds and fragments (20,000 under -short; FuzzExpressionShapes goes
// further), and at least one text in fifty is a match, so the comparison
// is not of two refusals.
func TestExpressionShapesMatchTheirExpressions(t *testing.T) {
	n := 200_000
	if testing.Short() {
		n = 20_000
	}

	names := make([]string, 0, len(expressionShapes))
	for name := range expressionShapes {
		names = append(names, name)
	}

	slices.Sort(names)

	for i, name := range names {
		shape := expressionShapes[name]
		rng := rand.New(rand.NewPCG(uint64(i), 7))
		texts := slices.Clone(expressionShapeCases)

		for range n {
			texts = append(texts, randomExpression(rng, &shape))
		}

		failures, matched := 0, 0

		_, none := shape.compare("")

		for _, s := range texts {
			got, want := shape.compare(s)
			if got != want {
				t.Errorf("%s(%q) = %s, want %s", name, s, got, want)

				failures++
				if failures == 5 {
					break
				}
			}

			if want != none {
				matched++
			}
		}

		if matched < len(texts)/50 {
			t.Errorf("%s: %d of %d texts matched; the generated texts do not reach the shape", name, matched, len(texts))
		}
	}
}

// FuzzExpressionShapes holds each reader to its regular expression on any
// text.
func FuzzExpressionShapes(f *testing.F) {
	for _, s := range expressionShapeCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if unfoldedRune(s) {
			t.Skip()
		}

		for name, shape := range expressionShapes {
			if got, want := shape.compare(s); got != want {
				t.Errorf("%s(%q) = %s, want %s", name, s, got, want)
			}
		}
	})
}
