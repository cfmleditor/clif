package server

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
)

func (s *Server) handleDocumentColor(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.DocumentColorParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	docURI := params.TextDocument.URI

	content, ok := s.getDocument(docURI)
	if !ok {
		return []protocol.ColorInformation{}, nil
	}

	return findDocumentColors(content), nil
}

func (s *Server) handleColorPresentation(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.ColorPresentationParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	inOutput := false

	if content, ok := s.getDocument(params.TextDocument.URI); ok {
		line := int(params.Range.Start.Line)
		col := byteCol(content, line, params.Range.Start.Character)
		lines := parser.NewLineTable(content)

		if lineStart, exists := lines.Start(line); exists {
			inOutput = parser.InOutputContext(content, min(lineStart+col, len(content)))
		}
	}

	return buildColorPresentations(params.Color, params.Range, inOutput), nil
}

type cssColorMatch struct {
	start, end int
	color      protocol.Color
}

func findDocumentColors(content string) []protocol.ColorInformation {
	matches := findCSSColors(content)
	if len(matches) == 0 {
		return []protocol.ColorInformation{}
	}

	lines := parser.NewLineTable(content)
	out := make([]protocol.ColorInformation, 0, len(matches))

	for _, m := range matches {
		out = append(out, protocol.ColorInformation{
			Range: offsetRange(content, lines, m.start, m.end),
			Color: m.color,
		})
	}

	return out
}

func findCSSColors(content string) []cssColorMatch {
	outSpans := parser.OutputSpans(content)

	var matches []cssColorMatch

	for pos := 0; pos < len(content); {
		rel := strings.IndexByte(content[pos:], '<')
		if rel < 0 {
			break
		}

		i := pos + rel

		if strings.HasPrefix(content[i:], "<!---") {
			pos = parser.SkipCFMLComment(content, i)

			continue
		}

		if strings.HasPrefix(content[i:], "<!--") {
			if closeIdx := strings.Index(content[i+4:], "-->"); closeIdx >= 0 {
				pos = i + 4 + closeIdx + 3

				continue
			}

			break
		}

		// Skip <cfscript>...</cfscript> blocks so script code is not read as markup.
		if isOpenTagNamed(content, i, "cfscript") {
			end := findTagClose(content, i)
			if end < 0 {
				break
			}

			bodyStart := end + 1
			if closeIdx := findCloseTag(content, bodyStart, "cfscript"); closeIdx >= 0 {
				pos = closeIdx

				continue
			}

			break
		}

		// <style ...>...</style> block.
		if isOpenTagNamed(content, i, "style") {
			end := findTagClose(content, i)
			if end < 0 {
				break
			}

			bodyStart := end + 1
			bodyEnd, nextPos := findStyleBodyEnd(content, bodyStart)
			matches = scanStyleSheet(content, bodyStart, bodyEnd, outSpans, matches)
			pos = nextPos

			continue
		}

		// Any tag with an inline style="..." or style='...' attribute.
		if i+1 < len(content) && content[i+1] != '/' {
			end := findTagClose(content, i)
			if end < 0 {
				pos = i + 1

				continue
			}

			if valStart, valEnd, ok := findStyleAttrSpan(content, i, end); ok {
				matches = scanCSSDeclarations(content, valStart, valEnd, outSpans, matches)
			}

			pos = end + 1

			continue
		}

		pos = i + 1
	}

	return matches
}

func isOpenTagNamed(s string, at int, name string) bool {
	end := at + 1 + len(name)
	if end > len(s) || !strings.EqualFold(s[at+1:end], name) {
		return false
	}

	if end == len(s) {
		return true
	}

	switch s[end] {
	case '>', '/', ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func findTagClose(s string, open int) int {
	var quote byte

	for i := open + 1; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}

			continue
		}

		switch c {
		case '"', '\'':
			quote = c
		case '>':
			return i
		}
	}

	return -1
}

