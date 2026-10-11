package config

// CallerInferenceLimits is an optional batch inference policy. Positive values
// cap work; zero or negative values are unlimited. Exceeding a limit withholds
// the inferred type rather than accepting a partial set of callers.
// The block replaces an inherited policy as a whole; {} resets all limits.
type CallerInferenceLimits struct {
	MaxCandidateFiles   int `json:"maxCandidateFiles,omitempty"`
	MaxCallerFiles      int `json:"maxCallerFiles,omitempty"`
	MaxCallSites        int `json:"maxCallSites,omitempty"`
	MaxConstructorFiles int `json:"maxConstructorFiles,omitempty"`
}
