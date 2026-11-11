package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// CFTag is one CF tag in a text, read as the parse reads tags: outside
// <!--- ---> comments, its extent past a '>' inside a quoted value
// (tagEnd), its attributes by the CF-mode attribute scanner. The resolver
// read <cfloop>, <cfargument>, <cffunction> and <cfreturn> with regular
// expressions of its own, `<cfloop\b([^>]*)>` and the like, which ended a tag
// at the first '>' wherever it stood, read a commented-out tag as live, and
// took a value to either quote.
type CFTag struct {
	// Name is the tag's name, lowercased: "cfloop".
	Name string
	// Close is true for an end tag, </cfloop>.
	Close bool
	// Start is the offset of the tag's '<', End the offset just past it:
	// past its '>', or where an unclosed tag stops.
	Start, End int

	text string
}

// Text is the tag's text, from its '<' to its End.
func (t *CFTag) Text() string { return t.text }

// Attr is the value of attribute name, without its quotes, and whether the
// tag gives it.
func (t *CFTag) Attr(name string) (string, bool) {
	start, end, ok := tagAttrValue(t.text, name, cfAttrs)
	if !ok {
		return "", false
	}

	return t.text[start:end], true
}

// Attrs is every attribute the tag gives a value, by lowercased name; a name
// given twice keeps its first value, as the grammar does.
func (t *CFTag) Attrs() map[string]string { return tagAttrMap(t.text, cfAttrs) }

// Body is the tag's text after its name, up to its '>' (or "/>"), trimmed:
// the expression of an expression tag, <cfreturn x[i]>.
func (t *CFTag) Body() string {
	body := t.text[min(len(t.text), len("<")+len(t.Name)):]
	body = strings.TrimSuffix(body, ">")
	body = strings.TrimSuffix(strings.TrimSpace(body), "/")

	return strings.TrimSpace(body)
}

// CFTags is the opening and closing tags named one of names (lowercased,
// "cfloop") in content, in order. With no names it is every CF tag.
func CFTags(content string, names ...string) []CFTag {
	var (
		out  []CFTag
		ends tagEnds
		made bool
	)

	for pos := 0; pos < len(content); {
		i := strings.IndexByte(content[pos:], '<')
		if i < 0 {
			break
		}

		i += pos

		if strings.HasPrefix(content[i:], "<!---") {
			pos = SkipCFMLComment(content, i)

			continue
		}

		j := i + 1
		closing := j < len(content) && content[j] == '/'

		if closing {
			j++
		}

		k := j
		for k < len(content) && isTagNameByte(content[k]) {
			k++
		}

		name := content[j:k]
		if !lex.HasPrefixFold(name, "cf") || !namedTag(name, names) {
			pos = i + 1

			continue
		}

		if !made {
			ends, made = newTagEnds(content), true
		}

		end := tagEnd(&ends, k)
		out = append(out, CFTag{Name: strings.ToLower(name), Close: closing, Start: i, End: end, text: content[i:end]})
		pos = end
	}

	return out
}

// namedTag reports whether name is one of names, ignoring case; any name
// when there are none.
func namedTag(name string, names []string) bool {
	if len(names) == 0 {
		return true
	}

	for _, n := range names {
		if strings.EqualFold(name, n) {
			return true
		}
	}

	return false
}

// WithoutCFTags is content with the tags named names removed: a function
// body without its <cfargument> tags.
func WithoutCFTags(content string, names ...string) string {
	tags := CFTags(content, names...)
	if len(tags) == 0 {
		return content
	}

	var b strings.Builder

	last := 0

	for i := range tags {
		b.WriteString(content[last:tags[i].Start])
		last = tags[i].End
	}

	b.WriteString(content[last:])

	return b.String()
}