func findCloseTag(s string, from int, name string) int {
	needle := "</" + name

	for pos := from; pos < len(s); {
		rel := strings.IndexByte(s[pos:], '<')
		if rel < 0 {
			return -1
		}

		i := pos + rel
		if i+len(needle) <= len(s) && strings.EqualFold(s[i:i+len(needle)], needle) {
			after := i + len(needle)
			if after < len(s) && (s[after] == '>' || s[after] == ' ' || s[after] == '\t' || s[after] == '\n' || s[after] == '\r') {
				if gt := strings.IndexByte(s[after:], '>'); gt >= 0 {
					return after + gt + 1
				}

				return len(s)
			}
		}

		pos = i + 1
	}

	return -1
}

func findStyleBodyEnd(s string, from int) (bodyEnd, nextPos int) {
	for pos := from; pos < len(s); {
		rel := strings.IndexByte(s[pos:], '<')
		if rel < 0 {
			return len(s), len(s)
		}

		i := pos + rel
		if strings.HasPrefix(s[i:], "<!---") {
			pos = parser.SkipCFMLComment(s, i)

			continue
		}

		if i+7 <= len(s) && strings.EqualFold(s[i:i+7], "</style") {
			after := i + 7
			if after < len(s) && (s[after] == '>' || s[after] == ' ' || s[after] == '\t' || s[after] == '\n' || s[after] == '\r') {
				if gt := strings.IndexByte(s[after:], '>'); gt >= 0 {
					return i, after + gt + 1
				}

				return i, len(s)
			}
		}

		pos = i + 1
	}

	return len(s), len(s)
}

func findStyleAttrSpan(s string, tagStart, tagEnd int) (start, end int, ok bool) {
	i := tagStart + 1
	for i < tagEnd && !isCSSSpace(s[i]) && s[i] != '/' {
		i++
	}

	for i < tagEnd {
		for i < tagEnd && (isCSSSpace(s[i]) || s[i] == '/') {
			i++
		}

		nameStart := i
		for i < tagEnd && !isCSSSpace(s[i]) && s[i] != '=' && s[i] != '/' {
			i++
		}

		name := s[nameStart:i]

		for i < tagEnd && isCSSSpace(s[i]) {
			i++
		}

		if i >= tagEnd || s[i] != '=' {
			continue
		}

		i++
		for i < tagEnd && isCSSSpace(s[i]) {
			i++
		}

		if i >= tagEnd {
			break
		}

		var valStart, valEnd int

		if s[i] == '"' || s[i] == '\'' {
			q := s[i]
			i++
			valStart = i

			for i < tagEnd && s[i] != q {
				i++
			}

			valEnd = i
			if i < tagEnd {
				i++
			}
		} else {
			valStart = i
			for i < tagEnd && !isCSSSpace(s[i]) {
				i++
			}

			valEnd = i
		}

		if strings.EqualFold(name, "style") {
			return valStart, valEnd, true
		}
	}

	return 0, 0, false
}

// scanStyleSheet finds CSS colors inside declaration blocks ({ ... }) of a
// <style> body while skipping selectors, comments, and quoted strings.
func scanStyleSheet(content string, from, to int, outSpans [][2]int, matches []cssColorMatch) []cssColorMatch {
	depth := 0
	inValue := false

	for i := from; i < to; {
		if next, skipped := skipCSSNoise(content, i, to); skipped {
			i = next

			continue
		}

		switch content[i] {
		case '{':
			depth++
			inValue = false
			i++
		case '}':
			depth = max(depth-1, 0)
			inValue = false
			i++
		case ':':
			if depth > 0 {
				inValue = true
			}

			i++
		case ';':
			inValue = false
			i++
		default:
			if depth > 0 && inValue {
				if m, next, ok := matchCSSColorAt(content, i, to, outSpans); ok {
					matches = append(matches, m)
					i = next

					continue
				}
			}

			i++
		}
	}

	return matches
}

// scanCSSDeclarations finds CSS colors in an inline style attribute value.
func scanCSSDeclarations(content string, from, to int, outSpans [][2]int, matches []cssColorMatch) []cssColorMatch {
	inValue := false

	for i := from; i < to; {
		if next, skipped := skipCSSNoise(content, i, to); skipped {
			i = next

			continue
		}

		switch content[i] {
		case ':':
			inValue = true
			i++
		case ';':
			inValue = false
			i++
		default:
			if inValue {
				if m, next, ok := matchCSSColorAt(content, i, to, outSpans); ok {
					matches = append(matches, m)
					i = next

					continue
				}
			}

			i++
		}
	}

	return matches
}

