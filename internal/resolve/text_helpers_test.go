package resolve

import "testing"

func TestPrecededByLocalDetectsPrefix(t *testing.T) {
	text := "local.foo"
	if !precededByLocal(text, len("local.")) {
		t.Fatal("local. prefix not detected")
	}

	if precededByLocal("xlocal.foo", len("xlocal.")) {
		t.Fatal("prefix inside a larger word reported")
	}

	if precededByLocal("foo", 3) {
		t.Fatal("short text reported")
	}
}

func TestIsRegexSpaceCoversWhitespace(t *testing.T) {
	for _, c := range []byte{' ', '\t', '\n', '\f', '\r'} {
		if !isRegexSpace(c) {
			t.Errorf("isRegexSpace(%q) = false", c)
		}
	}

	if isRegexSpace('a') {
		t.Fatal("letter reported as space")
	}
}

func TestWordStartBoundary(t *testing.T) {
	if !wordStart("foo", 0) {
		t.Fatal("start of string not a boundary")
	}

	if wordStart("afoo", 1) {
		t.Fatal("inside a word reported as boundary")
	}

	if !wordStart("a foo", 2) {
		t.Fatal("after space not a boundary")
	}
}
