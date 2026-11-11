package parser

import (
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// MappingSources reads statically knowable application mappings in execution
// order, including literal includes and parent constructors. It never runs CFML.
func MappingSources(file string, initial map[string]string, read func(string) ([]byte, error)) map[string]string {
	return MappingSourcesWithParent(file, initial, read, nil)
}

// MappingSourcesWithParent accepts a bootstrap path lookup that must not depend
// on the mappings being computed. A nil lookup retains declaring-relative parents.
func MappingSourcesWithParent(file string, initial map[string]string, read func(string) ([]byte, error), parent func(string, string) string) map[string]string {
	return ApplicationSources(file, initial, read, parent).Mappings
}

// AppSources is what an Application file's constructor states that a lookup
// needs: its mappings, and the directories this.customTagPaths names, each
// absolute. Both come from one walk of the file (ApplicationSources).
type AppSources struct {
	Mappings       map[string]string
	CustomTagPaths []string
}

// ApplicationSources reads an Application file's mappings and custom tag paths
// in execution order, through literal includes and the parent Application.cfc,
// as MappingSourcesWithParent describes, in one walk: a caller that wants both
// does not read the file twice.
func ApplicationSources(file string, initial map[string]string, read func(string) ([]byte, error), parent func(string, string) string) AppSources {
	s := &mappingSourceState{mappings: map[string]string{}, defaults: initial, env: map[string]string{}, seen: map[string]bool{}, read: read, base: file, parent: parent}
	s.file(file, 0)

	if source := s.requestMethod; source != nil {
		if body, proven := requestMappingBody(source.content); proven {
			s.content(body, source.file, 0)
		}
	}

	return AppSources{Mappings: s.mappings, CustomTagPaths: s.customTags}
}

type mappingSourceState struct {
	mappings, defaults, env map[string]string
	seen                    map[string]bool
	read                    func(string) ([]byte, error)
	base                    string
	parent                  func(string, string) string
	requestMethod           *mappingRequestSource
	// customTags is this.customTagPaths, each directory absolute; see
	// ApplicationSources.
	customTags []string
}

type mappingRequestSource struct{ file, content string }

func mappingParentName(content string) string {
	sc := NewSourceScanner(content)
	for {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF {
			return ""
		}

		if t.Kind != TokIdent || !identEq(t.Value, "component") && !identEq(t.Value, "cfcomponent") {
			continue
		}

		for {
			t = sc.NextSkipComments()
			if t.Kind == TokEOF || t.Kind == TokGT || t.Kind == TokLBrace {
				return ""
			}

			if t.Kind == TokIdent && identEq(t.Value, "extends") && sc.NextSkipComments().Kind == TokEquals {
				t = sc.NextSkipComments()
				if t.Kind == TokString {
					value, ok := stringLiteral(t.Value)
					if ok && !strings.Contains(value, "#") {
						return value
					}
				}

				if t.Kind == TokIdent {
					return dottedRest(sc, t.Value)
				}
			}
		}
	}
}

func (s *mappingSourceState) file(file string, depth int) {
	file = filepath.Clean(file)
	if depth >= 16 || len(s.seen) >= 64 || s.seen[file] {
		return
	}

	s.seen[file] = true

	data, err := s.read(file)
	if err != nil {
		return
	}

	content := string(data)
	if strings.EqualFold(filepath.Ext(file), ".cfc") {
		if parent := mappingParentName(content); parent != "" {
			path := relativeMappingParent(parent, file)

			if s.parent != nil {
				path = s.parent(parent, filepath.Dir(file))
			}

			if path != "" {
				s.file(path, depth+1)
			}
		}
	}

	s.content(content, file, depth)

	if strings.EqualFold(filepath.Ext(file), ".cfc") {
		if method, ok := ApplicationMethod(content, "onrequest"); ok {
			s.requestMethod = &mappingRequestSource{file, method}
		}
	}
}

func relativeMappingParent(parent, file string) string {
	relative := strings.ReplaceAll(parent, "\\", "/")
	if strings.HasPrefix(relative, "./") || strings.HasPrefix(relative, "../") {
		if !strings.HasSuffix(strings.ToLower(relative), ".cfc") {
			relative += ".cfc"
		}

		return filepath.Join(filepath.Dir(file), filepath.FromSlash(relative))
	}

	return filepath.Join(filepath.Dir(file), strings.ReplaceAll(parent, ".", string(filepath.Separator))+".cfc")
}

func (s *mappingSourceState) content(content, file string, depth int) {
	s.env["@template"] = file
	s.env["@baseTemplate"] = s.base
	sc := NewSourceScanner(content)
	previous := TokEOF

	var includes []IncludeSite
	if s.read != nil {
		includes = literalIncludeSites(content)
	}

	// skipTo drops the includes the walk stepped over without running: those
	// in a method's body, which a declaration does not execute.
	skipTo := func(offset int) {
		for len(includes) > 0 && includes[0].Offset < offset {
			includes = includes[1:]
		}
	}

	for {
		t := sc.NextSkipComments()
		before := previous

		previous = t.Kind
		if t.Kind == TokEOF {
			return
		}

		// An include runs where it stands, between the assignments before and
		// after it, so each is followed once the walk reaches it.
		for len(includes) > 0 && includes[0].Offset <= t.Offset {
			s.file(s.includePath(includes[0].Path, file), depth+1)
			s.env["@template"] = file
			includes = includes[1:]
		}

		if t.Kind != TokIdent || before == TokDot {
			continue
		}

		// Declaring a method does not execute its mapping assignments or includes.
		// Constructor extraction deliberately excludes even lifecycle methods.
		if identEq(t.Value, "function") {
			skipMappingFunction(sc)
			skipTo(sc.Pos())

			continue
		}

		if identEq(t.Value, "cffunction") {
			for {
				end := sc.NextSkipComments()
				if end.Kind == TokEOF {
					break
				}

				if end.Kind == TokSlash && identEq(sc.PeekSkipComments().Value, "cffunction") {
					sc.NextSkipComments()

					break
				}
			}

			skipTo(sc.Pos())

			continue
		}

		if identEq(t.Value, "component") || identEq(t.Value, "cfcomponent") {
			for {
				header := sc.NextSkipComments()
				if header.Kind == TokLBrace || header.Kind == TokGT || header.Kind == TokEOF {
					break
				}
			}

			continue
		}

		cursor := *sc
		cursor.Restore(ScannerState{pos: t.Offset, line: int(t.Line)})

		target, key, ok := mappingSourceTarget(&cursor)
		if !ok || cursor.NextSkipComments().Kind != TokEquals || cursor.PeekSkipComments().Kind == TokEquals {
			continue
		}

		expression := mappingSourceExpression(&cursor)
		*sc = cursor

		s.assignment(target, key, expression, filepath.Dir(file))
	}
}

// literalIncludeSites is the includes of content that name one template
// literally, in source order. They are the document walk's (IncludeSites), so
// an include in a comment, a string or a page's prose is none. A computed
// path, which IncludeSites reads as a glob or keeps as `#attributes.x#`, names
// no one file the walk could read.
func literalIncludeSites(content string) []IncludeSite {
	sites := IncludeSites(content)

	out := sites[:0]
	for _, site := range sites {
		if !strings.ContainsAny(site.Path, "#*") {
			out = append(out, site)
		}
	}

	slices.SortStableFunc(out, func(a, b IncludeSite) int { return a.Offset - b.Offset })

	return out
}

func skipMappingFunction(sc *Scanner) {
	paren, depth := 0, 0

	for {
		t := sc.NextSkipComments()
		switch t.Kind {
		case TokEOF:
			return
		case TokLParen:
			paren++
		case TokRParen:
			paren--
		case TokLBrace:
			if paren == 0 {
				depth++
			}
		case TokRBrace:
			if paren == 0 {
				depth--
				if depth == 0 {
					return
				}
			}
		default:
		}
	}
}

func mappingSourceTarget(sc *Scanner) (target, key string, ok bool) {
	t := sc.NextSkipComments()
	if t.Kind != TokIdent {
		return "", "", false
	}

	if identEq(t.Value, "var") {
		t = sc.NextSkipComments()
		if t.Kind != TokIdent {
			return "", "", false
		}
	}

	var path chainBuilder
	path.reset(t.Value)

	for sc.PeekSkipComments().Kind == TokDot {
		sc.NextSkipComments()

		t = sc.NextSkipComments()
		if t.Kind != TokIdent {
			return "", "", false
		}

		path.writeDot()
		path.writeString(t.Value)
	}

	target = path.String()

	if sc.PeekSkipComments().Kind == TokLBracket {
		sc.NextSkipComments()

		t = sc.NextSkipComments()
		if t.Kind != TokString {
			return "", "", false
		}

		key = strings.Trim(t.Value, `"'`)
		if strings.Contains(key, "#") || sc.NextSkipComments().Kind != TokRBracket {
			return "", "", false
		}
	}

	return target, key, true
}

func mappingSourceExpression(sc *Scanner) string {
	start := sc.PeekSkipComments().Offset
	depth, lastLine := 0, sc.PeekSkipComments().Line
	previous := TokEOF

	for {
		t := sc.PeekSkipComments()
		if t.Kind == TokEOF || depth == 0 && (t.Kind == TokSemicolon || t.Kind == TokGT || t.Kind == TokRBrace || t.Line > lastLine && previous != TokAmpersand) {
			return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(sc.src[start:t.Offset]), "/"))
		}

		t = sc.NextSkipComments()
		lastLine = t.Line

		previous = t.Kind
		switch t.Kind {
		case TokLBrace, TokLBracket, TokLParen:
			depth++
		case TokRBrace, TokRBracket, TokRParen:
			depth--
		default:
		}
	}
}

