package parser

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The regular expressions text_shapes.go replaced, kept as the oracle each
// replacement is held to.
var (
	oracleBeanPath       = regexp.MustCompile(`(?i)this\.beanPaths\[\s*["']([^"']*)["']\s*\]\s*=\s*(?:expandPath\(\s*["']([^"']+)["']\s*\)|["']([^"']+)["'])`)
	oracleDILocations    = regexp.MustCompile(`(?i)(?:variables\.)?framework\.diLocations\s*=\s*["']([^"']+)["']`)
	oracleFW1App         = regexp.MustCompile(`(?i)extends\s*=\s*["']framework\.one["']`)
	oracleMappingTarget  = regexp.MustCompile(`(?i)^this\.mappings\[\s*["']([^"']+)["']\s*\]$`)
	oracleServiceProp    = regexp.MustCompile(`@serviceproperty\s+([A-Za-z_][A-Za-z0-9_.\[\]]*)\s+([A-Za-z]+)\|(\S+)`)
	oracleMockClassName  = regexp.MustCompile(`(?i)\bclassName\s*[=:]\s*["']([^"'#]+)["']`)
	oracleMarkup         = regexp.MustCompile(`(?i)</[a-z][a-z0-9-]*\s*>|<!doctype|<!--[^-]`)
	oraclePlaceholder    = regexp.MustCompile(`\$(\d+)`)
	oracleServiceLoop    = regexp.MustCompile(`(?is)^for\s*\(\s*((?:variables\.|local\.)?[\w$]+)\s+in\s+listToArray\s*\(\s*([^()]+)\s*\)\s*\)\s*\{`)
	oracleServiceWrite   = regexp.MustCompile(`(?i)^(application|request|session|server)\s*\[\s*["']#\s*([\w.$]+)\s*#["']\s*\]\s*=\s*([\w.$]+)\.getBean\s*\(\s*["']#\s*([\w.$]+)\s*#["']\s*\)\s*;`)
	oracleServiceAppend  = regexp.MustCompile(`(?is)^listAppend\s*\(\s*((?:variables\.|local\.)?[\w$]+)\s*,\s*(["'][^"'#]*["'])\s*\)$`)
	shapeOracleFragments = []string{
		"this.beanPaths[", "THIS.BEANPATHS[", "expandPath(", "variables.", "local.", "framework.diLocations",
		"extends", "framework.one", "this.mappings[", "@serviceproperty", "className", "classname",
		"</", "<!doctype", "<!--", "<!---", "for", "in", "listToArray", "listAppend", "application",
		"request", "session", "server", ".getBean", ".getbean", "#", "$1", "$", "12",
		`"`, `'`, "[", "]", "(", ")", "{", "}", "=", ":", ",", ";", "|", ".", "-", "/", "<", ">",
		" ", "  ", "\t", "\n", "\r", "\f", "a", "Z", "x1", "_", "div", "é", "\x00", "k",
	}
)

// shapeCases are the hand-written texts every shape is checked on, beside the
// random ones.
var shapeCases = []string{
	`this.beanPaths["svc"] = expandPath("./model/services"); this.beanPaths['x']='y'`,
	`this.beanpaths[ "" ] = "z" this.beanPaths["a"] = expandPath( 'b' ) `,
	`variables.framework.diLocations = "model, controllers"`,
	`framework.diLocations="" framework.diLocations = 'a'`,
	`component extends = 'framework.one' {}`,
	`component extends="Framework.One"`,
	`this.mappings[ "/root" ]`, `this.mappings["/root"] x`, `THIS.MAPPINGS['a']`,
	`<!--- @serviceproperty donorObj package|sandbox.tasscom.donor --->`,
	`@serviceproperty result.data[i] service|gljournal @serviceproperty _x k|y`,
	`createMock(className="a.b.C")`, `createMock( classname : 'x' )`, `myclassName="no"`, `className="#dyn#"`,
	`</div >`, `<!DOCTYPE html>`, `<!-- c -->`, `<!--- cf --->`, `</1>`,
	`$1.get$2()`, `get$1()`, `$$12x$`, `no placeholder`,
	`for ( name in listToArray( "a,b" ) ) {`, `for(variables.svc in listToArray(   )){`,
	`for ( local.x in listToArray( svcList ) ) { application[ "#x#" ] = beanFactory.getBean( "#x#" ); }`,
	`application[ "#svc#" ] = variables.factory.getBean( "#svc#" );`,
	`SERVER['#a.b#']=f.getbean.getBean('#a.b#') ;`,
	`application["#x#"] = .getBean("#x#");`,
	// A match inside another's text is no match: the search goes on from
	// where the first ended.
	`this.beanPaths['']='this.beanPaths['a']='b'`,
	`listAppend( variables.list, "item" )`, `listAppend(list,'')`, `listAppend(list, "#x#")`,
}

// randomShapeText is a text built from the shapes' own fragments, so the
// random cases reach deep into each shape.
func randomShapeText(rng *rand.Rand) string {
	var b strings.Builder

	for range 1 + rng.IntN(24) {
		b.WriteString(shapeOracleFragments[rng.IntN(len(shapeOracleFragments))])
	}

	return b.String()
}

