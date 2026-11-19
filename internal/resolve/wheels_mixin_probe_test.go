package resolve

import (
	"path/filepath"
	"testing"
)

func TestWheelsMixinSkipsHostWithoutAncestor(t *testing.T) {
	fsys := &sourceStatCounter{}
	r := &Resolver{FS: fsys}
	controller := filepath.Join(t.TempDir(), "Controller.cfc")

	if r.wheelsObjectMixinFunc([]string{controller}, "render") != nil {
		t.Fatal("a controller without an ancestor supplied a Wheels mixin")
	}

	if got := fsys.stats.Load(); got != 0 {
		t.Fatalf("checked source metadata %d times for a host that cannot inherit the loader", got)
	}
}
