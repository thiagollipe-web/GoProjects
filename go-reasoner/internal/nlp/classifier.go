package nlp

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

type Intent string

const (
	IntentGreeting    Intent = "greeting"
	IntentFarewell    Intent = "farewell"
	IntentThanks      Intent = "thanks"
	IntentQuestion    Intent = "question"
	IntentCodeRequest Intent = "code_request"
	IntentTeach       Intent = "teach"
	IntentWhy         Intent = "why"
	IntentPositive    Intent = "feedback_positive"
	IntentNegative    Intent = "feedback_negative"
	IntentSmallTalk   Intent = "smalltalk"
	IntentUnknown     Intent = "unknown"
)

type Prediction struct {
	Intent     Intent
	Confidence float64
	Margin     float64
}

type NB struct {
	classCount    map[Intent]int
	vocab         map[Intent]map[string]int
	total         int
	vocabSize     map[string]bool
	MinConfidence float64
	UniformPriors bool
	exact         map[string]Intent // normalized training example -> intent
}

func NewNB() *NB {
	return &NB{
		classCount:    make(map[Intent]int),
		vocab:         make(map[Intent]map[string]int),
		vocabSize:     make(map[string]bool),
		exact:         make(map[string]Intent),
		MinConfidence: 0.20,
		UniformPriors: true,
	}
}

func (nb *NB) Train(intent Intent, examples ...string) {
	if nb.vocab[intent] == nil {
		nb.vocab[intent] = make(map[string]int)
	}
	for _, ex := range examples {
		nb.exact[normalizePhrase(ex)] = intent
		nb.classCount[intent]++
		nb.total++
		for _, t := range Tokenize(ex) {
			nb.vocab[intent][t]++
			nb.vocabSize[t] = true
		}
	}
}

func (nb *NB) Classify(text string) (Intent, float64) {
	pred := nb.ClassifyDetailed(text)
	return pred.Intent, pred.Confidence
}

// normalizePhrase lowercases, strips accents and punctuation and collapses spaces.
func normalizePhrase(s string) string {
	var b strings.Builder
	for _, r := range Normalize(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func (nb *NB) ClassifyDetailed(text string) Prediction {
	// Fast path: a single-word or short phrase identical to a training example
	// is unambiguous, while Naive Bayes confidence is naturally low for 1 token.
	if in, ok := nb.exact[normalizePhrase(text)]; ok {
		return Prediction{Intent: in, Confidence: 1, Margin: 1}
	}
	toks := Tokenize(text)
	if len(toks) == 0 || len(nb.classCount) == 0 {
		return Prediction{Intent: IntentUnknown, Confidence: 0}
	}

	knownTokens := 0
	for _, t := range toks {
		if nb.vocabSize[t] {
			knownTokens++
		}
	}
	if knownTokens == 0 {
		return Prediction{Intent: IntentUnknown, Confidence: 0}
	}

	V := float64(len(nb.vocabSize))
	numClasses := float64(len(nb.classCount))

	type scoredIntent struct {
		intent Intent
		score  float64
	}
	var ranked []scoredIntent

	for intent, cc := range nb.classCount {
		var s float64
		if nb.UniformPriors {
			s = math.Log(1.0 / numClasses)
		} else {
			s = math.Log(float64(cc) / float64(nb.total))
		}
		var tot int
		for _, c := range nb.vocab[intent] {
			tot += c
		}
		for _, t := range toks {
			c := float64(nb.vocab[intent][t])
			s += math.Log((c + 1.0) / (float64(tot) + V))
		}
		ranked = append(ranked, scoredIntent{intent: intent, score: s})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].intent < ranked[j].intent
		}
		return ranked[i].score > ranked[j].score
	})

	best := ranked[0]
	var sum float64
	for _, r := range ranked {
		sum += math.Exp(r.score - best.score)
	}
	conf := 1.0 / sum

	var secondConf float64
	if len(ranked) > 1 {
		secondConf = math.Exp(ranked[1].score-best.score) / sum
	}
	margin := conf - secondConf

	if conf < nb.MinConfidence {
		return Prediction{Intent: IntentUnknown, Confidence: conf, Margin: margin}
	}
	return Prediction{Intent: best.intent, Confidence: conf, Margin: margin}
}

func Seed(nb *NB) {
	nb.Train(IntentGreeting,
		"hello", "hi", "hey", "good morning", "good evening", "howdy", "hi there", "greetings",
		"oi", "ola", "bom dia", "boa tarde", "boa noite", "e ai", "salve", "opa")
	nb.Train(IntentFarewell,
		"bye", "goodbye", "see you", "later", "exit", "quit", "i'm done",
		"tchau", "ate mais", "ate logo", "sair", "encerrar", "falou")
	nb.Train(IntentThanks,
		"thanks", "thank you", "thx", "appreciate it",
		"obrigado", "obrigada", "valeu", "agradecido", "muito obrigado")
	nb.Train(IntentQuestion,
		"what is", "who is", "tell me about", "explain", "how does",
		"what are", "where is", "when did", "define", "meaning of",
		"o que e", "quem e", "me fale sobre", "explique", "como funciona",
		"defina", "qual o significado de", "onde fica")
	nb.Train(IntentCodeRequest,
		"write a function", "generate code", "write go", "code for",
		"implement", "create a function", "write a program", "show me code",
		"give me a snippet", "write a handler", "worker pool code",
		"escreva uma funcao", "gere codigo", "implemente", "crie um handler",
		"codigo em go", "faca um programa", "mostre um exemplo de codigo")
	nb.Train(IntentTeach,
		"remember that", "from now on", "always use", "never use",
		"i prefer", "use instead of", "learn this", "note that", "keep in mind", "don't use",
		"lembre que", "a partir de agora", "sempre use", "nunca use",
		"eu prefiro", "use em vez de", "aprenda isso", "anote que", "nao use")
	nb.Train(IntentWhy,
		"why did you", "justify", "explain your reasoning", "proof", "how did you conclude",
		"por que", "justifique", "explique o raciocinio", "qual a prova", "como chegou nisso")
	nb.Train(IntentPositive,
		"good job", "nice", "great", "perfect", "that's right", "correct", "well done", "awesome",
		"bom trabalho", "otimo", "perfeito", "isso mesmo", "correto", "muito bom", "excelente", "mandou bem")
	nb.Train(IntentNegative,
		"no", "wrong", "that's not right", "incorrect", "not what i wanted", "bad", "nope", "try again", "fix it",
		"nao", "errado", "incorreto", "nao era isso", "ruim", "tente de novo", "corrija isso", "pessimo")
	nb.Train(IntentSmallTalk,
		"how are you", "what's up", "how's it going", "you ok", "tell me a joke", "what do you do",
		"como vai voce", "tudo bem", "como estao as coisas", "conte uma piada", "o que voce faz")
}
