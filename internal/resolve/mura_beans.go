package resolve

import (
	"regexp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// assumedMuraBeans applies the documented single-host Mura factory convention.
// Only actual IDs from one discovered startup policy receive rules; unknown IDs
// retain their ordinary diagnostics. Explicit rules precede these defaults.
func (r *Resolver) assumedMuraBeans(configured map[string]string) []parser.Resolver {
	if len(configured) > 0 || len(r.BeanPaths) > 0 || len(r.StartupFiles) > 0 {
		return nil
	}

	enabled := false

	for i := range r.Resolvers {
		rule := &r.Resolvers[i]
		if rule.NameOnly && strings.EqualFold(rule.Resolve, "mura.MuraScope") {
			enabled = true

			break
		}
	}

	if !enabled {
		return nil
	}

	roots := map[string]bool{}

	read := TrackDiscoveryFS(r.fs(), r.Discovery).ReadFile
	for _, root := range r.WorkspaceFolders {
		if mura := cfpath.AssumedProjectMappings(root, read)["mura"]; mura != "" {
			roots[mura] = true
		}
	}

	if len(roots) != 1 {
		return nil
	}

	all := r.diPolicySet()

	var policy *diPolicy

	for i := range all {
		p := &all[i]
		if len(p.roots) != 1 || !roots[p.roots[0]] {
			continue
		}

		if policy != nil {
			return nil
		}

		policy = p
	}

	if policy == nil || policy.blocked {
		return nil
	}

	ids := map[string]bool{}
	for id := range policy.beans {
		ids[id] = true
	}

	for id := range policy.entries {
		ids[id] = true
	}

	keys := make([]string, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}

	slices.Sort(keys)

	var rules []parser.Resolver

	for _, id := range keys {
		component := policy.dependency(id, "", r, map[string]bool{}, false)
		if component == "" || assumedMuraExcluded(component) {
			continue
		}

		rules = append(rules, parser.Resolver{Match: `(?i)(?:^|\.)getBean\(\s*(?:beanName\s*[=:]\s*)?["']` + regexp.QuoteMeta(id) + `["']\s*(?:,[^()]*)?\)$`, Resolve: component, Prefix: "getBean"})
	}

	return rules
}

// DI/1 always excludes infrastructure components, in addition to the caller's
// configured substring exclusions. Do not publish these as ordinary bean IDs.
func assumedMuraExcluded(component string) bool {
	path := strings.ToLower(strings.ReplaceAll(component, `\`, "/"))
	for _, part := range []string{"/web-inf", "/application.cfc", "/myapplication.cfc", "framework.cfc", "ioc.cfc", "/framework/"} {
		if strings.Contains(path, part) {
			return true
		}
	}

	return false
}