func (s *mappingSourceState) assignment(target, key, expr, dir string) {
	if strings.EqualFold(target, "this.customTagPaths") && key == "" || strings.EqualFold(target, "this") && strings.EqualFold(key, "customTagPaths") {
		s.customTagPaths(expr, dir)

		return
	}

	// Lucee also takes a struct, keyed like this.mappings (its LDEV-1276
	// test): each key adds one directory.
	if strings.EqualFold(target, "this.customTagPaths") {
		if v, ok := evalPathExpr(expr, s.env, dir); ok && strings.TrimSpace(v) != "" {
			s.customTags = append(s.customTags, cleanMappingPath(strings.TrimSpace(v), dir))
			s.env[customTagPathsVar] = strings.Trim(s.env[customTagPathsVar]+","+strings.TrimSpace(v), ",")
		}

		return
	}

	if strings.EqualFold(target, "this.mappings") && key == "" {
		entries, _ := mappingLiteral(expr)
		// Evaluate the whole RHS against the old struct, before replacing it.
		// Iteration order cannot make a sibling literal key visible prematurely.
		values := map[string]string{}
		unknownRoot := false

		for k, expression := range entries {
			if value, ok := evalPathExpr(expression, s.env, dir); ok && k != "" {
				values[strings.ToLower(strings.Trim(k, "/"))] = value
			} else if k == "/" {
				unknownRoot = true
			}
		}

		for name := range s.env {
			if strings.HasPrefix(name, "mapping:") {
				delete(s.env, name)
			}
		}

		s.mappings = map[string]string{}

		for name, value := range values {
			s.mappings[name] = cleanMappingPath(value, dir)
			s.env["mapping:"+name] = value
		}

		if unknownRoot {
			s.mappings[""] = ""
		}

		return
	}

	if strings.EqualFold(target, "this.mappings") {
		s.mapping(key, expr, dir)

		return
	}

	if lex.HasPrefixFold(target, "this.mappings.") {
		s.mapping(target[len("this.mappings."):], expr, dir)

		return
	}

	if key != "" {
		return
	}

	value, ok := evalPathExpr(expr, s.env, dir)
	if ok {
		s.env[envName(target)] = value
	} else {
		delete(s.env, envName(target))
	}
}

