package resolve

import (
	"slices"
	"sort"
	"strings"

	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
)

type queryOutput struct {
	kind   string
	fields map[string][]parser.Token
	end    int
}

// Read only the lifetime after the selected initializer and before the lookup.
// A completed output in an enclosing lexical block dominates this lookup on its
// normal path. Closed branches, catches, closures and unknown uses withhold it.
// Markup, shared fields and same-line output/lookups need separate contracts.
func (r *Resolver) literalQueryOutput(pr *parser.ParseResult, name string, assigned, header int, caller, baseDir string, call *parser.CallSite) (string, bool) {
	starts := r.sourceLineStarts(pr)
	if assigned < 0 || header <= assigned+1 || header >= len(starts) {
		return "", false
	}

	from, to := starts[assigned+1], starts[header]-1
	if !r.hasQueryOutputWord(pr, from, to) {
		return "", false
	}

	tokens := producerTokens(pr.Content[from:to])
	if tokens == nil {
		return "", true
	}

	prefix := len(tokens)

	offset := starts[header]
	headerLine := sourceLine(pr.Content, starts, header)

	if call != nil && int(call.StartByte) >= offset && int(call.StartByte) <= offset+len(headerLine) {
		headerTokens := producerTokens(pr.Content[offset:call.StartByte])
		if headerTokens == nil && int(call.StartByte) > offset {
			return "", true
		}

		tokens = append(tokens, headerTokens...)
	} else if strings.ContainsAny(headerLine, "{}") {
		return "", true
	}

	return r.scanQueryOutputs(tokens, prefix, name, caller, pr, baseDir)
}

func (r *Resolver) scanQueryOutputs(tokens []parser.Token, prefix int, name, caller string, pr *parser.ParseResult, baseDir string) (string, bool) {
	var stack, owner []int

	nextFrame := 0
	component := ""
	written, unsafe := false, false

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token.Kind == parser.TokLT && i+1 < len(tokens) && strings.HasPrefix(strings.ToLower(tokens[i+1].Value), "cf") {
			return "", false
		}

		if output, ok := readQueryOutput(tokens, i); ok {
			comp, matched := queryOutputComponent(output, name)
			if matched {
				written = true

				component = comp
				if i >= prefix || !queryOutputStatement(tokens, i) {
					component = ""
				}

				owner = slices.Clone(stack)

				if token.Kind == parser.TokIdent && !strings.EqualFold(token.Value, "query") && (pr.Declares(token.Value) || argumentOf(pr, caller, token.Value) != nil || r.bareFunc(token.Value, pr, baseDir, caller) != nil) {
					component = ""
				}
			}

			if matched {
				i = output.end

				continue
			}
		}

		switch token.Kind {
		case parser.TokLBrace:
			nextFrame++
			stack = append(stack, nextFrame)
		case parser.TokRBrace:
			if len(stack) == 0 {
				unsafe = true
			} else {
				stack = stack[:len(stack)-1]
			}
		case parser.TokIdent:
			if strings.EqualFold(token.Value, "function") || queryUnbracedControl(tokens, i) {
				unsafe = true
			}

			if written && strings.EqualFold(token.Value, name) && (i < prefix || i != len(tokens)-2 || tokens[i+1].Kind != parser.TokDot) && !queryReadonlyUse(tokens, i) && !queryPendingAssignment(tokens, i, prefix) {
				component = ""
			}
		default:
		}
	}

	if unsafe || len(owner) > len(stack) || !slices.Equal(owner, stack[:min(len(owner), len(stack))]) {
		component = ""
	}

	return component, written
}

// Only standalone tag statements are known to execute on the normal path;
// a callable writer inside a short-circuit or ternary expression may be skipped.
func queryOutputStatement(tokens []parser.Token, at int) bool {
	if at == 0 {
		return true
	}

	switch tokens[at-1].Kind {
	case parser.TokSemicolon, parser.TokLBrace, parser.TokRBrace:
		return true
	default:
		return false
	}
}

