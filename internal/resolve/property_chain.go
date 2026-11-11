package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// Only assignments to this expose a field on another component. Variables
// fields and generated getX() methods do not imply a public x field. An
// unknown/changed write shadows an inherited field rather than borrowing it.
func (r *Resolver) publicPropertyComponent(component, name, baseDir string) string {
	path := r.ComponentPath(component, baseDir)

	seen := make(map[string]bool)
	for len(seen) < 8 && path != "" && !seen[path] {
		seen[path] = true
		r.EnsureIndexed(path)
		refs := r.Index.RefsForFile(cfpath.ToURI(path))

		text, err := r.readText(path)
		if err != nil {
			return ""
		}

		writes := publicPropertyWrites(r.assignmentsIn(path, text), name)
		if len(writes) > 0 {
			value := ""

			for line, count := range writes {
				if count != 1 {
					return "$any"
				}

				atLine := ""

				for _, ref := range refs {
					if !ref.This || !strings.EqualFold(ref.Variable, name) || int(ref.Line) != line {
						continue
					}

					if atLine != "" && !strings.EqualFold(atLine, ref.Component) {
						return "$any"
					}

					atLine = ref.Component
				}

				if atLine == "" || value != "" && !strings.EqualFold(value, atLine) {
					return "$any"
				}

				value = atLine
			}

			return value
		}

		next, ok := r.extendsOf(path, cfpath.ToURI(path))
		if !ok {
			return ""
		}

		path = r.ComponentPath(next, filepath.Dir(path))
	}

	return ""
}

// publicPropertyWrites counts, by line, the writes in as to this.name: `this.name
// = …` and `this[ "name" ] = …`, and `this[ key ] = …` with any other key,
// which may be name.
func publicPropertyWrites(as []parser.Assignment, name string) map[int]int {
	writes := make(map[int]int)

	for i := range as {
		a := &as[i]
		if a.Op != parser.AssignSet {
			continue
		}

		steps := targetSteps(a.Target)
		if len(steps) != 2 || !strings.EqualFold(steps[0], "this") {
			continue
		}

		field := steps[1]
		if strings.HasPrefix(field, "[") {
			field = name

			key := strings.TrimSpace(steps[1][1 : len(steps[1])-1])
			if len(key) >= 2 && (key[0] == '"' || key[0] == '\'') && key[len(key)-1] == key[0] && !strings.ContainsAny(key, "#") {
				field = key[1 : len(key)-1]
			}
		}

		if strings.EqualFold(field, name) {
			writes[a.Line]++
		}
	}

	return writes
}
