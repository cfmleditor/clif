package resolve

import (
	"slices"
	"testing"
)

func TestAncestorTraversalBoundsCyclesAndMatches(t *testing.T) {
	for _, tc := range []struct {
		name, start, target string
		limit               int
		want                []string
		matched             bool
	}{
		{"cycle", "a", "missing", 12, []string{"a", "b", "c"}, false},
		{"depth bound", "a", "c", 2, []string{"a", "b"}, false},
		{"last permitted ancestor", "a", "c", 3, []string{"a", "b", "c"}, true},
		{"current file match", "a", "a", 1, []string{"a"}, true},
		{"empty path", "", "a", 12, nil, false},
		{"zero limit", "a", "a", 0, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var visited []string

			next := map[string]string{"a": "b", "b": "c", "c": "a"}

			got := walkAncestors(tc.start, tc.limit, func(path string) (string, bool) {
				visited = append(visited, path)

				return next[path], path == tc.target
			})
			if got != tc.matched || !slices.Equal(visited, tc.want) {
				t.Fatalf("matched %v, visited %v; want %v, %v", got, visited, tc.matched, tc.want)
			}
		})
	}
}