func (s *mappingSourceState) mapping(key, expr, dir string) {
	if key == "" {
		return
	}

	key = strings.ToLower(strings.Trim(key, "/"))

	value, ok := evalPathExpr(expr, s.env, dir)
	if !ok {
		delete(s.mappings, key)

		if _, fallback := s.defaults[key]; fallback || key == "" {
			s.mappings[key] = ""
		}

		delete(s.env, "mapping:"+strings.ToLower(key))

		return
	}

	s.mappings[key] = cleanMappingPath(value, dir)
	s.env["mapping:"+strings.ToLower(key)] = value
}

func mappingLiteral(expr string) (map[string]string, bool) {
	if emptyCollection(expr) {
		return map[string]string{}, true
	}

	sc := NewScanner(expr)
	if sc.NextSkipComments().Kind != TokLBrace {
		return nil, false
	}

	out := map[string]string{}

	for sc.PeekSkipComments().Kind != TokRBrace {
		key := sc.NextSkipComments()
		if key.Kind != TokIdent && key.Kind != TokString {
			return nil, false
		}

		separator := sc.NextSkipComments()
		if separator.Kind != TokColon && separator.Kind != TokEquals {
			return nil, false
		}

		start := sc.PeekSkipComments().Offset
		depth := 0

		for {
			t := sc.PeekSkipComments()
			if t.Kind == TokEOF {
				return nil, false
			}

			if depth == 0 && (t.Kind == TokComma || t.Kind == TokRBrace) {
				out[strings.Trim(key.Value, `"'`)] = strings.TrimSpace(expr[start:t.Offset])

				break
			}

			sc.NextSkipComments()

			switch t.Kind {
			case TokLParen, TokLBrace, TokLBracket:
				depth++
			case TokRParen, TokRBrace, TokRBracket:
				depth--
			default:
			}
		}

		if sc.PeekSkipComments().Kind == TokComma {
			sc.NextSkipComments()
		}
	}

	sc.NextSkipComments()

	return out, sc.NextSkipComments().Kind == TokEOF
}

