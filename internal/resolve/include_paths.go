package resolve

import "github.com/cfmleditor/clif/internal/parser"

// includePaths remembers the include paths of the texts asked about most
// recently. They are a function of the text alone, and withIncludes asked for
// the same template's once for each place that walks the includes of a page
// (the Wheels globals, the helpers), each time reading the whole file's
// references again: 1,071 of cfwheels' 1,080 repeat reads of one were these.
var includePaths = &textCache[[]string]{size: stringsSize}

// includePathsOf is parser.ExtractIncludes(text), remembered. What it returns
// is shared and must not be changed.
func includePathsOf(text string) []string {
	return includePaths.of(text, parser.ExtractIncludes)
}

// stringsSize is roughly what a list of strings holds beyond its text.
func stringsSize(list []string) int {
	n := 24 + 16*cap(list)
	for _, s := range list {
		n += len(s)
	}

	return n
}
