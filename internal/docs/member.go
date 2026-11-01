package docs

import "strings"

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

// members maps a type to its member functions' lowercased names.
var members map[string]map[string]bool

func init() {
	buildMemberFuncs()
	buildMembers()
}

func buildMembers() {
	members = map[string]map[string]bool{}

	for _, m := range builtinMembers() {
		if members[m.Type] == nil {
			members[m.Type] = map[string]bool{}
		}

		members[m.Type][strings.ToLower(m.Name)] = true
	}
}

// IsMemberFunction reports whether the doc sources record method as a member
// function of a value of type typ, a native type's name ("string", "date"),
// or of any value. Lucee's `"abc".reFindNoCase( "b" )` is one; cfdocs records
// few, so the hand-kept contracts in internal/nativetype stay the first answer.
func IsMemberFunction(typ, method string) bool {
	typ, method = strings.ToLower(typ), strings.ToLower(method)

	switch typ {
	case "void":
		// Nothing is called on the result of a function returning nothing.
		return false
	case "date":
		// The sources call a date a datetime.
		typ = "datetime"
	}

	return members[typ][method] || members["any"][method]
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
