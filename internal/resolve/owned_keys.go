package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// A factory may cache what it builds in a shared scope under a key it
// computes, and hand back whatever the key holds:
//
//	var key = "_presideObjectService" & Hash( SerializeJson( arguments ) );
//	if ( arguments.forceNewInstance || !request.keyExists( key ) ) {
//		request[ key ] = new preside.system.services.presideObjects.PresideObjectService( … );
//		request[ key ] = getMockbox().createMock( object=request[ key ] );
//	}
//	return request[ key ];
//
// The key starts with a string the file spells nowhere else, so only this
// function writes it, and whatever is present was stored by one of its
// writes. Another file spelling the same string must hold a factory storing
// the same component. When every write stores the same
// component (or a MockBox mock of what is already there), that is the
// return, whichever way the guard went. Preside's test helper builds its
// PresideObjectService this way, and 849 calls in PresideObjectServiceTest
// were on a variable it returned.
//
// It is a batch-scan rule: whether another file spells the prefix needs the
// workspace's file list. A mention of the key variable that could hand it
// elsewhere (an argument other than a presence test or delete on the scope)
// withholds it.

// ownedKeyScopes are the shared scopes a factory caches in. variables is
// left out: a component's own generic setters write it under keys they are
// handed.
var ownedKeyScopes = map[string]bool{"request": true, "session": true, "application": true, "server": true}

// ownedKeyReturn is the component fd returns when its every return is
// `scope[ key ]` for a key only it writes, as above, or "".
func (r *Resolver) ownedKeyReturn(fd *parser.FunctionDef) string {
	if fd == nil || !fd.URI.IsFile() || len(r.owner().InferArgsFiles) == 0 {
		return ""
	}

	o := r.owner()
	key := fd.URI.Path() + "\t" + strings.ToLower(fd.Name)

	o.mu.RLock()
	component, known := o.ownedKeys[key]
	o.mu.RUnlock()

	if known {
		return component
	}

	component, prefix := r.ownedKeyFactory(fd)
	if component != "" && !strings.EqualFold(r.prefixComponent(prefix), component) {
		component = ""
	}

	o.mu.Lock()
	if o.ownedKeys == nil {
		o.ownedKeys = map[string]string{}
	}

	o.ownedKeys[key] = component
	o.mu.Unlock()

	return component
}

// ownedKeyFactory is what fd stores under its key and the key's literal
// prefix, lowercased, when fd has the shape above and its file spells the
// prefix in a string once; "" otherwise.
func (r *Resolver) ownedKeyFactory(fd *parser.FunctionDef) (component, prefix string) {
	file := fd.URI.Path()
	source := r.sourceInfo(file)

	body := functionSource(&source, fd.Name)
	if body == "" {
		return "", ""
	}

	tokens := allTokens(body)

	key, scope, ok := ownedKeyShape(tokens)
	if !ok {
		return "", ""
	}

	prefix, ok = ownedKeyPrefix(tokens, key)
	if !ok || quotedCount(source.content, prefix) != 1 {
		return "", ""
	}

	writes := 0

	for i := 0; i+4 < len(tokens); i++ {
		if !isScopeKey(tokens, i, scope, key) || !isAssignment(tokens, i+4) {
			continue
		}

		value := tokens[i+5 : ownedStatementEnd(tokens, i+5)]
		writes++

		if isSelfMock(value, scope, key) {
			continue
		}

		c := r.storedComponent(fd, value)
		if c == "" || strings.HasPrefix(c, "$") {
			return "", ""
		}

		if p := r.ComponentPath(c, filepath.Dir(file)); p != "" {
			c = p
		}

		if component != "" && !strings.EqualFold(c, component) {
			return "", ""
		}

		component = c
	}

	if writes == 0 {
		return "", ""
	}

	return component, prefix
}

