package path

import (
	"io/fs"
	"maps"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The expressions AssumedMuraPlugin read a plugin's config.xml.cfm with before
// it read tags, kept as the oracle the reader is compared to.
var (
	oracleMuraPluginTag     = regexp.MustCompile(`(?i)<plugin\s*>`)
	oracleMuraPluginPackage = regexp.MustCompile(`(?i)<package>\s*([A-Za-z_][\w-]*)\s*</package>`)
	oracleMuraPluginMapping = regexp.MustCompile(`(?i)<mapping\s+name\s*=\s*["']([A-Za-z_][\w-]*)["']\s+directory\s*=\s*["']([^"'#<>]+)["']`)
)

// oracleMuraPlugin is AssumedMuraPlugin as it was, on one file's content.
func oracleMuraPlugin(dir string, data []byte) (MuraPlugin, bool) {
	if !oracleMuraPluginTag.Match(data) {
		return MuraPlugin{}, false
	}

	m := oracleMuraPluginPackage.FindSubmatch(data)
	if m == nil {
		return MuraPlugin{}, false
	}

	p := MuraPlugin{Root: dir, Package: string(m[1]), Mappings: map[string]string{string(m[1]): dir}}

	for _, mm := range oracleMuraPluginMapping.FindAllSubmatch(data, -1) {
		target := filepath.Clean(filepath.Join(dir, filepath.FromSlash(string(mm[2]))))
		if rel, err := filepath.Rel(dir, target); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}

		if _, dup := p.Mappings[string(mm[1])]; !dup {
			p.Mappings[string(mm[1])] = target
		}
	}

	return p, true
}

func samePlugin(a, b MuraPlugin, aok, bok bool) bool {
	return aok == bok && a.Root == b.Root && a.Package == b.Package && maps.Equal(a.Mappings, b.Mappings)
}

// readPlugin is AssumedMuraPlugin on content, as the file is read from dir/plugin.
func readPlugin(dir string, content []byte) (MuraPlugin, bool) {
	return AssumedMuraPlugin(dir, func(string) ([]byte, error) { return content, nil })
}

