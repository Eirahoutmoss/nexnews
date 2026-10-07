package core

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type eventGraphNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Score    int    `json:"score,omitempty"`
	Region   string `json:"region,omitempty"`
	Category string `json:"category,omitempty"`
}

type eventGraphEdge struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Weight float64 `json:"weight"`
	Type   string  `json:"type"`
}

type eventGraph struct {
	Updated int              `json:"updated"`
	Nodes   []eventGraphNode `json:"nodes"`
	Edges   []eventGraphEdge `json:"edges"`
}

func graphTokenSimilarity(a, b *Story) float64 {
	if a == nil || b == nil {
		return 0
	}
	at, bt := clusterTokens(a.Title), clusterTokens(b.Title)
	if len(at) == 0 || len(bt) == 0 {
		return 0
	}
	shared := 0
	for t := range at {
		if bt[t] {
			shared++
		}
	}
	den := len(at)
	if len(bt) < den {
		den = len(bt)
	}
	if den == 0 {
		return 0
	}
	return float64(shared) / float64(den)
}

func graphSafeID(prefix, value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return prefix + "-" + strings.Trim(b.String(), "-")
}

func (s *Store) eventGraphJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := map[string]bool{}
	items := make([]*Story, 0, len(s.important))
	for _, st := range s.important {
		if st == nil || s.isHidden(st) || st.EventID == "" || seen[st.EventID] {
			continue
		}
		seen[st.EventID] = true
		items = append(items, st)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].Published.After(items[j].Published)
	})
	if len(items) > 10 {
		items = items[:10]
	}

	g := eventGraph{Updated: int(time.Now().Unix())}
	for _, st := range items {
		region, _ := pulseRegionForCategory(st.Category)
		g.Nodes = append(g.Nodes, eventGraphNode{ID: st.EventID, Label: st.Title, Type: "event", Score: st.Score, Region: region, Category: st.Category})
	}

	seenSource := map[string]bool{}
	for _, st := range items {
		for _, src := range st.Sources {
			if len(seenSource) >= 24 {
				break
			}
			id := graphSafeID("src", src)
			if src == "" || seenSource[id] {
				continue
			}
			seenSource[id] = true
			g.Nodes = append(g.Nodes, eventGraphNode{ID: id, Label: src, Type: "source"})
			g.Edges = append(g.Edges, eventGraphEdge{Source: st.EventID, Target: id, Weight: 1, Type: "source"})
		}
		region, _ := pulseRegionForCategory(st.Category)
		rid := graphSafeID("region", region)
		if !seenSource[rid] {
			seenSource[rid] = true
			g.Nodes = append(g.Nodes, eventGraphNode{ID: rid, Label: region, Type: "region"})
		}
		g.Edges = append(g.Edges, eventGraphEdge{Source: st.EventID, Target: rid, Weight: 0.9, Type: "region"})
		cid := graphSafeID("cat", st.Category)
		if !seenSource[cid] {
			seenSource[cid] = true
			g.Nodes = append(g.Nodes, eventGraphNode{ID: cid, Label: st.Category, Type: "category"})
		}
		g.Edges = append(g.Edges, eventGraphEdge{Source: st.EventID, Target: cid, Weight: 0.7, Type: "category"})
	}

	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			sim := graphTokenSimilarity(items[i], items[j])
			if sim >= 0.34 {
				g.Edges = append(g.Edges, eventGraphEdge{Source: items[i].EventID, Target: items[j].EventID, Weight: sim, Type: "related"})
			}
		}
	}
	return json.Marshal(g)
}
