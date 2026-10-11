package parser

import "strings"

// ApplicationSections separates executable constructor text from declarations.
// Byte positions and newlines are retained in each projection. The method map
// is case insensitive; duplicate declarations are withheld rather than selected.
func ApplicationSections(content string) (constructor string, methods map[string]string) {
	masked := []byte(content)
	methods = map[string]string{}
	duplicate := map[string]bool{}

	sc := NewScanner(content)

	var header []Token

	for {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF {
			break
		}

		if t.Kind != TokIdent {
			if t.Kind == TokDot && len(header) > 0 {
				header = append(header, t)
			} else {
				header = nil
			}

			continue
		}

		start, name := t.Offset, ""

		var end int

		switch {
		case identEq(t.Value, "function"):
			if applicationFunctionHeader(header) {
				start = header[0].Offset
			}

			if next := sc.PeekSkipComments(); next.Kind == TokIdent {
				name = strings.ToLower(next.Value)
			}

			skipMappingFunction(sc)
			end = sc.pos
		case identEq(t.Value, "cffunction"):
			start = max(0, start-1)

			name = applicationTagMethod(sc)

			end = sc.pos
		default:
			if len(header) < 16 {
				header = append(header, t)
			} else {
				header = nil
			}

			continue
		}

		header = nil

		projection := []byte(content)
		for i := range projection {
			if (i < start || i >= end) && projection[i] != '\n' && projection[i] != '\r' {
				projection[i] = ' '
			}
		}

		for i := start; i < end; i++ {
			if masked[i] != '\n' && masked[i] != '\r' {
				masked[i] = ' '
			}
		}

		if name != "" {
			if _, exists := methods[name]; exists || duplicate[name] {
				methods[name] = ""
				duplicate[name] = true
			} else {
				methods[name] = string(projection)
			}
		}
	}

	return string(masked), methods
}

// Access/modifier and return-type tokens are part of a declaration, not
// executable constructor text. Keep arbitrary qualified component return types
// while rejecting statement keywords as a purported return type.
func applicationFunctionHeader(tokens []Token) bool {
	if len(tokens) == 0 {
		return false
	}

	start := 0
	for start < len(tokens) {
		switch strings.ToLower(tokens[start].Value) {
		case "public", "private", "package", "remote", "static", "final", "abstract":
			start++
		default:
			goto returnType
		}
	}

	return true

returnType:
	for i := start; i < len(tokens); i++ {
		if (i-start)%2 == 0 {
			if tokens[i].Kind != TokIdent {
				return false
			}
		} else if tokens[i].Kind != TokDot {
			return false
		}
	}

	switch strings.ToLower(tokens[start].Value) {
	case "return", "var", "new", "component", "include", "import", "throw":
		return false
	}

	return (len(tokens)-start)%2 == 1
}

func applicationTagMethod(sc *Scanner) (name string) {
	for {
		a := sc.NextSkipComments()
		if a.Kind == TokEOF || a.Kind == TokGT {
			break
		}

		if identEq(a.Value, "name") && sc.NextSkipComments().Kind == TokEquals {
			value := sc.NextSkipComments()
			if value.Kind == TokString {
				name = strings.ToLower(strings.Trim(value.Value, `"'`))
			}
		}
	}

	for {
		a := sc.NextSkipComments()
		if a.Kind == TokEOF {
			break
		}

		if a.Kind == TokSlash && identEq(sc.PeekSkipComments().Value, "cffunction") {
			sc.NextSkipComments()

			if sc.PeekSkipComments().Kind == TokGT {
				sc.NextSkipComments()
			}

			break
		}
	}

	return name
}
