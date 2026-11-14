package parser

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// matcherCase is a matcher held to the pattern it replaced: the expression the
// built-in rule spelled its match with, compiled with (?i) as the resolver
// compiled it.
type matcherCase struct {
	name    string
	matcher Matcher
	oracle  *regexp.Regexp
	seeds   []string
}

func oracle(pattern string) *regexp.Regexp { return regexp.MustCompile("(?i)" + pattern) }

var wireIDPattern = func(fn string) string {
	return `(?:^|\.)` + fn + `\(\s*(?:\w+\s*[=:]\s*)?["'](?:id:|model:)?([A-Za-z_][\w.]*(?:@[\w.-]+)?)@?["']\s*(?:,[^()]*)?\)$`
}

var matcherCases = []matcherCase{
	{
		"names", NameSet("prc.oCurrentAuthor", "author", "site"),
		oracle(`^(?:variables\.|arguments\.)?(?:` + regexp.QuoteMeta("prc.oCurrentAuthor") + `|author|site)$`),
		[]string{"author", "Author", "variables.author", "arguments.site", "prc.oCurrentAuthor", "variables.prc.oCurrentAuthor", "prcXoCurrentAuthor", "authors", "x.author", "author ", " author", "variables.", "variables.variables.site", "SITE"},
	},
	{
		"empty call", EmptyCall("content"),
		oracle(`(?:^|\.)\s*content\s*\(\s*\)$`),
		[]string{"content()", "content( )", " content ( ) ", "x.content()", "$.content()", "x . content()", "x.\tcontent\n(\n)", "content", "content(1)", "contents()", "xcontent()", "content()()", "a.b.content()", "content().content()", "content(", "CONTENT()"},
	},
	{
		"wire id", &LiteralCall{Func: "getInstance", Named: AnyNamedArg, Form: IDWire, Rest: true},
		oracle(wireIDPattern("getInstance")),
		[]string{
			`getInstance("Foo")`, `getInstance( 'Foo' )`, `getInstance(name="Foo")`, `getInstance( name : "pkg.Foo" )`, `getInstance("Foo@mod")`, `getInstance("Foo@")`, `getInstance("Foo@mod@")`,
			`getInstance("id:Foo")`, `getInstance("model:Foo@m")`, `getInstance("ID:Foo")`, `getInstance("id:")`, `getInstance("id")`, `getInstance("Foo", true)`, `getInstance("Foo",x)`, `getInstance("Foo", f(1))`,
			`x.getInstance("Foo")`, `a.b.getInstance("Foo")`, `getInstance("Foo").bar()`, `getInstance("1Foo")`, `getInstance("_Foo")`, `getInstance("Foo-bar")`, `getInstance("Foo@b-c.d")`, `getInstance("Foo@b@c")`,
			`getInstance("Foo')`, `getInstance('Foo")`, `getInstance(Foo)`, `getInstance( )`, `getInstance("Foo"`, `getInstance("Foo" )`, `getInstance("Foo" , 1 )`, `getInstance("a.b.c@d.e")`, `GETINSTANCE("Foo")`,
		},
	},
	{
		"dsl", &LiteralCall{Func: "getInstance", Bare: true, Named: AnyNamedArg, Form: IDLiteral, Literal: "logbox"},
		oracle(`^getInstance\(\s*(?:\w+\s*[=:]\s*)?["']` + regexp.QuoteMeta("logbox") + `["']\s*\)$`),
		[]string{`getInstance("logbox")`, `getInstance('LogBox')`, `getInstance( dsl = "logbox" )`, `x.getInstance("logbox")`, `getInstance("logbox", 1)`, `getInstance("logboxx")`, `getInstance("logbox").x()`, ` getInstance("logbox")`},
	},
	{
		"mura scope", &LiteralCall{Func: "getValue", Form: IDLiteral, Literal: "muraScope"},
		oracle(`(?:^|\.)getValue\(\s*["']muraScope["']\s*\)$`),
		[]string{`getValue("muraScope")`, `x.getValue('MuraScope')`, `getValue( "muraScope" )`, `getValue(name="muraScope")`, `getValue("muraScope", 1)`, `getValue("x")`, `getValue("muraScope").a()`},
	},
	{
		"slatwall service", &LiteralCall{Func: "getService", Named: AnyNamedArgEquals, Form: IDWord},
		oracle(`(?:^|\.)getService\(\s*(?:\w+\s*=\s*)?["']([A-Za-z_]\w*)["']\s*\)$`),
		[]string{`getService("hibachiService")`, `x.getService('Foo')`, `getService( name = "Foo" )`, `getService(name: "Foo")`, `getService("Foo", 1)`, `getService("1Foo")`, `getService("a.b")`, `getService("Foo").x()`, `getService("")`, `getService(x="a"`},
	},
	{
		"bean rule", &LiteralCall{Func: "getBean", Named: AnyNamedArg, Form: IDLiteral, Literal: "userService", Rest: true},
		oracle(`(?:^|\.)getBean\(\s*(?:\w+\s*[=:]\s*)?["']` + regexp.QuoteMeta("userService") + `["']\s*(?:,[^()]*)?\)$`),
		[]string{`getBean("userService")`, `$.getBean('UserService')`, `getBean( id = "userService" )`, `getBean("userService", true)`, `getBean("userService",a,b)`, `getBean("userService", f())`, `getBean("userServices")`, `getBean("userService").x()`, `x.y.getBean("userService")`},
	},
	{
		"mura bean rule", &LiteralCall{Func: "getBean", Named: ExactNamedArg, ArgName: "beanName", Form: IDLiteral, Literal: "content.x", Rest: true},
		oracle(`(?:^|\.)getBean\(\s*(?:beanName\s*[=:]\s*)?["']` + regexp.QuoteMeta("content.x") + `["']\s*(?:,[^()]*)?\)$`),
		[]string{`getBean("content.x")`, `getBean(beanName="content.x")`, `getBean( BEANNAME : 'content.x' )`, `getBean(name="content.x")`, `getBean("content.x", 1)`, `getBean("contentXx")`, `getBean("content.x"`},
	},
	{
		"path id", &LiteralCall{Func: "getBean", Named: AnyNamedArg, Form: IDPath, Rest: true},
		oracle(`(?:^|\.)getBean\(\s*(?:\w+\s*[=:]\s*)?["']([A-Za-z_][\w.]*)["']\s*(?:,[^()]*)?\)$`),
		[]string{`getBean("userService")`, `$.getBean('content.x')`, `getBean( id = "a.b.c" )`, `getBean("a.b", true)`, `getBean("a-b")`, `getBean("1a")`, `getBean("a@b")`, `getBean("")`, `getBean("a.b").c()`, `x.getBean("a")`, `getBean("a"`, `getBean(a="b", c)`},
	},
	{
		"java class", JavaClass(),
		oracle(`createObject\s*\(\s*['"]java['"]\s*,\s*['"](.+?)['"]\s*\)`),
		[]string{
			`createObject("java", "java.io.File")`, `createObject( 'java' , 'a.B' )`, `x = createObject("java","a.B").init()`, `createObject("component","a.B")`, `createObject("java", "a'b")`, `createObject("java", "a" )`,
			`createObject("java", "a") createObject("java", "b")`, `createObject("java","")`, "createObject(\"java\", \"a\nb\")", `createobject("JAVA","a.B")`, `createObject("java", "a.B"`, `y(createObject("java", "x.Y"))`, `createObject ( "java" , "a.B" )`, `createObject("java", 'a")`,
		},
	},
}

