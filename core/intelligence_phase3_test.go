package core

import (
	"testing"
	"time"
)

func TestEvidenceIndependentDomains(t *testing.T) {
	c := &Story{Title: "test", Category: "Test", Confidence: 72, IndependentSources: 3, PrimarySources: 1}
	c.items = []Item{{Title: "x", Source: "AFAD", Link: "https://afad.gov.tr/a"}, {Title: "x", Source: "BBC", Link: "https://www.bbc.com/a"}, {Title: "x", Source: "BBC", Link: "https://www.bbc.com/b"}, {Title: "x", Source: "USGS", Link: "https://usgs.gov/a"}}
	e := evidenceForStory(c)
	if len(e.Sources) != 3 || e.IndependentSources != 3 || e.PrimarySources != 2 {
		t.Fatalf("unexpected evidence: %+v", e)
	}
}
func TestPulseRange(t *testing.T) {
	c := &Story{Score: 80, Confidence: 80, Published: nowForTest()}
	p := pulseScore([]*Story{c})
	if p < 0 || p > 100 {
		t.Fatalf("pulse out of range: %d", p)
	}
}
func nowForTest() time.Time { return time.Now() }
