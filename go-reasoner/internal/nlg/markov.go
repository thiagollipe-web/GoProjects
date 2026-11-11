package nlg

import (
	"math/rand"
	"strings"
)

type Markov struct {
	order  int
	chain  map[string][]string
	starts []string
}

func NewMarkov(order int) *Markov {
	if order < 1 {
		order = 2
	}
	return &Markov{order: order, chain: map[string][]string{}}
}

func (m *Markov) Train(corpus []string) {
	for _, line := range corpus {
		w := strings.Fields(line)
		if len(w) <= m.order {
			continue
		}
		for i := 0; i <= len(w)-m.order-1; i++ {
			key := strings.Join(w[i:i+m.order], " ")
			if i == 0 {
				m.starts = append(m.starts, key)
			}
			m.chain[key] = append(m.chain[key], w[i+m.order])
		}
	}
}

func (m *Markov) Generate(max int) string {
	return m.GenerateSeeded("", max)
}

func (m *Markov) GenerateSeeded(topic string, max int) string {
	if len(m.starts) == 0 || max <= 0 {
		return ""
	}
	key := ""
	if topic != "" {
		lowTopic := strings.ToLower(topic)
		for _, st := range m.starts {
			if strings.Contains(strings.ToLower(st), lowTopic) {
				key = st
				break
			}
		}
	}
	if key == "" {
		key = m.starts[rand.Intn(len(m.starts))]
	}

	out := strings.Fields(key)
	visited := map[string]int{key: 1}

	for i := 0; i < max; i++ {
		nexts := m.chain[key]
		if len(nexts) == 0 {
			break
		}
		n := nexts[rand.Intn(len(nexts))]
		out = append(out, n)
		key = strings.Join(out[len(out)-m.order:], " ")
		visited[key]++
		if visited[key] > 2 {
			break
		}
	}
	return strings.Join(out, " ")
}