// prefixComponent is the one component every file of the batch spelling the
// string prefix stores under keys built from it, or "": each must spell it
// once, in a factory of the shape above. Preside's two test base classes
// each hold the same PresideObjectService factory, and a TestBox run may
// load both in one request.
func (r *Resolver) prefixComponent(prefix string) string {
	o := r.owner()

	o.mu.RLock()
	component, known := o.keyPrefixes[prefix]
	o.mu.RUnlock()

	if known {
		return component
	}

	component = ""

	for _, path := range o.InferArgsFiles {
		text, err := r.readText(path)
		if err != nil || quotedCount(text, prefix) == 0 {
			continue
		}

		c := r.prefixFactoryIn(path, prefix)
		if c == "" || component != "" && !strings.EqualFold(c, component) {
			component = ""

			break
		}

		component = c
	}

	o.mu.Lock()
	if o.keyPrefixes == nil {
		o.keyPrefixes = map[string]string{}
	}

	o.keyPrefixes[prefix] = component
	o.mu.Unlock()

	return component
}

// prefixFactoryIn is what the factory in file whose body spells the string
// prefix stores, or "".
func (r *Resolver) prefixFactoryIn(file, prefix string) string {
	source := r.sourceInfo(file)
	names := make([]string, 0, len(source.methodMap()))

	for name := range source.methodMap() {
		names = append(names, name)
	}

	for _, fn := range tagFunctionNames(source.content) {
		names = append(names, fn.name)
	}

	for _, name := range names {
		if quotedCount(functionSource(&source, name), prefix) == 0 {
			continue
		}

		fd := r.funcInFile(file, name)
		if fd == nil {
			return ""
		}

		component, p := r.ownedKeyFactory(fd)
		if p != prefix {
			return ""
		}

		return component
	}

	return ""
}

// quotedCount counts the strings in text that start with prefix.
func quotedCount(text, prefix string) int {
	lower := strings.ToLower(text)

	return strings.Count(lower, `"`+prefix) + strings.Count(lower, `'`+prefix)
}

// storedComponent is the one component value evaluates to: a `new` of a
// written path, or what the producer makes of it.
func (r *Resolver) storedComponent(fd *parser.FunctionDef, value []parser.Token) string {
	if len(value) > 2 && strings.EqualFold(value[0].Value, "new") && value[len(value)-1].Kind == parser.TokRParen {
		if open := slices.IndexFunc(value, func(t parser.Token) bool { return t.Kind == parser.TokLParen }); open > 1 && producerGroupEnd(value, open, parser.TokLParen, parser.TokRParen) == len(value)-1 {
			if producerPath(value[1:open]) != "" {
				return strings.ReplaceAll(producerText(value[1:open]), " ", "")
			}
		}

		return ""
	}

	budget := 1024

	return r.producerExpressionValue(fd, producerText(value), filepath.Dir(fd.URI.Path()), "", 0, &budget).component()
}

// ownedKeyShape is the key variable and scope of a body whose every return
// is `scope[ key ]` and whose every other mention of key is one ownedKeyUse
// allows; ok false otherwise.
func ownedKeyShape(tokens []parser.Token) (key, scope string, ok bool) {
	returns := 0

	for i := range tokens {
		if !strings.EqualFold(tokens[i].Value, "return") && !strings.EqualFold(tokens[i].Value, "cfreturn") {
			continue
		}

		end := ownedStatementEnd(tokens, i+1)

		value := tokens[i+1 : end]
		if n := len(value); n > 0 && value[n-1].Kind == parser.TokSlash {
			value = value[:n-1]
		}

		if len(value) != 4 || value[0].Kind != parser.TokIdent || value[1].Kind != parser.TokLBracket || value[2].Kind != parser.TokIdent || value[3].Kind != parser.TokRBracket {
			return "", "", false
		}

		s, k := strings.ToLower(value[0].Value), strings.ToLower(value[2].Value)
		if !ownedKeyScopes[s] || key != "" && (k != key || s != scope) {
			return "", "", false
		}

		key, scope = k, s
		returns++
	}

	if returns == 0 {
		return "", "", false
	}

	for i := range tokens {
		if tokens[i].Kind == parser.TokIdent && strings.EqualFold(tokens[i].Value, key) && !ownedKeyUse(tokens, i, scope) {
			return "", "", false
		}
	}

	return key, scope, true
}