func skipCSSNoise(content string, i, to int) (int, bool) {
	if i+5 <= to && content[i:i+5] == "<!---" {
		end := parser.SkipCFMLComment(content, i)

		return min(end, to), true
	}

	if i+2 <= to && content[i:i+2] == "/*" {
		if closeIdx := strings.Index(content[i+2:to], "*/"); closeIdx >= 0 {
			return i + 2 + closeIdx + 2, true
		}

		return to, true
	}

	if content[i] == '"' || content[i] == '\'' {
		q := content[i]
		j := i + 1

		for j < to {
			if content[j] == '\\' && j+1 < to {
				j += 2

				continue
			}

			if content[j] == q {
				return j + 1, true
			}

			j++
		}

		return to, true
	}

	return i, false
}

func matchCSSColorAt(content string, i, to int, outSpans [][2]int) (cssColorMatch, int, bool) {
	if content[i] == '#' {
		return matchHexColorAt(content, i, to, inOutputSpans(outSpans, i))
	}

	if i > 0 && (lex.IsWord(content[i-1]) || content[i-1] == '-') {
		return cssColorMatch{}, i, false
	}

	switch lex.LowerASCII(content[i]) {
	case 'r':
		return matchFuncColorAt(content, i, to, "rgb", parseRGBArgs)
	case 'h':
		return matchFuncColorAt(content, i, to, "hsl", parseHSLArgs)
	default:
		return cssColorMatch{}, i, false
	}
}

func matchHexColorAt(content string, i, to int, inOutput bool) (cssColorMatch, int, bool) {
	if i > 0 && (lex.IsWord(content[i-1]) || content[i-1] == '#') {
		return cssColorMatch{}, i, false
	}

	digitsStart := i + 1

	if inOutput {
		if i+1 >= to || content[i+1] != '#' {
			return cssColorMatch{}, i, false
		}

		digitsStart = i + 2
	} else if i+1 < to && content[i+1] == '#' {
		return cssColorMatch{}, i, false
	}

	j := digitsStart
	for j < to && isHexDigit(content[j]) {
		j++
	}

	if j < to && (lex.IsWord(content[j]) || content[j] == '#') {
		return cssColorMatch{}, j, false
	}

	color, ok := parseHexColor(content[digitsStart:j])
	if !ok {
		return cssColorMatch{}, i, false
	}

	return cssColorMatch{start: i, end: j, color: color}, j, true
}

func matchFuncColorAt(content string, i, to int, base string, parse func(string) (protocol.Color, bool)) (cssColorMatch, int, bool) {
	var open int

	switch {
	case i+len(base)+1 < to && strings.EqualFold(content[i:i+len(base)], base) && content[i+len(base)] == '(':
		open = i + len(base)
	case i+len(base)+2 < to && strings.EqualFold(content[i:i+len(base)], base) && lex.LowerASCII(content[i+len(base)]) == 'a' && content[i+len(base)+1] == '(':
		open = i + len(base) + 1
	default:
		return cssColorMatch{}, i, false
	}

	closeIdx := strings.IndexByte(content[open+1:to], ')')
	if closeIdx < 0 {
		return cssColorMatch{}, i, false
	}

	end := open + 1 + closeIdx + 1

	color, ok := parse(content[open+1 : open+1+closeIdx])
	if !ok {
		return cssColorMatch{}, i, false
	}

	return cssColorMatch{start: i, end: end, color: color}, end, true
}

