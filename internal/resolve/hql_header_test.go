package resolve

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

// The expressions hql_header.go replaced, kept as the oracle each reader is
// held to.
var (
	oracleHQLEntity = regexp.MustCompile(`(?is)^\s*(?:select\s+(\w+)\s+)?from\s+([A-Za-z_][\w.]*)(?:\s+(?:as\s+)?(\w+))?(\s|$)`)
	oracleHQLJoin   = regexp.MustCompile(`(?i)\bjoin\b`)
	oracleHQLWhere  = regexp.MustCompile(`(?i)\bwhere\b`)
)

var hqlCases = []string{
	"FROM Booking", "from Booking WHERE id = :id", "SELECT b FROM Booking b WHERE b.id = :id", "select b from Booking as b",
	"from Booking b", "from Booking as b where x", "from Booking as where x", "from Booking as", "from Booking as  ", "from Booking as\n",
	"from Booking b, Other o", "from Booking, Other", "from Booking b , Other", "from Booking as b, Other", "from Booking.x", "from Booking.",
	"from my.pkg.Booking b", "from _Booking", "from 1Booking", "from .Booking", "from Booking\n b\n where", "from\tBooking\tb",
	"  from Booking", "\n\tfrom Booking", "select b.name from Booking b", "select b, c from Booking b", "select from Booking", "select b from",
	"select b from Booking", "select b from Booking x", "select b  from  Booking  b", "select\tb\nfrom Booking b", "selectb from Booking",
	"select b fromBooking", "select from from Booking", "select b from from Booking", "from", "from ", "select", "select ", "", " ",
	"from Booking inner join b.items i", "from Booking b join b.items", "from Booking b JOIN b.items", "from Booking b joint", "from Booking b rejoin",
	"from Booking b left_join c", "from Booking b\njoin\nb.items", "from Booking b where b.x = 1 and b.y in (select 1 from X x, Y y)",
	"from Booking b where b.a, b.b", "from Booking b WHERE x", "from Booking b where", "where from Booking", "from Booking b nowhere", "from Booking b somewhere x",
	"from Booking b ELSEWHERE", "from Booking b\twhere\t:x", "from Booking b where_ x", "from Booking _where b", "FROM Booking B WHERE B.x=1",
	"from Booking b order by b.name", "from Booking b group by b.x", "from Booking order by name", "from Booking where", "from Booking é", "from é",
	"from Booking b\x00", "from\x00Booking", "from Booking\xff", "from Booking b", "from Booking",
}

var hqlFragments = []string{
	"select", "SELECT", "from", "FROM", "as", "AS", "where", "WHERE", "join", "JOIN", "inner", "left", "Booking", "b", "x", "my.pkg.Entity", ".", ",", " ", "  ",
	"\t", "\n", "\r", "\f", "\v", "(", ")", "=", ":id", "1", "_", "é", "\xff", "#", "'", "\"", "joins", "rejoin", "somewhere", "where_", "_where",
}

func randomHQL(rng *rand.Rand) string {
	if rng.IntN(2) == 0 {
		s := hqlCases[rng.IntN(len(hqlCases))]

		for range rng.IntN(4) {
			i := rng.IntN(len(s) + 1)
			j := min(len(s), i+rng.IntN(3))

			switch rng.IntN(3) {
			case 0:
				s = s[:i] + hqlFragments[rng.IntN(len(hqlFragments))] + s[i:]
			case 1:
				s = s[:i] + s[j:]
			default:
				s = s[:i] + hqlFragments[rng.IntN(len(hqlFragments))] + s[j:]
			}
		}

		return s
	}

	var b strings.Builder

	for range 1 + rng.IntN(12) {
		b.WriteString(hqlFragments[rng.IntN(len(hqlFragments))])
	}

	return b.String()
}

