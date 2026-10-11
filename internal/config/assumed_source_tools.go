package config

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

// These tooling contexts supplement one recognized project owner. The presets
// are project-wide conventions; explicit editor configuration remains decisive.
func assumedSourceTools(owner string, read func(string) ([]byte, error), list func(string) ([]fs.DirEntry, error)) []string {
	var out []string

	for _, relative := range []string{"migrations", "modules/contentbox/migrations"} {
		if toolingDirectory(filepath.Join(owner, filepath.FromSlash(relative)), 0, read, list, migrationEntryPoints) {
			out = append(out, "cfmigrations")

			break
		}
	}

	if data, err := read(filepath.Join(owner, "task.cfc")); err == nil && commandBoxEntryPoint(parser.ParseWithOptions(uri.File(filepath.Join(owner, "task.cfc")), string(data), &parser.ParseOptions{ExtractCalls: true})) {
		return append(out, "commandbox")
	}

	for _, folder := range []struct {
		relative string
		depth    int
	}{{"build", 0}, {"commands", 5}, {"cli/src/commands", 5}} {
		if toolingDirectory(filepath.Join(owner, filepath.FromSlash(folder.relative)), folder.depth, read, list, commandBoxEntryPoint) {
			out = append(out, "commandbox")

			break
		}
	}

	return out
}

// Bound both directory traversal and source reads. DiscoveryFS observes missing
// directories as well as files so adding a tooling entry point invalidates it.
func toolingDirectory(root string, maxDepth int, read func(string) ([]byte, error), list func(string) ([]fs.DirEntry, error), match func(*parser.ParseResult) bool) bool {
	type folder struct {
		path  string
		depth int
	}

	queue := []folder{{root, 0}}
	files := 0

	for i := 0; i < len(queue) && i < 128; i++ {
		dir := queue[i]

		entries, err := list(dir.path)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			file := filepath.Join(dir.path, entry.Name())
			if entry.IsDir() {
				if dir.depth < maxDepth && len(queue) < 128 {
					queue = append(queue, folder{file, dir.depth + 1})
				}

				continue
			}

			if !strings.EqualFold(filepath.Ext(entry.Name()), ".cfc") {
				continue
			}

			files++
			if files > 128 {
				return false
			}

			data, err := read(file)
			if err != nil {
				continue
			}

			if match(parser.ParseWithOptions(uri.File(file), string(data), &parser.ParseOptions{ExtractCalls: true})) {
				return true
			}
		}
	}

	return false
}

func migrationEntryPoints(pr *parser.ParseResult) bool {
	found := map[string]bool{}

	for i := range pr.Funcs {
		fn := &pr.Funcs[i]

		name := strings.ToLower(fn.Name)
		if name != "up" && name != "down" || len(fn.Arguments) < 2 {
			continue
		}

		schema, query := fn.Arguments[0], fn.Arguments[1]
		if !strings.EqualFold(schema.Name, "schema") || !strings.EqualFold(query.Name, "qb") && !strings.EqualFold(query.Name, "query") {
			continue
		}

		if !toolingArgumentType(schema.Type, "qb.models.Schema.SchemaBuilder") || !toolingArgumentType(query.Type, "qb.models.Query.QueryBuilder") {
			continue
		}

		found[name] = true
	}

	return found["up"] && found["down"]
}

func toolingArgumentType(typ, component string) bool {
	return typ == "" || strings.EqualFold(typ, "any") || strings.EqualFold(typ, component)
}

func commandBoxEntryPoint(pr *parser.ParseResult) bool {
	hasRun := false

	for i := range pr.Funcs {
		if strings.EqualFold(pr.Funcs[i].Name, "run") {
			hasRun = true

			break
		}
	}

	if !hasRun {
		return false
	}

	if strings.HasPrefix(strings.ToLower(pr.Extends), "commandbox.") {
		return true
	}

	if pr.Declares("getCWD") {
		return false
	}

	calls := pr.AllCalls()
	for i := range calls {
		call := &calls[i]
		if call.Variable == "" && strings.EqualFold(call.FuncName, "getCWD") || (strings.EqualFold(call.Variable, "shell") || strings.EqualFold(call.Variable, "variables.shell")) && strings.EqualFold(call.FuncName, "getUtil") {
			return true
		}
	}

	return false
}
