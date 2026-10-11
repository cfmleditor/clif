package resolve

// SpeculativeReceiver identifies a receiver within a file and calling function.
// It intentionally cannot distinguish separate values assigned to that spelling.
type SpeculativeReceiver struct {
	File     string
	Caller   string
	Variable string
}
