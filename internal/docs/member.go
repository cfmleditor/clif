package docs

import (
	"slices"
	"strings"
)

// MemberFunction represents a function callable via dot notation on a type.
type MemberFunction struct {
	Name  string // e.g. "append"
	Entry *Entry // the underlying built-in entry (e.g. arrayAppend)
}

// typePrefixes maps CFML type names to their function name prefixes.
var typePrefixes = []string{
	"array", "string", "struct", "query", "list", "image", "spreadsheet",
}

var memberFuncs map[string][]MemberFunction // type -> member functions

// Member is a function's member-function form as the doc sources record it
// (`member_details`): Name called on a value of Type is Function.
type Member struct {
	Type     string // the doc sources' type name, lowercased: "string", "datetime", "any"
	Name     string // e.g. "reFindNoCase"
	Function string // e.g. "rEFindNoCase"
}

// members maps a type to its member functions, by lowercased name, each to
// the native type its function returns ("" when that is no help: any, void,
// or a boolean a chaining member does not really return).
var members map[string]map[string]string

func init() {
	buildMemberFuncs()
	buildMembers()
}

func buildMembers() {
	members = map[string]map[string]string{}

	for _, m := range builtinMembers() {
		if members[m.Type] == nil {
			members[m.Type] = map[string]string{}
		}

		ret := ""
		if e := funcMap[strings.ToLower(m.Function)]; e != nil {
			ret = memberReturn(e.Returns)
		}

		members[m.Type][strings.ToLower(m.Name)] = ret
	}

	// Completion and the report's built-in names read memberFuncs, so the
	// recorded members join it beside the ones read from a name's prefix.
	for _, m := range builtinMembers() {
		e := funcMap[strings.ToLower(m.Function)]
		if e == nil || slices.ContainsFunc(memberFuncs[m.Type], func(mf MemberFunction) bool { return strings.EqualFold(mf.Name, m.Name) }) {
			continue
		}

		memberFuncs[m.Type] = append(memberFuncs[m.Type], MemberFunction{Name: m.Name, Entry: e})
	}
}

// memberReturn is the native type name a member's function returns, when it
// is one a value keeps: Lucee's chaining members (`arr.append( 1 ).append( 2 )`)
// return the receiver where the function returns a boolean, so a boolean, a
// void and an any say nothing.
func memberReturn(returns string) string {
	switch strings.ToLower(returns) {
	case "string":
		return "string"
	case "number", "numeric":
		return "numeric"
	case "array", "struct", "query":
		return strings.ToLower(returns)
	case "date", "datetime":
		return "date"
	default:
		return ""
	}
}

// memberValueTypes are the CFML values a member recorded for any value is
// called on. A Java handle (a connection, a result set) has its own contract.
var memberValueTypes = map[string]bool{
	"string": true, "numeric": true, "boolean": true, "array": true, "struct": true,
	"query": true, "datetime": true, "xml": true, "image": true,
}

// IsMemberOf reports whether the doc sources record method as a member
// function of a value of type typ, a native type's name ("string", "date"),
// and the native type it returns ("" when that says nothing). Lucee's
// `"abc".reFindNoCase( "b" )` is one; cfdocs records few, so the hand-kept
// contracts in internal/nativetype stay the first answer.
func IsMemberOf(typ, method string) (ret string, ok bool) {
	var tbuf, mbuf [64]byte

	t, m := lowerASCII(tbuf[:0], typ), lowerASCII(mbuf[:0], method)

	switch string(t) {
	case "void":
		// Nothing is called on the result of a function returning nothing.
		return "", false
	case "date":
		// The sources call a date a datetime.
		t = append(t[:0], "datetime"...)
	}

	if ret, ok = members[string(t)][string(m)]; ok {
		return ret, true
	}

	if memberValueTypes[string(t)] {
		ret, ok = members["any"][string(m)]
	}

	return ret, ok
}

// lowerASCII appends s lowercased to buf, which a caller keeps on the stack,
// so a lookup by a mixed-case name allocates nothing.
func lowerASCII(buf []byte, s string) []byte {
	for i := range len(s) {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}

		buf = append(buf, c)
	}

	return buf
}

func buildMemberFuncs() {
	memberFuncs = make(map[string][]MemberFunction)

	for i := range entries {
		e := &entries[i]
		if e.Type != "function" {
			continue
		}

		lower := strings.ToLower(e.Name)
		for _, prefix := range typePrefixes {
			if strings.HasPrefix(lower, prefix) && len(lower) > len(prefix) {
				memberName := strings.ToLower(e.Name[len(prefix):len(prefix)+1]) + e.Name[len(prefix)+1:]
				memberFuncs[prefix] = append(memberFuncs[prefix], MemberFunction{
					Name:  memberName,
					Entry: e,
				})

				break
			}
		}
	}
}

// AllMemberFunctions returns member functions for all types (used when type is unknown).
func AllMemberFunctions() []MemberFunction {
	var out []MemberFunction
	for _, mfs := range memberFuncs {
		out = append(out, mfs...)
	}

	return out
}

// MemberFunctionsForType returns member functions for a specific type.
func MemberFunctionsForType(typeName string) []MemberFunction {
	return memberFuncs[strings.ToLower(typeName)]
}
