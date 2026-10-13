package unresolved

import (
	"fmt"
	"strings"

	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
	"go.lsp.dev/uri"
)

// SpeculativeConnection records a proposed receiver fallback, not a verified fix.
// Line is the first baseline diagnostic's zero-based source line.
type SpeculativeConnection struct {
	File      string   `json:"file"`
	Line      uint32   `json:"line"`
	Caller    string   `json:"caller,omitempty"`
	Variable  string   `json:"variable"`
	Component string   `json:"component"`
	Policy    string   `json:"policy"`
	Ownership string   `json:"ownership"`
	Methods   []string `json:"methods"`
}

// scanSpeculative takes one strict snapshot and performs one guessed scan.
// Rebuilding isolates inference caches; guesses never mutate the strict resolver.
func scanSpeculative(fsys vfs.FS, files, targets []string, opt *Options) Report {
	strict := *opt
	strict.Speculative = false
	strict.SpeculativeNative = false
	strict.SpeculativeRanked = false
	strict.receiverRoles = opt.SpeculativeRanked
	strict.unlimitedCandidates = opt.SpeculativeRanked
	strict.SpeculativeReceivers = nil
	strict.Candidates = true
	strict.Verbose = nil
	baseline := Scan(fsys, files, targets, &strict)

	groups := map[resolve.SpeculativeReceiver]*speculativeGroup{}
	parsed := map[string][]parser.CallSite{}

	var order []resolve.SpeculativeReceiver

	for i := range baseline.Calls {
		call := &baseline.Calls[i]
		if call.Variable == "" || !strings.Contains(call.Reason, "no component ref") {
			continue
		}

		key := resolve.SpeculativeReceiver{File: call.File, Caller: call.Caller, Variable: strings.ToLower(call.Variable)}

		g := groups[key]
		if g == nil {
			g = &speculativeGroup{first: *call}
			groups[key] = g
			order = append(order, key)
		}

		method := call.Function
		if opt.SpeculativeRanked {
			role, ok := speculativeReceiverRole(speculativeSites(fsys, parsed, call.File), call)
			if !ok {
				g.unsafe = true
			} else {
				method = role
			}
		}

		g.methods = append(g.methods, method)
		g.addCandidates(call, opt.SpeculativeRanked)
	}

	guessed := *opt
	guessed.Speculative = false
	guessed.SpeculativeNative = false
	guessed.SpeculativeRanked = false
	guessed.SpeculativeReceivers = map[resolve.SpeculativeReceiver]string{}

	var connections []SpeculativeConnection

	for _, key := range order {
		g := groups[key]
		if g.unsafe {
			continue
		}

		policy := "sole-cfc-candidate"
		label := "sole candidate"

		if opt.SpeculativeRanked && !g.blocked {
			g.component = g.bestCandidate()
			policy, label = "ranked-cfc-role", "unique best receiver-role candidate"
		}

		if (!opt.Speculative && !opt.SpeculativeRanked) || g.blocked || g.component == "" {
			g.component = ""
			if opt.SpeculativeNative {
				g.component = nativetype.Guess(g.methods)
			}

			if g.component == "" {
				continue
			}

			policy = "sole-native-family"
			label = "sole native family"

			switch g.component {
			case nativetype.Type("string"), nativetype.Type("array"), nativetype.Type("struct"), nativetype.Type("stringbuilder"):
				policy = "sole-primitive-family"
				label = "primitive family fallback"
			}
		}

		guessed.SpeculativeReceivers[key] = g.component

		connections = append(connections, SpeculativeConnection{
			File: key.File, Line: g.first.Line, Caller: key.Caller, Variable: key.Variable,
			Component: g.component, Policy: policy, Ownership: "unproven", Methods: g.methods,
		})
		if opt.Verbose != nil {
			_, _ = fmt.Fprintf(opt.Verbose, "  ? speculative %s:%d [%s] %s -> %s (%s; origin unproven)\n", key.File, g.first.Line+1, key.Caller, key.Variable, g.component, label)
		}
	}

	rep := Scan(fsys, files, targets, &guessed)
	rep.SpeculativeConnections = connections
	rep.IndexTime += baseline.IndexTime
	rep.ScanTime += baseline.ScanTime

	return rep
}

// Without call byte offsets in a diagnostic, different chains on the same
// line may match it. Withhold guessing unless every matching site has one role.
func speculativeReceiverRole(sites []parser.CallSite, call *Call) (string, bool) {
	role := ""

	for i := range sites {
		site := &sites[i]
		if site.Line != call.Line || site.FuncName != call.Function || site.Variable != call.Variable || site.Caller != call.Caller {
			continue
		}

		method := site.FuncName
		if len(site.Chain) > 0 {
			if strings.HasPrefix(site.Chain[0], "$property:") {
				return "", false
			}

			method = parser.CallHopName(site.Chain[0])
		}

		if role != "" && !strings.EqualFold(role, method) {
			return "", false
		}

		role = method
	}

	return role, role != ""
}

type speculativeGroup struct {
	component  string
	first      Call
	blocked    bool
	methods    []string
	candidates map[string]int
	unsafe     bool
}

func speculativeSites(fsys vfs.FS, cache map[string][]parser.CallSite, file string) []parser.CallSite {
	if sites, ok := cache[file]; ok {
		return sites
	}

	var sites []parser.CallSite
	if data, err := fsys.ReadFile(file); err == nil {
		sites = parser.ParseWithOptions(uri.File(file), string(data), &parser.ParseOptions{ExtractCalls: true}).AllCalls()
	}

	cache[file] = sites

	return sites
}

func (g *speculativeGroup) addCandidates(call *Call, ranked bool) {
	if !ranked {
		if call.CandidateCount != 1 || len(call.Candidates) != 1 {
			g.blocked = true

			return
		}

		component := call.Candidates[0].File
		if g.component != "" && g.component != component {
			g.blocked = true
		}

		g.component = component

		return
	}

	if len(call.Candidates) == 0 || call.CandidateCount != len(call.Candidates) {
		g.blocked = true

		return
	}

	current := map[string]int{}

	for _, candidate := range call.Candidates {
		score := 1

		switch candidate.Confidence {
		case ConfidenceHigh:
			score = 3
		case ConfidenceMedium:
			score = 2
		}

		current[candidate.File] = score
	}

	if g.candidates == nil {
		g.candidates = current

		return
	}

	for path, score := range g.candidates {
		if next, ok := current[path]; ok {
			g.candidates[path] = score + next
		} else {
			delete(g.candidates, path)
		}
	}
}

func (g *speculativeGroup) bestCandidate() string {
	best, tie, component := 0, false, ""

	for path, score := range g.candidates {
		switch {
		case score > best:
			component, best, tie = path, score, false
		case score == best:
			tie = true
		}
	}

	if tie {
		return ""
	}

	return component
}
