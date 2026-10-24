package core

// Kriz modu (RC97).
//
// Büyük bir deprem ya da afette insanlar haberi saniyeler içinde paylaşır;
// doğrulanmamış bilgi, eski fotoğraflar ve söylentiler resmî açıklamalardan
// hızlı yayılır. Kullanıcının ülkesinde büyük bir afet algılandığında akışın
// en üstünde sade bir kriz kartı çıkar: yalnızca resmî ve çok kaynakla
// doğrulanmış gelişmeler, acil numaralar ve paylaşmadan önce doğrulama
// hatırlatması. Eşikler bilerek yüksektir: sık yaşanan küçük depremler kriz
// modunu açmaz.

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type crisisInfo struct {
	Key     string            `json:"key"`   // kapatma için kimlik (aynı afet → aynı anahtar)
	Kind    string            `json:"kind"`  // deprem / sel / yangın / tsunami / fırtına / heyelan
	Level   string            `json:"level"` // red / orange
	Mag     float64           `json:"mag,omitempty"`
	Title   string            `json:"title"`
	Link    string            `json:"link"`
	TS      int64             `json:"ts"`
	Country string            `json:"country"`
	Stories []outItem         `json:"stories"` // çok kaynaklı ya da resmî gelişmeler
	Numbers []emergencyNumber `json:"numbers"`
}

var (
	reMagUSGS = regexp.MustCompile(`(?i)^\s*M\s*([0-9]+(?:[.,][0-9])?)\b`)
	reMagText = regexp.MustCompile(`(?i)([0-9]+[.,][0-9])\s*(?:büyüklüğünde|büyüklüğündeki|magnitude|şiddetinde|richter|mw|ml)\b|(?:magnitude|büyüklük)\s*:?\s*([0-9]+[.,][0-9])`)
	reGDACS   = regexp.MustCompile(`(?i)\b(red|orange)\s+(earthquake|flood|tropical cyclone|cyclone|tsunami|volcano|drought|wildfire|forest fire)\b[^.]*\balert\b`)
)

var crisisKinds = []struct {
	kind  string
	terms []string
}{
	{"deprem", []string{"deprem", "earthquake", "quake", "sarsıntı", "erdbeben", "séisme", "terremoto", "sismo", "землетрясение", "زلزال"}},
	{"tsunami", []string{"tsunami", "tsunamisi"}},
	{"sel", []string{" sel ", " seli ", " selde", " sele ", "sel felaketi", "su baskını", "flood", "hochwasser", "inondation", "inundación", "наводнение", "فيضان"}},
	{"yangın", []string{"orman yangını", "wildfire", "forest fire", "waldbrand", "incendie de forêt", "incendio forestal", "лесной пожар"}},
	{"fırtına", []string{"kasırga", "hurricane", "typhoon", "tayfun", "cyclone", "siklon"}},
	{"heyelan", []string{"heyelan", "landslide", "erdrutsch", "glissement de terrain", "deslizamiento"}},
}

func crisisKindOf(text string) string {
	low := " " + strings.ToLower(text) + " "
	for _, k := range crisisKinds {
		for _, t := range k.terms {
			if strings.Contains(low, t) {
				return k.kind
			}
		}
	}
	return ""
}

func parseMag(title string) float64 {
	if m := reMagUSGS.FindStringSubmatch(title); m != nil {
		v, _ := strconv.ParseFloat(strings.Replace(m[1], ",", ".", 1), 64)
		return v
	}
	if m := reMagText.FindStringSubmatch(title); m != nil {
		s := m[1]
		if s == "" {
			s = m[2]
		}
		v, _ := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
		if v > 0 && v < 10 {
			return v
		}
	}
	return 0
}

func titleMentionsCountry(title, code string) bool {
	p, ok := countryProfile(code)
	if !ok {
		return false
	}
	t := normalizeGeo(title)
	for _, a := range append(append([]string{}, p.Aliases...), p.Name) {
		if a = normalizeGeo(a); a != "" && geoHas(t, a) {
			return true
		}
	}
	return false
}

