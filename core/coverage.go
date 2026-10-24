package core

// Kör nokta göstergesi (RC93).
//
// Bir olayı hangi bölgelerin kaynakları haber yapmış, hangileri (o sırada
// başka haberler yayımladığı hâlde) hiç değinmemiş? İnsanlar genellikle
// neyi okumadıklarını fark etmez; gündem belirleme yalnızca neyin anlatıldığı
// değil neyin anlatılmadığıyla da işler. Eşleşme, olay gruplamasındaki aynı
// kelime örtüşmesiyle ve (varsa) arayüz diline çevrilmiş başlıklarla yapılır;
// böylece farklı dillerdeki haberler de aynı olay olarak tanınabilir.

import (
	"strings"
	"time"
)

type regionCov struct {
	Region  string `json:"region"`
	Sources int    `json:"sources"`
	Title   string `json:"title"`
	Link    string `json:"link"`
}

type coverage struct {
	Covered []regionCov `json:"covered"`
	Silent  []string    `json:"silent"`
}

var regionalCats = map[string]bool{"Türkiye": true, "Avrupa": true, "Asya": true, "Ortadoğu": true, "Orta Doğu": true,
	"Afrika": true, "Amerika": true, "Kuzey Amerika": true, "Güney Amerika": true, "Okyanusya": true, "Rusya": true,
	"Küresel / Uluslararası": true}

func isRegionalCat(name string) bool {
	return regionalCats[name] || strings.HasPrefix(name, localCatPrefix)
}

// storyTokens: özgün ve (önbellekte varsa) çevrilmiş başlığın kelimeleri.
func (s *Store) storyTokens(st *Story) map[string]bool {
	tok := map[string]bool{}
	for k := range clusterTokens(st.Title) {
		tok[k] = true
	}
	if t := s.trTitle(st.Title); t != st.Title {
		for k := range clusterTokens(t) {
			tok[k] = true
		}
	}
	return tok
}

// coverageFor: çağıran s.mu kilidini tutuyor olmalı.
func (s *Store) coverageFor(st *Story) *coverage {
	if st == nil {
		return nil
	}
	base := s.storyTokens(st)
	if len(base) < minShared {
		return nil
	}
	cv := &coverage{}
	ref := st.Published
	if !st.HasDate {
		ref = time.Now()
	}
	for cat, list := range s.storiesByCat {
		if !isRegionalCat(cat) {
			continue
		}
		var best *Story
		bestSim, active := simThreshold, false
		for _, o := range list {
			if o.HasDate {
				d := ref.Sub(o.Published)
				if d < 0 {
					d = -d
				}
				if d > 48*time.Hour {
					continue
				}
			}
			active = true
			if o == st {
				best, bestSim = o, 2
				break
			}
			sim, inter := overlapCoef(base, s.storyTokens(o))
			if inter >= minShared && sim >= bestSim {
				best, bestSim = o, sim
			}
		}
		switch {
		case best != nil:
			cv.Covered = append(cv.Covered, regionCov{Region: cat, Sources: len(best.Sources), Title: s.trTitle(best.Title), Link: best.Link})
		case active:
			cv.Silent = append(cv.Silent, cat)
		}
	}
	if len(cv.Covered)+len(cv.Silent) < 2 {
		return nil // tek bölgeli kurulumda anlamlı değil
	}
	sortRegionCov(cv)
	return cv
}

func sortRegionCov(cv *coverage) {
	// En çok kaynakla haber yapan önce; sessizler alfabetik.
	for i := 1; i < len(cv.Covered); i++ {
		for j := i; j > 0 && cv.Covered[j].Sources > cv.Covered[j-1].Sources; j-- {
			cv.Covered[j], cv.Covered[j-1] = cv.Covered[j-1], cv.Covered[j]
		}
	}
	for i := 1; i < len(cv.Silent); i++ {
		for j := i; j > 0 && cv.Silent[j] < cv.Silent[j-1]; j-- {
			cv.Silent[j], cv.Silent[j-1] = cv.Silent[j-1], cv.Silent[j]
		}
	}
}
