package path

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// AssumedWheelsRoot applies the documented single-deployment Wheels public/
// convention. It is a layout assumption, not a declaration of the runtime web
// root. Explicit deployment config suppresses it; explicit editor/application
// mappings retain their usual priority. Reads include missing candidates so
// changing the layout invalidates discovery. See PROJECT-ASSUMPTIONS.md.
func AssumedWheelsRoot(dir string, read func(string) ([]byte, error)) string {
	for range 32 {
		for _, name := range []string{"server.json", ".cfconfig.json"} {
			if _, err := read(filepath.Join(dir, name)); err == nil {
				return ""
			}
		}

		public := filepath.Join(dir, "public")

		app, err := read(filepath.Join(public, "Application.cfc"))
		if err == nil {
			proxy, proxyErr := read(filepath.Join(public, "ApplicationProxy.cfc"))
			injector, injectorErr := read(filepath.Join(dir, "vendor", "wheels", "Injector.cfc"))

			binder, binderErr := read(filepath.Join(dir, "vendor", "wheels", "Bindings.cfc"))
			if proxyErr == nil && injectorErr == nil && binderErr == nil && len(injector) > 0 && len(binder) > 0 && rootTokens(string(proxy)) == rootTokens(`component extends="Application" {}`) && assumedWheelsBootstrap(string(app)) && parser.MappingSources(filepath.Join(public, "Application.cfc"), nil, read)["wheels"] == filepath.Join(dir, "vendor", "wheels") {
				return public
			}
			// An encountered public application belongs to the nearest project;
			// failure does not borrow an enclosing project's deployment context.
			return ""
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return ""
}

func assumedWheelsBootstrap(content string) bool {
	constructor, methods := parser.ApplicationSections(content)
	if !strings.Contains(rootTokens(constructor), rootTokens(`this.wheels.rootPath=GetDirectoryFromPath(GetCurrentTemplatePath());`)) {
		return false
	}
	// The vendor mapping must actually point to the expected sibling tree.
	// MappingSources has no ancestry dependency for this literal constructor.
	return outerRootStatementPair(methods["onapplicationstart"])
}

func outerRootStatementPair(content string) bool {
	sc := parser.NewScanner(content)
	depth := 0
	previous := parser.TokEOF
	needle := rootTokens(`application.wheelsdi=new wheels.Injector("wheels.Bindings"); application.wo=application.wheelsdi.getInstance("global");`)

	for {
		t := sc.NextSkipComments()
		if t.Kind == parser.TokEOF {
			return false
		}

		if t.Kind == parser.TokLBrace {
			depth++
		}

		if t.Kind == parser.TokRBrace {
			depth--
		}

		if depth == 1 && (previous == parser.TokLBrace || previous == parser.TokRBrace || previous == parser.TokSemicolon) && strings.EqualFold(t.Value, "application") {
			cursor := *sc

			candidate := rootTokens(t.Value)
			for len(candidate) < len(needle) {
				next := cursor.NextSkipComments()
				if next.Kind == parser.TokEOF {
					break
				}

				candidate += rootTokens(next.Value)
			}

			if candidate == needle {
				return true
			}
		}

		previous = t.Kind
	}
}

func rootTokens(content string) string {
	sc := parser.NewScanner(content)

	var out strings.Builder

	for {
		t := sc.NextSkipComments()
		if t.Kind == parser.TokEOF {
			return out.String()
		}

		value := t.Value
		if t.Kind == parser.TokString && len(value) > 1 {
			value = `"` + value[1:len(value)-1] + `"`
		}

		out.WriteString(strings.ToLower(value))
		out.WriteByte('\t')
	}
}

func bootstrapMappings(appDir string, initial map[string]string, read func(string) ([]byte, error)) map[string]string {
	out := loadServerMappings(appDir)
	for key, value := range AssumedProjectMappings(appDir, read) {
		explicit := false

		for configured := range initial {
			if strings.EqualFold(strings.Trim(configured, "/"), key) {
				explicit = true

				break
			}
		}

		if explicit {
			continue
		}

		if _, explicit := out[key]; !explicit {
			out[key] = value
		}
	}

	for key := range initial {
		if strings.Trim(key, "/") == "" {
			return out
		}
	}

	if _, exists := out[""]; !exists {
		if root := AssumedWheelsRoot(appDir, read); root != "" {
			out[""] = root
		}
	}

	return out
}