// ownedKeyUse reports whether the mention of the key variable at tokens[i]
// cannot write the scope under another name or hand the key elsewhere: its
// declaration, an index into the scope, a presence test or delete on the
// scope, or an operand of & building another string.
func ownedKeyUse(tokens []parser.Token, i int, scope string) bool {
	prev := func(n int) parser.Token {
		if i-n < 0 {
			return parser.Token{}
		}

		return tokens[i-n]
	}

	var next parser.Token
	if i+1 < len(tokens) {
		next = tokens[i+1]
	}

	switch {
	case isAssignment(tokens, i+1):
		return prev(1).Kind != parser.TokDot || strings.EqualFold(prev(2).Value, "local")
	case prev(1).Kind == parser.TokLBracket && next.Kind == parser.TokRBracket:
		return strings.EqualFold(prev(2).Value, scope)
	case prev(1).Kind == parser.TokAmpersand || next.Kind == parser.TokAmpersand:
		return true
	case prev(1).Kind == parser.TokComma && strings.EqualFold(prev(2).Value, scope) && prev(3).Kind == parser.TokLParen && next.Kind == parser.TokRParen:
		name := strings.ToLower(prev(4).Value)

		return name == "structkeyexists" || name == "structdelete"
	case prev(1).Kind == parser.TokLParen && next.Kind == parser.TokRParen && prev(3).Kind == parser.TokDot && strings.EqualFold(prev(4).Value, scope):
		name := strings.ToLower(prev(2).Value)

		return name == "keyexists" || name == "delete"
	}

	return false
}

// ownedKeyPrefix is the literal, lowercased, the key variable's one
// assignment starts with: `key = "_prefix" & …`.
func ownedKeyPrefix(tokens []parser.Token, key string) (string, bool) {
	prefix := ""
	assignments := 0

	for i := range tokens {
		if tokens[i].Kind != parser.TokIdent || !strings.EqualFold(tokens[i].Value, key) || !isAssignment(tokens, i+1) {
			continue
		}

		assignments++

		if i+2 >= len(tokens) || tokens[i+2].Kind != parser.TokString {
			return "", false
		}

		prefix = strings.ToLower(strings.Trim(tokens[i+2].Value, `"'`))
	}

	if assignments != 1 || len(prefix) < 4 || strings.ContainsAny(prefix, "#") {
		return "", false
	}

	return prefix, true
}

// isScopeKey reports whether tokens[i:] starts `scope [ key ]`.
func isScopeKey(tokens []parser.Token, i int, scope, key string) bool {
	return i+3 < len(tokens) && strings.EqualFold(tokens[i].Value, scope) && tokens[i+1].Kind == parser.TokLBracket &&
		strings.EqualFold(tokens[i+2].Value, key) && tokens[i+3].Kind == parser.TokRBracket && (i == 0 || tokens[i-1].Kind != parser.TokDot)
}

// isAssignment reports whether tokens[i] is an `=` that assigns, not `==`.
func isAssignment(tokens []parser.Token, i int) bool {
	return i < len(tokens) && tokens[i].Kind == parser.TokEquals && (i+1 >= len(tokens) || tokens[i+1].Kind != parser.TokEquals) && (i == 0 || tokens[i-1].Kind != parser.TokEquals)
}

