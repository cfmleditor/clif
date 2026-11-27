package lint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/vfs"
)

// rcEntry represents one entry in a .cflintrc "includes" or "excludes" array.
type rcEntry struct {
	Code     string `json:"code"`
	Severity string `json:"severity,omitempty"`
}

// rcParam represents one `{name, value}` entry in a `.cflintrc` rule's parameter list.
type rcParam struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// rcRule represents one entry in a `.cflintrc` "rule" array.
type rcRule struct {
	Name      string    `json:"name"`
	ClassName string    `json:"className"`
	Parameter []rcParam `json:"parameter"`
}

// rcFile is the on-disk JSON schema of a .cflintrc file.
type rcFile struct {
	Rule          []rcRule       `json:"rule"`
	Includes      []rcEntry      `json:"includes"`
	Excludes      []rcEntry      `json:"excludes"`
	InheritParent *bool          `json:"inheritParent"`
	Parameters    map[string]any `json:"parameters"`
	ExcludeFiles  []string       `json:"excludeFiles"`
}

// RCConfig is the effective .cflintrc rule configuration for a directory after
// walking upward and merging parent configs (stopping at inheritParent: false).
type RCConfig struct {
	// HasIncludes is true when at least one config in the chain specified a
	// non-empty "includes" list (meaning only listed rules are active).
	HasIncludes bool
	// Includes maps uppercased rule code -> optional severity override ("" for default).
	Includes map[string]string
	// Excludes holds uppercased rule codes explicitly disabled.
	Excludes map[string]bool
	// Parameters holds merged rule parameters (child overrides parent).
	Parameters map[string]any
	// ExcludeFiles holds compiled excludeFiles regexes.
	ExcludeFiles []*regexp.Regexp
}

// LoadRC resolves the effective .cflintrc for filePath by walking upward from
// filepath.Dir(filePath) until a config sets "inheritParent": false or the
// filesystem root is reached.
func LoadRC(filePath string) (*RCConfig, error) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	return LoadRCForDir(filepath.Dir(abs))
}

// LoadRCForDir resolves the effective .cflintrc for dir.
func LoadRCForDir(dir string) (*RCConfig, error) {
	var chain []rcFile

	for cur := dir; ; {
		rc, found, err := readRCFile(filepath.Join(cur, cflint.ConfigFile))
		if err != nil {
			return nil, err
		}

		if found {
			chain = append(chain, rc)
			if rc.InheritParent != nil && !*rc.InheritParent {
				break
			}
		}

		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}

		cur = parent
	}

	cfg := &RCConfig{
		Includes:   make(map[string]string),
		Excludes:   make(map[string]bool),
		Parameters: make(map[string]any),
	}

	// Merge parent -> child so child entries override parent entries.
	for i := range slices.Backward(chain) {
		rc := &chain[i]
		if len(rc.Includes) > 0 {
			cfg.HasIncludes = true

			for _, inc := range rc.Includes {
				code := strings.ToUpper(strings.TrimSpace(inc.Code))
				if code == "" {
					continue
				}

				cfg.Includes[code] = strings.ToUpper(strings.TrimSpace(inc.Severity))
				delete(cfg.Excludes, code)
			}
		}

		for _, exc := range rc.Excludes {
			code := strings.ToUpper(strings.TrimSpace(exc.Code))
			if code == "" {
				continue
			}

			cfg.Excludes[code] = true
		}

		maps.Copy(cfg.Parameters, rc.Parameters)

		for _, r := range rc.Rule {
			ruleKey := strings.TrimSpace(r.Name)
			if ruleKey == "" {
				ruleKey = strings.TrimSpace(r.ClassName)
			}

			if ruleKey == "" {
				continue
			}

			for _, p := range r.Parameter {
				pName := strings.TrimSpace(p.Name)
				if pName != "" {
					cfg.Parameters[ruleKey+"."+pName] = p.Value
				}
			}
		}

		for _, pat := range rc.ExcludeFiles {
			if strings.TrimSpace(pat) == "" {
				continue
			}

			re, err := regexp.Compile(pat)
			if err != nil {
				return nil, fmt.Errorf(".cflintrc invalid excludeFiles regex %q: %w", pat, err)
			}

			cfg.ExcludeFiles = append(cfg.ExcludeFiles, re)
		}
	}

	return cfg, nil
}

func readRCFile(path string) (rcFile, bool, error) {
	data, err := vfs.ReadRegular(path)
	if errors.Is(err, os.ErrNotExist) {
		return rcFile{}, false, nil
	}

	if err != nil {
		return rcFile{}, false, fmt.Errorf("%s: %w", path, err)
	}

	// Strip optional UTF-8 BOM, matching cflint.CheckConfigs.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	var rc rcFile
	if err := json.Unmarshal(data, &rc); err != nil {
		return rcFile{}, false, fmt.Errorf("%s is not valid JSON (%w)", path, err)
	}

	return rc, true, nil
}

// RuleEnabled reports whether rule code is enabled by this .cflintrc.
func (c *RCConfig) RuleEnabled(code string) bool {
	if c == nil {
		return true
	}

	k := strings.ToUpper(strings.TrimSpace(code))
	if c.Excludes[k] {
		return false
	}

	if c.HasIncludes {
		_, ok := c.Includes[k]

		return ok
	}

	return true
}

// RuleSeverity returns the configured severity override for code, or
// defaultSev when none is specified.
func (c *RCConfig) RuleSeverity(code, defaultSev string) string {
	if c != nil {
		if sev := c.Includes[strings.ToUpper(strings.TrimSpace(code))]; sev != "" {
			return sev
		}
	}

	return defaultSev
}

// ParamString returns a string parameter value (or "" if absent or non-string).
func (c *RCConfig) ParamString(key string) string {
	if c == nil {
		return ""
	}

	v, ok := c.Parameters[key]
	if !ok {
		return ""
	}

	s, _ := v.(string)

	return s
}

// FileExcluded reports whether filePath matches any excludeFiles pattern.
func (c *RCConfig) FileExcluded(filePath string) bool {
	if c == nil || len(c.ExcludeFiles) == 0 {
		return false
	}

	slashed := filepath.ToSlash(filePath)
	for _, re := range c.ExcludeFiles {
		if re.MatchString(filePath) || re.MatchString(slashed) {
			return true
		}
	}

	return false
}
