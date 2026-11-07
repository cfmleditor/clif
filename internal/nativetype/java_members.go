package nativetype

// The JDK surface a Lucee value shows through. A CFML date is a
// java.util.Date, so its setters are calls on it; a Duration, a Class and an
// array answer the methods of their Java types. Names a contract already types
// keep their type; the rest are accepted with an erased result.
func init() {
	erasedMembers("duration", "get getunits todayspart tohourspart tomillispart tominutespart tosecondspart tonanospart "+
		"todays tohours tominutes tomillis toseconds tonanos iszero isnegative abs negated plus minus plusdays plushours "+
		"plusminutes plusseconds plusmillis plusnanos minusdays minushours minusminutes minusseconds minusmillis minusnanos "+
		"getseconds getnano compareto dividedby multipliedby withseconds withnanos")
	erasedMembers("date", "setdate setminutes setseconds sethours setmonth setyear settime duplicate clone tokey after before")
	erasedMembers("class", "getclassloader getdeclaredfield getdeclaredfields getdeclaredmethod getdeclaredmethods "+
		"getdeclaredconstructor getdeclaredconstructors getfield getfields getmethod getmethods getconstructor getconstructors "+
		"getresourceasstream getresource getsimplename getpackage getsuperclass getinterfaces isinstance isinterface "+
		"isassignablefrom newinstance getmodifiers isarray getcomponenttype getenumconstants getannotations")
	erasedMembers("array", "clone tokey")
	erasedMembers("list", "clone tokey")
	erasedMembers("lucee.pagecontext", "getservlet hascfsession getrelativepagesourceexisting getpagesources getpagesource "+
		"settimeoutstacktrace getrequest getresponse getsession getout getservletcontext getservletconfig getpage getexception "+
		"getattribute setattribute removeattribute findattribute forward include release initialize handlepageexception "+
		"getattributenamesinscope getattributesscope")
	erasedMembers("lucee.config", "getconfigserverimpl getconfigserver flushapplicationpathcache getadminmode getdebuggerpool")
	erasedMembers("lucee.factory", "getservlet checktimeout")
	erasedMembers("lucee.pagesource", "isimmutable")
	erasedMembers("lucee.datasource", "getclassdefinition")
}
