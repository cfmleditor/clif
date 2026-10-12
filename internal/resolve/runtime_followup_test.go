package resolve

import "testing"

func TestRuntimeFollowupChainsAndCallbacks(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){
 var data=structNew();
 data.some(function(key,value,collection){key.repeatString(2).len();collection.keyArray();value.ready();}).toString();
 data.every(function(otherKey,otherValue,otherCollection){otherKey.len();otherCollection.keyList();otherValue.ready();}).toString();
 var text="1";
 text.repeatString(2).len();text.numberFormat().len();
 var number=12;
 number.numberFormat().len();
 var when=now();
 when.setYear(2025).setMonth(1).setDay(2).setHour(3).setMinute(4).setSecond(5).setMilliSecond(6).year();
 var values=[];
 values.removeDuplicates().len();values.getMetadata().keyArray();
 var pc=getPageContext();
 pc.getCurrentPageSource().getMapping().getVirtual().len();
 pc.getThread().getThreadGroup().getParent().getName().len();
 pc.getCurrentPageSource().getMapping().getPageSource("example.cfm").getFileName().len();
 pc.getConfig().getDatasourceConnectionPools().iterator().next().getFactory().getDatasource().getName().len();
 values.removeDuplicates().absent();text.removeDuplicates();number.setYear(2025);
 }}`})

	got := reasonsWith(t, &Resolver{NativeEngine: "lucee"}, dir, "Use.cfc")
	if reason, ok := got["pc.getConfig.getDatasourceConnectionPools.iterator.next.getFactory.getDatasource.getName.len"]; !ok || reason != "" {
		t.Errorf("pool factory datasource: %q present=%v", reason, ok)
	}

	if reason, ok := got["pc.getCurrentPageSource.getMapping.getPageSource.getFileName.len"]; !ok || reason != "" {
		t.Errorf("round-trip page source: %q present=%v", reason, ok)
	}

	for _, name := range []string{"data.some.toString", "data.every.toString", "key.repeatString.len", "collection.keyArray", "otherKey.len", "otherCollection.keyList", "text.repeatString.len", "text.numberFormat.len", "number.numberFormat.len", "when.setYear.setMonth.setDay.setHour.setMinute.setSecond.setMilliSecond.year", "values.removeDuplicates.len", "values.getMetadata.keyArray", "pc.getCurrentPageSource.getMapping.getVirtual.len", "pc.getThread.getThreadGroup.getParent.getName.len"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"value.ready", "otherValue.ready", "values.removeDuplicates.absent", "text.removeDuplicates", "number.setYear"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must remain diagnosed: %q present=%v", name, reason, ok)
		}
	}

	generic := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	if reason := generic["pc.getCurrentPageSource.getMapping.getVirtual.len"]; reason == "" {
		t.Fatal("generic engine borrowed Lucee PageContext")
	}
}