// The RHS lookup runs before its containing assignment writes the result.
func queryPendingAssignment(tokens []parser.Token, at, prefix int) bool {
	if at < prefix || at+1 >= len(tokens) || tokens[at+1].Kind != parser.TokEquals {
		return false
	}

	for _, token := range tokens[at+2:] {
		if token.Kind == parser.TokSemicolon {
			return false
		}
	}

	return true
}

func queryReadonlyUse(tokens []parser.Token, at int) bool {
	if at+3 >= len(tokens) || tokens[at+1].Kind != parser.TokDot || tokens[at+3].Kind != parser.TokLParen {
		return false
	}

	switch strings.ToLower(tokens[at+2].Value) {
	case "columndata", "columnarray", "getcolumnnames", "getcell", "getrow", "recordcount", "columnexists", "keyarray", "keylist", "keyexists", "count", "len":
		return true
	default:
		return false
	}
}

func readQueryOutput(tokens []parser.Token, at int) (queryOutput, bool) {
	if at >= len(tokens) {
		return queryOutput{}, false
	}

	token := tokens[at]
	if token.Kind != parser.TokIdent || at > 0 && tokens[at-1].Kind == parser.TokDot {
		return queryOutput{}, false
	}

	name := strings.ToLower(token.Value)
	if !queryOutputTag(name) {
		return queryOutput{}, false
	}

	start, end := at+1, -1
	if start >= len(tokens) {
		return queryOutput{}, false
	}

	if tokens[start].Kind == parser.TokLParen {
		// query(...) is the query literal function, not a tag output statement.
		if name == "query" {
			return queryOutput{}, false
		}

		end = producerGroupEnd(tokens, start, parser.TokLParen, parser.TokRParen)
		if end < 0 {
			return queryOutput{}, false
		}

		fields := queryOutputFields(tokens[start+1 : end])
		if end+1 < len(tokens) && tokens[end+1].Kind == parser.TokLBrace {
			end = producerGroupEnd(tokens, end+1, parser.TokLBrace, parser.TokRBrace)
			if end < 0 {
				return queryOutput{}, false
			}
		}

		return queryOutput{kind: name, fields: fields, end: end}, true
	}

	if start+1 >= len(tokens) || tokens[start].Kind != parser.TokIdent || tokens[start+1].Kind != parser.TokEquals {
		return queryOutput{}, false
	}

	depth := 0

	for i := start; i < len(tokens); i++ {
		switch tokens[i].Kind {
		case parser.TokLParen, parser.TokLBracket:
			depth++
		case parser.TokRParen, parser.TokRBracket:
			depth--
		case parser.TokLBrace, parser.TokSemicolon:
			if depth == 0 {
				end = i
			}
		default:
		}

		if end >= 0 {
			break
		}
	}

	if end < 0 {
		return queryOutput{}, false
	}

	fields := queryOutputFields(tokens[start:end])
	if tokens[end].Kind == parser.TokLBrace {
		end = producerGroupEnd(tokens, end, parser.TokLBrace, parser.TokRBrace)
		if end < 0 {
			return queryOutput{}, false
		}
	}

	return queryOutput{kind: name, fields: fields, end: end}, true
}

func queryOutputFields(tokens []parser.Token) map[string][]parser.Token {
	fields := map[string][]parser.Token{}
	start, key, depth := 0, "", 0

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if depth == 0 && tok.Kind == parser.TokIdent && i+1 < len(tokens) && tokens[i+1].Kind == parser.TokEquals {
			if key != "" {
				storeQueryOutputField(fields, key, tokens[start:i])
			}

			key = strings.ToLower(tok.Value)
			start = i + 2
			i++

			continue
		}

		switch tok.Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth--
		default:
		}
	}

	if key != "" {
		storeQueryOutputField(fields, key, tokens[start:])
	}

	return fields
}

func storeQueryOutputField(fields map[string][]parser.Token, key string, tokens []parser.Token) {
	if _, duplicate := fields[key]; duplicate {
		fields[key] = nil

		return
	}

	fields[key] = queryOutputFieldValue(tokens)
}

