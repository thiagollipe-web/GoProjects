package memory

import (
	"path/filepath"
	"testing"
)

func TestGraphPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mem.json")
	g, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	g.Upsert(&Node{ID: "concept:go", Type: NodeConcept, Label: "go", Content: "linguagem", Weight: 1})
	g.Upsert(&Node{ID: "concept:chan", Type: NodeConcept, Label: "channel", Content: "canal", Weight: 1})
	g.Link("concept:go", "concept:chan", "related", 1)
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	g2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := g2.Search("go"); !ok || n.Content != "linguagem" {
		t.Fatalf("node not restored: %+v %v", n, ok)
	}
	if ns := g2.Neighbors("concept:go", "related"); len(ns) != 1 || ns[0].ID != "concept:chan" {
		t.Fatalf("edge not restored: %v", ns)
	}
}

func TestAdjustWeightIsClamped(t *testing.T) {
	g, _ := Open(filepath.Join(t.TempDir(), "m.json"))
	g.Upsert(&Node{ID: "a", Label: "a", Weight: 1})
	g.AdjustWeight("a", 1000)
	hi, _ := g.Get("a")
	g.AdjustWeight("a", -1000)
	lo, _ := g.Get("a")
	if hi.Weight > 10 || lo.Weight < 0 {
		t.Fatalf("weight not clamped: hi=%v lo=%v", hi.Weight, lo.Weight)
	}
}
