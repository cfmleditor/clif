package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/frameworkapi"
)

// TestAWireBoxIDIsWhatItsModuleRegisters: ColdBox registers a module's models
// under its modelNamespace — the module's name unless ModuleConfig.cfc says
// otherwise — so `UserService@users` is the one in the users module, not the
// nearest file of that name. A binder's `map( id ).to( path )` is exact,
// `#moduleMapping#` being the module. A module's cfmapping is a mapping to it.
// And an id of a module the workspace does not have is the module not being
// installed, which is not a finding: cbstorages, cbmailservices and the rest
// were reported against the corpus.
func TestAWireBoxIDIsWhatItsModuleRegisters(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"modules/users/ModuleConfig.cfc": `component {
	this.cfmapping = "usersModule";
	function onLoad() {
		binder.map( "Mailer@users" )
			.to( "#moduleMapping#.models.mail.SmtpMailer" );
	}
}`,
		"modules/users/models/UserService.cfc":     `component { function list() {} }`,
		"modules/users/models/mail/SmtpMailer.cfc": `component { function send() {} }`,
		"modules/users/lib/Util.cfc":               `component { function help() {} }`,
		"modules/admin/ModuleConfig.cfc":           `component { this.modelNamespace = "cbadmin"; }`,
		"modules/admin/models/UserService.cfc":     `component { function adminOnly() {} }`,
		"config/WireBox.cfc":                       `component { function configure() { map( "Clock" ).to( "lib.SystemClock" ); } }`,
		"lib/SystemClock.cfc":                      `component { function now() {} }`,
		"handlers/Main.cfc": `component {
	property name="users" inject="UserService@users";
	property name="admin" inject="id:UserService@cbadmin";
	property name="mailer" inject="Mailer@users";
	property name="storage" inject="RequestStorage@cbstorages";
	property name="clock" inject="Clock";
	function index() {
		users.list();
		users.adminOnly();
		admin.adminOnly();
		mailer.send();
		storage.getSessionVar( "a" );
		clock.now();
		var u = new usersModule.lib.Util();
		u.help();
	}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "handlers/Main.cfc"), map[string]string{
		"users.list":            "",
		"users.adminOnly":       "method 'adminOnly' not found in UserService@users",
		"admin.adminOnly":       "",
		"mailer.send":           "",
		"storage.getSessionVar": "",
		"clock.now":             "",
		"u.help":                "",
	})
}

func TestModuleModelDiscoveryEligibility(t *testing.T) {
	for _, tc := range []struct {
		name, settings, target string
		enabled                bool
	}{
		{"default", "", "models", true},
		{"disabled", `this.autoMapModels=false;`, "models", false},
		{"computed flag", `this.autoMapModels=runtime();`, "models", false},
		{"conditional flag", `if (runtime()) this.autoMapModels=true;`, "models", false},
		{"custom directory", `function configure(){conventions={modelsLocation:"domain"};}`, "domain", true},
		{"computed directory", `function configure(){conventions={modelsLocation:settings.path};}`, "models", false},
		{"empty namespace", `this.modelNamespace="";`, "models", false},
		{"child injector", `this.moduleInjector=true;`, "models", false},
		{"unused assignment", `function unused(){this.autoMapModels=false;this.modelNamespace="other";}`, "models", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"modules/users/ModuleConfig.cfc":   `component {` + tc.settings + `}`,
				"modules/users/models/Service.cfc": `component {function old(){}}`,
				"modules/users/domain/Service.cfc": `component {function custom(){}}`,
				"other/Service.cfc":                `component {function unrelated(){}}`,
			})

			r := &Resolver{}
			// Populate the same index a batch scan supplies before ID resolution.
			reasonsWith(t, r, dir, "modules/users/models/Service.cfc")
			got := r.ComponentPath("Service@users", dir)

			want := ""
			if tc.enabled {
				want = filepath.Join(dir, "modules", "users", tc.target, "Service.cfc")
			}

			if got != want {
				t.Fatalf("%q, want %q", got, want)
			}
		})
	}
}

func TestModuleDiscoveryDoesNotPickDuplicateBasenames(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"modules/users/ModuleConfig.cfc":     `component {}`,
		"modules/users/models/a/Service.cfc": `component {}`,
		"modules/users/models/b/Service.cfc": `component {}`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "modules/users/models/a/Service.cfc")

	if got := r.ComponentPath("Service@users", dir); got != "" {
		t.Fatal("unproven registration order", got)
	}
}

// TestACommandBoxIDIsOneOfItsServicesOrUtils: CommandBox's WireBox maps its
// services and util packages by file name, so a CommandBox module's
// `inject="FileSystem"` is commandbox.system.util.FileSystem, as
// cfwheels' CLI injects it, while a project's own file of the name wins.
func TestACommandBoxIDIsOneOfItsServicesOrUtils(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"commands/Build.cfc": `component {
	property name="fileSystemUtil" inject="FileSystem";
	property name="serverService" inject="ServerService";
	property name="mine" inject="MyOwn";
	function run() {
		fileSystemUtil.resolvePath( "x" );
		fileSystemUtil.notAMethod();
		serverService.resolveServerDetails( {} );
		mine.go();
	}
}`,
		"models/MyOwn.cfc": `component { function go() {} }`,
	})

	r := &Resolver{Stubs: frameworkapi.For([]string{"commandbox"})}
	expectReasons(t, reasonsWith(t, r, dir, "commands/Build.cfc"), map[string]string{
		"fileSystemUtil.resolvePath":         "",
		"fileSystemUtil.notAMethod":          "method 'notAMethod' not found in FileSystem",
		"serverService.resolveServerDetails": "",
		"mine.go":                            "",
	})
}
