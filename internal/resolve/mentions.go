package resolve

import "github.com/cfmleditor/clif/internal/parser"

// mentions reports whether text holds literal, case-insensitively. The
// readers run over whole files and function bodies each need a literal (a
// name, `prc.`, `this.memento`) that a file without it cannot hold, so asking
// this first skips the reader where it would find nothing, which is almost
// everywhere, at the cost of one vectorised search.
func mentions(text, literal string) bool {
	return parser.IndexFold(text, literal) >= 0
}