func (s *mappingSourceState) includePath(raw, file string) string {
	if !strings.HasPrefix(raw, "/") {
		return filepath.Join(filepath.Dir(file), filepath.FromSlash(raw))
	}

	for prefix := strings.TrimPrefix(raw, "/"); prefix != ""; {
		root, ok := s.mappings[strings.ToLower(prefix)]
		if !ok {
			root, ok = s.defaults[strings.ToLower(prefix)]
		}

		if ok {
			return filepath.Join(root, strings.TrimPrefix(strings.TrimPrefix(raw, "/"), prefix))
		}

		index := strings.LastIndexByte(prefix, '/')
		if index < 0 {
			break
		}

		prefix = prefix[:index]
	}

	return filepath.Join(filepath.Dir(s.base), filepath.FromSlash(strings.TrimPrefix(raw, "/")))
}

// Path slicing is common in shared application templates. Only literal length
// arithmetic on an already known path is accepted.
func slicedMappingPath(term string, env map[string]string, dir string) (string, bool) {
	for _, fn := range []string{"left", "right"} {
		args, ok := callArg(term, fn)
		if !ok {
			continue
		}

		parts := splitPathArguments(args)
		if len(parts) != 2 {
			return "", false
		}

		value, ok := evalPathExpr(parts[0], env, dir)
		if !ok {
			return "", false
		}

		n, ok := mappingLength(parts[1], env, dir)

		runes := []rune(value)
		if !ok || n < 0 || n > len(runes) {
			return "", false
		}

		if fn == "left" {
			return string(runes[:n]), true
		}

		return string(runes[len(runes)-n:]), true
	}

	return "", false
}

func mappingLength(expr string, env map[string]string, dir string) (int, bool) {
	expr = strings.TrimSpace(expr)
	if n, err := strconv.Atoi(expr); err == nil {
		return n, true
	}

	index := strings.LastIndexAny(expr, "+-")
	delta := 0

	if index >= 0 {
		n, err := strconv.Atoi(strings.TrimSpace(expr[index+1:]))
		if err != nil {
			return 0, false
		}

		delta = n
		if expr[index] == '-' {
			delta = -n
		}

		expr = strings.TrimSpace(expr[:index])
	}

	arg, ok := callArg(expr, "len")
	if !ok {
		return 0, false
	}

	value, ok := evalPathExpr(arg, env, dir)

	return len([]rune(value)) + delta, ok
}

func splitPathArguments(args string) []string {
	sc := NewScanner(args)
	start, depth := 0, 0

	var out []string

	for {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF {
			return append(out, args[start:])
		}

		if t.Kind == TokComma && depth == 0 {
			out = append(out, args[start:t.Offset])
			start = t.Offset + 1
		}

		switch t.Kind {
		case TokLParen:
			depth++
		case TokRParen:
			depth--
		default:
		}
	}
}

