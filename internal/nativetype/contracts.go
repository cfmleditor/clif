// Package nativetype describes public runtime interfaces, separately from CFCs.
// Contracts are intentionally bounded; unknown Java classes retain dynamic handling.
package nativetype

import (
	"maps"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// Prefix distinguishes runtime interfaces from CFC paths.
const Prefix = parser.NativeTypePrefix

// Type constructs a synthetic native interface identity.
func Type(name string) string { return Prefix + name }

// Java recognizes supported Java class identities.
func Java(class string) string { return parser.NativeJavaComponent(class) }

// Producer identifies supported CFML global builtin results.
func Producer(name string) string {
	switch strings.ToLower(name) {
	case "ormgetsession":
		return Type("hibernatesession")
	case "ormgetsessionfactory":
		return Type("sessionfactory")
	case "arraynew":
		return Type("array")
	case "structnew":
		return Type("struct")
	case "now", "createdatetime", "createdate", "parsedatetime", "dateadd":
		return Type("date")
	}

	if stringProducers[strings.ToLower(name)] {
		return Type("string")
	}

	if family := collectionProducers[strings.ToLower(name)]; family != "" {
		return Type(family)
	}

	return ""
}

// Method returns the result type and whether a public method is supported.
// $any is used only for genuinely erased values such as Map.get and Iterator.next.
func Method(component, method string) (string, bool) {
	name := strings.TrimPrefix(component, Prefix)
	if name == component {
		switch component {
		case "$builtin.querynew", "$builtin.queryexecute":
			name = "query"
		default:
			return "", false
		}
	}

	// A struct property is an erased value, not a method on the struct.
	// Its name alone cannot establish the value's runtime interface.
	if name == "struct" {
		if _, ok := parser.PropertyName(method); ok {
			return "$any", true
		}
	}

	m := strings.ToLower(method)
	if name != "void" {
		switch m {
		case "tostring":
			return Type("string"), true
		case "equals":
			return Type("boolean"), true
		case "hashcode":
			return Type("numeric"), true
		case "getclass":
			return Type("class"), true
		}
	}

	if m == "init" && name != "void" {
		return component, true
	}

	if name == "hashmap" {
		if m == "clone" {
			return Type("hashmap"), true
		}

		name = "map"
	}

	if methods := contracts[name]; methods != nil {
		if ret, ok := methods[m]; ok {
			return ret, true
		}
	}

	// A CFML struct is a java.util.Map on both engines, so it answers the
	// Map methods its own contract does not list: ColdBox's MockProvider asks
	// a struct `containsValue()`.
	if name == "struct" {
		if ret, ok := contracts["map"][m]; ok {
			return ret, true
		}
	}

	return "", false
}

func methods(result string, names string) map[string]string {
	out := map[string]string{}
	for name := range strings.FieldsSeq(names) {
		out[strings.ToLower(name)] = result
	}

	return out
}

func contract(groups ...map[string]string) map[string]string {
	out := map[string]string{}

	for _, group := range groups {
		maps.Copy(out, group)
	}

	return out
}

// elements is the interface each struct-notation element of a native
// handle holds, when the interface states one.
var elements = map[string]string{
	// A database handle's db["collection"] is the collection the method
	// call names.
	"mongodb": Type("mongodbcollection"),
}

// Element answers the interface each struct-notation element of the native
// handle holds, or ok=false: a database handle's db["collection"] is the
// collection the ReadMe's struct-notation example names.
func Element(component string) (string, bool) {
	name := strings.ToLower(strings.TrimPrefix(component, Prefix))
	if name == component {
		return "", false
	}

	ret, ok := elements[name]

	return ret, ok
}

var contracts = map[string]map[string]string{
	"object": {}, "boolean": {}, "void": {},
	// CFML member-function syntax on a numeric receiver: a.sgn(), y.abs(),
	// a.aCos() — every entry is a generated-catalogue function taking a
	// number first (booleanFormat/formatBaseN return strings).
	"numeric":           contract(methods(Type("numeric"), "abs acos asin atn bitand bitmaskclear bitmaskread bitmaskset bitnot bitor bitshln bitshrn bitxor ceiling cos fix floor int len log log10 max min round sgn sin sqr tan"), methods(Type("string"), "booleanformat formatbasen")),
	"string":            contract(methods(Type("string"), "trim strip stripleading striptrailing substring substr replace replaceall replacefirst tolowercase touppercase concat repeat intern tolist"), methods(Type("numeric"), "length len indexof lastindexof compareto comparetoignorecase codepointat"), methods(Type("boolean"), "isempty isblank contains equalsignorecase startswith endswith matches"), methods("$any", "split tochararray getbytes"), methods(Type("string"), "charat")),
	"stringbuilder":     contract(methods(Type("stringbuilder"), "append insert delete deletecharat replace reverse"), methods(Type("numeric"), "length capacity indexof lastindexof"), methods(Type("string"), "substring charat"), methods(Type("void"), "setlength setcharat ensurecapacity trimtosize")),
	"class":             contract(methods(Type("string"), "getname getsimplename getcanonicalname gettypename"), methods(Type("class"), "getcomponenttype getsuperclass forname asSubclass"), methods("$any", "getinterfaces getdeclaredmethods getmethods getdeclaredfields getfields getconstructors getdeclaredconstructors"), methods(Type("boolean"), "isarray isassignablefrom isinstance isinterface isprimitive isenum"), methods("$any", "newinstance cast")),
	"map":               contract(methods(Type("set.entry"), "entryset"), methods(Type("set"), "keyset"), methods(Type("collection"), "values"), methods(Type("numeric"), "size"), methods(Type("boolean"), "isempty containskey containsvalue"), methods(Type("void"), "clear putall"), methods("$any", "get put remove replace putifabsent getordefault compute computeifabsent computeifpresent merge")),
	"list":              contract(methods(Type("iterator"), "iterator listiterator"), methods(Type("numeric"), "size length len indexof lastindexof"), methods(Type("boolean"), "isempty contains add addall removeall retainall"), methods("$any", "get set remove"), methods(Type("list"), "sublist"), methods("$any", "toarray"), methods(Type("void"), "clear sort")),
	"collection":        contract(methods(Type("iterator"), "iterator"), methods(Type("numeric"), "size"), methods(Type("boolean"), "isempty contains add addall remove removeall retainall"), methods("$any", "toarray"), methods(Type("void"), "clear")),
	"set":               contract(methods(Type("iterator"), "iterator"), methods(Type("numeric"), "size"), methods(Type("boolean"), "isempty contains add addall remove removeall retainall"), methods("$any", "toarray"), methods(Type("void"), "clear")),
	"set.entry":         contract(methods(Type("iterator.entry"), "iterator"), methods(Type("numeric"), "size"), methods(Type("boolean"), "isempty contains"), methods("$any", "toarray")),
	"iterator":          contract(methods(Type("boolean"), "hasnext"), methods("$any", "next"), methods(Type("void"), "remove foreachremaining")),
	"iterator.entry":    contract(methods(Type("boolean"), "hasnext"), methods(Type("entry"), "next"), methods(Type("void"), "remove foreachremaining")),
	"entry":             methods("$any", "getkey getvalue setvalue"),
	"query":             contract(methods("$any", "getcolumn"), methods(Type("array"), "getcolumnnames columnarray"), methods(Type("numeric"), "getrecordcount recordcount columncount"), methods("$any", "getcell getrow rowdata"), methods(Type("boolean"), "columnexists"), methods(Type("query"), "filter map"), methods("$any", "addrow addcolumn setcell sort each deleterow deletecolumn")),
	"drivermanager":     contract(methods(Type("connection"), "getconnection"), methods(Type("numeric"), "getlogintimeout"), methods(Type("void"), "setlogintimeout")),
	"connection":        contract(methods(Type("statement"), "createstatement"), methods(Type("preparedstatement"), "preparestatement"), methods(Type("databasemetadata"), "getmetadata"), methods(Type("boolean"), "isclosed isvalid getautocommit isreadonly"), methods(Type("string"), "getcatalog getschema"), methods(Type("void"), "close commit rollback setautocommit setreadonly setcatalog setschema")),
	"statement":         contract(methods(Type("resultset"), "executequery getresultset getgeneratedkeys"), methods(Type("numeric"), "executeupdate getupdatecount"), methods(Type("boolean"), "execute isclosed"), methods(Type("connection"), "getconnection"), methods(Type("void"), "close cancel setquerytimeout addbatch clearbatch")),
	"preparedstatement": contract(methods(Type("resultset"), "executequery getresultset getgeneratedkeys"), methods(Type("numeric"), "executeupdate getupdatecount"), methods(Type("boolean"), "execute isclosed"), methods(Type("void"), "close cancel setstring setint setlong setboolean setobject setnull setdate settimestamp addbatch clearparameters")),
	"resultset":         contract(methods(Type("boolean"), "next previous first last absolute relative wasnull isclosed"), methods(Type("string"), "getstring"), methods(Type("numeric"), "getint getlong getdouble getfloat getrow findcolumn"), methods(Type("boolean"), "getboolean"), methods(Type("date"), "getdate gettimestamp"), methods("$any", "getobject"), methods(Type("resultsetmetadata"), "getmetadata"), methods(Type("void"), "close beforefirst afterlast")),
	"databasemetadata":  contract(methods(Type("resultset"), "gettables getcolumns getprimarykeys getimportedkeys getexportedkeys getcatalogs getschemas"), methods(Type("string"), "getdatabaseproductname getdatabaseproductversion getdrivername getdriverversion geturl getusername")),
	"resultsetmetadata": contract(methods(Type("numeric"), "getcolumncount getcolumntype getprecision getscale"), methods(Type("string"), "getcolumnname getcolumnlabel getcolumntypename gettablename getschemaname getcolumnclassname")),
	"hibernatesession": contract(methods(Type("sessionstatistics"), "getstatistics"), methods(Type("sessionfactory"), "getsessionfactory"), methods(Type("boolean"), "isdirty isopen contains isconnected"), methods(Type("string"), "getentityname"), methods("$any", "getidentifier get load save merge"), methods(Type("void"), "close flush clear evict delete update persist saveorupdate refresh"), methods(Type("hibernatequery"), "createquery createsqlquery createnativequery getnamedquery"),
		// The rest of org.hibernate.Session cborm and applications call, and
		// Lucee's wrapper's getActualSession(); their results are left dynamic.
		methods("$any", "begintransaction gettransaction createcriteria createfilter enablefilter getenabledfilter getactualsession getflushmode getcachemode getlockmode dowork connection"),
		methods(Type("void"), "setflushmode setcachemode setreadonly disablefilter lock replicate cancelquery disconnect reconnect"),
		methods(Type("boolean"), "isreadonly isdefaultreadonly")),
	"sessionfactory":    contract(methods(Type("statistics"), "getstatistics"), methods(Type("hibernatesession"), "opensession getcurrentsession"), methods(Type("classmetadata"), "getclassmetadata"), methods(Type("void"), "close"), methods(Type("boolean"), "isclosed")),
	"sessionstatistics": contract(methods(Type("numeric"), "getentitycount getcollectioncount"), methods(Type("set"), "getentitykeys getcollectionkeys")),
	"statistics":        contract(methods(Type("void"), "setstatisticsenabled clear"), methods(Type("boolean"), "isstatisticsenabled"), methods(Type("numeric"), "getsessionopencount getsessionclosecount getentityloadcount getentityfetchcount getentityinsertcount getentityupdatecount getentitydeletecount getqueryexecutioncount getsecondlevelcachehitcount getsecondlevelcachemisscount getsecondlevelcacheputcount"), methods("$any", "getsecondlevelcacheregionnames getentitynames getqueries"), methods(Type("cachestatistics"), "getsecondlevelcachestatistics getcacheregionstatistics")),
	"cachestatistics":   methods(Type("numeric"), "gethitcount getmisscount getputcount getelementcountinmemory getelementcountondisk getsizeinmemory"),
	"classmetadata": contract(
		// org.hibernate.metadata.ClassMetadata, plus getTableName from the
		// entity persister Hibernate actually returns as the metadata.
		methods(Type("string"), "gettablename"),
		methods(Type("boolean"), "hasidentifierproperty isversioned hasproxy ismutable isinherited hasnaturalidentifier hassubclasses"),
		methods("$any", "getidentifiertype getpropertytype getpropertynullability getpropertylaziness getversionproperty getmappedclass getnaturalidentifierproperties getpropertyvalues getpropertyindex"),
		methods(Type("string"), "getentityname getidentifierpropertyname"), methods("$any", "getpropertynames getpropertytypes"), methods("$any", "getidentifier getpropertyvalue")),
	"hibernatequery": contract(methods(Type("hibernatequery"), "setparameter setparameterlist setmaxresults setfirstresult settimeout setcacheable setcacheregion"), methods(Type("list"), "list getresultlist"), methods("$any", "uniqueresult getsingleresult"), methods(Type("numeric"), "executeupdate")),
	"date":           contract(methods(Type("instant"), "toinstant"), methods(Type("numeric"), "gettime getyear getmonth getdate getday gethours getminutes getseconds compareto"), methods(Type("boolean"), "before after"), methods(Type("string"), "format")),
	"instant":        contract(methods(Type("instant"), "now parse ofepochmilli ofepochsecond plus minus plusseconds plusmillis minusseconds minusmillis truncatedto"), methods(Type("zoneddatetime"), "atzone"), methods(Type("offsetdatetime"), "atoffset"), methods(Type("numeric"), "toepochmilli getepochsecond getnano")),
	"localdatetime":  contract(methods(Type("localdatetime"), "now of parse ofinstant ofepochsecond plus minus plusdays plushours plusminutes plusseconds plusmonths plusyears minusdays minushours minusminutes minusseconds minusmonths minusyears with withhour withminute withsecond withnano withdayofmonth withdayofyear withmonth withyear truncatedto"), methods(Type("localdate"), "tolocaldate"), methods(Type("localtime"), "tolocaltime"), methods(Type("zoneddatetime"), "atzone"), methods(Type("offsetdatetime"), "atoffset"), methods(Type("instant"), "toinstant"), methods(Type("string"), "format"), methods(Type("numeric"), "getyear getmonthvalue getdayofmonth getdayofyear gethour getminute getsecond getnano compareto")),
	"localdate":      contract(methods(Type("localdate"), "now of parse plusdays plusmonths plusyears minusdays minusmonths minusyears with withdayofmonth withmonth withyear"), methods(Type("localdatetime"), "attime"), methods("$any", "atstartofday"), methods(Type("numeric"), "getyear getmonthvalue getdayofmonth getdayofyear toepochday compareto"), methods(Type("string"), "format")),
	"localtime":      contract(methods(Type("localtime"), "now of parse plushours plusminutes plusseconds withhour withminute withsecond"), methods(Type("numeric"), "gethour getminute getsecond getnano compareto"), methods(Type("string"), "format")),
	"zoneddatetime":  contract(methods(Type("zoneddatetime"), "now of parse ofinstant plusdays plushours plusminutes plusseconds with withhour withminute withsecond withnano withzoneSameInstant withzoneSameLocal"), methods(Type("localdatetime"), "tolocaldatetime"), methods(Type("localdate"), "tolocaldate"), methods(Type("localtime"), "tolocaltime"), methods(Type("instant"), "toinstant"), methods(Type("zoneid"), "getzone getoffset"), methods(Type("string"), "format")),
	"offsetdatetime": contract(methods(Type("offsetdatetime"), "now of parse ofinstant plusdays plushours plusminutes plusseconds with withoffsetsameinstant"), methods(Type("localdatetime"), "tolocaldatetime"), methods(Type("instant"), "toinstant"), methods(Type("zoneddatetime"), "tozoneddatetime"), methods(Type("string"), "format")),
	"zoneid":         contract(methods(Type("zoneid"), "of systemdefault ofhours ofhoursminutes"), methods(Type("string"), "getid")),
	"duration":       contract(methods(Type("duration"), "ofdays ofhours ofminutes ofseconds ofmillis between parse plus minus"), methods(Type("numeric"), "todays tohours tominutes toseconds tomillis getseconds getnano"), methods(Type("boolean"), "iszero isnegative")),
	"enum":           contract(methods(Type("numeric"), "ordinal getvalue"), methods(Type("string"), "name"), methods(Type("enum"), "valueof"), methods(Type("collection"), "values")),
	"adjusters":      methods(Type("object"), "firstinmonth lastinmonth firstdayofmonth lastdayofmonth next previous nextorsame previousorsame"),
	"formatter":      contract(methods(Type("formatter"), "ofpattern withzone withlocale"), methods(Type("string"), "format"), methods("$any", "parse")),

	// The Lucee MongoDB extension's shell-like API, documented in the
	// extension's own ReadMe: the database handle MongoDBConnect returns,
	// the collection its getCollection (and its struct notation) holds, the
	// cursor find chains sort/limit onto, the aggregation output, the
	// write result the writes return, the ObjectId MongoDBID returns and
	// the write concern the collection reads back. The collection also has
	// the methods DBCollectionImpl dispatches (lucee/extension-mongodb at
	// a146e45) that the ReadMe leaves out: the stats and sizes, rename, index
	// info, bulk writes and the search indexes. mapReduce is dispatched only
	// to throw (removed in the 5.x driver), so it is left out and a call to it
	// stays reported.
	"mongodb":                contract(methods(Type("mongodbcollection"), "getcollection"), methods(Type("mongodb"), "getsiblingdb getsisterdb"), methods(Type("string"), "getname"), methods(Type("numeric"), "size"), methods(Type("boolean"), "containskey"), methods(Type("void"), "clear"), methods("$any", "authenticate")),
	"mongodbcollection":      contract(methods(Type("mongocursor"), "find"), methods(Type("mongo.writeresult"), "insert insertmany save update remove"), methods(Type("mongoaggregationoutput"), "aggregate"), methods(Type("struct"), "findone findandmodify findandremove stats getstats bulkwrite"), methods(Type("array"), "distinct getindexes getindexinfo createsearchindexes listsearchindexes"), methods(Type("mongowriteconcern"), "getwriteconcern"), methods(Type("numeric"), "count datasize storagesize totalindexsize"), methods(Type("string"), "createsearchindex"), methods(Type("void"), "drop rename renamecollection dropsearchindex updatesearchindex"), methods("$any", "createindex ensureindex dropindex dropindexes setwriteconcern")),
	"mongocursor":            contract(methods(Type("mongocursor"), "sort limit"), methods(Type("boolean"), "hasnext"), methods(Type("struct"), "next"), methods(Type("numeric"), "size count"), methods(Type("void"), "close remove")),
	"mongoaggregationoutput": contract(methods(Type("boolean"), "hasnext"), methods(Type("struct"), "next results"), methods(Type("mongoaggregationoutput"), "batchsize"), methods(Type("numeric"), "getbatchsize"), methods(Type("void"), "close")),
	"mongo.writeresult":      contract(methods(Type("numeric"), "getn size"), methods(Type("boolean"), "isok isupdateofexisting containskey"), methods(Type("string"), "geterror"), methods(Type("set"), "keyset"), methods(Type("collection"), "duplicate"), methods(Type("void"), "clear"), methods("$any", "get remove set setel call callwithnamedvalues")),
	"mongoid":                contract(methods(Type("string"), "tostring"), methods(Type("date"), "getdate")),
	"mongowriteconcern":      methods(Type("boolean"), "isacknowledged"),
}

// EngineProducer selects an engine-specific API only when the scan identifies it.
func EngineProducer(name, engine string) string {
	if engine == "lucee" {
		switch strings.ToLower(name) {
		case "mongodbconnect":
			return Type("mongodb")
		case "mongodbid":
			return Type("mongoid")
		}
	}

	if strings.EqualFold(name, "getPageContext") && engine == "lucee" {
		return Type("lucee.pagecontext")
	}

	return Producer(name)
}

func init() {
	contracts["lucee.pagecontext"] = contract(methods(Type("lucee.config"), "getconfig"), methods(Type("lucee.factory"), "getcfmlfactory"), methods("$any", "getrequest getresponse getsession getservletcontext getapplicationcontext getvariablescope geturlscope getformscope gethttpsession"), methods(Type("string"), "getwebcharset getresourcecharset"), methods(Type("void"), "flush write forcewrite"))
	contracts["lucee.factory"] = contract(methods(Type("lucee.config"), "getconfig getconfigserver"), methods("$any", "getengine getscopecontext"))
	contracts["lucee.config"] = contract(methods(Type("list.extension"), "getallrhextensions"), methods(Type("lucee.jdbcdriver"), "getjdbcdriverbyid getjdbcdriverbyclassname"), methods(Type("map"), "getcacheconnections"), methods(Type("list"), "getlognames"), methods(Type("lucee.log"), "getlog"), methods("$any", "getdatasource getdatasources getmappings getconfigdir getrootdirectory getsecuritymanager getdebugentries getdebugentry"))
	contracts["list.extension"] = contract(methods(Type("iterator.extension"), "iterator"), methods(Type("numeric"), "size"))
	contracts["iterator.extension"] = contract(methods(Type("boolean"), "hasnext"), methods(Type("lucee.extension"), "next"))
	contracts["lucee.extension"] = contract(methods(Type("lucee.extensionmetadata"), "getmetadata"), methods(Type("string"), "getid getname getversion"))
	contracts["lucee.extensionmetadata"] = methods(Type("list.map"), "getjdbcs")
	contracts["list.map"] = contract(methods(Type("iterator.map"), "iterator"), methods(Type("numeric"), "size"))
	contracts["iterator.map"] = contract(methods(Type("boolean"), "hasnext"), methods(Type("map"), "next"))
	contracts["lucee.jdbcdriver"] = methods(Type("lucee.classdefinition"), "getclassdefinition")
	contracts["lucee.classdefinition"] = contract(methods(Type("boolean"), "isbundle ismaven"), methods(Type("string"), "getclassname getname getversionasstring getmavenraw"))
	contracts["lucee.log"] = methods(Type("void"), "debug info warn error fatal trace log")
}

// Guess first preserves a unique existing interface-family answer, then tries
// String/Array/Struct/StringBuilder as a separate fallback family set. Priority
// between these sets is an opt-in policy assumption, never origin evidence.
// CFML query methods overlapping a primitive collection cannot select query.
func Guess(observed []string) string {
	if !nativeGuessDiscriminated(observed) {
		return ""
	}

	if existing := guessFamily(observed, []string{"map", "list", "set", "iterator", "connection", "statement", "preparedstatement", "resultset", "query", "hibernatesession", "sessionfactory", "sessionstatistics", "statistics", "instant", "localdatetime", "localdate", "zoneddatetime"}); existing != "" {
		if existing == Type("query") {
			for _, family := range []string{"string", "array", "struct"} {
				if guessFamily(observed, []string{family}) != "" {
					return ""
				}
			}
		}

		return existing
	}

	primitive := guessFamily(observed, []string{"string", "array", "struct", "stringbuilder"})
	if primitive == Type("string") && guessFamily(observed, []string{"numeric"}) != "" {
		return ""
	}

	return primitive
}

// Every observed method must match, and competing answers in the selected set
// withhold a guess. Common Object names alone cannot select a primitive family.
func guessFamily(observed, families []string) string {
	if len(observed) == 0 {
		return ""
	}

	answer := ""

	for _, family := range families {
		match := true

		for _, method := range observed {
			if _, ok := Method(Type(family), method); !ok {
				match = false

				break
			}
		}

		if match {
			if answer != "" {
				return ""
			}

			answer = Type(family)
		}
	}

	return answer
}

func init() {
	contracts["array"] = contract(methods(Type("boolean"), "append prepend clear delete deleteat insertat isempty isdefined indexexists resize set sort swap contains containsnocase every some"), methods(Type("numeric"), "len length max min avg median sum find findnocase"), methods(Type("array"), "map filter slice mid reverse duplicate copy merge findall findallnocase"), methods(Type("string"), "tolist"), methods("$any", "first last getat pop shift reduce"), methods(Type("void"), "each"))

	contracts["struct"] = contract(methods(Type("boolean"), "keyexists isempty insert delete clear iscasesensitive isordered"), methods(Type("numeric"), "count len"), methods(Type("array"), "keyarray valuearray findkey findvalue"), methods(Type("string"), "keylist toquerystring"), methods(Type("struct"), "copy duplicate filter map append update"), methods("$any", "find get reduce"), methods(Type("void"), "each"))

	// Public collection APIs used by Lucee and BoxLang. As with the other
	// native contracts, this interface does not verify engine availability.
	// Java List overloads of add/remove have different results. Keep those and
	// get erased; collection identity alone supplies no element contract.
	maps.Copy(contracts["array"], methods("$any", "add get remove"))
	maps.Copy(contracts["array"], methods(Type("numeric"), "size getdimension"))
	maps.Copy(contracts["array"], methods("$any", "toarray"))
	maps.Copy(contracts["array"], methods(Type("list"), "sublist"))
	maps.Copy(contracts["array"], methods(Type("iterator"), "iterator"))
	maps.Copy(contracts["array"], methods(Type("boolean"), "addall containsall removeall retainall"))
	maps.Copy(contracts["array"], methods(Type("string"), "tojson"))
	maps.Copy(contracts["query"], methods(Type("string"), "valuelist tojson"))

	// String character operations and delimited-list operations have distinct
	// names but share these documented result interfaces. Reducers remain erased.
	maps.Copy(contracts["string"], methods(Type("string"), "hash listgetduplicates listmap listfilter map filter"))
	maps.Copy(contracts["string"], methods(Type("boolean"), "listsome listevery some every"))
	maps.Copy(contracts["string"], methods(Type("void"), "each listeach"))
	maps.Copy(contracts["string"], methods("$any", "reduce listreduce deserializejson"))
	maps.Copy(contracts["struct"], methods(Type("struct"), "tosorted"))
	maps.Copy(contracts["struct"], methods(Type("string"), "tojson"))
	maps.Copy(contracts["query"], methods(Type("array"), "columndata"))
	maps.Copy(contracts["query"], methods(Type("boolean"), "some every"))
	maps.Copy(contracts["query"], methods("$any", "reduce"))
	// Lucee registers these hidden StringHasPrefix/StringHasSuffix members.
	maps.Copy(contracts["string"], methods(Type("boolean"), "hasprefix hassuffix"))

	maps.Copy(contracts["string"], methods(Type("array"), "listtoarray listqualifiedtoarray reMatch reMatchNoCase"))

	maps.Copy(contracts["string"], methods(Type("numeric"), "find findnocase listlen listfind listfindnocase listvaluecount listvaluecountnocase compare comparenocase"))

	maps.Copy(contracts["string"], methods(Type("string"), "lcase ucase ltrim rtrim reverse rereplace rereplacenocase replacenocase left right mid listfirst listlast listgetat listappend listprepend listchangedelims listdeleteat listinsertat listsetat listsort gettoken"))

	maps.Copy(contracts["date"], methods(Type("date"), "add convert"))

	maps.Copy(contracts["date"], methods(Type("numeric"), "diff compare daysinmonth daysinyear day dayofweek dayofyear hour minute second month year week quarter getnumericdate"))

	maps.Copy(contracts["date"], methods(Type("string"), "dateformat timeformat datetimeformat lsdateformat lstimeformat lsdatetimeformat"))

	for _, family := range []string{"localdatetime", "localdate", "localtime", "zoneddatetime", "offsetdatetime", "instant"} {
		maps.Copy(contracts[family], methods(Type("boolean"), "isbefore isafter issupported"))

		maps.Copy(contracts[family], methods(Type("numeric"), "until get getlong compareto"))

		if family != "instant" && family != "localtime" {
			contracts[family]["getdayofweek"] = Type("enum")
			maps.Copy(contracts[family], methods(Type(family), "plusweeks minusweeks"))
		}
	}

	for _, family := range []string{"localdate", "localdatetime", "zoneddatetime", "offsetdatetime"} {
		contracts[family]["isequal"] = Type("boolean")
	}

	maps.Copy(contracts["duration"], methods(Type("duration"), "abs negated multipliedby withnanos withseconds from"))

	contracts["duration"]["dividedby"] = "$any" // Duration and numeric overloads.
	contracts["duration"]["compareto"] = Type("numeric")
	contracts["lucee.pagecontext"]["getthread"] = Type("thread")
	contracts["lucee.pagecontext"]["getout"] = Type("writer")
	contracts["thread"] = contract(methods(Type("string"), "getname"), methods(Type("numeric"), "getid getpriority"), methods(Type("boolean"), "isalive isinterrupted"), methods(Type("void"), "join interrupt"))
	contracts["writer"] = contract(methods(Type("string"), "getstring tostring"), methods(Type("void"), "write flush close clear clearbuffer"))
	contracts["lucee.config"]["getdatasourceconnectionpools"] = Type("list.pool")
	contracts["list.pool"] = contract(methods(Type("iterator.pool"), "iterator"), methods(Type("numeric"), "size"))
	contracts["iterator.pool"] = contract(methods(Type("boolean"), "hasnext"), methods(Type("pool"), "next"))
	contracts["pool"] = contract(methods(Type("numeric"), "getnumactive getnumidle getmaxidle getminidle getmaxwaitmillis"), methods(Type("datasource"), "getfactory"))
	contracts["datasource"] = methods(Type("datasource"), "getdatasource")
}
