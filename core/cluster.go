package core

// Deduplicator: groups near-duplicate headlines (the same event reported by
// several sources) into a Story. Clustering is greedy — each item joins the
// first existing cluster whose seed title is similar enough and close in time,
// otherwise it starts a new cluster. Items are expected pre-sorted newest-first.

import (
	"sort"
	"time"
)

const (
	simThreshold  = 0.50           // overlap coefficient to be "the same story"
	minShared     = 2              // min shared significant tokens to merge
	clusterWindow = 36 * time.Hour // max time gap to merge into one story
)

// Story is a cluster of items about one event.
type Story struct {
	Title     string
	Link      string
	Category  string
	Sources   []string // unique source names, in first-seen order
	Count     int      // number of member headlines
	Published time.Time
	FirstSeen time.Time
	HasDate   bool
	Breaking  bool
	Score     int
	Kind      string
	Summary   string // representative item's feed summary (reader fallback)
	Image     string // first thumbnail found among the members

	// Event Intelligence fields. These are derived locally from the cluster.
	EventID            string  `json:"eventId"`
	IndependentSources int     `json:"independentSources"`
	PrimarySources     int     `json:"primarySources"`
	Confidence         int     `json:"confidence"`
	Velocity           float64 `json:"velocity"`
	Lifecycle          string  `json:"lifecycle"`

	items   []Item          // members (for timeline in later phases)
	seedTok map[string]bool // tokens of the seed title
	srcSeen map[string]bool // dedupe sources
}

// clusterItems groups a category's items into stories.
func clusterItems(items []Item) []*Story {
	// Do not let rolling data/list pages manufacture event clusters or timeline
	// entries. The raw feed remains cached separately for traceability.
	items = filterEventItems(items)
	var clusters []*Story
	for _, it := range items {
		tok := clusterTokens(it.Title)
		var best *Story
		bestSim := simThreshold
		for _, c := range clusters {
			if !withinWindow(c, it) {
				continue
			}
			sim, inter := overlapCoef(tok, c.seedTok)
			if inter >= minShared && sim >= bestSim {
				bestSim = sim
				best = c
			}
		}
		if best == nil {
			c := newStory(it, tok)
			clusters = append(clusters, c)
		} else {
			best.add(it)
		}
	}
	for _, c := range clusters {
		finalizeStory(c)
	}
	return clusters
}

func withinWindow(c *Story, it Item) bool {
	if !c.HasDate || !it.HasDate {
		return true // undated: don't block merging on time
	}
	d := c.Published.Sub(it.Published)
	if d < 0 {
		d = -d
	}
	return d <= clusterWindow
}

func newStory(it Item, tok map[string]bool) *Story {
	c := &Story{
		Title:     it.Title,
		Link:      it.Link,
		Category:  it.Category,
		Published: it.Published,
		FirstSeen: it.Published,
		HasDate:   it.HasDate,
		Breaking:  it.Breaking,
		Summary:   it.Summary,
		Image:     it.Image,
		items:     []Item{it},
		seedTok:   tok,
		srcSeen:   map[string]bool{},
	}
	c.Count = 1
	if it.Source != "" && !c.srcSeen[it.Source] {
		c.srcSeen[it.Source] = true
		c.Sources = append(c.Sources, it.Source)
	}
	return c
}

func (c *Story) add(it Item) {
	c.items = append(c.items, it)
	c.Count++
	if it.Source != "" && !c.srcSeen[it.Source] {
		c.srcSeen[it.Source] = true
		c.Sources = append(c.Sources, it.Source)
	}
	if it.Breaking {
		c.Breaking = true
	}
	if c.Image == "" && it.Image != "" {
		c.Image = it.Image
	}
	// newest published wins for the representative time; prefer a breaking or
	// newer item's title as the representative headline.
	if it.HasDate && (!c.HasDate || it.Published.After(c.Published)) {
		c.Published = it.Published
		c.HasDate = true
		c.Title = it.Title
		c.Link = it.Link
		if it.Summary != "" {
			c.Summary = it.Summary
		}
	}
	if it.HasDate && it.Published.Before(c.FirstSeen) {
		c.FirstSeen = it.Published
	}
}

// rankStories orders stories by score (desc), newest first on ties.
func rankStories(cl []*Story) {
	sort.SliceStable(cl, func(i, j int) bool {
		if cl[i].Score != cl[j].Score {
			return cl[i].Score > cl[j].Score
		}
		return cl[i].Published.After(cl[j].Published)
	})
}

// sortItemsNewest orders items newest-first (dated before undated).
func sortItemsNewest(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].HasDate != items[j].HasDate {
			return items[i].HasDate
		}
		return items[i].Published.After(items[j].Published)
	})
}

func finalizeStory(c *Story) {
	// Keep members newest-first for later timeline use.
	sort.SliceStable(c.items, func(i, j int) bool {
		return c.items[i].Published.After(c.items[j].Published)
	})
	c.Kind = kindOf(c)
	finalizeEvent(c)
}
