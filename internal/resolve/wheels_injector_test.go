package resolve

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestWheelsInjectorStartup(t *testing.T) {
	data, err := os.ReadFile("testdata/wheels-injector/Injector.cfc")
	if err != nil {
		t.Fatal(err)
	}

	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(wheelsTokens(string(data))))); got != wheelsInjectorContract {
		t.Fatalf("fixture does not match reviewed contract: %s", got)
	}

	for _, mode := range []string{"literal", "different target", "changed injector", "intervening mutation", "conditional", "computed binder", "computed alias", "binder effects", "duplicate binding", "replacement", "unused lifecycle", "flags", "later rebind", "after conditional", "effectful default", "duplicate result"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			injector := string(data)
			binder := `component {public void function configure(required any injector){arguments.injector.map("global").to("wheels.Global");}}`
			body := `application.container = new wheels.Injector("wheels.Bindings"); application.wo = application.container.getInstance("global");`
			method := "onApplicationStart"

			switch mode {
			case "different target":
				binder = strings.ReplaceAll(binder, "wheels.Global", "wheels.Other")
			case "changed injector":
				injector = strings.Replace(injector, "return local.instance;", "return {};", 1)
			case "intervening mutation":
				body = strings.Replace(body, " application.wo", ` application.container.map("global").to("wheels.Other"); application.wo`, 1)
			case "conditional":
				body = "if(flag){" + body + "}"
			case "computed binder":
				body = strings.Replace(body, `"wheels.Bindings"`, "selected", 1)
			case "computed alias":
				body = strings.Replace(body, `getInstance("global")`, "getInstance(selected)", 1)
			case "binder effects":
				binder = strings.Replace(binder, "arguments.injector.map", "unknown(arguments.injector); arguments.injector.map", 1)
			case "duplicate binding":
				binder = strings.Replace(binder, ";}}", `.map("global").to("wheels.Other");}}`, 1)
			case "replacement":
				body += " application.wo=unknown;"
			case "unused lifecycle":
				method = "unused"
			case "flags":
				binder = strings.Replace(binder, ";}}", ".asSingleton();}}", 1)
			case "after conditional":
				body = `if(flag){local.ignore=true;}` + body
			case "effectful default":
				binder = strings.Replace(binder, "required any injector", "any injector=unknown()", 1)
			case "duplicate result":
				body += `if(flag){application.wo=application.container.getInstance("global");}`
			case "later rebind":
				body += ` application.container.map("global").to("wheels.Other");`
			}

			writeFiles(t, dir, map[string]string{"Application.cfc": fmt.Sprintf("component {function %s(){%s}}", method, body), "wheels/Injector.cfc": injector, "wheels/Bindings.cfc": binder, "wheels/Global.cfc": `component {function ready(){}}`, "wheels/Other.cfc": `component {function other(){}}`, "Spec.cfc": `component {function run(){
g=application.wo;
application.wo.ready();
g.ready();
g.missing();
}}`})
			r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
			got := reasonsWith(t, r, dir, "Spec.cfc")

			want := mode == "literal" || mode == "later rebind" || mode == "after conditional"
			for _, call := range []string{"application.wo.ready", "g.ready"} {
				if (got[call] == "") != want {
					t.Fatalf("%s: %v", call, got)
				}
			}

			if mode == "different target" {
				if got["g.missing"] != "method 'missing' not found in Other" {
					t.Fatalf("binding target was not preserved: %v", got)
				}
			}

			if got["g.missing"] == "" {
				t.Fatal("missing method accepted")
			}
		})
	}
}
