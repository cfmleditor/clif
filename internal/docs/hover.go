package docs

import (
	"fmt"
	"strings"
)

// FunctionPreviews returns CFScript declaration previews for the documented call forms.
func (e *Entry) FunctionPreviews() []string {
	previews := []string{}
	seen := make(map[string]bool)

	for line := range strings.SplitSeq(e.Syntax, "\n") {
		params, ok := e.callParameters(strings.TrimSpace(line))
		if !ok {
			continue
		}

		preview := e.functionPreview(params)
		if !seen[preview] {
			seen[preview] = true

			previews = append(previews, preview)
		}
	}

	if len(previews) == 0 {
		previews = append(previews, e.functionPreview(e.Params))
	}

	return previews
}

func (e *Entry) callParameters(syntax string) ([]Param, bool) {
	syntax = strings.TrimSpace(strings.TrimSuffix(syntax, ";"))

	name, params, ok := strings.Cut(syntax, "(")
	if !ok || !strings.EqualFold(strings.TrimSpace(name), e.Name) || !strings.HasSuffix(params, ")") {
		return nil, false
	}

	params = strings.TrimSpace(strings.TrimSuffix(params, ")"))
	if strings.ContainsAny(params, "()") {
		return nil, false
	}

	if params == "" {
		return []Param{}, true
	}

	fields := strings.Split(params, ",")
	ordered := make([]Param, 0, len(fields))
	seen := make(map[string]bool, len(fields))

	for _, field := range fields {
		words := strings.Fields(strings.Trim(field, "[] "))
		if len(words) == 0 {
			return nil, false
		}

		paramName := strings.ToLower(strings.Trim(words[len(words)-1], "[]"))
		if seen[paramName] {
			return nil, false
		}

		seen[paramName] = true
		found := false

		for i := range e.Params {
			p := &e.Params[i]
			if strings.EqualFold(p.Name, paramName) {
				ordered = append(ordered, *p)
				found = true

				break
			}
		}

		if !found {
			return nil, false
		}
	}

	return ordered, true
}

func (e *Entry) functionPreview(params []Param) string {
	returnType := e.Returns
	if returnType == "" {
		returnType = "any"
	}

	var out strings.Builder
	fmt.Fprintf(&out, "%s function %s(", returnType, e.Name)

	for i := range params {
		if i > 0 {
			out.WriteString(", ")
		}

		p := &params[i]
		if p.Required {
			out.WriteString("required ")
		}

		parameterType := p.Type
		if parameterType == "" {
			parameterType = "any"
		}

		fmt.Fprintf(&out, "%s %s", parameterType, p.Name)
	}

	out.WriteByte(')')

	return out.String()
}
