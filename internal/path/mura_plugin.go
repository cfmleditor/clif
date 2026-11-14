package path

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// A Mura plugin is deployed as plugins/<directory>, and Mura's plugin manager
// writes plugins/mappings.cfm for it: the plugin's package maps to its
// directory, and each <mapping name directory> its plugin/config.xml declares
// maps to that directory inside it. Every Mura application includes that file,
// so the plugin's own components name each other through the package
// (Slatwall's `extends="Slatwall.com.utility.BaseObject"`). A checkout of a
// plugin is that directory, and nothing in it spells the mapping as CFML.

// MuraPlugin is a Mura plugin recognised at Root.
type MuraPlugin struct {
	// Root is the plugin's directory, holding plugin/config.xml(.cfm).
	Root string
	// Package is the plugin's <package>.
	Package string
	// Mappings are the mappings Mura gives the plugin, by name: the package
	// and each declared <mapping>, to absolute directories.
	Mappings map[string]string
}

// AssumedMuraPlugin is the Mura plugin whose plugin/config.xml.cfm or
// plugin/config.xml is in dir, if one is. It needs the <plugin> element and a
// literal <package>: a Mura module's config.xml.cfm has neither.
func AssumedMuraPlugin(dir string, read func(string) ([]byte, error)) (MuraPlugin, bool) {
	for _, name := range []string{"config.xml.cfm", "config.xml"} {
		data, err := read(filepath.Join(dir, "plugin", name))
		if err != nil {
			continue
		}

		content := string(data)

		tags := muraPluginTags(content)
		if !slices.ContainsFunc(tags, func(t muraTag) bool { return t.is("plugin") && t.bare() }) {
			return MuraPlugin{}, false
		}

		pkg := muraPluginPackage(content, tags)
		if pkg == "" {
			return MuraPlugin{}, false
		}

		p := MuraPlugin{Root: dir, Package: pkg, Mappings: map[string]string{pkg: dir}}

		for _, t := range tags {
			name, directory, ok := t.mapping()
			if !ok {
				continue
			}

			target := filepath.Clean(filepath.Join(dir, filepath.FromSlash(directory)))
			// A mapping is a directory inside the plugin.
			if rel, err := filepath.Rel(dir, target); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				continue
			}

			if _, dup := p.Mappings[name]; !dup {
				p.Mappings[name] = target
			}
		}

		return p, true
	}

	return MuraPlugin{}, false
}

// NearestMuraPlugin is the Mura plugin dir is in: AssumedMuraPlugin of dir or
// the nearest directory above it that is one.
func NearestMuraPlugin(dir string, read func(string) ([]byte, error)) (MuraPlugin, bool) {
	for range 32 {
		if p, ok := AssumedMuraPlugin(dir, read); ok {
			return p, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return MuraPlugin{}, false
}

// muraTag is a start tag of a plugin's config.xml.cfm: a template, so not
// always well-formed XML (`<cfsilent><cfset v = "1"></cfsilent>` stands before
// the <plugin>), and read as a tag reader reads one, not as an XML decoder
// would.
type muraTag struct {
	name string // as written
	text string // from the '<' through the '>'
	end  int    // the offset just past the '>'
}

func (t muraTag) is(name string) bool { return strings.EqualFold(t.name, name) }

// bare reports whether the tag is closed and has no attributes: `<plugin>` and
// `<plugin >`, not `<plugin` where another tag begins.
func (t muraTag) bare() bool {
	inner, closed := strings.CutSuffix(t.text, ">")

	return closed && lex.SkipRegexpSpace(inner, len("<")+len(t.name)) == len(inner)
}

// mapping is the name and directory of a <mapping>, when it has a name that is
// a word and a literal directory: one with no expression, quote or bracket in
// it, which Mura's plugin manager would write as the mapping it is.
func (t muraTag) mapping() (name, directory string, ok bool) {
	if !t.is("mapping") {
		return "", "", false
	}

	name = parser.TagAttribute(t.text, "name")
	directory = parser.TagAttribute(t.text, "directory")

	if !muraPluginName(name) || directory == "" || strings.ContainsAny(directory, "\"'#<>") {
		return "", "", false
	}

	return name, directory, true
}

// muraPluginName reports whether s is a package or mapping name: a letter or
// an underscore, then word characters and hyphens.
func muraPluginName(s string) bool {
	if s == "" || !lex.IsWordStart(s[0]) {
		return false
	}

	for i := 1; i < len(s); i++ {
		if !lex.IsWord(s[i]) && s[i] != '-' {
			return false
		}
	}

	return true
}

// muraPluginPackage is the text of the first <package> that holds a name and
// nothing else, or "".
func muraPluginPackage(content string, tags []muraTag) string {
	for _, t := range tags {
		if !t.is("package") || !t.bare() {
			continue
		}

		rel := strings.IndexByte(content[t.end:], '<')
		if rel < 0 {
			continue
		}

		rel += t.end

		rest, ok := strings.CutPrefix(strings.ToLower(content[rel:]), "</package")
		if !ok || !strings.HasPrefix(strings.TrimLeft(rest, " \t\r\n\f"), ">") {
			continue
		}

		if name := strings.Trim(content[t.end:rel], " \t\r\n\f"); muraPluginName(name) {
			return name
		}
	}

	return ""
}

// muraPluginTags is the start tags of content in order, those in a comment
// (`<!-- -->`, and CFML's `<!--- --->`) or a CDATA section left out.
func muraPluginTags(content string) []muraTag {
	var tags []muraTag

	for i := 0; i < len(content); {
		lt := strings.IndexByte(content[i:], '<')
		if lt < 0 {
			break
		}

		i += lt

		switch rest := content[i:]; {
		case strings.HasPrefix(rest, "<!---"):
			i = parser.SkipCFMLComment(content, i)
		case strings.HasPrefix(rest, "<!--"):
			i = skipPast(content, i+len("<!--"), "-->")
		case strings.HasPrefix(rest, "<![CDATA["):
			i = skipPast(content, i+len("<![CDATA["), "]]>")
		case len(rest) > 1 && lex.IsASCIILetter(rest[1]):
			name := 1
			for name < len(rest) && (lex.IsWord(rest[name]) || rest[name] == '-' || rest[name] == ':' || rest[name] == '.') {
				name++
			}

			end := muraTagEnd(rest)
			if end < 0 {
				// A tag that is never closed runs to the end of the file.
				end = len(rest)
			}

			tags = append(tags, muraTag{name: rest[1:name], text: rest[:end], end: i + end})
			i += end
		default:
			i++
		}
	}

	return tags
}

// skipPast is the offset just past the first sep at or after from, or the end.
func skipPast(s string, from int, sep string) int {
	at := strings.Index(s[from:], sep)
	if at < 0 {
		return len(s)
	}

	return from + at + len(sep)
}

// muraTagEnd is the length of the start tag at the start of s, through its
// '>', which a quoted attribute value may contain. A tag with no '>' ends where
// the next tag begins, since a '<' stands in no tag; a tag whose quotes never
// close (a stray apostrophe in a CF tag) ends at its first '>' or '<'; -1 when
// there is neither, and the tag runs to the end of the text.
func muraTagEnd(s string) int {
	var quote byte

	for i := 1; i < len(s); i++ {
		switch c := s[i]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case lex.IsQuote(c):
			quote = c
		case c == '>':
			return i + 1
		case c == '<':
			return i
		}
	}

	if end := strings.IndexAny(s[1:], "<>"); end >= 0 {
		if s[1+end] == '>' {
			return end + 2
		}

		return end + 1
	}

	return -1
}
