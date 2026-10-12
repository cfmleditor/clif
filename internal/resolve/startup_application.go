package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

type startupSource struct{ file, content string }

// Framework ancestry uses the same frozen bootstrap lookup as startup. A
// missing intermediate parent or a cycle cannot establish framework ownership.
func (r *Resolver) applicationInheritsFramework(file, framework string) bool {
	read := func(path string) ([]byte, error) {
		r.Discovery.add(path)

		return r.fs().ReadFile(path)
	}
	_, parent := cfpath.ApplicationParentLookup(filepath.Dir(file), r.Mappings, read)

	seen := map[string]bool{}
	for len(seen) < 16 && file != "" && !seen[pathKey(file)] {
		seen[pathKey(file)] = true

		data, err := read(file)
		if err != nil {
			return false
		}

		pr := parser.Parse(cfpath.ToURI(file), string(data), nil)
		if strings.EqualFold(pr.Extends, framework) {
			return true
		}

		if pr.Extends == "" {
			return false
		}

		file = parent(pr.Extends, filepath.Dir(file))
	}

	return false
}

// Application constructors execute from parent to child. Lifecycle methods
// select the nearest declaration; ancestor bodies enter only through an
// explicit, standalone super call in the selected body's outer statement list.
func (r *Resolver) applicationStartupSources(file string) []startupSource {
	read := func(path string) ([]byte, error) {
		r.Discovery.add(path)

		return r.fs().ReadFile(path)
	}
	_, parent := cfpath.ApplicationParentLookup(filepath.Dir(file), r.Mappings, read)

	var (
		chain   []startupSource
		methods []map[string]startupSource
	)

	seen := map[string]bool{}
	fw1 := false

	for len(chain) < 16 && file != "" && !seen[pathKey(file)] {
		seen[pathKey(file)] = true

		data, err := read(file)
		if err != nil {
			break
		}

		content := string(data)
		constructor := parser.ApplicationConstructor(content)
		declared := r.applicationDeclarations(file, content, read)
		chain = append(chain, startupSource{file, constructor})
		methods = append(methods, declared)

		pr := parser.Parse(cfpath.ToURI(file), content, nil)

		fw1 = fw1 || strings.EqualFold(pr.Extends, "framework.one")
		if pr.Extends == "" {
			break
		}

		file = parent(pr.Extends, filepath.Dir(file))
	}

	var out []startupSource
	for _, c := range slices.Backward(chain) {
		out = append(out, c)
	}

	var selected func(string, int) []startupSource

	selected = func(name string, from int) []startupSource {
		for i := from; i < len(chain); i++ {
			method, ok := methods[i][name]
			if !ok {
				continue
			}

			body := method.content
			if body == "" {
				return nil
			}

			tokens := allTokens(body)
			depth, last := 0, 0

			var parts []startupSource

			for j, t := range tokens {
				if t.Kind == parser.TokLBrace {
					depth++
				}

				if t.Kind == parser.TokRBrace {
					depth--
				}

				if depth != 1 || !strings.EqualFold(t.Value, "super") || j == 0 || j+5 >= len(tokens) {
					continue
				}

				if tokens[j-1].Kind != parser.TokLBrace && tokens[j-1].Kind != parser.TokSemicolon {
					continue
				}

				if tokens[j+1].Kind != parser.TokDot || !strings.EqualFold(tokens[j+2].Value, name) || tokens[j+3].Kind != parser.TokLParen || tokens[j+4].Kind != parser.TokRParen || tokens[j+5].Kind != parser.TokSemicolon {
					continue
				}

				parts = append(parts, startupSource{method.file, body[last:t.Offset]})
				parts = append(parts, selected(name, i+1)...)
				last = tokens[j+5].Offset + 1
			}

			return append(parts, startupSource{method.file, body[last:]})
		}

		return nil
	}
	for _, name := range []string{"onapplicationstart", "onsessionstart", "onrequeststart"} {
		out = append(out, selected(name, 0)...)
	}

	if fw1 {
		for _, name := range []string{"setupapplication", "setupsession", "setuprequest"} {
			out = append(out, selected(name, 0)...)
		}
	}

	return out
}

func (r *Resolver) startupSources(file, content string) []startupSource {
	if strings.EqualFold(filepath.Base(file), "Application.cfc") {
		return r.applicationStartupSources(file)
	}

	constructor := parser.ApplicationConstructor(content)

	return []startupSource{{file, constructor}}
}

// Literal constructor includes can declare the selected lifecycle method in
// a separate template. They retain that template as the declaration origin.
// Includes inside declarations or guarded constructor blocks supply no methods.
func (r *Resolver) applicationDeclarations(file, content string, read func(string) ([]byte, error)) map[string]startupSource {
	methods := map[string]startupSource{}
	seen := map[string]bool{}

	var collect func(string, string, int)

	collect = func(file, content string, depth int) {
		if depth >= 16 || len(seen) >= maxStartupTemplates {
			return
		}

		if seen[pathKey(file)] {
			// Reexecuting a declaration template can replace methods or fail.
			// Do not keep a previously collected lifecycle as a stable target.
			for name, method := range methods {
				method.content = ""
				methods[name] = method
			}

			return
		}

		seen[pathKey(file)] = true

		constructor, declared := parser.ApplicationSections(content)
		for name, body := range declared {
			if _, exists := methods[name]; exists {
				body = ""
			}

			methods[name] = startupSource{file, body}
		}

		unconditional := constructorDeclarationIncludes(constructor)

		remaining := map[string]int{}
		for _, raw := range unconditional {
			remaining[raw]++
			if target := r.IncludePath(raw, file); target != "" {
				if data, err := read(target); err == nil {
					collect(target, string(data), depth+1)
				}
			}
		}

		for _, raw := range constructorIncludes(constructor, false) {
			if remaining[raw] > 0 {
				remaining[raw]--

				continue
			}

			if target := r.IncludePath(raw, file); target != "" {
				if data, err := read(target); err == nil {
					for _, name := range parser.ApplicationMethodNames(string(data)) {
						methods[name] = startupSource{target, ""}
					}
				}
			}
		}
	}
	collect(file, content, 0)

	return methods
}

func constructorDeclarationIncludes(content string) []string {
	return constructorIncludes(content, true)
}

func constructorIncludes(content string, outerOnly bool) []string {
	var includes []string

	for _, region := range parser.ClassifyRegions(content) {
		if region.Kind != parser.RegionScript {
			if !outerOnly {
				includes = append(includes, parser.ExtractIncludes(region.Text)...)
			}

			continue
		}

		sc := parser.NewScanner(region.Text)
		depth, owner := 0, 0
		previous := parser.TokEOF

		for {
			t := sc.NextSkipComments()
			if t.Kind == parser.TokEOF {
				break
			}

			if strings.EqualFold(t.Value, "component") {
				owner = 1
			}

			if t.Kind == parser.TokLBrace {
				depth++
			}

			if t.Kind == parser.TokRBrace {
				depth--
			}

			if strings.EqualFold(t.Value, "include") && (!outerOnly || depth == owner && (previous == parser.TokEOF || previous == parser.TokLBrace || previous == parser.TokSemicolon)) {
				value := sc.NextSkipComments()
				if value.Kind == parser.TokString && !strings.Contains(value.Value, "#") {
					includes = append(includes, strings.Trim(value.Value, `"'`))
				}

				previous = value.Kind

				continue
			}

			previous = t.Kind
		}
	}

	return includes
}
