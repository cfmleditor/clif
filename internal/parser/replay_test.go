package parser

import "testing"

// TestReplayReadsWhatTheScannerReads: a replaying scanner hands back the
// tokens a live one scans, peeks as it does, and restores to a token's offset
// (the token just read, the one in front, or any other) as it does. The
// passes after a parse read a region's tokens this way, and a difference
// would change what they find without any test of theirs noticing.
func TestReplayReadsWhatTheScannerReads(t *testing.T) {
	src := "component {\n\tfunction f( a ) {\n\t\t/* c */ var x = a.b[ 1 ].c( \"s #g(\"q\")# t\" ); // n\n\t\treturn x;\n\t}\n}\n"
	toks := tokenize(src)
	replay := newReplayScanner(src, &toks)

	live := NewScanner(src)
	live.interpStrings = true

	for i := 0; ; i++ {
		if lp, rp := live.PeekSkipComments(), replay.PeekSkipComments(); lp != rp {
			t.Fatalf("peek %d: live %+v, replay %+v", i, lp, rp)
		}

		lt, rt := live.NextSkipComments(), replay.NextSkipComments()
		if lt != rt {
			t.Fatalf("token %d: live %+v, replay %+v", i, lt, rt)
		}

		if lt.Kind == TokEOF {
			break
		}
	}

	for _, back := range []int{0, 5, len(toks) - 3, len(toks) - 2} {
		at := ScannerState{pos: toks[back].Offset, line: toks[back].Line}
		live.Restore(at)
		replay.Restore(at)

		if lt, rt := live.NextSkipComments(), replay.NextSkipComments(); lt != rt {
			t.Errorf("after restoring to token %d: live %+v, replay %+v", back, lt, rt)
		}
	}
}
