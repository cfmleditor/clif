package resolve

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// This fingerprint pins the complete lexical Injector contract in testdata/
// wheels-injector/Injector.cfc. In particular init empties all caches, calls the
// selected binder once and returns this; getInstance resolves the alias and
// $constructInstance returns the created object, ignoring init's return value.
// Constructor code, overrides and modified helper/default bodies invalidate it.
// A fresh transient lookup immediately following construction cannot observe
// a later rebind. This is not a contract for arbitrary live container lookups.
const wheelsInjectorContract = "0fc2d8da940fe1c1b25cf546992195f9be50db12948625f2fba5733ba35dd7ce"

func (r *Resolver) bindWheelsStartup(assigns []startupAssign, content, dir string) {
	tokens := allTokens(content)
	depth := 0

	for i, t := range tokens {
		if t.Kind == parser.TokLBrace {
			depth++
		}

		if t.Kind == parser.TokRBrace {
			depth--
		}

		if depth != 1 || i == 0 || (tokens[i-1].Kind != parser.TokLBrace && tokens[i-1].Kind != parser.TokRBrace && tokens[i-1].Kind != parser.TokSemicolon) {
			continue
		}

		key, rhs, end := wheelsStartupStatement(tokens, i)
		if key == "" || len(rhs) != 7 || !strings.EqualFold(rhs[0].Value, "new") || producerPath(rhs[1:4]) != "wheels.injector" || rhs[4].Kind != parser.TokLParen || rhs[5].Kind != parser.TokString || rhs[6].Kind != parser.TokRParen {
			continue
		}

		binder, ok := wheelsLiteral(rhs[5])
		if !ok {
			continue
		}

		result, lookup, last := wheelsStartupStatement(tokens, end+1)
		if result == "" || len(lookup) != 8 || producerPath(lookup[:5]) != key+".getinstance" || lookup[5].Kind != parser.TokLParen || lookup[6].Kind != parser.TokString || lookup[7].Kind != parser.TokRParen {
			continue
		}

		alias, ok := wheelsLiteral(lookup[6])
		if !ok {
			continue
		}

		injector := r.ComponentPath("wheels.Injector", dir)
		if injector == "" {
			continue
		}

		digest := sha256.Sum256([]byte(wheelsTokens(r.wheelsSource(injector).content)))
		if hex.EncodeToString(digest[:]) != wheelsInjectorContract {
			continue
		}

		component := r.wheelsLiteralBinding(binder, alias, dir)
		if component == "" {
			continue
		}
		// Match the exact assignment rather than another write to the same slot.
		expression := content[lookup[0].Offset:tokens[last].Offset]
		match, count := -1, 0

		for j := range assigns {
			if assigns[j].key == result && wheelsTokens(assigns[j].rhs) == wheelsTokens(expression) {
				match, count = j, count+1
			}
		}

		if count == 1 {
			assigns[match].component = component
		}
	}
}

func wheelsStartupStatement(tokens []parser.Token, start int) (string, []parser.Token, int) {
	if start+4 >= len(tokens) || tokens[start+3].Kind != parser.TokEquals {
		return "", nil, 0
	}

	key, ok := splitSharedScope(producerPath(tokens[start : start+3]))
	if !ok {
		return "", nil, 0
	}

	for end := start + 4; end < len(tokens) && end < start+32; end++ {
		if tokens[end].Kind == parser.TokSemicolon {
			return key, tokens[start+4 : end], end
		}

		if tokens[end].Kind == parser.TokLBrace || tokens[end].Kind == parser.TokRBrace {
			break
		}
	}

	return "", nil, 0
}

func wheelsLiteral(token parser.Token) (string, bool) {
	if token.Kind != parser.TokString || len(token.Value) < 2 || strings.Contains(token.Value, "#") {
		return "", false
	}

	value := token.Value[1 : len(token.Value)-1]

	return value, value != "" && !strings.ContainsAny(value, "\"'")
}

// A closed configure body contains only literal fluent registrations against
// its actual injector parameter. No conditional entries, flags/caches, inherited
// declarations, calls/escapes, duplicate aliases or computed paths are admitted.
func (r *Resolver) wheelsLiteralBinding(binder, alias, dir string) string {
	file := r.ComponentPath(binder, dir)
	if file == "" {
		return ""
	}

	source := r.wheelsSource(file)
	pr := parser.Parse(cfpath.ToURI(file), source.content, nil)

	constructor, _ := parser.ApplicationSections(source.content)
	if pr.Extends != "" || len(pr.Funcs) != 1 || wheelsTokens(constructor) != wheelsTokens(`component {}`) {
		return ""
	}

	fd := &pr.Funcs[0]
	if !strings.EqualFold(fd.Name, "configure") || len(fd.Arguments) != 1 || !strings.EqualFold(fd.Arguments[0].Name, "injector") || !source.methods["configure"].public || !source.methods["configure"].literalDefaults {
		return ""
	}

	tokens := allTokens(source.methods["configure"].body)
	bindings := map[string]string{}

	for i := 0; i < len(tokens); {
		if i+3 > len(tokens) || producerPath(tokens[i:i+3]) != "arguments.injector" {
			return ""
		}

		i += 3
		for {
			if i+10 > len(tokens) || tokens[i].Kind != parser.TokDot || (tokens[i+1].Value != "map" && tokens[i+1].Value != "bind" && tokens[i+1].Value != "mapinstance") || tokens[i+2].Kind != parser.TokLParen || tokens[i+4].Kind != parser.TokRParen || tokens[i+5].Kind != parser.TokDot || tokens[i+6].Value != "to" || tokens[i+7].Kind != parser.TokLParen || tokens[i+9].Kind != parser.TokRParen {
				return ""
			}

			name, knownName := wheelsLiteral(tokens[i+3])
			target, knownTarget := wheelsLiteral(tokens[i+8])

			name = strings.ToLower(name)
			if !knownName || !knownTarget || bindings[name] != "" || len(bindings) >= 64 {
				return ""
			}

			bindings[name] = target

			i += 10
			if i >= len(tokens) {
				return ""
			}

			if tokens[i].Kind == parser.TokSemicolon {
				i++

				break
			}
		}
	}

	return r.ComponentPath(bindings[strings.ToLower(alias)], filepath.Dir(file))
}

// Preserve a proved assignment's strict conflict policy before falling back to
// Application.cfc's class-level ComponentRefs.
func (r *Resolver) wheelsStartupComponent(variable, dir string, tr *callTrace) (string, bool) {
	key, shared := splitSharedScope(variable)
	if !shared {
		return "", false
	}

	assigns := r.startupAssigns(dir)
	for i := range assigns {
		if assigns[i].key == key && assigns[i].component != "" {
			return r.startupComponent(variable, dir, tr), true
		}
	}

	return "", false
}
