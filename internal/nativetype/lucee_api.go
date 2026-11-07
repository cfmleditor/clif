package nativetype

import "strings"

// The public API of Lucee's PageContext and configuration objects, read from the
// interfaces at Lucee commit 26aef68 (the corpus checkout) (loader/src/main/java/lucee/runtime/
// PageContext.java, config/Config.java and ConfigWeb.java, and core/src/main/java/
// lucee/runtime/config/ConfigPro.java). Lucee's tests and its admin call
// getPageContext().getId(), getConfig().getMavenDir() and the like; the contracts
// in contracts.go name the members typed results are known for, and a method of
// the interface that is not one of them is a method the object has with a result
// no contract describes, so it is accepted and erased ("$any"). A name a contract
// already types keeps its type.
const luceePageContextAPI = "" +
	"addpagesource applicationscope argumentsscope cgiscope clear clearcatch clearremoteuser clientscope " +
	"clientscopeel close compile cookiescope doinclude evaluate execute executecfml executerest exelogend " +
	"exelogstart flush forcewrite formscope get getactivecomponent getactiveudf getapplicationcontext " +
	"getbasepagesource getcachedaftertimerange getcachedwithin getcatch getcfid getcfmlfactory getcftoken " +
	"getcollection getconfig getcurrentlevel getcurrentpagesource getcurrenttag getcurrenttemplatedialect " +
	"getcurrenttemplatepagesource getdatasource getdatasourcemanager getdebugger getexecutiontime " +
	"getfullnullsupport getfunction getfunctionwithnamedvalues gethttpservletrequest gethttpservletresponse getid " +
	"getiterator getjsessionid getlocale getmailservers getormsession getparentpagecontext getpsq getquery " +
	"getreference getremoteuser getrequestdialect getrequesttimeout getrequesttimeoutexception getresourcecharset " +
	"getresponsestream getroottemplatedirectory getrootwriter getsessiontype getstarttime gettemplatepath getthread " +
	"getthreadscope getthreadscopenames gettimezone geturltoken getvariable getvariableutil getwebcharset " +
	"handlepageexception hasfamily ignorescopes initbody issilent loadcomponent loadinline localget localscope " +
	"localtouch outputend outputstart param releasebody removelastpagesource removevariable requestscope scope " +
	"serialize serverscope sessionscope set setapplicationcontext setcatch setcfoutputonly seterrorpage " +
	"setexecutiontime setfunctionscopes setheader setlocale setpsq setremoteuser setrequesttimeout setsilent " +
	"setthreadscope settimezone setvariable setvariablesscope subparam throwcatch topagesource touch undefinedscope " +
	"unsetsilent urlformscope urlscope us variablesscope write writeencodefor writepsq"

