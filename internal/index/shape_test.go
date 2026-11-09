package index

import (
	"testing"

	"go.lsp.dev/uri"
)

// shapeMoves runs write and reports whether ShapeGeneration moved.
func shapeMoves(idx *Index, write func()) bool {
	before := idx.ShapeGeneration()

	write()

	return idx.ShapeGeneration() != before
}

func TestShapeIgnoresAFileNoNameCanReach(t *testing.T) {
	t.Parallel()

	idx := New()
	idx.IndexFile(uri.File("/ws/Base.cfc"), `component {}`)
	idx.IndexFile(uri.File("/ws/Kid.cfc"), `component extends="Base" {}`)

	if shapeMoves(idx, func() { idx.IndexFile(uri.File("/ws/Plain.cfc"), `component { function go(){} }`) }) {
		t.Error("a new file that extends nothing, which no extends names, moved the shape")
	}

	if shapeMoves(idx, func() { idx.IndexFile(uri.File("/ws/Plain.cfc"), `component { function other(){} }`) }) {
		t.Error("re-indexing it moved the shape")
	}

	if shapeMoves(idx, func() { idx.RemoveFile(uri.File("/ws/Plain.cfc")) }) {
		t.Error("removing it moved the shape")
	}

	if shapeMoves(idx, func() { idx.SetExtends(uri.File("/ws/Unindexed.cfc"), "") }) {
		t.Error("recording that an unindexed file extends nothing moved the shape")
	}
}

func TestShapeMovesWhenAnExtendsCouldReachTheFile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		setup func(*Index)
		write func(*Index)
	}{
		{"a file that extends", func(*Index) {}, func(i *Index) { i.IndexFile(uri.File("/ws/Kid.cfc"), `component extends="Base" {}`) }},
		{
			"a file whose name an extends spells", func(i *Index) { i.IndexFile(uri.File("/ws/Kid.cfc"), `component extends="a.b.Helper" {}`) },
			func(i *Index) { i.IndexFile(uri.File("/ws/other/Helper.cfc"), `component {}`) },
		},
		{
			"a file whose name a module id spells", func(i *Index) { i.IndexFile(uri.File("/ws/Kid.cfc"), `component extends="Svc@mod" {}`) },
			func(i *Index) { i.IndexFile(uri.File("/ws/Svc.cfc"), `component {}`) },
		},
		{
			"a file whose entity name an extends spells", func(i *Index) { i.IndexFile(uri.File("/ws/Kid.cfc"), `component extends="cbAuthor" {}`) },
			func(i *Index) {
				i.IndexFile(uri.File("/ws/Author.cfc"), `component persistent="true" entityname="cbAuthor" {}`)
			},
		},
		{"a ModuleConfig, which WireBox ids are read from", func(*Index) {}, func(i *Index) { i.IndexFile(uri.File("/ws/m/ModuleConfig.cfc"), `component {}`) }},
		{
			"a file once a computed extends could name anything", func(i *Index) { i.IndexFile(uri.File("/ws/Kid.cfc"), `component extends="#base#" {}`) },
			func(i *Index) { i.IndexFile(uri.File("/ws/Plain.cfc"), `component {}`) },
		},
		{"a named file's removal", func(i *Index) {
			i.IndexFile(uri.File("/ws/Kid.cfc"), `component extends="Helper" {}`)
			i.IndexFile(uri.File("/ws/Helper.cfc"), `component {}`)
		}, func(i *Index) { i.RemoveFile(uri.File("/ws/Helper.cfc")) }},
		{
			"a file that stops extending", func(i *Index) { i.IndexFile(uri.File("/ws/Kid.cfc"), `component extends="Base" {}`) },
			func(i *Index) { i.IndexFile(uri.File("/ws/Kid.cfc"), `component {}`) },
		},
		{
			"an extends recorded for an indexed file", func(i *Index) { i.IndexFile(uri.File("/ws/Kid.cfc"), `component {}`) },
			func(i *Index) { i.SetExtends(uri.File("/ws/Kid.cfc"), "Base") },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			idx := New()
			tc.setup(idx)

			if !shapeMoves(idx, func() { tc.write(idx) }) {
				t.Error("the shape did not move, so a cached answer would outlive the change")
			}
		})
	}
}

func TestShapeStaysWhenAFileIsIndexedAgainUnchanged(t *testing.T) {
	t.Parallel()

	idx := New()
	u := uri.File("/ws/Kid.cfc")
	idx.IndexFile(u, `component extends="Base" { function a(){} }`)

	if shapeMoves(idx, func() { idx.IndexFile(u, `component extends="Base" { function b(){} }`) }) {
		t.Error("re-indexing a file with the same extends moved the shape")
	}
}

func TestMentionsFollowTheExtendsRecorded(t *testing.T) {
	t.Parallel()

	idx := New()
	idx.IndexFile(uri.File("/ws/Kid.cfc"), `component extends="x.Helper" {}`)
	idx.IndexFile(uri.File("/ws/Kid.cfc"), `component extends="x.Other" {}`)

	if idx.mentions["helper"] != 0 || idx.mentions["other"] != 1 {
		t.Errorf("mentions = %v, want other alone", idx.mentions)
	}

	idx.RemoveFile(uri.File("/ws/Kid.cfc"))

	if len(idx.mentions) != 0 || idx.mentionAll != 0 {
		t.Errorf("mentions = %v (%d computed) after the only extender went", idx.mentions, idx.mentionAll)
	}
}
