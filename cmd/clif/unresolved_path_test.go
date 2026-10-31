package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The index and the report have to name a file the same way. Scanning `.` built
// the index from relative paths while the files to report were absolute, so a
// function's own siblings were not found under the name the scan asked with:
// `s = zoned( … ); s.toLocalDate()` was "variable 's' has no component ref"
// for `.` and resolved for the same directory spelled out.
func TestUnresolvedDoesNotDependOnHowTheRootIsSpelled(t *testing.T) {
	dir := t.TempDir()

	src := `<cfcomponent output="false">

<cffunction name="whenText" access="public" returntype="string" output="false">
	<cfargument name="ev" type="struct" required="true">
	<cfset var s = zoned(arguments.ev.startUtc, arguments.ev.timeZone)>
	<cfset var e = zoned(arguments.ev.endUtc, arguments.ev.timeZone)>
	<cfreturn s.toLocalDate().equals(e.toLocalDate())>
</cffunction>

<cffunction name="zoned" access="private" returntype="any" output="false">
	<cfargument name="utc" type="string" required="true">
	<cfargument name="zoneId" type="string" required="true">
	<cfreturn createObject("java", "java.time.Instant").parse(arguments.utc).atZone(createObject("java", "java.time.ZoneId").of(arguments.zoneId))>
</cffunction>

</cfcomponent>
`
	if err := os.WriteFile(filepath.Join(dir, "Events.cfc"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	absolute, _, _ := scanUnresolved([]string{dir}, &unresolvedFlags{})

	t.Chdir(dir)

	for _, spelling := range []string{".", "./", "Events.cfc"} {
		relative, _, _ := scanUnresolved([]string{spelling}, &unresolvedFlags{})

		if len(relative.Calls) != len(absolute.Calls) {
			t.Fatalf("%q reports %d unresolved calls, the absolute path %d: %v", spelling, len(relative.Calls), len(absolute.Calls), relative.Calls)
		}
	}

	if len(absolute.Calls) != 0 {
		t.Fatalf("the fixture should resolve completely, got %v", absolute.Calls)
	}
}
