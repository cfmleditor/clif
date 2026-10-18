package index

import (
	"testing"

	"go.lsp.dev/uri"
)

// keyMatches answers uriKey(u) == fk without building u's key; over escaped,
// unescaped, mixed-case, malformed and non-ASCII spellings it must agree.
func TestKeyMatchesAgreesWithURIKey(t *testing.T) {
	spellings := []string{
		"file:///w/My%20Svc.cfc", "file:///w/My Svc.cfc", "file:///W/MY%20SVC.cfc",
		"file:///w/my%2fsvc.cfc", "file:///w/my%2Fsvc.cfc", "file:///w/my/svc.cfc",
		"file:///w/%zzbad.cfc", "file:///w/trail%2", "file:///w/caf%C3%A9.cfc",
		"file:///A%20%zz.cfc", "file:///a%20%zz.cfc", "file:///A %zz.cfc",
		"file:///w/CAFÉ.cfc", "file:///w/café.cfc", "file:///w/Svc.cfc", "",
	}

	for _, u := range spellings {
		for _, k := range spellings {
			fk := uriKey(uri.URI(k))
			if got, want := keyMatches(u, fk), uriKey(uri.URI(u)) == fk; got != want {
				t.Errorf("keyMatches(%q, key of %q) = %v, uriKey says %v", u, k, got, want)
			}
		}
	}
}
