package core

// Okuma aynası (RC97).
//
// İnsanlar genellikle kendi görüşlerini doğrulayan kaynaklara yönelir
// ("seçici maruz kalma"); algoritmalar da bunu pekiştirince yankı odaları
// oluşur. Okuma aynası, son 30 günde gerçekten neyi okuduğunu yalnızca bu
// cihazdaki okuma geçmişinden hesaplar ve kişiye kendi haber "diyetini"
// gösterir: hangi kaynaklar, hangi bölgeler, hangi sahiplik türü. Hiçbir veri
// cihazdan çıkmaz; amaç yargılamak değil, fark ettirmek.

import (
	"encoding/json"
	"math"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type mirrorCount struct {
	Name  string `json:"name"`
	Label string `json:"label,omitempty"`
	Count int    `json:"count"`
	Pct   int    `json:"pct"`
}

type mirrorNote struct {
	Kind string `json:"kind"` // tekKaynak / bolgeYok / devlet / kategori / cesitli / az
	Name string `json:"name,omitempty"`
	Pct  int    `json:"pct,omitempty"`
}

type mirrorBoard struct {
	Days          int           `json:"days"`
	Total         int           `json:"total"`
	Sources       []mirrorCount `json:"sources"`
	SourceCount   int           `json:"sourceCount"`
	Diversity     int           `json:"diversity"` // 0–100: kaynak çeşitliliği (normalleştirilmiş entropi)
	Regions       []mirrorCount `json:"regions"`
	SilentRegions []string      `json:"silentRegions"`
	Owners        []mirrorCount `json:"owners"`
	OwnerUnknown  int           `json:"ownerUnknown"`
	Cats          []mirrorCount `json:"cats"`
	Notes         []mirrorNote  `json:"notes"`
	From          int64         `json:"from"`
}

// ---- sahiplik araması (Go tarafı) ----

type ownershipCat struct {
	Types map[string]string   `json:"types"`
	D     map[string][]string `json:"d"`
}

var (
	ownCatMu   sync.Mutex
	ownCatData *ownershipCat
	ownCatSrc  int
)

func ownershipCatalog() *ownershipCat {
	src := ownershipSource()
	ownCatMu.Lock()
	defer ownCatMu.Unlock()
	if ownCatData != nil && ownCatSrc == len(src) {
		return ownCatData
	}
	c := &ownershipCat{}
	_ = json.Unmarshal(src, c)
	ownCatData, ownCatSrc = c, len(src)
	return c
}

// ownershipOf: bağlantının alan adına göre [tür, sahip, ülke]; bilinmiyorsa nil.
func ownershipOf(link string) []string {
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return nil
	}
	h := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	d := ownershipCatalog().D
	for h != "" {
		if o := d[h]; len(o) >= 2 {
			return o
		}
		i := strings.IndexByte(h, '.')
		if i < 0 {
			break
		}
		h = h[i+1:]
		if !strings.Contains(h, ".") {
			break
		}
	}
	return nil
}

// ---- ayna ----

func pctOf(n, total int) int {
	if total <= 0 {
		return 0
	}
	return int(math.Round(float64(n) * 100 / float64(total)))
}

func sortedCounts(m map[string]int, total int) []mirrorCount {
	out := make([]mirrorCount, 0, len(m))
	for k, v := range m {
		out = append(out, mirrorCount{Name: k, Count: v, Pct: pctOf(v, total)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func readingMirrorJSON() ([]byte, error) {
	const days = 30
	b := mirrorBoard{Days: days, Sources: []mirrorCount{}, Regions: []mirrorCount{}, SilentRegions: []string{}, Owners: []mirrorCount{}, Cats: []mirrorCount{}, Notes: []mirrorNote{}}
	if favs == nil {
		return json.Marshal(b)
	}
	cut := time.Now().Add(-days * 24 * time.Hour).Unix()
	favs.mu.Lock()
	hist := append([]savedItem(nil), favs.hist...)
	favs.mu.Unlock()

	src, reg, own, cat := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	from := time.Now().Unix()
	for _, h := range hist {
		if h.Added < cut {
			continue
		}
		b.Total++
		if h.Added < from {
			from = h.Added
		}
		if s := strings.TrimSpace(h.Source); s != "" {
			src[s]++
		}
		if h.Cat != "" {
			cat[h.Cat]++
			if isRegionalCat(h.Cat) {
				reg[h.Cat]++
			}
		}
		if o := ownershipOf(h.Link); o != nil {
			own[o[0]]++
		} else {
			b.OwnerUnknown++
		}
	}
	b.From = from
	b.Sources = sortedCounts(src, b.Total)
	b.SourceCount = len(src)
	if len(b.Sources) > 8 {
		b.Sources = b.Sources[:8]
	}
	b.Regions = sortedCounts(reg, b.Total)
	b.Cats = sortedCounts(cat, b.Total)
	known := b.Total - b.OwnerUnknown
	b.Owners = sortedCounts(own, known)
	types := ownershipCatalog().Types
	for i := range b.Owners {
		b.Owners[i].Label = types[b.Owners[i].Name]
	}

	// Kaynak çeşitliliği: Shannon entropisi / log(kaynak sayısı).
	if len(src) > 1 {
		h := 0.0
		for _, v := range src {
			p := float64(v) / float64(b.Total)
			h -= p * math.Log(p)
		}
		b.Diversity = int(math.Round(h / math.Log(float64(len(src))) * 100))
		// Az kaynakla "çok çeşitli" görünmesin: 10 kaynaktan azı orantılı kırpılır.
		if len(src) < 10 {
			b.Diversity = b.Diversity * len(src) / 10
		}
	}

	// Akışta bulunan ama hiç okunmamış bölgeler.
	if cfg != nil {
		cats, _ := cfg.snapshot()
		for _, c := range cats {
			if isRegionalCat(c.Name) && reg[c.Name] == 0 && len(c.Sources) > 0 {
				b.SilentRegions = append(b.SilentRegions, c.Name)
			}
		}
	}

	// Notlar (yargı değil, fark ettirme).
	if b.Total < 10 {
		b.Notes = append(b.Notes, mirrorNote{Kind: "az"})
		return json.Marshal(b)
	}
	if len(b.Sources) > 0 && b.Sources[0].Pct >= 40 {
		b.Notes = append(b.Notes, mirrorNote{Kind: "tekKaynak", Name: b.Sources[0].Name, Pct: b.Sources[0].Pct})
	}
	if known >= 10 && pctOf(own["devlet"], known) >= 40 {
		b.Notes = append(b.Notes, mirrorNote{Kind: "devlet", Pct: pctOf(own["devlet"], known)})
	}
	if len(b.Cats) > 0 && b.Cats[0].Pct >= 60 {
		b.Notes = append(b.Notes, mirrorNote{Kind: "kategori", Name: b.Cats[0].Name, Pct: b.Cats[0].Pct})
	}
	if len(b.SilentRegions) > 0 {
		b.Notes = append(b.Notes, mirrorNote{Kind: "bolgeYok", Name: strings.Join(b.SilentRegions, ", ")})
	}
	if b.Diversity >= 70 && len(b.Notes) == 0 {
		b.Notes = append(b.Notes, mirrorNote{Kind: "cesitli", Pct: b.Diversity})
	}
	return json.Marshal(b)
}
