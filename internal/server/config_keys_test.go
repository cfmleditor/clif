package server

import (
	"testing"

	"github.com/cfmleditor/clif/internal/config"
)

// Both keys are parsed by config.JSON, merged by config.Merge and — since this
// change — carried by config.Resolve, but applyConfig is the last hop into the
// Server. A key it does not copy is a key the user can set, the loader can
// read, and nothing will ever act on.
func TestInitializeAppliesExpressionMappings(t *testing.T) {
	s := initializeWithConfig(t, `{
		"mappings": {"models": "./src/models"},
		"expressionMappings": {"#VARIABLES._core#": "packages.tass.core."}
	}`)

	if got := s.ExpressionMappings["#VARIABLES._core#"]; got != "packages.tass.core." {
		t.Errorf("expressionMappings not applied to the server: got %q", got)
	}
}

func TestInitializeAppliesServicePropertyResolvers(t *testing.T) {
	s := initializeWithConfig(t, `{
		"mappings": {"models": "./src/models"},
		"servicePropertyResolvers": {"package": "packages.${name}"}
	}`)

	if got := s.ServicePropertyResolvers["package"]; got != "packages.${name}" {
		t.Errorf("servicePropertyResolvers not applied to the server: got %q", got)
	}
}

func TestInitializeAppliesCallerInferenceLimits(t *testing.T) {
	s := initializeWithConfig(t, `{"callerInferenceLimits":{"maxCandidateFiles":12,"maxCallSites":4}}`)
	if s.CallerInferenceLimits.MaxCandidateFiles != 12 || s.CallerInferenceLimits.MaxCallSites != 4 {
		t.Fatalf("limits did not reach session: %+v", s.CallerInferenceLimits)
	}

	// Restoration is tested on an idle session; initialization starts indexing.
	s = &Server{CallerInferenceLimits: s.CallerInferenceLimits}
	prev := s.optionalBlocks()
	s.CallerInferenceLimits.MaxCallSites = 0
	s.restoreUnmentionedBlocks(&config.JSON{}, &prev)

	if s.CallerInferenceLimits.MaxCallSites != 4 {
		t.Fatal("omitted block lost limits")
	}

	s.CallerInferenceLimits.MaxCallSites = 0
	s.restoreUnmentionedBlocks(&config.JSON{CallerInferenceLimits: &config.CallerInferenceLimits{}}, &prev)

	if s.CallerInferenceLimits.MaxCallSites != 0 {
		t.Fatal("explicit empty block failed to reset limits")
	}
}
