package nlp

import "testing"

func TestTokenizeNormalizesAndDropsStopwords(t *testing.T) {
	got := Tokenize("O Que é uma Goroutine?")
	for _, tok := range got {
		if tok == "o" || tok == "uma" || tok == "e" {
			t.Fatalf("stopword leaked: %v", got)
		}
	}
	if len(got) == 0 {
		t.Fatal("expected tokens")
	}
}

func TestClassifier(t *testing.T) {
	nb := NewNB()
	Seed(nb)
	cases := map[string]Intent{
		"oi":                 IntentGreeting,
		"Olá!":               IntentGreeting,
		"bom dia":            IntentGreeting,
		"valeu":              IntentThanks,
		"tchau":              IntentFarewell,
		"escreva uma funcao": IntentCodeRequest,
		"por que":            IntentWhy,
		"me fale sobre":      IntentQuestion,
	}
	for in, want := range cases {
		if got, _ := nb.Classify(in); got != want {
			t.Errorf("Classify(%q) = %s, want %s", in, got, want)
		}
	}
	if in, _ := nb.Classify("zzzz qqqq"); in != IntentUnknown {
		t.Errorf("gibberish should be unknown, got %s", in)
	}
}