// isSelfMock reports whether value is MockBox decorating what scope[ key ]
// already holds: createMock( object=… ) or prepareMock( … ), on whatever
// they are called. The object keeps its class.
func isSelfMock(value []parser.Token, scope, key string) bool {
	n := len(value)
	if n < 6 || value[n-1].Kind != parser.TokRParen {
		return false
	}

	open := -1

	for i := n - 1; i >= 0; i-- {
		if producerGroupEnd(value, i, parser.TokLParen, parser.TokRParen) == n-1 && value[i].Kind == parser.TokLParen {
			open = i
		}
	}

	if open < 1 {
		return false
	}

	name := strings.ToLower(value[open-1].Value)
	inner := value[open+1 : n-1]

	if len(inner) > 2 && strings.EqualFold(inner[0].Value, "object") && inner[1].Kind == parser.TokEquals {
		inner = inner[2:]
	} else if name != "preparemock" {
		return false
	}

	return (name == "createmock" || name == "preparemock") && len(inner) == 4 && isScopeKey(inner, 0, scope, key)
}

// ownedStatementEnd is where the statement starting at tokens[from] ends: a
// semicolon or a tag's closing > outside brackets, or the next line once
// every bracket is closed.
func ownedStatementEnd(tokens []parser.Token, from int) int {
	depth := 0

	for i := from; i < len(tokens); i++ {
		t := tokens[i]

		if depth == 0 && i > from && t.Line > tokens[i-1].Line {
			return i
		}

		switch t.Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			if depth == 0 {
				return i
			}

			depth--
		case parser.TokSemicolon, parser.TokGT:
			if depth == 0 {
				return i
			}
		default:
		}
	}

	return len(tokens)
}

// functionSource is the body of the file's function name: a script
// function's, or the text between a <cffunction name="…"> and its close.
func functionSource(source *sourceInfo, name string) string {
	if m, ok := source.methodMap()[strings.ToLower(name)]; ok && m.body != "" {
		return m.body
	}

	return source.tagFunctionBodies()[strings.ToLower(name)]
}

// tagFunctionBodies is the body of each `<cffunction>` in the source by its
// lowercased name, the first declaration of a name winning, read once per
// source: functionSource is asked for every owned-key candidate, and reading
// the whole file for each was 4% of tassweb's scan.
func (s *sourceInfo) tagFunctionBodies() map[string]string {
	if s.derived == nil {
		return tagFunctionBodies(s.content)
	}

	s.derived.tagBodiesOnce.Do(func() { s.derived.tagBodies = tagFunctionBodies(s.content) })

	return s.derived.tagBodies
}

func tagFunctionBodies(content string) map[string]string {
	out := map[string]string{}

	for _, fn := range tagFunctionNames(content) {
		if _, seen := out[fn.name]; !seen {
			out[fn.name] = fn.body
		}
	}

	return out
}

// tagFunction is one <cffunction> of a tag component: its lowercased name and
// the text between its tag and its </cffunction>, "" when it has none.
type tagFunction struct{ name, body string }

// tagFunctionNames is the <cffunction>s in content whose name is a plain
// identifier ([\w$]+), in order, read with parser.CFTags: a commented-out
// function, or a '>' inside a hint, no longer ends or makes one.
func tagFunctionNames(content string) []tagFunction {
	var out []tagFunction

	tags := parser.CFTags(content, "cffunction")
	for i := range tags {
		if tags[i].Close {
			continue
		}

		name, _ := tags[i].Attr("name")
		if !plainFunctionName(name) {
			continue
		}

		body := ""
		if i+1 < len(tags) && tags[i+1].Close {
			body = content[tags[i].End:tags[i+1].Start]
		}

		// A clone: the name is a slice of the whole file, which the map
		// would otherwise keep alive.
		out = append(out, tagFunction{strings.Clone(strings.ToLower(name)), body})
	}

	return out
}

// plainFunctionName reports whether name is [\w$]+, the names
// tagFunctionNames reads.
func plainFunctionName(name string) bool {
	if name == "" {
		return false
	}

	for i := range len(name) {
		if c := name[i]; !lex.IsWord(c) && c != '$' {
			return false
		}
	}

	return true
}
