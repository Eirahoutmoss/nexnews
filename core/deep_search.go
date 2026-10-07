package core

import (
	"encoding/json"
	"sort"
	"strings"
)

type deepEntity struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type deepSource struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type deepEvent struct {
	Title      string `json:"title"`
	EventID    string `json:"eventId,omitempty"`
	Score      int    `json:"score"`
	Confidence int    `json:"confidence"`
	Sources    int    `json:"sources"`
	Lifecycle  string `json:"lifecycle"`
	Category   string `json:"category"`
	Reason     string `json:"reason,omitempty"`
	Link       string `json:"link"`
}

type deepSearchBoard struct {
	Query       string       `json:"query"`
	WindowLabel string       `json:"windowLabel"`
	Total       int          `json:"total"`
	SourceCount int          `json:"sourceCount"`
	Entities    []deepEntity `json:"entities"`
	Sources     []deepSource `json:"sources"`
	Events      []deepEvent  `json:"events"`
}

// deepSearchJSON turns the existing event search into a compact research dossier.
// It deliberately reuses the same clustering/ranking pipeline as normal search,
// so Deep Search never invents a second interpretation of the news stream.
func (s *Store) deepSearchJSON(query string) ([]byte, error) {
	b, err := s.searchJSON(query)
	if err != nil {
		return nil, err
	}
	var base outSearch
	if err := json.Unmarshal(b, &base); err != nil {
		return nil, err
	}
	board := deepSearchBoard{Query: base.Query, WindowLabel: base.WindowLabel, Total: base.Total, SourceCount: base.SourceCount}

	entityCounts := map[string]*deepEntity{}
	sourceCounts := map[string]int{}
	for _, it := range base.Items {
		for _, src := range it.Sources {
			if strings.TrimSpace(src) != "" {
				sourceCounts[src]++
			}
		}
		for _, e := range entityMatches(it.Title + " " + it.Summary + " " + it.Category) {
			id := normEntity(e.Type + ":" + e.Name)
			if entityCounts[id] == nil {
				entityCounts[id] = &deepEntity{Name: e.Name, Type: e.Type}
			}
			entityCounts[id].Count++
		}
		board.Events = append(board.Events, deepEvent{
			Title: it.Title, EventID: it.EventID, Score: it.Score, Confidence: it.Confidence,
			Sources: it.IndependentSources, Lifecycle: it.Lifecycle, Category: it.Category,
			Reason: it.Reason, Link: it.Link,
		})
	}
	for _, e := range entityCounts {
		board.Entities = append(board.Entities, *e)
	}
	sort.SliceStable(board.Entities, func(i, j int) bool {
		if board.Entities[i].Count != board.Entities[j].Count {
			return board.Entities[i].Count > board.Entities[j].Count
		}
		return board.Entities[i].Name < board.Entities[j].Name
	})
	if len(board.Entities) > 16 {
		board.Entities = board.Entities[:16]
	}
	for name, count := range sourceCounts {
		board.Sources = append(board.Sources, deepSource{Name: name, Count: count})
	}
	sort.SliceStable(board.Sources, func(i, j int) bool { return board.Sources[i].Count > board.Sources[j].Count })
	if len(board.Sources) > 10 {
		board.Sources = board.Sources[:10]
	}
	if len(board.Events) > 10 {
		board.Events = board.Events[:10]
	}
	return json.Marshal(board)
}
