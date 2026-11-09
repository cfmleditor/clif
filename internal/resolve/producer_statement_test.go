package resolve

import "testing"

// firstStatement reads text as the producer plan reads a function body and
// returns the first statement's kind, whether the plan is still valid after it
// (reading it did not fail it), and how many statements the first one's body and alternative hold.
func firstStatement(text string) (kind string, valid bool, body, alternative int) {
	p := producerScript{tokens: significantTokens(text)}
	node := p.statement()

	return node.kind, !p.failed, len(node.body), len(node.alternative)
}

// Each form of statement the plan reads, and the ways it can run out of input:
// a statement cut short fails the plan rather than being read as shorter.
func TestProducerStatementForms(t *testing.T) {
	for _, c := range []struct {
		src             string
		kind            string
		valid           bool
		body, alternate int
	}{
		{`;`, "noop", true, 0, 0},
		{`{ x = 1; }`, "block", true, 1, 0},
		{`{ x = 1;`, "", false, 0, 0},
		{`if ( a ) b = 1;`, "if", true, 1, 0},
		{`if ( a ) b = 1; else c = 2;`, "if", true, 1, 1},
		{`if ( a )`, "", false, 0, 0},
		{`if ( a ) b = 1; else`, "", false, 0, 0},
		{`while ( a ) b = 1;`, "loop", true, 1, 0},
		{`for ( var item in items ) x = item;`, "loop", true, 1, 0},
		{`for ( var i = 0; i < 3; i = i + 1 ) x = i;`, "unsafe", true, 0, 0},
		{`for ( i = 0; i < 3; i++ ) x = i;`, "unsafe", true, 0, 0},
		{`while ( a )`, "", false, 0, 0},
		{`try`, "", false, 0, 0},
		{`try { a = 1; }`, "", false, 0, 0},
		{`try { a = 1; } catch ( any e )`, "", false, 0, 0},
		{`try { a = 1; } catch ( any e ) { b = 2; }`, "try", true, 1, 1},
		{`try { a = 1; } catch ( any e ) { b = 2; } catch ( x e ) { c = 3; }`, "try", true, 1, 1},
		{`try { a = 1; } catch ( any e ) { b = 2; } catch ( x e )`, "", false, 0, 0},
		{`try { a = 1; } catch ( any e ) { b = 2; } finally { c = 3; }`, "try", false, 1, 1},
		{`function f() {}`, "", false, 0, 0},
		{`do { a = 1; } while ( b );`, "", false, 0, 0},
		{`break;`, "", false, 0, 0},
		{`continue;`, "", false, 0, 0},
		{`x = 1;`, "set", true, 0, 0},
	} {
		kind, valid, body, alternate := firstStatement(c.src)
		if kind != c.kind || valid != c.valid || body != c.body || alternate != c.alternate {
			t.Errorf("%q = kind %q valid %v body %d alternative %d, want %q %v %d %d", c.src, kind, valid, body, alternate, c.kind, c.valid, c.body, c.alternate)
		}
	}
}
