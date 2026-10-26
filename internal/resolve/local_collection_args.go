package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// Mura hands its publisher its arguments in a struct built a key at a time:
//
//	<cfset var sArgs = structNew()>
//	<cfset sArgs.Bundle = Bundle />
//	<cfset publisher.getToWork( argumentCollection = sArgs )>
//
// localCollectionArgument reads what such a call passes argName: the value
// the struct's key was written, when the text between the struct's
// declaration and the call shows every reference to it. The struct is
// declared in the caller (`var name = structNew()`, `{}` or a struct
// literal); every reference after that writes a key (`name.key = value`,
// `name[ "key" ] = value`) or reads one, and anything else (passing it on,
// structDelete, replacing it) leaves the argument untyped. A key written more
// than once, in branches or not, is untyped, as is a written name the caller
// assigns again before the call. A struct that never holds the key does not
// pass it.

// localCollectionArgument is what the call whose name is tokens[call] passes
// argName through argumentCollection = name: the value's tokens and whether it
// is passed at all, or ok false when the text cannot show it.
func localCollectionArgument(tokens []parser.Token, call int, name, argName string) (expr []parser.Token, passed, ok bool) {
	decl := localStructDeclaration(tokens, call, name)
	if decl < 0 {
		return nil, false, false
	}

	var (
		value []parser.Token
		at    = -1
	)

	// The declaration's own literal keys.
	if end := declarationEnd(tokens, decl); end > 0 && tokens[decl+3].Kind == parser.TokLBrace {
		for _, piece := range producerSplit(tokens[decl+4 : end]) {
			if len(piece) > 2 && (piece[1].Kind == parser.TokColon || piece[1].Kind == parser.TokEquals) && strings.EqualFold(strings.Trim(piece[0].Value, `"'`), argName) {
				value, at = piece[2:], decl
			}
		}
	}

	for j := decl + 3; j < call; j++ {
		t := tokens[j]
		if t.Kind != parser.TokIdent || !strings.EqualFold(t.Value, name) || j > 0 && tokens[j-1].Kind == parser.TokDot {
			continue
		}

		key, rhs, next, read := structKeyUse(tokens, j)

		switch {
		case key == "" && !read:
			return nil, false, false
		case read:
			j = next
		case strings.EqualFold(key, argName):
			if at >= 0 {
				return nil, false, false
			}

			value, at, j = tokens[rhs:next], j, next-1
		default:
			j = next - 1
		}
	}

	if at < 0 {
		return nil, false, true
	}

	// A name the key was written and the caller then assigns again holds
	// something else at the call.
	if len(value) == 1 && value[0].Kind == parser.TokIdent && reassignedBetween(tokens, at, call, value[0].Value) {
		return nil, false, false
	}

	return value, len(value) > 0, len(value) > 0
}

// localStructDeclaration is the index of the `var` of the nearest
// `var name = structNew()`, `var name = {…}` before tokens[call], or -1: one
// past a function's start, or of anything else, is not this struct.
func localStructDeclaration(tokens []parser.Token, call int, name string) int {
	for j := call - 1; j >= 2; j-- {
		t := tokens[j]
		if t.Kind == parser.TokIdent && (strings.EqualFold(t.Value, "function") || strings.EqualFold(t.Value, "cffunction")) {
			return -1
		}

		if t.Kind != parser.TokEquals || tokens[j-1].Kind != parser.TokIdent || !strings.EqualFold(tokens[j-1].Value, name) || !strings.EqualFold(tokens[j-2].Value, "var") {
			continue
		}

		if j+1 >= len(tokens) {
			return -1
		}

		switch rhs := tokens[j+1]; {
		case rhs.Kind == parser.TokLBrace:
			return j - 2
		case rhs.Kind == parser.TokIdent && strings.EqualFold(rhs.Value, "structNew") && j+3 < len(tokens) && tokens[j+2].Kind == parser.TokLParen && tokens[j+3].Kind == parser.TokRParen:
			return j - 2
		default:
			return -1
		}
	}

	return -1
}

// declarationEnd is the index of the `}` closing a declaration's struct
// literal, or -1.
func declarationEnd(tokens []parser.Token, decl int) int {
	if decl+3 >= len(tokens) || tokens[decl+3].Kind != parser.TokLBrace {
		return -1
	}

	return producerGroupEnd(tokens, decl+3, parser.TokLBrace, parser.TokRBrace)
}

// structKeyUse reads the reference to a struct at tokens[j]: a write to a key,
// `name.key = value` or `name[ "key" ] = value`, gives the key, where its value
// starts and where the statement ends; a read of a key, `name.key` not
// followed by `=`, gives read and the index past it. Anything else gives
// neither.
func structKeyUse(tokens []parser.Token, j int) (key string, rhs, next int, read bool) {
	k := j + 1

	switch {
	case k+1 < len(tokens) && tokens[k].Kind == parser.TokDot && tokens[k+1].Kind == parser.TokIdent:
		key, k = tokens[k+1].Value, k+2
	case k+2 < len(tokens) && tokens[k].Kind == parser.TokLBracket && tokens[k+1].Kind == parser.TokString && tokens[k+2].Kind == parser.TokRBracket:
		key, k = strings.Trim(tokens[k+1].Value, `"'`), k+3
	default:
		return "", 0, 0, false
	}

	if key == "" || strings.Contains(key, "#") {
		return "", 0, 0, false
	}

	if k >= len(tokens) || tokens[k].Kind != parser.TokEquals || k+1 < len(tokens) && tokens[k+1].Kind == parser.TokEquals {
		// A read, unless what follows reaches into the value and writes it.
		if k < len(tokens) && (tokens[k].Kind == parser.TokDot || tokens[k].Kind == parser.TokLBracket || tokens[k].Kind == parser.TokLParen) {
			return "", 0, 0, false
		}

		return "", 0, k - 1, true
	}

	return key, k + 1, assignmentValueEnd(tokens, k+1), false
}

// assignmentValueEnd is the index past an assignment's value starting at
// tokens[from]: the first `;`, tag end or line break outside a group.
func assignmentValueEnd(tokens []parser.Token, from int) int {
	depth := 0

	for j := from; j < len(tokens); j++ {
		t := tokens[j]

		switch t.Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			if depth == 0 {
				return j
			}

			depth--
		case parser.TokSemicolon, parser.TokGT:
			if depth == 0 {
				return j
			}
		case parser.TokSlash:
			if depth == 0 && j+1 < len(tokens) && tokens[j+1].Kind == parser.TokGT {
				return j
			}
		default:
		}

		if depth == 0 && j > from && t.Line != tokens[j-1].Line {
			return j
		}
	}

	return len(tokens)
}

// reassignedBetween reports whether name is assigned between tokens[from]
// and tokens[to]: `name =` or `var name =`, outside a member's own name.
func reassignedBetween(tokens []parser.Token, from, to int, name string) bool {
	for j := from + 1; j < to && j+1 < len(tokens); j++ {
		t := tokens[j]
		if t.Kind == parser.TokIdent && strings.EqualFold(t.Value, name) && tokens[j-1].Kind != parser.TokDot && tokens[j+1].Kind == parser.TokEquals && (j+2 >= len(tokens) || tokens[j+2].Kind != parser.TokEquals) {
			return true
		}
	}

	return false
}