// TestMuraPluginReaderAgreesOnRealPlugins compares the reader to the
// expressions it replaced on every plugin/config.xml(.cfm) under
// CLIF_MURAPLUGIN_CORPUS (directories, as PATH is). Skipped without it.
func TestMuraPluginReaderAgreesOnRealPlugins(t *testing.T) {
	corpus := os.Getenv("CLIF_MURAPLUGIN_CORPUS")
	if corpus == "" {
		t.Skip("CLIF_MURAPLUGIN_CORPUS not set")
	}

	files, plugins, mappings := 0, 0, 0

	for _, root := range filepath.SplitList(corpus) {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}

			if base := filepath.Base(path); (base != "config.xml" && base != "config.xml.cfm") || filepath.Base(filepath.Dir(path)) != "plugin" {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil //nolint:nilerr // a file that cannot be read is not compared
			}

			dir := filepath.Dir(filepath.Dir(path))
			files++

			got, gok := readPlugin(dir, data)
			want, wok := oracleMuraPlugin(dir, data)

			if !samePlugin(got, want, gok, wok) {
				t.Errorf("%s:\n  reader %v %+v\n  oracle %v %+v", path, gok, got, wok, want)
			}

			if wok {
				plugins++
				mappings += len(want.Mappings)
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Logf("%d config files, %d plugins, %d mappings compared", files, plugins, mappings)
}

// plainPluginText is a config.xml.cfm built from pieces, in the forms the
// expressions read: no comments, the attributes of a <mapping> in the order
// name, directory, each value quoted by one kind of quote. On those the
// reader must answer what the expressions did.
func plainPluginText(rng *rand.Rand) string {
	pick := func(s []string) string { return s[rng.IntN(len(s))] }

	noise := []string{"", "\n", " ", "\t", "<cfsilent><cfset v = \"1\"></cfsilent>", "<cfoutput>", "</cfoutput>", "#x#", "<name>X</name>", "<mappings>", "</mappings>", "text", "<cfinclude template=\"a.cfm\" />"}
	plugin := []string{"<plugin>", "<PLUGIN>", "<plugin >", "<plugin\n>", "<plugin x=\"1\">", "<plugins>", "<plugin", "</plugin>", "<pluginx>", "<plugin-x>"}
	pkg := []string{
		"<package>Slat</package>", "<package> A-b_1 </package>", "<package>#x#</package>", "<package>1x</package>", "<package></package>", "<package>a b</package>",
		"<PACKAGE>X</PACKAGE>", "<package>X", "<package x=\"y\">X</package>", "<package>\n\tPkg\n</package>", "<package>a.b</package>", "<package>a</PACKAGE>",
	}
	names := []string{"VT", "a-b", "_x", "1x", "x y", "", "x.y", "é"}
	// No value holds a quote: the expression reads `'a"b'` as the directory a,
	// which the reader refuses (TestMuraPluginReaderReadsTagsNotText).
	dirs := []string{"org/VT", "../out", "a/../b", "", "x#y#", "a<b", "a>b", "/abs", "a b", "plain", "lib\\x", "./self", ".."}
	gaps := []string{" ", "\n\t", "  ", "\r\n"}
	eq := []string{"=", " = ", "= ", " ="}
	attr := func(name string) string {
		return pick([]string{name, strings.ToUpper(name), strings.ToUpper(name[:1]) + name[1:]})
	}

	var b strings.Builder

	for range 1 + rng.IntN(9) {
		switch rng.IntN(4) {
		case 0:
			b.WriteString(pick(noise))
		case 1:
			b.WriteString(pick(plugin))
		case 2:
			b.WriteString(pick(pkg))
		default:
			q := pick([]string{`"`, `'`})
			tag := pick([]string{"mapping", "MAPPING", "mappings", "mapping2"})
			b.WriteString("<" + tag + pick(gaps) + attr("name") + pick(eq) + q + pick(names) + q +
				pick(gaps) + attr("directory") + pick(eq) + q + pick(dirs) + q + pick([]string{"/>", ">", " />", ""}))
		}
	}

	return b.String()
}

// TestMuraPluginReaderMatchesTheExpressionsOnPlainText holds the reader to the
// expressions on generated config files with the features they share, as the
// readers of the other shapes are. The documents are not all plugins; at least
// one in twenty is, and holds a mapping besides the package.
func TestMuraPluginReaderMatchesTheExpressionsOnPlainText(t *testing.T) {
	n := 200_000
	if testing.Short() {
		n = 20_000
	}

	rng := rand.New(rand.NewPCG(1, 7))
	dir := filepath.Join(t.TempDir(), "plug")

	plugins, withMapping, failures := 0, 0, 0

	for range n {
		text := plainPluginText(rng)
		got, gok := readPlugin(dir, []byte(text))
		want, wok := oracleMuraPlugin(dir, []byte(text))

		if !samePlugin(got, want, gok, wok) {
			t.Errorf("%q:\n  reader %v %v\n  oracle %v %v", text, gok, got.Mappings, wok, want.Mappings)

			failures++

			if failures == 8 {
				return
			}
		}

		if wok {
			plugins++

			if len(want.Mappings) > 1 {
				withMapping++
			}
		}
	}

	if plugins < n/20 || withMapping < n/100 {
		t.Errorf("%d plugins and %d with a mapping of %d texts; the generated texts do not reach the reader", plugins, withMapping, n)
	}
}

// TestMuraPluginReaderReadsTagsNotText: what the expressions read as text and
// the reader as tags. A <mapping> in a comment is no mapping (Mura's own plugin
// template ships one as an example, `myMapping`), the attributes of a <mapping>
// may come in either order with others between, and a value whose quotes do not
// match is not a value.
func TestMuraPluginReaderReadsTagsNotText(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "plug")

	for _, c := range []struct {
		name, text string
		want       map[string]string // by name, relative to the plugin; the package maps to ""
	}{
		{"a mapping", `<plugin><package>P</package><mapping name="M" directory="lib"/></plugin>`, map[string]string{"P": "", "M": "lib"}},
		{"in an XML comment", "<plugin><package>P</package><!--\n<mapping name=\"M\" directory=\"lib\"/>\n--></plugin>", map[string]string{"P": ""}},
		{"in a CFML comment", "<plugin><package>P</package><!--- <mapping name=\"M\" directory=\"lib\"/> ---></plugin>", map[string]string{"P": ""}},
		{"in a nested CFML comment", "<plugin><package>P</package><!--- <!--- x ---> <mapping name=\"M\" directory=\"lib\"/> ---></plugin>", map[string]string{"P": ""}},
		{"after a comment", "<plugin><package>P</package><!-- x --><mapping name=\"M\" directory=\"lib\"/></plugin>", map[string]string{"P": "", "M": "lib"}},
		{"directory first", `<plugin><package>P</package><mapping directory="lib" name="M"/></plugin>`, map[string]string{"P": "", "M": "lib"}},
		{"another attribute between", `<plugin><package>P</package><mapping name="M" note="x" directory="lib"/></plugin>`, map[string]string{"P": "", "M": "lib"}},
		{"a quote in the value", `<plugin><package>P</package><mapping name="M" directory="li'b"/></plugin>`, map[string]string{"P": ""}},
		{"a > in an earlier tag's value", `<cfset x = "a>b"><plugin><package>P</package><mapping name="M" directory="lib"/></plugin>`, map[string]string{"P": "", "M": "lib"}},
		{"a stray apostrophe before the plugin", `<cfset x = 'it's'><plugin><package>P</package></plugin>`, map[string]string{"P": ""}},
		{"a package in a comment, a literal after", "<plugin><!-- <package>Old</package> --><package>New</package></plugin>", map[string]string{"New": ""}},
		{"the plugin only in a comment", "<!-- <plugin> --><package>P</package>", nil},
		{"a package that is an expression", "<plugin><package>#variables.settings.package#</package></plugin>", nil},
		{"a package with attributes", `<plugin><package x="y">P</package></plugin>`, nil},
	} {
		got, ok := readPlugin(dir, []byte(c.text))
		if c.want == nil {
			if ok {
				t.Errorf("%s: read as a plugin: %v", c.name, got.Mappings)
			}

			continue
		}

		if !ok {
			t.Errorf("%s: not read as a plugin", c.name)

			continue
		}

		want := map[string]string{}
		for name, rel := range c.want {
			want[name] = filepath.Clean(filepath.Join(dir, rel))
		}

		if !maps.Equal(got.Mappings, want) {
			t.Errorf("%s: mappings %v, want %v", c.name, got.Mappings, want)
		}
	}
}