// shapeChecks compare each replacement with its oracle on one text, and say
// how they differ.
var shapeChecks = map[string]func(s string) (got, want string){
	"beanPathAssignments": func(s string) (string, string) {
		var want []string
		for _, m := range oracleBeanPath.FindAllStringSubmatch(s, -1) {
			want = append(want, m[1]+"="+m[2]+m[3])
		}

		var got []string
		for _, a := range beanPathAssignments(s) {
			got = append(got, a.namespace+"="+a.dir)
		}

		return fmt.Sprint(got), fmt.Sprint(want)
	},
	"diLocations": func(s string) (string, string) {
		want := "none"
		if m := oracleDILocations.FindStringSubmatch(s); m != nil {
			want = m[1]
		}

		got := "none"
		if v, ok := diLocations(s); ok {
			got = v
		}

		return got, want
	},
	"extendsFW1": func(s string) (string, string) {
		return strconv.FormatBool(extendsFW1(s)), strconv.FormatBool(oracleFW1App.MatchString(s))
	},
	"mappingTargetKey": func(s string) (string, string) {
		want := "none"
		if m := oracleMappingTarget.FindStringSubmatch(s); m != nil {
			want = m[1]
		}

		got := "none"
		if v, ok := mappingTargetKey(s); ok {
			got = v
		}

		return got, want
	},
	"servicePropertyAnnotations": func(s string) (string, string) {
		var want []string
		for _, m := range oracleServiceProp.FindAllStringSubmatchIndex(s, -1) {
			want = append(want, fmt.Sprint(m[0], s[m[2]:m[3]], s[m[4]:m[5]], s[m[6]:m[7]]))
		}

		var got []string
		for _, a := range servicePropertyAnnotations(s) {
			got = append(got, fmt.Sprint(a.start, a.name, a.kind, a.target))
		}

		return fmt.Sprint(got), fmt.Sprint(want)
	},
	"mockClassName": func(s string) (string, string) {
		want := ""
		if m := oracleMockClassName.FindStringSubmatch(s); m != nil {
			want = m[1]
		}

		return mockClassName(s), want
	},
	"hasMarkupEvidence": func(s string) (string, string) {
		return strconv.FormatBool(hasMarkupEvidence(s)), strconv.FormatBool(oracleMarkup.MatchString(s))
	},
	"placeholders": func(s string) (string, string) {
		want := fmt.Sprint(oraclePlaceholder.MatchString(s), oraclePlaceholder.ReplaceAllString(s, ""), oraclePlaceholder.Split(s, -1))
		got := fmt.Sprint(hasPlaceholder(s), withoutPlaceholders(s), splitPlaceholders(s))

		return got, want
	},
	"serviceLoopHead": func(s string) (string, string) {
		want := "none"
		if m := oracleServiceLoop.FindStringSubmatch(s); m != nil {
			want = fmt.Sprintf("%q %q %d", m[1], m[2], len(m[0]))
		}

		got := "none"
		if l, ok := serviceLoopHead(s); ok {
			got = fmt.Sprintf("%q %q %d", l.iterator, l.list, l.bodyStart)
		}

		return got, want
	},
	"serviceMemberWriteAt": func(s string) (string, string) {
		want := "none"
		if m := oracleServiceWrite.FindStringSubmatch(s); m != nil {
			want = fmt.Sprintf("%q", m[1:])
		}

		got := "none"
		if w, ok := serviceMemberWriteAt(s); ok {
			got = fmt.Sprintf("%q", []string{w.scope, w.key, w.factory, w.bean})
		}

		return got, want
	},
	"listAppendCall": func(s string) (string, string) {
		want := "none"
		if m := oracleServiceAppend.FindStringSubmatch(s); m != nil {
			want = m[1] + " " + m[2]
		}

		got := "none"
		if list, item, ok := listAppendCall(s); ok {
			got = list + " " + item
		}

		return got, want
	},
}

// unfolded reports whether s holds a character a regular expression's (?i)
// folds to ASCII and the shapes, folding in ASCII alone, do not: the Kelvin
// sign and the long s.
func unfolded(s string) bool { return strings.ContainsAny(s, "Kſ") }

// Every shape reads what its regular expression read, on the hand-written
// cases and on 200,000 texts built from the shapes' own fragments (10,000
// under -short; FuzzTextShapes goes further).
func TestTextShapesMatchTheirExpressions(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	texts := slices.Clone(shapeCases)

	n := 200_000
	if testing.Short() {
		n = 10_000
	}

	for range n {
		texts = append(texts, randomShapeText(rng))
	}

	names := make([]string, 0, len(shapeChecks))
	for name := range shapeChecks {
		names = append(names, name)
	}

	slices.Sort(names)

	for _, name := range names {
		check := shapeChecks[name]
		failures := 0

		for _, s := range texts {
			if got, want := check(s); got != want {
				t.Errorf("%s(%q) = %s, want %s", name, s, got, want)

				failures++
				if failures == 5 {
					break
				}
			}
		}
	}
}

// FuzzTextShapes holds each shape to its regular expression on any text.
func FuzzTextShapes(f *testing.F) {
	for _, s := range shapeCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if unfolded(s) {
			t.Skip()
		}

		for name, check := range shapeChecks {
			if got, want := check(s); got != want {
				t.Errorf("%s(%q) = %s, want %s", name, s, got, want)
			}
		}
	})
}
