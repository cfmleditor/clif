package parser

import "strings"

// DuplicateArgument is an argument a function declares a second (or later)
// time, at byte offsets of the page: Start to End is that later declaration.
type DuplicateArgument struct {
	Name     string
	Function string
	Start    int
	End      int
}

// DuplicateArguments are the arguments each of spans declares more than once,
// one entry for every declaration after the first, in the order they stand. A
// script function's are read from its parameter list, a <cffunction>'s from
// the name of its <cfargument> tags.
func DuplicateArguments(content string, spans []FunctionSpan) []DuplicateArgument {
	var out []DuplicateArgument

	for i := range spans {
		s := &spans[i]
		seen := map[string]bool{}

		note := func(name string, start, end int) {
			lower := strings.ToLower(name)
			if seen[lower] {
				out = append(out, DuplicateArgument{Name: name, Function: s.Name, Start: start, End: end})
			}

			seen[lower] = true
		}

		if s.Params >= 0 {
			for _, p := range parseParams(s.ParamsText(content)) {
				note(p.name, s.Params+p.nameAt, s.Params+p.nameAt+len(p.name))
			}

			continue
		}

		if s.Body < 0 {
			continue
		}

		body := s.BodyText(content)
		for _, tag := range CFTags(body, "cfargument") {
			if tag.Close {
				continue
			}

			if start, end, ok := attrSpan(tag.text, "name"); ok && start < end {
				note(tag.text[start:end], s.Body+tag.Start+start, s.Body+tag.Start+end)
			}
		}
	}

	return out
}
