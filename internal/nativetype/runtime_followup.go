package nativetype

import (
	"maps"
	"strings"
)

// Metadata is available on unrelated runtime objects. Its presence, even with
// common Object methods, cannot distinguish an array from an extension or JDBC
// object. Keep known-receiver metadata contracts separate from family guesses.
func nativeGuessDiscriminated(observed []string) bool {
	for _, method := range observed {
		switch strings.ToLower(method) {
		case "getmetadata", "tostring", "equals", "hashcode", "getclass", "init":
		default:
			return true
		}
	}

	return false
}

// These bounded interfaces follow Lucee's function library and public loader
// interfaces, and Java's Thread API. As with the existing CFML member contracts,
// a method contract does not prove availability in every engine/version.
func init() {
	maps.Copy(contracts["struct"], methods(Type("boolean"), "some every"))
	maps.Copy(contracts["string"], methods(Type("string"), "repeatstring numberformat"))
	maps.Copy(contracts["numeric"], methods(Type("string"), "numberformat"))
	maps.Copy(contracts["date"], methods(Type("date"), "setday sethour setmillisecond setminute setmonth setsecond setyear"))
	maps.Copy(contracts["array"], methods(Type("array"), "removeduplicates"))
	maps.Copy(contracts["array"], methods(Type("struct"), "getmetadata"))
	maps.Copy(contracts["struct"], methods(Type("struct"), "getmetadata"))

	// PageContext is produced only with Lucee engine evidence. Keep these chains
	// separate from generic Object/CFML struct and filesystem identities.
	contracts["lucee.pagecontext"]["getcurrentpagesource"] = Type("lucee.pagesource")
	contracts["lucee.pagesource"] = contract(methods(Type("lucee.mapping"), "getmapping"), methods(Type("string"), "getrealpath getrealpathwithvirtual getfilename getclassname getjavaname getcomponentname getdisplaypath"), methods(Type("boolean"), "exists executable"))
	contracts["lucee.mapping"] = contract(methods(Type("string"), "getvirtual getvirtuallowercase getvirtuallowercasewithslash getstrarchive getstrphysical"), methods(Type("boolean"), "hasarchive hasphysical ishidden isphysicalfirst isreadonly istrusted istoplevel"))
	contracts["lucee.mapping"]["getpagesource"] = Type("lucee.pagesource")
	contracts["thread"]["getthreadgroup"] = Type("threadgroup")
	contracts["threadgroup"] = contract(methods(Type("string"), "getname"), methods(Type("threadgroup"), "getparent"), methods(Type("numeric"), "activecount activegroupcount getmaxpriority"))
	contracts["pool"]["getfactory"] = Type("lucee.datasourcefactory")
	contracts["lucee.datasourcefactory"] = methods(Type("lucee.datasource"), "getdatasource")
	contracts["lucee.datasource"] = methods(Type("string"), "getname")
}
