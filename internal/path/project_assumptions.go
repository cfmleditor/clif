package path

import (
	"hash/maphash"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/parser"
)

// AssumedProjectMappings makes a project's own declared library namespace
// available to its nested source files. The documented assumption is that
// those library components run in this standard project's host context, even
// when their physical directory has an otherwise empty/restrictive Application.
// It does not import lifecycle methods, beans or shared variables from that host.
func AssumedProjectMappings(dir string, read func(string) ([]byte, error)) map[string]string {
	out := map[string]string{}
	seen := map[string]bool{}

	for range 32 {
		for _, name := range []string{"server.json", ".cfconfig.json"} {
			if _, err := read(filepath.Join(dir, name)); err == nil {
				return out
			}
		}

		if !seen["admin"] {
			if admin := assumedLuceeAdmin(dir, read); admin != "" {
				seen["admin"] = true
				out["admin"] = admin
			}
		}

		// A Mura plugin's checkout: the mappings Mura's plugin manager writes
		// for it, from its plugin/config.xml. The nearest plugin owns a name.
		if plugin, ok := AssumedMuraPlugin(dir, read); ok {
			for _, name := range slices.Sorted(maps.Keys(plugin.Mappings)) {
				if key := strings.ToLower(name); !seen[key] {
					seen[key] = true
					out[name] = plugin.Mappings[name]
				}
			}
		}

		for _, profile := range []struct {
			namespace, source, target string
			markers                   []string
		}{
			{"testbox", "tests/Application.cfc", "", []string{"system/BaseSpec.cfc", "system/Assertion.cfc"}},
			{"coldbox", "tests/Application.cfc", "", []string{"system/Bootstrap.cfc", "system/web/Controller.cfc"}},
			{"framework", "Application.cfc", "framework", []string{"framework/one.cfc", "framework/ioc.cfc"}},
			{"mura", "Application.cfc", "core/mura", []string{"core/mura/configBean.cfc", "core/mura/bean/beanFactory.cfc", "core/appcfc/applicationSettings.cfm"}},
			{"murawrm", "Application.cfc", "", []string{"core/mura/configBean.cfc", "core/mura/bean/beanFactory.cfc", "core/appcfc/applicationSettings.cfm"}},
			{"contentbox", "Application.cfc", "modules/contentbox", []string{"modules/contentbox/ModuleConfig.cfc", "modules/contentbox/models/content/ContentService.cfc"}},
			// Preside's own checkout: its test suite maps /preside to the root,
			// as the Bootstrap every Preside application extends does.
			{"preside", "tests/Application.cfc", "", []string{"system/Bootstrap.cfc", "system/services/PresideSuperClass.cfc"}},
			// CommandBox's own checkout: its tests map /commandbox to src/cfml,
			// where commandbox.system.* lives.
			{"commandbox", "tests/Application.cfc", "src/cfml", []string{"src/cfml/system/BaseCommand.cfc", "src/cfml/system/Shell.cfc"}},
			// The WireBox, LogBox and CacheBox it ships under system/wirebox,
			// which its own source names wirebox.system.*.
			{"wirebox", "tests/Application.cfc", "src/cfml/system/wirebox", []string{"src/cfml/system/BaseCommand.cfc", "src/cfml/system/wirebox/system/ioc/Injector.cfc"}},
		} {
			if seen[profile.namespace] {
				continue
			}

			known := true

			for _, marker := range profile.markers {
				if _, err := read(filepath.Join(dir, marker)); err != nil {
					known = false

					break
				}
			}

			if !known {
				continue
			}
			// A recognized nearer layout owns this namespace even if its own
			// mapping is different or cannot be evaluated.
			seen[profile.namespace] = true
			// Read the declaring application's own mapping expressions, without
			// calling bootstrapMappings and recursively manufacturing defaults.
			mapping := assumedSourceMappings(filepath.Join(dir, profile.source), read)[profile.namespace]
			if mapping == filepath.Join(dir, profile.target) {
				out[profile.namespace] = mapping
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return out
}

// assumedSourceMemo is assumedSourceMappings' answers by source file.
var assumedSourceMemo sync.Map

// sourceRead is one file a mapping walk read, and what it held.
type sourceRead struct {
	path string
	ok   bool
	sum  uint64
}

type assumedSources struct {
	reads    []sourceRead
	mappings map[string]string
}

var sourceSeed = maphash.MakeSeed()

func readStamp(path string, data []byte, err error) sourceRead {
	if err != nil {
		return sourceRead{path: path}
	}

	return sourceRead{path: path, ok: true, sum: maphash.Bytes(sourceSeed, data)}
}

// assumedSourceMappings is parser.MappingSources(file, nil, read), kept by
// file. Profiles share a source (mura and murawrm both read Application.cfc,
// testbox, coldbox and preside tests/Application.cfc), and every directory
// walk from below a project reads the same ones again: each walk followed the
// file's includes in full.
//
// An answer is kept with what each file the walk read held, and is used again
// only when reading them now gives the same: read is called again for each,
// so a caller that records what it reads (Discovery) sees them as it would,
// and an edit to any of them is noticed.
func assumedSourceMappings(file string, read func(string) ([]byte, error)) map[string]string {
	if held, ok := assumedSourceMemo.Load(file); ok {
		memo, _ := held.(*assumedSources)

		fresh := true

		for _, r := range memo.reads {
			data, err := read(r.path)
			if readStamp(r.path, data, err) != r {
				fresh = false

				break
			}
		}

		if fresh {
			return memo.mappings
		}
	}

	var (
		mu    sync.Mutex
		reads []sourceRead
	)

	recording := func(path string) ([]byte, error) {
		data, err := read(path)

		stamp := readStamp(path, data, err)

		mu.Lock()

		reads = append(reads, stamp)
		mu.Unlock()

		return data, err
	}

	mappings := parser.MappingSources(file, nil, recording)
	assumedSourceMemo.Store(file, &assumedSources{reads: reads, mappings: mappings})

	return mappings
}
