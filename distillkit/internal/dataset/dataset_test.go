package dataset

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func sample() *Base {
	return &Base{Articles: map[string]Article{
		"a": {Question: "O que é um buraco negro?", Area: "Astronomia", Answer: strings.Repeat("Resposta longa. ", 5) + "\nNOVAS PERGUNTAS:\n1. algo"},
		"b": {Question: "O que é um  Buraco Negro", Area: "Astronomia", Answer: strings.Repeat("Outra resposta longa. ", 5)}, // duplicate of a
		"c": {Question: "Curta?", Area: "Física", Answer: "curta"},
		"d": {Question: "O que é DNA?", Answer: strings.Repeat("DNA guarda informação genética. ", 3)},
	}}
}

func TestBuildCleansFiltersAndDedupes(t *testing.T) {
	ex, rep := Build(sample(), Options{})
	if rep.Total != 4 || rep.Kept != 2 || rep.TooShort != 1 || rep.Duplicates != 1 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	for _, e := range ex {
		if strings.Contains(e.Output, "NOVAS PERGUNTAS") {
			t.Fatal("trailing marker not removed")
		}
	}
	if rep.PerArea["Ciência Universal"] != 1 {
		t.Fatalf("default area missing: %v", rep.PerArea)
	}
}

func TestFormats(t *testing.T) {
	e := Example{Instruction: "Q?", Area: "A", Output: "R"}
	for _, f := range []string{"alpaca", "sharegpt", "gemma"} {
		var buf bytes.Buffer
		if err := Write(&buf, e, f); err != nil {
			t.Fatal(f, err)
		}
		if !json.Valid(bytes.TrimSpace(buf.Bytes())) || !strings.HasSuffix(buf.String(), "\n") {
			t.Fatalf("%s: not a JSON line: %q", f, buf.String())
		}
	}
	var g bytes.Buffer
	_ = Write(&g, e, "gemma")
	if !strings.Contains(g.String(), "<start_of_turn>model") || !strings.Contains(g.String(), "<end_of_turn>") {
		t.Fatalf("gemma template wrong: %s", g.String())
	}
	if err := Write(&bytes.Buffer{}, e, "xml"); err == nil {
		t.Fatal("unknown format must error")
	}
}

func TestSplitIsDeterministicAndRoughlyProportional(t *testing.T) {
	if IsValidation("q", 0) {
		t.Fatal("0 must disable validation")
	}
	n, hits := 5000, 0
	for i := 0; i < n; i++ {
		q := "pergunta número " + strings.Repeat("x", i%7) + string(rune('a'+i%26)) + string(rune('A'+i/26%26)) + string(rune('0'+i/676))
		if IsValidation(q, 0.1) != IsValidation(q, 0.1) {
			t.Fatal("not deterministic")
		}
		if IsValidation(q, 0.1) {
			hits++
		}
	}
	if hits < n/20 || hits > n/5 {
		t.Fatalf("validation share off: %d/%d", hits, n)
	}
}

func TestExportFiles(t *testing.T) {
	ex, _ := Build(sample(), Options{})
	dir := t.TempDir()
	tr, va, err := Export(ex, "alpaca", filepath.Join(dir, "t.jsonl"), filepath.Join(dir, "v.jsonl"), 0)
	if err != nil || tr != len(ex) || va != 0 {
		t.Fatalf("tr=%d va=%d err=%v", tr, va, err)
	}
}