const luceeConfigAPI = "" +
	"allowcompression allowimplicidquerycall allowrealpath allowrequesttimeout checkforchangesinconfigfile " +
	"checkpassword checkpermgenspace cleandatasourceconnectionpools clearapplicationcache clearcomponentcache " +
	"clearctcache clearfunctioncache closeconnection contentlength createclusterscope debug debuglogoutput " +
	"docomponentdeepsearch docustomtagdeepsearch dolocalcustomtag getactionmonitor getadminsync getadminsyncclass " +
	"getaiengine getaienginenames getaienginepool getallextensionbundledefintions getallrhextensions getamfengine " +
	"getantisamypolicy getapplicationlistener getapplicationpagesource getapplicationpathcachetimeout " +
	"getapplicationtimeout getbasecomponentpagesource getbasecomponenttemplate getbufferoutput getcacheconnections " +
	"getcachedaftertimerange getcachedefaultconnection getcachedefaultconnectionname getcachedefinition " +
	"getcachedefinitions getcachedir getcachedirsize getcachedpage getcachedwithin getcachehandlercollection " +
	"getcachehandlers getcfmlwritertype getcfxtagpool getcgiscopereadonly getclassdirectory getclassesdirectory " +
	"getclassloader getclassloadercore getclassloaderenv getclassloaderloader getclientcluster getclientscopedir " +
	"getclientscopedirsize getclientstorage getclienttimeout getclienttype getclusterclass getcompiletype " +
	"getcomponentdatamemberdefaultaccess getcomponentdefaultimport getcomponentdumptemplate getcomponentlocalsearch " +
	"getcomponentmappings getcomponentrootsearch getconfigdir getconfigfile getconfigserver getconfigserverdir " +
	"getconstants getcoretaglib getctinitfile getcustomtagextensions getcustomtagmappings getdatasource " +
	"getdatasourceconnectionpool getdatasourceconnectionpools getdatasources getdatasourcesasmap " +
	"getdatecasterclassicstyle getdebugentries getdebugentry getdebugmaxrecordslogged getdebugtemplate " +
	"getdefaultdatasource getdefaultdumpwriter getdefaultencoding getdefaultfunctionmapping " +
	"getdefaultfunctionoutput getdefaultresourceprovider getdefaulttagmapping getdeploydirectory " +
	"getdirectclassloader getdotnotationuppercase getdumpwriter getengine geterrorstatuscode geterrortemplate " +
	"geterrwriter geteventgatewaydirectory getexecutionlogenabled getexecutionlogfactory getextensionavailabledir " +
	"getextensionbundledefintions getextensiondirectory getextensioninstalleddir getextensionproviders " +
	"getextensions getexternalizestringgte getfactory getfldfile getflds getformurlasstruct getfromfunctioncache " +
	"getfullnullsupport getfunctionmapping getfunctionmappings gethandleunquotedattrvalueasstring getid " +
	"getidentification getinspecttemplate getinspecttemplateautointerval getintervallmonitor getintervallmonitors " +
	"getjavasettings getjdbcdriverbybundle getjdbcdriverbycd getjdbcdriverbyclassname getjdbcdriverbyid " +
	"getjdbcdrivers getlabel getlibrarydirectory getloadtime getlocale getlocalextensionproviderdirectory " +
	"getlocalmode getlockmanager getlog getlogdirectory getlogengine getlogincaptcha getlogindelay getlognames " +
	"getmaildefaultcharset getmailservers getmailspoolinterval getmailtimeout getmainlogger getmappings getmavendir " +
	"getormconfig getormengine getormengineclassdefintion getoutwriter getpagesource getpagesourceexisting " +
	"getpagesources getpasswordorigin getpasswordsalt getpasswordtype getplugindirectory getprecisemath " +
	"getproxydata getpsql getqueryvarusage getqueueenable getqueuemax getqueuetimeout getregex getrememberme " +
	"getremoteclientdirectory getremoteclients getremoteclientusage getrequestmonitor getrequestmonitors " +
	"getrequesttimeout getresource getresourcecharset getresourcecharset getresourceproviderfactories " +
	"getresourceproviders getresources getrestlist getrestmappings getrestsetting getreturnformat " +
	"getrhextensionproviders getrhextensions getrootdirectory getrpcclassloader getsalt getscheduler " +
	"getscopecascadingtype getscriptprotect getsearchengine getsearchengineclassdefinition getsearchenginedirectory " +
	"getsecretprovider getsecretproviders getsecuritydirectory getsecuritymanager getserialnumber " +
	"getserverrhextensions getsessioncluster getsessionscopedir getsessionstorage getsessiontimeout getsessiontype " +
	"getshowdebug getshowdoc getshowmetric getshowtest getspoolerengine getstartups getsuppresswsbeforearg " +
	"gettagdefaultattributevalues gettagmapping gettagmappings gettempdirectory gettemplatecharset getthreadqueue " +
	"gettimeserver gettimeserveroffset gettimezone gettldfile gettlds gettriggercomponentdatamember gettypechecking " +
	"getupdatelocation getupdatetype getversion getvideodirectory getvideoexecuterclass getwebcharset getwebcharset " +
	"getwriter hasdebugoptions hasormengine haspassword hasresourceprovider hasserverpassword " +
	"isallowurlrequesttimeout isclientcookies isclientmanagement isdevelopmode isdomaincookies isextensionenabled " +
	"isloggingloaded ismailsendpartial ismailspoolenable ismonitoringenabled ispasswordequal isproxyenablefor " +
	"issessionmanagement isshowversion issuppresscontent issuppresswhitespace isuserset lastmodified " +
	"limitevaluation listcomponentcache listctcache loadlocalextensions mergeformandurl passwordequal preservecase " +
	"putapplicationpagesource putcachedpagesource putctinitfile puttofunctioncache reloadtimeserveroffset " +
	"removedatasourceconnectionpool reset resetmappings resetormengine resetrestmappings setjavasettings " +
	"setlastmodified setpassword topagesource usecomponentpathcache usecomponentshadow usectpathcache"

func init() {
	erasedMembers("lucee.pagecontext", luceePageContextAPI)
	erasedMembers("lucee.config", luceeConfigAPI)
	erasedMembers("query", luceeQueryAPI)
}

// luceeQueryAPI is the Java surface of lucee.runtime.type.Query and QueryImpl,
// which a Lucee test calls on a query directly.
const luceeQueryAPI = "addcolumn addrow call callwithnamedvalues charat clear clone containskey enableshowqueryusage " +
	"executiontime get getat getcachetype getcolumncount getcolumnlist getcolumnnames getcolumnnamesasstring " +
	"getexecutiontime getgeneratedkeys getmeta getmetadatasimple getname getsql getstring gettemplate gettypes " +
	"gettypesasmap getupdatecount go iscached keys length remove removecolumn removecolumnel removeel " +
	"removerow removerowel rename set setat setatel setcachetype setcached setel setexecutiontime size"

// erasedMembers adds each name to the contract of component that it does not
// already have, with an erased result.
func erasedMembers(component, names string) {
	c := contracts[component]
	if c == nil {
		return
	}

	for name := range strings.FieldsSeq(names) {
		if _, typed := c[name]; !typed {
			c[name] = "$any"
		}
	}
}
