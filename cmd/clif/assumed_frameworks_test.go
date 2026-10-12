package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/daemon"
	"github.com/cfmleditor/clif/internal/parser"
	"github.com/cfmleditor/clif/internal/unresolved"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestUnconfiguredContentBoxAssumesFrameworkContext(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{"Application.cfc": `component {this.mappings["/contentbox"]=expandPath("./modules/contentbox");}`, "modules/contentbox/ModuleConfig.cfc": `component {}`, "modules/contentbox/models/content/ContentService.cfc": `component {}`, "handlers/Main.cfc": `component {function index(event){event.getValue("x");event.definitelyMissing();}}`}

	var sources []string

	for name, content := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		sources = append(sources, p)
	}

	opt := unresolvedOptions(nil, []string{root}, &unresolvedFlags{})

	report := unresolved.Scan(vfs.OS{}, sources, []string{filepath.Join(root, "handlers", "Main.cfc")}, opt)
	for _, call := range report.Calls {
		if call.Function == "getValue" {
			t.Fatalf("framework context not inferred: %+v", report.Calls)
		}
	}

	missing := false

	for _, call := range report.Calls {
		if call.Function == "definitelyMissing" {
			missing = true
		}
	}

	if !missing {
		t.Fatal("unknown method was accepted")
	}

	explicit := unresolvedOptions(&daemon.Config{Path: filepath.Join(root, ".clif.json")}, []string{root}, &unresolvedFlags{})
	if explicit.Stubs != nil || explicit.ImplicitExtends != nil {
		t.Fatal("config did not disable inferred presets")
	}
}

func TestUnconfiguredMuraAssumesScopeContext(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"Application.cfc":                     `component {this.mappings["/mura"]=expandPath("./core/mura");}`,
		"core/mura/configBean.cfc":            `component {}`,
		"core/mura/bean/beanFactory.cfc":      `component {}`,
		"core/appcfc/applicationSettings.cfm": `<cfscript></cfscript>`,
		"core/mura/MuraScope.cfc":             `component {function known(){}}`,
		"index.cfm":                           `<cfscript>$.known(); $.definitelyMissing();</cfscript>`,
	}

	var sources []string

	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}

		sources = append(sources, p)
	}

	opt := unresolvedOptions(nil, []string{root}, &unresolvedFlags{})
	report := unresolved.Scan(vfs.OS{}, sources, []string{filepath.Join(root, "index.cfm")}, opt)
	missing := false

	for _, call := range report.Calls {
		if call.Function == "known" {
			t.Fatalf("scope context missing: %+v", call)
		}

		if call.Function == "definitelyMissing" {
			missing = true
		}
	}

	if !missing {
		t.Fatal("unknown method accepted")
	}
}

func TestMuraAdminContextDefaultsCheckMethodsAndAllowOverrides(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"Application.cfc":                     `component {this.mappings["/mura"]=expandPath("./core/mura");}`,
		"core/mura/configBean.cfc":            `component {}`,
		"core/mura/bean/beanFactory.cfc":      `component {}`,
		"core/appcfc/applicationSettings.cfm": `<cfscript></cfscript>`,
		"core/mura/utility.cfc":               `component {function serviceKnown(){}}`,
		"core/mura/user/userBean.cfc":         `component {function standard(){}}`,
		"Override.cfc":                        `component {function replacement(){}}`,
		"index.cfm":                           `<cfscript>rc.userBean.standard();rc.userBean.replacement();rc.userBean.definitelyMissing();rc.bean.standard();application.utility.serviceKnown();application.utility.serviceMissing();</cfscript>`,
	}

	var sources []string

	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}

		sources = append(sources, p)
	}

	for _, override := range []bool{false, true} {
		opt := unresolvedOptions(nil, []string{root}, &unresolvedFlags{})
		if override {
			opt.Resolvers = append([]parser.Resolver{{Match: `(?i)^rc\.userBean$`, Resolve: filepath.Join(root, "Override.cfc"), Prefix: "rc.userBean", Anchored: true, NameOnly: true}}, opt.Resolvers...)
		}

		report := unresolved.Scan(vfs.OS{}, sources, []string{filepath.Join(root, "index.cfm")}, opt)

		seen := map[string]int{}
		for _, call := range report.Calls {
			seen[call.Function]++
		}

		wantStandard, wantReplacement := 1, 1
		if override {
			wantStandard++
			wantReplacement = 0
		} else {
			wantStandard = 1
		}

		if seen["standard"] != wantStandard || seen["replacement"] != wantReplacement || seen["definitelyMissing"] != 1 || seen["serviceKnown"] != 0 || seen["serviceMissing"] != 1 {
			t.Fatalf("override=%v: %+v", override, report.Calls)
		}
	}
}
