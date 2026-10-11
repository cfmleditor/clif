package parser

// NativeTypePrefix distinguishes runtime interfaces from CFC component paths.
const NativeTypePrefix = "$builtin.native."

// NativeJavaComponent identifies supported Java classes during type extraction.
// Explicit configured Java stubs are consulted before this fallback.
func NativeJavaComponent(class string) string {
	names := map[string]string{
		"java.lang.Object": "object", "java.lang.String": "string", "java.lang.StringBuilder": "stringbuilder", "java.lang.StringBuffer": "stringbuilder",
		"java.lang.Class": "class", "java.util.HashMap": "hashmap", "java.util.LinkedHashMap": "hashmap", "java.util.concurrent.ConcurrentHashMap": "map", "java.util.Map": "map",
		"java.util.ArrayList": "list", "java.util.LinkedList": "list", "java.util.List": "list", "java.util.HashSet": "set", "java.util.Set": "set", "java.util.Iterator": "iterator",
		"java.sql.Connection": "connection", "java.sql.Statement": "statement", "java.sql.PreparedStatement": "preparedstatement", "java.sql.ResultSet": "resultset", "java.sql.DriverManager": "drivermanager",
		"java.util.Date": "date", "java.sql.Date": "date", "java.sql.Timestamp": "date",
		"java.time.Instant": "instant", "java.time.LocalDateTime": "localdatetime", "java.time.LocalDate": "localdate", "java.time.LocalTime": "localtime", "java.time.ZonedDateTime": "zoneddatetime", "java.time.OffsetDateTime": "offsetdatetime",
		"java.time.ZoneId": "zoneid", "java.time.ZoneOffset": "zoneid", "java.time.Duration": "duration", "java.time.temporal.ChronoUnit": "enum", "java.time.temporal.ChronoField": "enum", "java.time.DayOfWeek": "enum", "java.time.temporal.TemporalAdjusters": "adjusters", "java.time.format.DateTimeFormatter": "formatter",
		"org.hibernate.Session": "hibernatesession", "org.hibernate.SessionFactory": "sessionfactory", "org.hibernate.stat.SessionStatistics": "sessionstatistics", "org.hibernate.stat.Statistics": "statistics",
	}
	if name := names[class]; name != "" {
		return NativeTypePrefix + name
	}

	return ""
}