func hqlDisagreement(s string) []string {
	var bad []string

	check := func(name, got, want string) {
		if got != want {
			bad = append(bad, fmt.Sprintf("%s(%q) = %s, want %s", name, s, got, want))
		}
	}

	selected, entity, alias, ok := hqlEntityHeader(s)
	m := oracleHQLEntity.FindStringSubmatch(s)

	var want string
	if m != nil {
		want = fmt.Sprintf("%q %q %q true", m[1], m[2], m[3])
	} else {
		want = `"" "" "" false`
	}

	check("hqlEntityHeader", fmt.Sprintf("%q %q %q %v", selected, entity, alias, ok), want)
	check("hqlHasJoin", strconv.FormatBool(hqlHasJoin(s)), strconv.FormatBool(oracleHQLJoin.MatchString(s)))
	check("hqlBeforeWhere", fmt.Sprintf("%q", hqlBeforeWhere(s)), fmt.Sprintf("%q", oracleHQLWhere.Split(s, 2)[0]))

	return bad
}

// unfoldedHQLRune is a character a regular expression's (?i) folds to ASCII and
// the readers, folding in ASCII alone, do not: the Kelvin sign and the long s.
func unfoldedHQLRune(s string) bool { return strings.ContainsAny(s, "Kſ") }

// Every reader answers what its expression answered, the groups included, on
// the queries a collection callback reads, their near misses, and texts built
// from fragments of them; one in twenty is a header, so the comparison is not
// of two refusals.
func TestHQLReadersMatchTheirExpressions(t *testing.T) {
	n := 300_000
	if testing.Short() {
		n = 30_000
	}

	texts := append([]string(nil), hqlCases...)

	rng := rand.New(rand.NewPCG(1, 7))
	for range n {
		texts = append(texts, randomHQL(rng))
	}

	failures, headers, joins := 0, 0, 0

	for _, s := range texts {
		if unfoldedHQLRune(s) {
			continue
		}

		for _, msg := range hqlDisagreement(s) {
			t.Error(msg)

			failures++

			if failures == 10 {
				return
			}
		}

		if oracleHQLEntity.MatchString(s) {
			headers++
		}

		if oracleHQLJoin.MatchString(s) {
			joins++
		}
	}

	if headers < len(texts)/20 || joins < len(texts)/50 {
		t.Errorf("%d headers and %d joins of %d texts; the generated texts do not reach the readers", headers, joins, len(texts))
	}
}

// FuzzHQLReaders holds each reader to its expression on any text.
func FuzzHQLReaders(f *testing.F) {
	for _, s := range hqlCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if unfoldedHQLRune(s) {
			t.Skip()
		}

		for _, msg := range hqlDisagreement(s) {
			t.Error(msg)
		}
	})
}

// hqlLiteral is a string literal in CFML source that reads like HQL: it starts
// with from or select. Found with an expression here, which is only a way to
// collect candidates for the comparison.
var hqlLiteral = regexp.MustCompile(`(?is)["']\s*((?:select|from)\s[^"']*)["']`)

// TestHQLReadersAgreeOnACorpus compares the readers to the expressions on every
// string literal that reads like HQL in the CFML files under CLIF_HQL_CORPUS
// (directories, as PATH is). Skipped without it, so it never runs in CI.
func TestHQLReadersAgreeOnACorpus(t *testing.T) {
	corpus := os.Getenv("CLIF_HQL_CORPUS")
	if corpus == "" {
		t.Skip("CLIF_HQL_CORPUS not set")
	}

	files, texts, headers := 0, 0, 0

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

			files++

			for _, m := range hqlLiteral.FindAllStringSubmatch(string(data), -1) {
				texts++

				for _, msg := range hqlDisagreement(m[1]) {
					t.Errorf("%s: %s", path, msg[:min(len(msg), 200)])
				}

				if oracleHQLEntity.MatchString(m[1]) {
					headers++
				}
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Logf("%d files, %d HQL-like literals compared, %d of them a header", files, texts, headers)
}
