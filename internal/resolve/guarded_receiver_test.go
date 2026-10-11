package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestUnknownReceiverUsesExactTypeGuard(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		resolved     bool
	}{
		{"later conjunction", `component {function run(x){if(runtime() && isInstanceOf(x,"Service")){x.go();}}}`, true},
		{"tag scoped suffix", `<cfif isDefined("session.mura.editBean") and isInstanceOf(session.mura.editBean,"Service") and session.mura.editBean.go() eq 1></cfif>`, true},
		{"scoped body", `<cfif isInstanceOf(session.mura.editBean,"Service")><cfset session.mura.editBean.go()></cfif>`, true},
		{"scoped reassignment", `<cfif isInstanceOf(session.mura.editBean,"Service")><cfset session.mura.editBean=unknown><cfset session.mura.editBean.go()></cfif>`, false},
		{"scoped parent replacement", `<cfif isInstanceOf(session.mura.editBean,"Service")><cfset session.mura={}><cfset session.mura.editBean.go()></cfif>`, false},
		{"scoped parent escape", `<cfif isInstanceOf(session.mura.editBean,"Service")><cfset mutate(session.mura)><cfset session.mura.editBean.go()></cfif>`, false},
		{"scoped bracket write", `<cfif isInstanceOf(session.mura.editBean,"Service")><cfset session.mura["editBean"]=unknown><cfset session.mura.editBean.go()></cfif>`, false},
		{"scoped interpolated escape", `<cfif isInstanceOf(session.mura.editBean,"Service")><cfset text="#mutate(session.mura)#"><cfset session.mura.editBean.go()></cfif>`, false},
		{"deferred callback", `component {function run(x){if(isInstanceOf(x,"Service")){deferred=function(){x.go();};}x=unknown;deferred();}}`, false},
		{"compared check", `component {function run(x){if(runtime() && isInstanceOf(x,"Service")==false){x.go();}}}`, false},
		{"earlier OR", `component {function run(x){if(runtime() || isInstanceOf(x,"Service")){x.go();}}}`, false},
		{"script conjunction", `component {function run(x){if(isInstanceOf(x,"Service") && runtime()){x.go();}}}`, true},
		{"tag conjunction", `<cfif isInstanceOf(x,"Service") and runtime()><cfset x.go()></cfif>`, true},
		{"disjunctive conjunction", `component {function run(x){if(isInstanceOf(x,"Service") && runtime() || other()){x.go();}}}`, false},
		{"condition escape", `component {function run(x){if(isInstanceOf(x,"Service") && mutate(x)){x.go();}}}`, false},
		{"script", `component {function run(x){if(isInstanceOf(x,"Service")){x.go();}}}`, true},
		{"tag", `<cfif isInstanceOf(x,"Service")><cfset x.go()></cfif>`, true},
		{"nested tag", `<cfif isInstanceOf(x,"Service")><cfif runtime()><cfset x.go()></cfif></cfif>`, true},
		{"arguments", `component {function run(x){if(isInstanceOf(arguments.x,"Service")){arguments.x.go();}}}`, true},
		{"or", `component {function run(x){if(isInstanceOf(x,"Service")||runtime()){x.go();}}}`, false},
		{"negated", `component {function run(x){if(!isInstanceOf(x,"Service")){x.go();}}}`, false},
		{"script else", `component {function run(x){if(isInstanceOf(x,"Service")){}else{x.go();}}}`, false},
		{"tag else", `<cfif isInstanceOf(x,"Service")><cfelse><cfset x.go()></cfif>`, false},
		{"tag elseif", `<cfif isInstanceOf(x,"Service")><cfelseif runtime()><cfset x.go()></cfif>`, false},
		{"reassigned", `component {function run(x){if(isInstanceOf(x,"Service")){x=unknown;x.go();}}}`, false},
		{"interpolated escape", `component {function run(x){if(isInstanceOf(x,"Service")){var text="#mutate(x)#";x.go();}}}`, false},
		{"interpolated scope escape", `component {function run(x){if(isInstanceOf(x,"Service")){var text="#mutate(variables)#";x.go();}}}`, false},
		{"scope escape", `component {function run(x){if(isInstanceOf(x,"Service")){mutate(variables);x.go();}}}`, false},
		{"include escape", `component {function run(x){if(isInstanceOf(x,"Service")){include "unknown.cfm";x.go();}}}`, false},
		{"escaped", `component {function run(x){if(isInstanceOf(x,"Service")){mutate(x);x.go();}}}`, false},
		{"implicit check", `component {function run(x){isInstanceOf=unknown;if(isInstanceOf(x,"Service")){x.go();}}}`, false},
		{"scoped check", `component {function run(x){variables.isInstanceOf=unknown;if(isInstanceOf(x,"Service")){x.go();}}}`, false},
		{"local check", `component {function run(x){var isInstanceOf=unknown;if(isInstanceOf(x,"Service")){x.go();}}}`, false},
		{"argument check", `component {function run(x,isInstanceOf){if(isInstanceOf(x,"Service")){x.go();}}}`, false},
		{"string guard", `component {function run(x){var text='if(isInstanceOf(x,"Service")){';x.go();}}`, false},
		{"comment guard", `component {function run(x){/* if(isInstanceOf(x,"Service")){ */ x.go();}}`, false},
		{"missing method", `component {function run(x){if(isInstanceOf(x,"Other")){x.go();}}}`, false},
		{"shadowed", `component {function isInstanceOf(value,type){return true;}function run(x){if(isInstanceOf(x,"Service")){x.go();}}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Service.cfc": `component {function go(){}}`, "Other.cfc": `component {function other(){}}`})
			r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
			pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Page.cfc")), tc.source, &parser.ParseOptions{ExtractCalls: true})
			checked := 0

			for _, call := range pr.AllCalls() {
				if call.FuncName == "go" {
					checked++

					why := r.CanResolveCall(&call, pr, dir)
					if (why == "") != tc.resolved {
						t.Fatalf("got %q want resolved %v", why, tc.resolved)
					}
				}
			}

			if checked != 1 {
				t.Fatalf("checked %d", checked)
			}
		})
	}
}