// crisisFor: kullanıcının ülkesinde süren büyük bir afet varsa kriz kartı.
// Çağıran s.mu kilidini tutuyor olmalı.
func (s *Store) crisisFor() *crisisInfo {
	if cfg == nil {
		return nil
	}
	cfg.mu.RLock()
	off := cfg.CrisisMode == "off"
	cfg.mu.RUnlock()
	if off {
		return nil
	}
	code := localNewsCountry()
	if code == "" {
		return nil
	}
	now := time.Now()
	var trig *crisisInfo
	consider := func(c *crisisInfo) {
		if trig == nil || (c.Level == "red" && trig.Level != "red") || (c.Level == trig.Level && c.Mag > trig.Mag) {
			trig = c
		}
	}

	// 1) Afet servisleri (GDACS turuncu/kırmızı, USGS önemli depremler).
	for name, list := range s.storiesByCat {
		if !strings.HasPrefix(name, "⚡") {
			continue
		}
		for _, st := range list {
			for _, it := range st.items {
				if it.HasDate && now.Sub(it.Published) > 12*time.Hour {
					continue
				}
				if !titleMentionsCountry(it.Title+" "+it.Summary, code) {
					continue
				}
				ts := it.Published.Unix()
				if m := reGDACS.FindStringSubmatch(it.Title); m != nil {
					kind := crisisKindOf(m[2])
					if kind == "" {
						kind = "afet"
					}
					consider(&crisisInfo{Kind: kind, Level: strings.ToLower(m[1]), Mag: parseMag(it.Title), Title: it.Title, Link: it.Link, TS: ts})
					continue
				}
				if mag := parseMag(it.Title); mag >= 5.5 {
					lvl := "orange"
					if mag >= 6.5 {
						lvl = "red"
					}
					consider(&crisisInfo{Kind: "deprem", Level: lvl, Mag: mag, Title: it.Title, Link: it.Link, TS: ts})
				}
			}
		}
	}

	// 2) Haber akışı: ülkeyle ilgili, son 6 saatte, çok kaynaklı afet haberi.
	for name, list := range s.storiesByCat {
		if strings.HasPrefix(name, "⚡") {
			continue
		}
		for _, st := range list {
			if !st.HasDate || now.Sub(st.Published) > 6*time.Hour || st.IndependentSources < 4 {
				continue
			}
			kind := ""
			for _, it := range st.items {
				if kind = crisisKindOf(it.Title); kind != "" {
					break
				}
			}
			if kind == "" || countryRelevance(st, code) < 0.8 {
				continue
			}
			mag := 0.0
			for _, it := range st.items {
				if m := parseMag(it.Title); m > mag {
					mag = m
				}
			}
			switch {
			case kind == "deprem" && mag >= 6.0:
				consider(&crisisInfo{Kind: kind, Level: "red", Mag: mag, Title: s.trTitle(st.Title), Link: st.Link, TS: st.Published.Unix()})
			case kind == "deprem" && mag >= 5.5:
				consider(&crisisInfo{Kind: kind, Level: "orange", Mag: mag, Title: s.trTitle(st.Title), Link: st.Link, TS: st.Published.Unix()})
			case kind != "deprem" && st.Breaking && st.IndependentSources >= 5 && st.Score >= 80:
				consider(&crisisInfo{Kind: kind, Level: "orange", Title: s.trTitle(st.Title), Link: st.Link, TS: st.Published.Unix()})
			}
		}
	}
	if trig == nil {
		return nil
	}
	trig.Country = code
	trig.Key = trig.Kind + ":" + time.Unix(trig.TS, 0).UTC().Format("2006-01-02")
	trig.Numbers = emergencyNumbers(code)

	// Doğrulanmış gelişmeler: aynı tür afet, son 24 saat, resmî kaynak ya da ≥3 bağımsız kaynak.
	seen := map[string]bool{}
	var picks []*Story
	for _, list := range s.storiesByCat {
		for _, st := range list {
			if seen[st.Link] || (st.HasDate && now.Sub(st.Published) > 24*time.Hour) {
				continue
			}
			if st.PrimarySources == 0 && st.IndependentSources < 3 {
				continue
			}
			match := false
			for _, it := range st.items {
				if crisisKindOf(it.Title) == trig.Kind {
					match = true
					break
				}
			}
			if !match || countryRelevance(st, code) < 0.6 {
				continue
			}
			seen[st.Link] = true
			picks = append(picks, st)
		}
	}
	sort.SliceStable(picks, func(i, j int) bool {
		if (picks[i].PrimarySources > 0) != (picks[j].PrimarySources > 0) {
			return picks[i].PrimarySources > 0
		}
		return picks[i].Published.After(picks[j].Published)
	})
	for _, st := range picks {
		if len(trig.Stories) >= 8 {
			break
		}
		trig.Stories = append(trig.Stories, s.storyOut(st))
	}
	return trig
}