// replacedMappingPath evaluates reReplace and reReplaceNoCase of a path by a
// literal pattern: ColdBox's test Application.cfc finds its root as
// `REReplaceNoCase( this.mappings[ "/tests" ], "tests(\\|/)", "" )`. The
// pattern must compile as Go reads it, the replacement must be a literal
// with no backreference and the scope, when given, "one" or "all"; anything
// else is declined.
func replacedMappingPath(term string, env map[string]string, dir string) (string, bool) {
	for _, fn := range []string{"reReplaceNoCase", "reReplace"} {
		args, ok := callArg(term, fn)
		if !ok {
			continue
		}

		parts := splitPathArguments(args)
		if len(parts) != 3 && len(parts) != 4 {
			return "", false
		}

		value, ok := evalPathExpr(parts[0], env, dir)
		if !ok {
			return "", false
		}

		pattern, ok := stringLiteral(strings.TrimSpace(parts[1]))
		if !ok || pattern == "" {
			return "", false
		}

		replacement, ok := stringLiteral(strings.TrimSpace(parts[2]))
		if !ok || strings.ContainsAny(replacement, "\\$") {
			return "", false
		}

		all := false

		if len(parts) == 4 {
			scope, ok := stringLiteral(strings.TrimSpace(parts[3]))
			if !ok || !strings.EqualFold(scope, "one") && !strings.EqualFold(scope, "all") {
				return "", false
			}

			all = strings.EqualFold(scope, "all")
		}

		if fn == "reReplaceNoCase" {
			pattern = "(?i)" + pattern
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", false
		}

		if all {
			return re.ReplaceAllLiteralString(value, replacement), true
		}

		loc := re.FindStringIndex(value)
		if loc == nil {
			return value, true
		}

		return value[:loc[0]] + replacement + value[loc[1]:], true
	}

	return "", false
}

// customTagPathsVar is where the walk keeps this.customTagPaths' value, as a
// list, so a later `this.customTagPaths & ",x"` or listAppend reads it.
const customTagPathsVar = "this.customtagpaths"

// customTagPaths reads an assignment to this.customTagPaths: a list (ColdFusion
// takes a comma-delimited string), an array literal (Lucee takes either), or
// listAppend( this.customTagPaths, dir ), which is how Mura adds its own. An
// element that is not a path the source states is left out; a whole value that
// is not clears what was known, as a mapping does.
func (s *mappingSourceState) customTagPaths(expr, dir string) {
	var raw []string

	switch {
	case strings.HasPrefix(strings.TrimSpace(expr), "["):
		elems, ok := arrayElements(expr)
		if !ok {
			break
		}

		for _, e := range elems {
			// An array element is one directory: only a list is split.
			if v, ok := evalPathExpr(e, s.env, dir); ok && strings.TrimSpace(v) != "" {
				raw = append(raw, strings.TrimSpace(v))
			}
		}
	default:
		if args, ok := callArg(strings.TrimSpace(expr), "listAppend"); ok {
			parts := splitTopLevel(args, ',')
			if len(parts) >= 2 && envName(parts[0]) == customTagPathsVar {
				raw = append(raw, splitPathList(s.env[customTagPathsVar])...)

				if v, ok := evalPathExpr(parts[1], s.env, dir); ok {
					raw = append(raw, splitPathList(v)...)
				}

				break
			}
		}

		if v, ok := evalPathExpr(expr, s.env, dir); ok {
			raw = splitPathList(v)
		}
	}

	s.env[customTagPathsVar] = strings.Join(raw, ",")
	s.customTags = s.customTags[:0]

	for _, p := range raw {
		s.customTags = append(s.customTags, cleanMappingPath(p, dir))
	}
}

// splitPathList is a comma-delimited list of paths, each trimmed, empty ones
// left out.
func splitPathList(v string) []string {
	var out []string

	for p := range strings.SplitSeq(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return out
}

// arrayElements is the element expressions of an array literal, alone.
func arrayElements(expr string) ([]string, bool) {
	expr = strings.TrimSpace(expr)
	if len(expr) < 2 || expr[0] != '[' || expr[len(expr)-1] != ']' {
		return nil, false
	}

	body := strings.TrimSpace(expr[1 : len(expr)-1])
	if body == "" {
		return nil, true
	}

	return splitTopLevel(body, ','), true
}

// splitTopLevel splits expr at sep outside strings and brackets.
func splitTopLevel(expr string, sep byte) []string {
	var (
		parts []string
		depth int
		quote byte
		start int
	)

	for i := range len(expr) {
		c := expr[i]

		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			depth--
		case c == sep && depth == 0:
			parts = append(parts, strings.TrimSpace(expr[start:i]))
			start = i + 1
		}
	}

	return append(parts, strings.TrimSpace(expr[start:]))
}
