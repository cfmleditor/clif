package nativetype

// Fixed collection interfaces verified against the generated catalogue and
// Lucee's function library. Values/elements remain erased; argument-dependent
// results such as directoryList, deserializeJSON and duplicate are excluded.
var collectionProducers = map[string]string{
	"arrayfindall": "array", "arrayfindallnocase": "array",
	"arrayfilter": "array", "arraymap": "array", "arraymid": "array",
	"arraymerge": "array", "arrayreverse": "array", "arrayslice": "array",
	"arraysplice": "array", "arrayremoveduplicates": "array",
	"listtoarray": "array", "listqualifiedtoarray": "array",
	"rematch": "array", "rematchnocase": "array",
	"structkeyarray": "array", "structvaluearray": "array",
	"structfindkey": "array", "structfindvalue": "array", "structsort": "array",
	"componentlistpackage": "array", "entitynamearray": "array",
	"querycolumnarray": "array", "querycolumndata": "array",
	"structfilter": "struct", "structmap": "struct", "structtosorted": "struct",
	"getapplicationsettings": "struct", "getapplicationmetadata": "struct",
	"gethttprequestdata": "struct", "gethttprequestheaders": "struct",
	"gettaglist": "struct", "gettagdata": "struct", "directoryinfo": "struct",
}