func parseHexColor(hex string) (protocol.Color, bool) {
	switch len(hex) {
	case 3:
		r, g, b := hexVal(hex[0])*17, hexVal(hex[1])*17, hexVal(hex[2])*17

		return rgbaColor(float64(r)/255, float64(g)/255, float64(b)/255, 1), true
	case 4:
		r, g, b, a := hexVal(hex[0])*17, hexVal(hex[1])*17, hexVal(hex[2])*17, hexVal(hex[3])*17

		return rgbaColor(float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255), true
	case 6:
		r := hexVal(hex[0])<<4 | hexVal(hex[1])
		g := hexVal(hex[2])<<4 | hexVal(hex[3])
		b := hexVal(hex[4])<<4 | hexVal(hex[5])

		return rgbaColor(float64(r)/255, float64(g)/255, float64(b)/255, 1), true
	case 8:
		r := hexVal(hex[0])<<4 | hexVal(hex[1])
		g := hexVal(hex[2])<<4 | hexVal(hex[3])
		b := hexVal(hex[4])<<4 | hexVal(hex[5])
		a := hexVal(hex[6])<<4 | hexVal(hex[7])

		return rgbaColor(float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255), true
	default:
		return protocol.Color{}, false
	}
}

func parseRGBArgs(raw string) (protocol.Color, bool) {
	parts, alpha, ok := splitCSSColorArgs(raw)
	if !ok {
		return protocol.Color{}, false
	}

	r, ok := parseRGBChannel(parts[0])
	if !ok {
		return protocol.Color{}, false
	}

	g, ok := parseRGBChannel(parts[1])
	if !ok {
		return protocol.Color{}, false
	}

	b, ok := parseRGBChannel(parts[2])
	if !ok {
		return protocol.Color{}, false
	}

	a := 1.0
	if alpha != "" {
		a, ok = parseAlpha(alpha)
		if !ok {
			return protocol.Color{}, false
		}
	}

	return rgbaColor(r, g, b, a), true
}

func parseHSLArgs(raw string) (protocol.Color, bool) {
	parts, alpha, ok := splitCSSColorArgs(raw)
	if !ok {
		return protocol.Color{}, false
	}

	h, ok := parseHue(parts[0])
	if !ok {
		return protocol.Color{}, false
	}

	s, ok := parsePercent(parts[1])
	if !ok {
		return protocol.Color{}, false
	}

	l, ok := parsePercent(parts[2])
	if !ok {
		return protocol.Color{}, false
	}

	a := 1.0
	if alpha != "" {
		a, ok = parseAlpha(alpha)
		if !ok {
			return protocol.Color{}, false
		}
	}

	r, g, b := hslToRGB(h, s, l)

	return rgbaColor(r, g, b, a), true
}

// splitCSSColorArgs handles both legacy comma syntax (a, b, c[, alpha]) and
// CSS Color 4 space/slash syntax (a b c [/ alpha]).
func splitCSSColorArgs(raw string) (parts [3]string, alpha string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return parts, "", false
	}

	if strings.Contains(raw, ",") {
		if strings.Contains(raw, "/") {
			return parts, "", false
		}

		items := strings.Split(raw, ",")
		if len(items) != 3 && len(items) != 4 {
			return parts, "", false
		}

		for i := range 3 {
			parts[i] = strings.TrimSpace(items[i])
		}

		if len(items) == 4 {
			alpha = strings.TrimSpace(items[3])
		}

		return parts, alpha, true
	}

	mainPart, alphaPart, hasSlash := strings.Cut(raw, "/")
	if hasSlash {
		alpha = strings.TrimSpace(alphaPart)
		if alpha == "" || strings.Contains(alpha, "/") {
			return parts, "", false
		}
	}

	fields := strings.Fields(mainPart)
	if len(fields) != 3 {
		return parts, "", false
	}

	copy(parts[:], fields)

	return parts, alpha, true
}

func parseRGBChannel(s string) (float64, bool) {
	if strings.HasSuffix(s, "%") {
		return parsePercent(s)
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}

	return clamp01(v / 255), true
}

