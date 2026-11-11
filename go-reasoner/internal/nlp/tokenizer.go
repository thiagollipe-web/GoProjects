package nlp

import (
	"strings"
	"unicode"
)

var stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"to": true, "of": true, "in": true, "on": true, "for": true,
	"with": true, "and": true, "or": true, "but": true, "if": true,
	"then": true, "so": true, "as": true, "at": true, "by": true,
	"from": true, "it": true, "this": true, "that": true, "these": true,
	"those": true, "i": true, "you": true, "we": true, "they": true,
	"he": true, "she": true, "do": true, "does": true, "did": true,
	"can": true, "could": true, "would": true, "should": true, "will": true,
	"have": true, "has": true, "had": true, "me": true, "my": true,
	"o": true, "os": true, "um": true, "uma": true, "uns": true, "umas": true,
	"de": true, "da": true, "dos": true, "das": true,
	"em": true, "no": true, "na": true, "nos": true, "nas": true,
	"por": true, "para": true, "pra": true, "com": true, "sem": true,
	"e": true, "ou": true, "mas": true, "se": true, "entao": true,
	"como": true, "que": true, "isso": true, "isto": true, "aquilo": true,
	"eu": true, "voce": true, "eles": true, "elas": true,
	"ele": true, "ela": true, "meu": true, "minha": true, "meus": true, "minhas": true,
	"ser": true, "estar": true, "foi": true, "era": true, "sao": true, "esta": true,
}

var accentMap = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a",
	"é", "e", "è", "e", "ê", "e",
	"í", "i", "ì", "i", "î", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o",
	"ú", "u", "ù", "u", "û", "u",
	"ç", "c",
)

func Normalize(s string) string {
	return accentMap.Replace(strings.ToLower(s))
}

func Tokenize(s string) []string {
	s = Normalize(s)
	var out []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		w := b.String()
		b.Reset()
		if stopwords[w] {
			return
		}
		out = append(out, stem(w))
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func stem(w string) string {
	suffixes := []string{
		"mente", "acoes", "acao", "ando", "endo", "indo",
		"aram", "eram", "iram", "edly", "ing", "ava", "iam",
		"ed", "es", "s",
	}
	for _, suf := range suffixes {
		if len(w) > len(suf)+2 && strings.HasSuffix(w, suf) {
			return w[:len(w)-len(suf)]
		}
	}
	return w
}
