package nativetype

import "strings"

// jdkMembers are public methods of the JDK classes a CFML page meets through
// createObject( "java", … ) and the objects the engine hands back, by family.
// They type nothing: no result type is claimed. They exist so a call on a value
// the source never typed can be told from a call that belongs to no known type
// (see Types). Only names specific to the JDK are listed; a name every
// component also uses is mostly left out.
var jdkMembers = func() map[string]bool {
	out := map[string]bool{}

	for _, names := range []string{
		// Throwable, StackTraceElement
		"getcause getstacktrace getsuppressed getlocalizedmessage initcause printstacktrace fillinstacktrace " +
			"getlinenumber getmethodname getclassname getfilename",
		// File, Path, Files, URL, URI
		"getabsolutepath getabsolutefile getcanonicalfile getparentfile getparent listfiles listroots " +
			"deleteonexit createnewfile createtempfile canread canwrite canexecute isdirectory isfile ishidden " +
			"lastmodified setlastmodified setreadonly mkdirs renameto topath tofile touri tourl tourl " +
			"readallbytes readalllines readstring writestring newbufferedreader newbufferedwriter newinputstream newoutputstream " +
			"toexternalform getprotocol gethost getport getpath getquery getref getauthority getuserinfo openstream openconnection " +
			"toabsolutepath torealpath getfilename getnameCount resolvesibling relativize normalize",
		// Class, reflection, class loading, security
		"getprotectiondomain getcodesource getlocation getdeclaredfield getdeclaredfields getdeclaredmethod getdeclaredmethods " +
			"getdeclaredconstructor getdeclaredconstructors getconstructor getconstructors getfield getfields getmethod getmethods " +
			"getsimplename getcanonicalname getpackagename getsuperclass getinterfaces getresource getresourceasstream " +
			"getclassloader forname newinstance isinstance isassignablefrom isinterface isenum isarray getcomponenttype " +
			"setaccessible trysetaccessible getmodifiers getreturntype getparametertypes getdeclaringclass invoke getgenerictype",
		// Enumeration, Iterator, Collection, Optional, Stream
		"nextelement hasmoreelements hasmoretokens nexttoken counttokens elements ispresent orelse orelseget orelsethrow ifpresent " +
			"parallelstream foreachordered collect flatmap mapto mapToint mapTolong mapTodouble anymatch allmatch nonematch " +
			"findfirst findany tolist toset retainall removeall containsall removeif sublist descendingmap headmap tailmap submap " +
			"firstkey lastkey firstentry lastentry floorkey ceilingkey higherkey lowerkey pollfirst polllast putifabsent " +
			"computeifabsent computeifpresent getordefault replaceall putall entryset keyset trimtosize ensurecapacity",
		// java.time, Date, Calendar, TimeZone
		"getepochsecond toepochsecond toepochmilli getnano getzone atzone atstartofday tolocaldate tolocaldatetime tolocaltime " +
			"toinstant plusdays plushours plusminutes plusseconds plusweeks plusmonths plusyears minusdays minushours minusminutes " +
			"minusseconds minusweeks minusmonths minusyears getdayofweek getdayofmonth getdayofyear getmonthvalue getyear gethour " +
			"getminute getsecond withzonesamelocal withzonesameinstant isbefore isafter ofpattern parse getrawoffset getdisplayname " +
			"getavailableids setfirstdayofweek gettimeinmillis settimeinmillis gettimezone settimezone getactualmaximum",
		// Atomics, concurrency, threads
		"incrementandget decrementandget getandincrement getanddecrement addandget getandadd compareandset getandset lazyset " +
			"updateandget getandupdate accumulateandget submitall invokeall invokeany shutdownnow awaittermination isterminated " +
			"isshutdown trylock unlock lockinterruptibly newcondition signalall countdown getcount awaitnanos acquire release " +
			"tryacquire availablepermits getthreadgroup setdaemon isdaemon isalive isinterrupted interrupt getstate currentthread " +
			"getallstacktraces getpriority setpriority getthreadid",
		// AWT, imaging, streams, readers, writers
		"derivefont getfontmetrics getstringbounds getheight getwidth getgraphics creategraphics setrgb getrgb getsubimage " +
			"drawimage drawstring fillrect drawrect setcolor setfont setrenderinghint dispose getbytes getchannel flush " +
			"readline readbyte readchar readint readlong readfully skipbytes mark reset marksupported available transferto " +
			"newline write append close",
		// regex, text, number formats, random, math
		"matches lookingat groupcount usepattern appendreplacement appendtail quotereplacement splitaspredicate asmatchpredicate " +
			"setmaximumfractiondigits setminimumfractiondigits setgroupingused setroundingmode formatto parseobject " +
			"nextint nextlong nextdouble nextboolean nextgaussian nextbytes setseed " +
			"bitlength bitcount testbit setbit clearbit flipbit shiftleft shiftright modpow modinverse gcd signum negate " +
			"intvalue longvalue doublevalue floatvalue shortvalue bytevalue",
		// Servlet, Rhino and engine internals reached from CFML
		"getrequesturi getrequesturl getquerystring getcontextpath getservletpath getpathinfo getremoteaddr getremotehost " +
			"getheadernames getheaders getparameternames getparametermap getattributenames getsession invalidate " +
			"setstatus senderror sendredirect addheader setheader addcookie setcharacterencoding setcontenttype " +
			"initstandardobjects evaluatestring compilestring newobject callfunction getcontext",
		// Misc
		"hashcode notify notifyall wait getcurrent tochararray codepointat codepoints chars intern compareto compareignorecase " +
			"regionmatches startswith endswith lastindexof indexof charat substring touppercase tolowercase getbytes",
	} {
		for name := range strings.FieldsSeq(names) {
			out[strings.ToLower(name)] = true
		}
	}

	return out
}()

// JDKMember reports whether name is a public method of a JDK class the page
// may be holding.
func JDKMember(name string) bool { return jdkMembers[strings.ToLower(name)] }