func parseHue(s string) (float64, bool) {
	lower := strings.ToLower(s)

	var scale float64 = 1

	switch {
	case strings.HasSuffix(lower, "deg"):
		lower = lower[:len(lower)-3]
	case strings.HasSuffix(lower, "grad"):
		lower = lower[:len(lower)-4]
		scale = 0.9
	case strings.HasSuffix(lower, "rad"):
		lower = lower[:len(lower)-3]
		scale = 180 / math.Pi
	case strings.HasSuffix(lower, "turn"):
		lower = lower[:len(lower)-4]
		scale = 360
	}

	v, err := strconv.ParseFloat(lower, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}

	deg := math.Mod(v*scale, 360)
	if deg < 0 {
		deg += 360
	}

	return deg, true
}

func parsePercent(s string) (float64, bool) {
	num, ok := strings.CutSuffix(s, "%")
	if !ok {
		return 0, false
	}

	v, err := strconv.ParseFloat(num, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}

	return clamp01(v / 100), true
}

func parseAlpha(s string) (float64, bool) {
	if strings.HasSuffix(s, "%") {
		return parsePercent(s)
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}

	return clamp01(v), true
}

func hslToRGB(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		return l, l, l
	}

	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2

	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return clamp01(r + m), clamp01(g + m), clamp01(b + m)
}

func rgbToHSL(r, g, b float64) (h, s, l int) {
	maxC := max(r, max(g, b))
	minC := min(r, min(g, b))
	delta := maxC - minC
	lightness := (maxC + minC) / 2

	var hue, sat float64
	if delta > 0 {
		sat = delta / (1 - math.Abs(2*lightness-1))

		switch maxC {
		case r:
			hue = 60 * math.Mod((g-b)/delta, 6)
		case g:
			hue = 60 * ((b-r)/delta + 2)
		default:
			hue = 60 * ((r-g)/delta + 4)
		}

		if hue < 0 {
			hue += 360
		}
	}

	return int(math.Round(hue)) % 360, int(math.Round(sat * 100)), int(math.Round(lightness * 100))
}

func buildColorPresentations(c protocol.Color, rng protocol.Range, inOutput bool) []protocol.ColorPresentation {
	r := channelByte(c.Red)
	g := channelByte(c.Green)
	b := channelByte(c.Blue)
	a := channelByte(c.Alpha)

	hash := "#"
	if inOutput {
		hash = "##"
	}

	var hexLabel, rgbLabel, hslLabel string

	h, s, l := rgbToHSL(clamp01(c.Red), clamp01(c.Green), clamp01(c.Blue))

	if a == 255 {
		hexLabel = fmt.Sprintf("%s%02x%02x%02x", hash, r, g, b)
		rgbLabel = fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
		hslLabel = fmt.Sprintf("hsl(%d, %d%%, %d%%)", h, s, l)
	} else {
		alphaText := formatColorAlpha(clamp01(c.Alpha))
		hexLabel = fmt.Sprintf("%s%02x%02x%02x%02x", hash, r, g, b, a)
		rgbLabel = fmt.Sprintf("rgba(%d, %d, %d, %s)", r, g, b, alphaText)
		hslLabel = fmt.Sprintf("hsla(%d, %d%%, %d%%, %s)", h, s, l, alphaText)
	}

	labels := []string{hexLabel, rgbLabel, hslLabel}
	out := make([]protocol.ColorPresentation, 0, len(labels))

	for _, label := range labels {
		out = append(out, protocol.ColorPresentation{
			Label:    label,
			TextEdit: protocol.TextEdit{Range: rng, NewText: label},
		})
	}

	return out
}

func formatColorAlpha(a float64) string {
	s := strconv.FormatFloat(math.Round(a*100)/100, 'f', 2, 64)
	s = strings.TrimRight(s, "0")

	return strings.TrimRight(s, ".")
}

func channelByte(v float64) int {
	return int(math.Round(clamp01(v) * 255))
}

func rgbaColor(r, g, b, a float64) protocol.Color {
	return protocol.Color{Red: r, Green: g, Blue: b, Alpha: a}
}

func clamp01(v float64) float64 {
	return min(max(v, 0), 1)
}

func isHexDigit(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	default:
		return int(c-'A') + 10
	}
}

func isCSSSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func inOutputSpans(spans [][2]int, offset int) bool {
	for _, span := range spans {
		if offset < span[0] {
			return false
		}

		if offset < span[1] {
			return true
		}
	}

	return false
}
