package nlp

import (
	"regexp"
	"strings"
)

type Entities struct {
	Topic       string
	Quoted      []string
	Preference  string
	UseXInstead string
	InsteadOfY  string
	NegatedPref bool
	CodeID      []string
	Raw         string
}

var (
	reQuote    = regexp.MustCompile(`"([^"]+)"|'([^']+)'`)
	reQuestion = regexp.MustCompile(`(?i)\b(?:what|who|where|when|why|how|o que|quem|onde|como)\s+(?:is|are|was|were|does|did|do|e|é|sao|são|foi|funciona)\s+(.+)`)
	reTellMe   = regexp.MustCompile(`(?i)\b(?:tell me about|explain|define|me fale sobre|explique|defina)\s+(.+)`)
	reUseX     = regexp.MustCompile(`(?i)\buse\s+([A-Za-z0-9_./:-]+)\s+(?:instead of|em vez de|no lugar de)\s+([A-Za-z0-9_./:-]+)`)
	reAlways   = regexp.MustCompile(`(?i)\b(?:always use|prefer|from now on use|sempre use|prefiro|a partir de agora use)\s+(.+)`)
	reNever    = regexp.MustCompile(`(?i)\b(?:never use|don't use|do not use|nunca use|nao use|não use)\s+(.+)`)
	reCodeReq  = regexp.MustCompile(`(?i)\b(?:write|generate|implement|create|show|escreva|gere|implemente|crie|mostre)\s+(?:me\s+)?(?:a\s+|um\s+|uma\s+)?(?:code\s+for\s+|codigo\s+para\s+|funcao\s+para\s+|function\s+for\s+)?(.+)`)
	reIdent    = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)+\b`)
)

func Extract(s string) Entities {
	e := Entities{Raw: s}
	for _, m := range reQuote.FindAllStringSubmatch(s, -1) {
		if m[1] != "" {
			e.Quoted = append(e.Quoted, m[1])
		} else if m[2] != "" {
			e.Quoted = append(e.Quoted, m[2])
		}
	}
	if m := reUseX.FindStringSubmatch(s); m != nil {
		e.UseXInstead, e.InsteadOfY = m[1], m[2]
		e.Preference = "replace:" + m[2] + "->" + m[1]
	}
	if e.Preference == "" {
		if m := reAlways.FindStringSubmatch(s); m != nil {
			e.Preference = cleanTopic(m[1])
		}
	}
	if e.Preference == "" {
		if m := reNever.FindStringSubmatch(s); m != nil {
			e.Preference = cleanTopic(m[1])
			e.NegatedPref = true
		}
	}
	if m := reQuestion.FindStringSubmatch(s); m != nil {
		e.Topic = cleanTopic(m[1])
	}
	if e.Topic == "" {
		if m := reTellMe.FindStringSubmatch(s); m != nil {
			e.Topic = cleanTopic(m[1])
		}
	}
	if e.Topic == "" {
		if m := reCodeReq.FindStringSubmatch(s); m != nil {
			e.Topic = cleanTopic(m[1])
		}
	}
	e.CodeID = reIdent.FindAllString(s, -1)
	return e
}

func cleanTopic(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, ".?!,;:")
	for _, suf := range []string{
		" in go", " in golang", " please", " for me",
		" em go", " em golang", " por favor", " pra mim", " para mim",
	} {
		if strings.HasSuffix(strings.ToLower(s), suf) {
			s = s[:len(s)-len(suf)]
		}
	}
	return strings.TrimSpace(s)
}