func queryOutputFieldValue(tokens []parser.Token) []parser.Token {
	if len(tokens) > 0 && tokens[len(tokens)-1].Kind == parser.TokComma {
		return tokens[:len(tokens)-1]
	}

	return tokens
}

func queryOutputComponent(output queryOutput, name string) (string, bool) {
	fields := output.fields
	if _, dynamic := fields["attributecollection"]; dynamic {
		return "", true
	}

	component, matched := "", false

	for _, key := range []string{"name", "result", "variable", "returnvariable", "output"} {
		value, ok := fields[key]
		if !ok {
			continue
		}

		if len(value) != 1 || value[0].Kind != parser.TokString || strings.Contains(value[0].Value, "#") {
			return "", true
		}

		target := strings.Trim(value[0].Value, "\"'")

		target = strings.TrimPrefix(strings.ToLower(target), "local.")
		if !strings.EqualFold(target, name) {
			continue
		}

		if matched {
			return "", true
		}

		matched = true

		if output.kind != "query" && output.kind != "cfquery" && output.kind != "cfdbinfo" {
			return "", true
		}

		if (output.kind == "query" || output.kind == "cfquery") && !querySynchronousOutput(fields) {
			return "", true
		}

		if key == "result" {
			component = nativetype.Type("struct")

			continue
		}

		if ret, exists := fields["returntype"]; exists && (len(ret) != 1 || ret[0].Kind != parser.TokString || !strings.EqualFold(strings.Trim(ret[0].Value, "\"'"), "query")) {
			return "", true
		}

		component = nativetype.Type("query")
	}

	return component, matched
}

// HQL defaults to entities, async execution may not have written its output,
// and a listener can change both the output name and the result interface.
func querySynchronousOutput(fields map[string][]parser.Token) bool {
	if _, exists := fields["listener"]; exists {
		return false
	}

	if value, exists := fields["async"]; exists && (len(value) != 1 || !strings.EqualFold(strings.Trim(value[0].Value, "\"'"), "false")) {
		return false
	}

	if value, exists := fields["dbtype"]; exists && (len(value) != 1 || value[0].Kind != parser.TokString || strings.EqualFold(strings.Trim(value[0].Value, "\"'"), "hql") || strings.Contains(value[0].Value, "#")) {
		return false
	}

	return true
}

func queryOutputTag(name string) bool {
	switch name {
	case "query", "cfquery", "cfdbinfo", "savecontent", "cfsavecontent", "cfhttp", "cfdirectory", "cffile", "cfxml", "cfinvoke", "cfldap", "cfsearch", "cfstoredproc", "cfprocresult", "cfexecute", "cfobject", "cfimage", "cfzip":
		return true
	default:
		return false
	}
}

func queryUnbracedControl(tokens []parser.Token, at int) bool {
	switch strings.ToLower(tokens[at].Value) {
	case "switch", "case", "default":
		return true
	case "else":
		return at+1 >= len(tokens) || tokens[at+1].Kind != parser.TokLBrace
	case "if", "for", "while":
		if at+1 >= len(tokens) || tokens[at+1].Kind != parser.TokLParen {
			return true
		}

		end := producerGroupEnd(tokens, at+1, parser.TokLParen, parser.TokRParen)

		return end < 0 || end+1 >= len(tokens) || tokens[end+1].Kind != parser.TokLBrace
	default:
		return false
	}
}

// Line-delimited windows have the same word boundaries as the complete source.
// Cache occurrences once and search their offsets instead of rescanning a
// potentially huge literal initializer for every later receiver lookup.
func (r *Resolver) hasQueryOutputWord(pr *parser.ParseResult, from, to int) bool {
	words := r.lexicalSource(pr).queryWords
	i := sort.Search(len(words), func(i int) bool { return words[i][0] >= from })

	return i < len(words) && words[i][1] <= to
}
