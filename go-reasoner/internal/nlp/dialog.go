package nlp

import "strings"

type Turn struct {
	User        string
	Bot         string
	Intent      Intent
	Topic       string
	NodeID      string
	FiredRules  []string
	Explanation string
}

type Dialog struct {
	History        []Turn
	LastAction     string
	LastNodeID     string
	LastRuleIDs    []string
	LastTopic      string
	LastProofTrace string
}

func (d *Dialog) Record(t Turn) {
	if t.Topic != "" {
		d.LastTopic = t.Topic
	}
	if t.NodeID != "" {
		d.LastNodeID = t.NodeID
	}
	if len(t.FiredRules) > 0 {
		d.LastRuleIDs = append([]string{}, t.FiredRules...)
	}
	if t.Explanation != "" {
		d.LastProofTrace = t.Explanation
	}
	d.History = append(d.History, t)
	if len(d.History) > 20 {
		d.History = d.History[len(d.History)-20:]
	}
}

func (d *Dialog) ResolveTopic(extracted string) string {
	cleaned := strings.ToLower(strings.TrimSpace(extracted))
	if cleaned == "" || cleaned == "it" || cleaned == "that" || cleaned == "this" ||
		cleaned == "isso" || cleaned == "isto" || cleaned == "ele" || cleaned == "ela" {
		return d.LastTopic
	}
	return extracted
}
