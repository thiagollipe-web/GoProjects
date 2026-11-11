package memory

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type NodeType string

const (
	NodeConcept NodeType = "concept"
	NodeFact    NodeType = "fact"
	NodeRule    NodeType = "rule"
	NodeCode    NodeType = "code"
)

type Node struct {
	ID        string    `json:"id"`
	Type      NodeType  `json:"type"`
	Label     string    `json:"label"`
	Content   string    `json:"content"`
	Weight    float64   `json:"weight"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Edge struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Rel    string  `json:"rel"`
	Weight float64 `json:"weight"`
}

type Graph struct {
	mu    sync.RWMutex
	Nodes map[string]*Node `json:"nodes"`
	Edges []Edge           `json:"edges"`
	path  string
}

func Open(path string) (*Graph, error) {
	g := &Graph{Nodes: map[string]*Node{}, path: path}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return g, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, g); err != nil {
			return nil, err
		}
		if g.Nodes == nil {
			g.Nodes = map[string]*Node{}
		}
	}
	return g, nil
}

func (g *Graph) Save() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.saveLocked()
}

func (g *Graph) saveLocked() error {
	if g.path == "" {
		return nil
	}
	tmp := g.path + ".tmp"
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, g.path)
}

func (g *Graph) Upsert(n *Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	if e, ok := g.Nodes[n.ID]; ok {
		e.Content, e.Label, e.Type = n.Content, n.Label, n.Type
		e.UpdatedAt = now
		if n.Weight != 0 {
			e.Weight = n.Weight
		}
		_ = g.saveLocked()
		return
	}
	n.CreatedAt, n.UpdatedAt = now, now
	if n.Weight == 0 {
		n.Weight = 1.0
	}
	g.Nodes[n.ID] = n
	_ = g.saveLocked()
}

func (g *Graph) Link(from, to, rel string, weight float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if weight == 0 {
		weight = 1.0
	}
	for i := range g.Edges {
		if g.Edges[i].From == from && g.Edges[i].To == to && g.Edges[i].Rel == rel {
			g.Edges[i].Weight = weight
			_ = g.saveLocked()
			return
		}
	}
	g.Edges = append(g.Edges, Edge{
		From:   from,
		To:     to,
		Rel:    rel,
		Weight: weight,
	})
	_ = g.saveLocked()
}

func (g *Graph) Neighbors(id, rel string) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []*Node
	for _, e := range g.Edges {
		if e.From == id && (rel == "" || e.Rel == rel) {
			if target, ok := g.Nodes[e.To]; ok {
				out = append(out, target)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Weight > out[j].Weight
	})
	return out
}

func (g *Graph) Get(id string) (*Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.Nodes[id]
	return n, ok
}

func (g *Graph) Search(label string) (*Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	label = strings.TrimSpace(label)
	if label == "" {
		return nil, false
	}
	for _, n := range g.Nodes {
		if strings.EqualFold(n.Label, label) {
			return n, true
		}
	}
	matches := g.searchFuzzyLocked(label)
	if len(matches) > 0 {
		return matches[0], true
	}
	return nil, false
}

func (g *Graph) SearchFuzzy(query string) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.searchFuzzyLocked(query)
}

func (g *Graph) searchFuzzyLocked(query string) []*Node {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	qToks := strings.Fields(q)
	type scored struct {
		node  *Node
		score float64
	}
	var hits []scored
	for _, n := range g.Nodes {
		if n.Type == NodeRule {
			continue
		}
		lbl := strings.ToLower(n.Label)
		var base float64
		if lbl == q {
			base = 3.0
		} else if strings.Contains(lbl, q) || strings.Contains(q, lbl) {
			base = 1.5
		} else {
			for _, qt := range qToks {
				if len(qt) > 2 && strings.Contains(lbl, qt) {
					base += 0.6
				}
			}
		}
		if base > 0 {
			hits = append(hits, scored{node: n, score: base * n.Weight})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score == hits[j].score {
			return hits[i].node.ID < hits[j].node.ID
		}
		return hits[i].score > hits[j].score
	})
	out := make([]*Node, len(hits))
	for i, h := range hits {
		out[i] = h.node
	}
	return out
}

func (g *Graph) All() []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]*Node, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Weight == out[j].Weight {
			return out[i].ID < out[j].ID
		}
		return out[i].Weight > out[j].Weight
	})
	return out
}

func (g *Graph) RulesWithMinWeight(minWeight float64) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []*Node
	for _, n := range g.Nodes {
		if n.Type == NodeRule && n.Weight >= minWeight {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Weight > out[j].Weight
	})
	return out
}

func (g *Graph) AdjustWeight(id string, delta float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if n, ok := g.Nodes[id]; ok {
		n.Weight = clampWeight(n.Weight + delta)
		n.UpdatedAt = time.Now()
		for i := range g.Edges {
			if g.Edges[i].From == id || g.Edges[i].To == id {
				g.Edges[i].Weight = clampWeight(g.Edges[i].Weight + 0.5*delta)
			}
		}
		_ = g.saveLocked()
	}
}

func clampWeight(w float64) float64 {
	if w < 0.1 {
		return 0.1
	}
	if w > 5.0 {
		return 5.0
	}
	return w
}
