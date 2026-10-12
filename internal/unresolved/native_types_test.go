package unresolved

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

func TestNativeTypesTrackResultsAndQualifiedBuiltinFailures(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Use.cfc")

	src := `component {
 function run(){
 var q=queryNew("id","integer");
 q.toString().trim();
 q.getColumn("id").size();
 q.toString().ready();
 var sessionStats=ormGetSession().getStatistics();
 sessionStats.getEntityCount();
 sessionStats.setStatisticsEnabled(true);
 var factoryStats=ormGetSessionFactory().getStatistics();
 factoryStats.setStatisticsEnabled(true);
 factoryStats.getSessionOpenCount();
 var map=createObject("java","java.util.HashMap").init();
 var entries=map.entrySet().iterator();
 entries.hasNext();
 entries.next().getKey().toString();
 var conn=createObject("java","java.sql.DriverManager").getConnection("jdbc:test");
 var rs=conn.createStatement().executeQuery("select 1");
 rs.next();
 rs.next().getKey();
 map.clone();
 map.finalize();
 var localTime=createObject("java","java.time.LocalDateTime").now();
 localTime.plusDays(1).toLocalDate().getYear();
 }
}`
	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	rep := Scan(vfs.OS{}, []string{file}, nil, &Options{WorkspaceFolders: []string{dir}})

	expected := map[string]string{"ready": "native", "setStatisticsEnabled": "sessionstatistics", "getKey": "boolean", "finalize": "native"}
	for _, call := range rep.Calls {
		want, ok := expected[call.Function]
		if !ok || !strings.Contains(strings.ToLower(call.Reason), want) {
			t.Errorf("unexpected finding: %+v", call)
		}

		delete(expected, call.Function)
	}

	if len(expected) > 0 {
		t.Errorf("missing diagnostics: %v; got %+v", expected, rep.Calls)
	}
}

func TestNativeBuiltinAndToStringContractsRespectSourceOverrides(t *testing.T) {
	dir := t.TempDir()

	var files []string

	for name, src := range map[string]string{
		"Product.cfc": `component {function ready(){}}`,
		"Use.cfc": `component {
   Product function ormGetSession(){return new Product();}
   Product function toString(){return new Product();}
   function run(){ormGetSession().ready();this.toString().ready();}
  }`,
	} {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		files = append(files, file)
	}

	rep := Scan(vfs.OS{}, files, nil, &Options{WorkspaceFolders: []string{dir}})
	if len(rep.Calls) != 0 {
		t.Fatalf("native rules displaced source contracts: %+v", rep.Calls)
	}
}

func TestNativeLiteralInitializerDoesNotTypeUnknownLoopElement(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Use.cfc")

	src := `component {function run(items){
 var obj="";
 for(obj in arguments.items){
  obj.save();
 }
 }}`
	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	rep := Scan(vfs.OS{}, []string{file}, nil, &Options{WorkspaceFolders: []string{dir}})
	if len(rep.Calls) != 1 || !strings.Contains(rep.Calls[0].Reason, "no component ref") {
		t.Fatalf("initializer leaked into loop: %+v", rep.Calls)
	}
}
