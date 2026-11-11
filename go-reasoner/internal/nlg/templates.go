package nlg

import (
	"math/rand"
	"strings"
	"sync"
)

var (
	greetings = []string{
		"E aí. O que vamos construir hoje?",
		"Oi! Pronto quando você estiver.",
		"Olá — o que está na cabeça?",
		"E aí. Pergunte qualquer coisa ou me diga o que construir.",
	}
	thanks = []string{
		"De nada.", "Disponha.", "Fico feliz que ajudou.",
	}
	farewells = []string{
		"Até a próxima.", "Cuide-se.", "Tchau — vou lembrar do que cobrimos.",
	}
	acks    = []string{"Anotado.", "Entendido.", "Beleza.", "Guardado."}
	posAcks = []string{
		"Bom — vou dar mais peso a essa abordagem.",
		"Que bom que funcionou. Reforçando esse padrão.",
	}
	negAcks = []string{
		"Certo. Vou evitar isso.",
		"Entendi — penalizando essa abordagem.",
	}
	smallTalk = []string{
		"Só um monte de mapas e canais por aqui. E você?",
		"Rodando em raciocínio simbólico e café. Em que posso ajudar?",
		"Tranquilo. Descobrindo as coisas uma regra de cada vez.",
	}
	notFound = []string{
		"Ainda não tenho isso na memória — deixa eu procurar.",
		"Vou checar uma fonte externa pra isso.",
	}

	mu       sync.Mutex
	lastPick = map[*[]string]int{}
)

func pick(xs *[]string) string {
	s := *xs
	if len(s) == 0 {
		return ""
	}
	if len(s) == 1 {
		return s[0]
	}
	mu.Lock()
	defer mu.Unlock()
	prev, hasPrev := lastPick[xs]
	idx := rand.Intn(len(s))
	if hasPrev && idx == prev {
		idx = (idx + 1) % len(s)
	}
	lastPick[xs] = idx
	return s[idx]
}

func Greet() string       { return pick(&greetings) }
func Thank() string       { return pick(&thanks) }
func Farewell() string    { return pick(&farewells) }
func Ack() string         { return pick(&acks) }
func PositiveAck() string { return pick(&posAcks) }
func NegativeAck() string { return pick(&negAcks) }
func SmallTalk() string   { return pick(&smallTalk) }
func NotFound() string    { return pick(&notFound) }

func Fill(tmpl string, vars map[string]string) string {
	for k, v := range vars {
		tmpl = strings.ReplaceAll(tmpl, "{"+k+"}", v)
	}
	return tmpl
}
