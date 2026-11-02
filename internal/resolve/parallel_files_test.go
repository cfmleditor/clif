package resolve

import (
	"fmt"
	"testing"
)

func TestForEachFileSurvivesAPanicOnOneFile(t *testing.T) {
	files := make([]string, 50)
	for i := range files {
		files[i] = fmt.Sprintf("f%02d.cfm", i)
	}

	done := make([]bool, len(files))

	forEachFile(files, func(i int, file string) {
		if file == "f07.cfm" {
			panic("a bad file")
		}

		done[i] = true
	})

	for i, ok := range done {
		if ok == (i == 7) {
			t.Fatalf("file %d: done %v", i, ok)
		}
	}
}
