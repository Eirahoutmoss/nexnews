package core

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Phase 3: Evidence and Source Intelligence. These signals are deliberately
// explainable and deterministic. NexNews describes the evidence structure; it
// never declares a story "true" solely from its score.

type SourceEvidence struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	Type        string `json:"type"` // primary / media / other
	Independent bool   `json:"independent"`
	Primary     bool   `json:"primary"`
	Via         string `json:"via,omitempty"` // wire agency whose copy this outlet ran
}

type EventEvidence struct {
	IndependentSources int              `json:"independentSources"`
	PrimarySources     int              `json:"primarySources"`
	Agreement          int              `json:"agreement"`
	Sources            []SourceEvidence `json:"sources"`
	Summary            string           `json:"summary"`
	Syndicated         int              `json:"syndicated"` // outlets echoing another origin
	Agencies           []string         `json:"agencies,omitempty"`
}

func sourceType(it Item) string {
	if isPrimarySource(it) {
		return "primary"
	}
	h := sourceIdentity(it)
	if h == "" {
		return "other"
	}
	return "media"
}

func evidenceForStory(c *Story) EventEvidence {
	seen := map[string]bool{}
	keys := originKeys(c)
	var out []SourceEvidence
	for i, it := range c.items {
		d := sourceIdentity(it)
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		_, via := wireOf(it)
		out = append(out, SourceEvidence{Name: it.Source, Domain: d, Type: sourceType(it), Independent: keys[i] == d, Primary: isPrimarySource(it), Via: via})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Primary != out[j].Primary {
			return out[i].Primary
		}
		return out[i].Name < out[j].Name
	})
	agreement := c.Confidence
	summary := "Sınırlı bağımsız kaynak desteği var."
	switch {
	case c.IndependentSources >= 4:
		summary = "Birden fazla bağımsız kaynak aynı olayı bildiriyor."
	case c.IndependentSources >= 2:
		summary = "En az iki bağımsız kaynak arasında mutabakat bulunuyor."
	}
	if c.PrimarySources > 0 {
		summary += " Birincil/resmî kaynak sinyali mevcut."
	}
	primaryCount := 0
	for _, src := range out {
		if src.Primary {
			primaryCount++
		}
	}
	synd, agencies := syndicatedCount(c)
	if synd > 0 {
		if len(agencies) > 0 {
			summary += " " + stringCount(synd) + " kaynak aynı ajans metnini (" + strings.Join(agencies, ", ") + ") kullanıyor; bunlar tek kaynak sayıldı."
		} else {
			summary += " " + stringCount(synd) + " kaynak aynı metni yayımlıyor; bunlar tek kaynak sayıldı."
		}
	}
	return EventEvidence{IndependentSources: c.IndependentSources, PrimarySources: primaryCount, Agreement: agreement, Sources: out, Summary: summary, Syndicated: synd, Agencies: agencies}
}

func eventReason(c *Story) string {
	reasons := []string{}
	if c.Breaking {
		reasons = append(reasons, "Son dakika sinyali")
	}
	if c.IndependentSources >= 2 {
		reasons = append(reasons, stringCount(c.IndependentSources)+" bağımsız kaynak")
	}
	if c.PrimarySources > 0 {
		reasons = append(reasons, "birincil/resmî kaynak")
	}
	if n, ag := syndicatedCount(c); n > 0 && len(ag) > 0 {
		reasons = append(reasons, strings.Join(ag, ", ")+" metni")
	}
	if c.Velocity >= 2 {
		reasons = append(reasons, "haber akışı hızlanıyor")
	}
	if c.Confidence >= 75 {
		reasons = append(reasons, "yüksek kaynak mutabakatı")
	}
	if len(reasons) == 0 {
		return "" // nothing beyond recency to explain; keep the list quiet
	}
	return strings.Join(reasons, " · ")
}

func stringCount(n int) string { return strconv.Itoa(n) }

func pulseScore(stories []*Story) int {
	if len(stories) == 0 {
		return 0
	}
	now := time.Now()
	weighted := 0.0
	total := 0.0
	for _, st := range stories {
		age := now.Sub(st.Published)
		if age < 0 {
			age = 0
		}
		hours := age.Hours()
		rec := 1.0
		switch {
		case hours <= 1:
			rec = 1.0
		case hours <= 6:
			rec = .75
		case hours <= 24:
			rec = .45
		default:
			rec = .15
		}
		w := float64(st.Score+st.Confidence) / 2 * rec
		weighted += w
		total += 100 * rec
	}
	v := int(weighted / total * 100)
	if v > 100 {
		v = 100
	}
	if v < 0 {
		v = 0
	}
	return v
}

func pulseBand(v int) string {
	switch {
	case v >= 85:
		return "çok yüksek"
	case v >= 70:
		return "yüksek"
	case v >= 45:
		return "normal"
	case v >= 25:
		return "düşük"
	default:
		return "sakin"
	}
}
