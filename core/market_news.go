package core

// Borsa Analiz sekmesi için piyasayı ilgilendiren gündem başlıkları.
// Ayrı bir kaynak çekilmez: zaten indirilen haberler anahtar kelimeyle süzülür.

import (
	"sort"
	"strings"
	"time"
	"unicode"
)

type marketNewsItem struct {
	Title     string `json:"title"`
	Link      string `json:"link"`
	Source    string `json:"source"`
	Published int64  `json:"published"`
	Tag       string `json:"tag"`
}

var marketNewsKeys = []struct{ key, tag string }{
	{"borsa", "Borsa"}, {"bist", "Borsa"}, {"hisse", "Borsa"}, {"halka arz", "Borsa"},
	{"merkez bankası", "Faiz"}, {"tcmb", "Faiz"}, {"faiz", "Faiz"}, {"politika faizi", "Faiz"},
	{"enflasyon", "Enflasyon"}, {"tüfe", "Enflasyon"},
	{"dolar", "Kur"}, {"avro", "Kur"}, {"euro/tl", "Kur"}, {"döviz", "Kur"}, {"kur", "Kur"},
	{"altın", "Altın"}, {"ons", "Altın"}, {"brent", "Emtia"}, {"petrol", "Emtia"},
	{"stock market", "Borsa"}, {"stocks", "Borsa"}, {"wall street", "Borsa"}, {"nasdaq", "Borsa"},
	{"interest rate", "Faiz"}, {"federal reserve", "Faiz"}, {"the fed", "Faiz"}, {"ecb", "Faiz"}, {"central bank", "Faiz"},
	{"inflation", "Enflasyon"}, {"oil price", "Emtia"}, {"gold price", "Altın"},
}

func lowerTR(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case 'I':
			return 'ı'
		case 'İ':
			return 'i'
		}
		return unicode.ToLower(r)
	}, s)
}

func marketTag(title string) string {
	t := " " + lowerTR(title) + " "
	for _, k := range marketNewsKeys {
		if strings.Contains(t, k.key) {
			// "ons" gibi kısa anahtarlar kelime olarak geçmeli.
			if len([]rune(k.key)) <= 3 && !strings.Contains(t, " "+k.key+" ") && !strings.Contains(t, " "+k.key+"'") {
				continue
			}
			return k.tag
		}
	}
	return ""
}

func (s *Store) marketNews(max int) []marketNewsItem {
	if s == nil {
		return nil
	}
	cutoff := time.Now().Add(-36 * time.Hour)
	seen := map[string]bool{}
	var out []marketNewsItem
	s.mu.RLock()
	for _, list := range s.byCat {
		for _, it := range list {
			if it.HasDate && it.Published.Before(cutoff) {
				continue
			}
			tag := marketTag(it.Title)
			if tag == "" {
				continue
			}
			key := lowerTR(strings.Join(strings.Fields(it.Title), " "))
			if seen[key] || seen[it.Link] {
				continue
			}
			seen[key], seen[it.Link] = true, true
			out = append(out, marketNewsItem{Title: it.Title, Link: it.Link, Source: it.Source, Published: it.Published.Unix(), Tag: tag})
		}
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Published > out[j].Published })
	if len(out) > max {
		out = out[:max]
	}
	return out
}
