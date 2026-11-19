package resolve

// walkAncestors bounds a traversal and stops cycles. visit keeps the framework's
// own matching, missing-base and path-resolution policy, returning the next path
// or matched=true. A current-file match need not read that file's extends.
func walkAncestors(path string, limit int, visit func(string) (next string, matched bool)) bool {
	seen := map[string]bool{}

	for depth := 0; path != "" && !seen[path] && depth < limit; depth++ {
		seen[path] = true

		next, matched := visit(path)
		if matched {
			return true
		}

		path = next
	}

	return false
}
