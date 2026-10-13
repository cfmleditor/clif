package server

import "github.com/cfmleditor/clif/internal/config"

// Settings is the per-session configuration a Server needs, gathered in one
// place so that every way of starting a session applies the same set.
//
// It exists because they did not. Daemon mode serves its first editor over
// stdio and every later editor over a socket, and the two were configured by
// separate code applying separate subsets of a twelve-argument parameter list.
// expressionMappings and servicePropertyResolvers reached the first editor and
// no other, so the same workspace resolved component paths differently
// depending on which window a file was opened in — and nothing reported it,
// because an unset mapping is indistinguishable from a genuinely dynamic
// expression.
//
// The configuration itself is [config.Resolved], embedded: a new key reaches every
// session by being added there once, and not by also remembering a field here
// and a line in Apply.
type Settings struct {
	config.Resolved
	// ConfigPath is the .clif.json these settings were read from, empty
	// when the daemon's own walk found none. It is what tells a session
	// whether its configuration has already been discovered for it — see
	// handleInitialize.
	ConfigPath string
}

// Apply copies the settings onto a freshly created Server.
func (set *Settings) Apply(s *Server) {
	s.ConfigPath = set.ConfigPath

	// On a server nothing has configured yet, the layered application of a
	// resolved config (fill what is unset, append resolvers) is an assignment,
	// and it is the one place the fields are copied.
	s.applyConfig(&set.Resolved)
}