var matcherFragments = []string{
	"getInstance", "getBean", "getService", "getValue", "createObject", "content", "author", "site", "variables.", "arguments.", "prc.oCurrentAuthor", ".", "(", ")", ",", " ", "  ", "\t", "\n", "\"", "'", "=", ":",
	"id:", "model:", "ID:", "@", "@mod", "-", "_", "1", "x", "Foo", "a.b", "name", "beanName", "logbox", "muraScope", "userService", "content.x", "java", "component", "é", "\xff", "()", "( )", "true", "f(1)",
}

func randomMatcherText(rng *rand.Rand, seeds []string) string {
	if rng.IntN(2) == 0 {
		s := seeds[rng.IntN(len(seeds))]

		for range rng.IntN(4) {
			i := rng.IntN(len(s) + 1)
			j := min(len(s), i+rng.IntN(3))

			switch rng.IntN(3) {
			case 0:
				s = s[:i] + matcherFragments[rng.IntN(len(matcherFragments))] + s[i:]
			case 1:
				s = s[:i] + s[j:]
			default:
				s = s[:i] + matcherFragments[rng.IntN(len(matcherFragments))] + s[j:]
			}
		}

		return s
	}

	var b strings.Builder

	for range 1 + rng.IntN(12) {
		b.WriteString(matcherFragments[rng.IntN(len(matcherFragments))])
	}

	return b.String()
}

func matcherDisagreement(c *matcherCase, expr string) string {
	got, ok := c.matcher.Match(expr)
	m := c.oracle.FindStringSubmatch(expr)

	var want []string
	if m != nil {
		want = m[1:]
	}

	if ok != (m != nil) || fmt.Sprintf("%q", got) != fmt.Sprintf("%q", want) {
		return fmt.Sprintf("%s: Match(%q) = %q %v, want %q %v", c.name, expr, got, ok, want, m != nil)
	}

	return ""
}

// unfoldedMatcherRune is a character a regular expression's (?i) folds to
// ASCII and the matchers, folding in ASCII alone, do not: the Kelvin sign and
// the long s.
func unfoldedMatcherRune(s string) bool { return strings.ContainsAny(s, "Kſ") }

// Each matcher answers what the pattern it replaced answered, captures
// included, on expressions built from the pattern's own forms, their near
// misses, and texts built from fragments; one text in ten is a match, so the
// comparison is not of two refusals.
func TestMatchersMatchTheirPatterns(t *testing.T) {
	n := 100_000
	if testing.Short() {
		n = 10_000
	}

	for i := range matcherCases {
		c := &matcherCases[i]
		rng := rand.New(rand.NewPCG(uint64(i), 7))

		texts := append([]string(nil), c.seeds...)
		for range n {
			texts = append(texts, randomMatcherText(rng, c.seeds))
		}

		failures, matched := 0, 0

		for _, s := range texts {
			if unfoldedMatcherRune(s) {
				continue
			}

			if msg := matcherDisagreement(c, s); msg != "" {
				t.Error(msg)

				failures++

				if failures == 6 {
					break
				}
			}

			if c.oracle.MatchString(s) {
				matched++
			}
		}

		if matched < len(texts)/50 {
			t.Errorf("%s: %d of %d texts matched; the generated texts do not reach the matcher", c.name, matched, len(texts))
		}
	}
}

// FuzzMatchers holds every matcher to its pattern on any text.
func FuzzMatchers(f *testing.F) {
	for i := range matcherCases {
		for _, s := range matcherCases[i].seeds {
			f.Add(s)
		}
	}

	f.Fuzz(func(t *testing.T, s string) {
		if unfoldedMatcherRune(s) {
			t.Skip()
		}

		for i := range matcherCases {
			if msg := matcherDisagreement(&matcherCases[i], s); msg != "" {
				t.Error(msg)
			}
		}
	})
}
