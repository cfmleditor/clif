package resolve

import (
	"fmt"
	"regexp"
)

// oracleTagSharedAssign is the expression tagSharedAssignIndex replaced, kept
// as the oracle it is held to, as the shapes in expression_shapes_test.go are.
var oracleTagSharedAssign = regexp.MustCompile(`(?i)^<cfset\s+(request|session|application|server)\.([A-Za-z_$][\w$]*)\s*=`)

func init() {
	expressionShapes["tagSharedAssign"] = expressionShape{
		[]string{
			"<cfset request.context = x>", "<CFSET Session.a=1>", "<cfset\tapplication.b\n=\n2>", "<cfset server.$c = 3>",
			"<cfset request.x == 1>", "<cfset requests.x = 1>", "<cfset local.x = 1>", "<cfset request..x = 1>",
			"<cfset request.1x = 1>", "<cfsetrequest.x = 1>", "<cfset  request . x = 1>", "<cfset request.x",
		},
		[]string{
			"<cfset", "<CFSET", " ", "\t", "\n", "\f", "\v", "request", "REQUEST", "Session", "application", "server", "serve",
			".", "x", "_a", "$b", "1", "=", "==", ">", "-", "é",
		},
		func(s string) (string, string) {
			return fmt.Sprint(tagSharedAssignIndex(s)), fmt.Sprint(oracleTagSharedAssign.FindStringSubmatchIndex(s))
		},
	}
}
