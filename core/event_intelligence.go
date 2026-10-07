package core

// Phase 2: Event Intelligence. This layer deliberately stays deterministic and
// local. It turns a Story cluster into an Event-like object with independent
// source count, confidence, importance, velocity and lifecycle signals.

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	eventDetected    = "detected"
	eventConfirmed   = "confirmed"
	eventDeveloping  = "developing"
	eventStabilizing = "stabilizing"
	eventArchived    = "archived"
)

func sourceIdentity(it Item) string {
	if it.Link != "" {
		if u, err := url.Parse(it.Link); err == nil && u.Hostname() != "" {
			h := strings.ToLower(u.Hostname())
			h = strings.TrimPrefix(h, "www.")
			if h != "" {
				return h
			}
		}
	}
	s := strings.ToLower(strings.TrimSpace(it.Source))
	s = strings.TrimPrefix(s, "www.")
	return s
}

// independentSourceKeys lists the distinct origins behind a story: outlets
// count separately unless they carry the same wire-agency copy or the same
// body text (see wire.go).
func independentSourceKeys(c *Story) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, k := range originKeys(c) {
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	return out
}

func independentSourceCount(c *Story) int { return len(independentSourceKeys(c)) }

// Primary sources are first-party/authoritative feeds. This is intentionally a
// conservative allowlist, not a claim that every article from these domains is true.
func isPrimarySource(it Item) bool {
	h := sourceIdentity(it)
	if h == "" {
		return false
	}
	primary := []string{
		"afad.gov.tr", "usgs.gov", "gdacs.org", "nasa.gov", "noaa.gov",
		"who.int", "un.org", "europa.eu", "ecb.europa.eu", "tcmb.gov.tr",
		"tuik.gov.tr", "cisa.gov", "cert.europa.eu", "nato.int",
	}
	for _, p := range primary {
		if h == p || strings.HasSuffix(h, "."+p) {
			return true
		}
	}
	return false
}

func primarySourceCount(c *Story) int {
	seen := map[string]bool{}
	n := 0
	for _, it := range c.items {
		k := sourceIdentity(it)
		if k == "" || seen[k] || !isPrimarySource(it) {
			continue
		}
		seen[k] = true
		n++
	}
	return n
}

func eventConfidence(c *Story) int {
	n := independentSourceCount(c)
	primary := primarySourceCount(c)
	score := 18
	switch {
	case n >= 6:
		score += 50
	case n >= 5:
		score += 44
	case n >= 4:
		score += 38
	case n >= 3:
		score += 31
	case n >= 2:
		// Two genuinely independent sources are enough to move an event
		// beyond the "limited corroboration" tier. Keep room for primary
		// source and volume bonuses without overstating certainty.
		score += 32
	}
	score += primary * 12
	if c.Count >= 2 {
		score += 5
	}
	if c.Count >= 5 {
		score += 4
	}
	if score > 100 {
		score = 100
	}
	return score
}

func eventVelocity(c *Story) float64 {
	if len(c.items) < 2 {
		return float64(len(c.items))
	}
	oldest, newest := c.items[0].Published, c.items[0].Published
	have := false
	for _, it := range c.items {
		if !it.HasDate {
			continue
		}
		if !have || it.Published.Before(oldest) {
			oldest = it.Published
		}
		if !have || it.Published.After(newest) {
			newest = it.Published
		}
		have = true
	}
	if !have {
		return float64(len(c.items))
	}
	hours := newest.Sub(oldest).Hours()
	if hours < 0.05 {
		hours = 0.05
	}
	return float64(len(c.items)) / hours
}

func eventLifecycle(c *Story) string {
	now := time.Now()
	age := now.Sub(c.Published)
	if age < 0 {
		age = 0
	}
	v := eventVelocity(c)
	n := independentSourceCount(c)
	switch {
	case age >= 72*time.Hour:
		return eventArchived
	case v >= 1.5 || (c.Breaking && age <= 2*time.Hour):
		return eventDeveloping
	case n >= 2:
		if age <= 12*time.Hour {
			return eventConfirmed
		}
		return eventStabilizing
	case age <= 2*time.Hour:
		return eventDetected
	default:
		return eventStabilizing
	}
}

func eventID(c *Story) string {
	toks := tokenize(c.Title)
	keys := make([]string, 0, len(toks))
	for t := range toks {
		keys = append(keys, t)
	}
	sort.Strings(keys)
	h := sha256.Sum256(append(append([]byte(strings.Join(keys, " ")), 0), []byte(strings.ToLower(c.Category))...))
	return hex.EncodeToString(h[:12])
}

func importanceScore(c *Story) int {
	s := scoreStory(c)
	independent := independentSourceCount(c)
	primary := primarySourceCount(c)
	velocity := eventVelocity(c)
	if independent >= 3 {
		s += 6
	}
	if independent >= 5 {
		s += 4
	}
	if primary > 0 {
		s += 8
	}
	if velocity >= 2 {
		s += 10
	} else if velocity >= 1 {
		s += 6
	}
	if eventLifecycle(c) == eventDeveloping {
		s += 8
	}
	if s > 100 {
		s = 100
	}
	return s
}

func finalizeEvent(c *Story) {
	c.EventID = eventID(c)
	c.IndependentSources = independentSourceCount(c)
	c.PrimarySources = primarySourceCount(c)
	c.Confidence = eventConfidence(c)
	c.Velocity = eventVelocity(c)
	c.Lifecycle = eventLifecycle(c)
	c.Score = importanceScore(c)
}
