package core

// RC97 ek verileri: doğrulama siteleri, gündem takvimi ve acil numaralar.
// Tek dosya (assets/extras.json); imzalı uzaktan güncellemeyle tazelenebilir,
// böylece yeni bir doğrulama sitesi ya da yeni yılın takvimi için uygulamayı
// yeniden yayımlamak gerekmez.

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed assets/extras.json
var extrasRaw []byte

type factSource struct {
	Name    string `json:"name"`
	Site    string `json:"site"`
	Feed    string `json:"feed"`
	Lang    string `json:"lang"`
	Country string `json:"country"`
}

type agendaFixed struct {
	Date    string `json:"date"` // YYYY-MM-DD (yerel tarih, tz'ye göre)
	Time    string `json:"time"` // HH:MM ya da boş
	TZ      string `json:"tz"`
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	Country string `json:"country"` // boş = herkes için
	Link    string `json:"link"`
	Note    string `json:"note,omitempty"`
}

type agendaRule struct {
	Rule    string `json:"rule"` // "monthly:<gün>" ya da "firstweekday:<0-6, 0=pazar>"
	Time    string `json:"time"`
	TZ      string `json:"tz"`
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	Country string `json:"country"`
	Link    string `json:"link"`
	Note    string `json:"note,omitempty"`
}

type emergencyNumber struct {
	Label  string `json:"label"`
	Number string `json:"number"`
}

type extrasFile struct {
	Version     int                          `json:"version"`
	FactCheck   []factSource                 `json:"factcheck"`
	Agenda      []agendaFixed                `json:"agenda"`
	AgendaRules []agendaRule                 `json:"agendaRules"`
	Emergency   map[string][]emergencyNumber `json:"emergency"`
	Observances []agendaRule                 `json:"observances"` // RC98: anma / dünya günleri
}

var (
	extrasMu   sync.Mutex
	extrasData *extrasFile
	extrasFrom []byte
)

// extras: geçerli ek veriler (uzaktan gelmişse o, yoksa gömülü olan).
func extras() *extrasFile {
	src := updatedAsset("extras.json")
	if src == nil {
		src = extrasRaw
	}
	extrasMu.Lock()
	defer extrasMu.Unlock()
	if extrasData != nil && string(extrasFrom) == string(src) {
		return extrasData
	}
	d := &extrasFile{}
	if err := json.Unmarshal(src, d); err != nil {
		_ = json.Unmarshal(extrasRaw, d)
	}
	extrasData, extrasFrom = d, src
	return d
}

// ExtrasAsset: imzalama aracı için.
func ExtrasAsset() []byte { return extrasRaw }

func emergencyNumbers(country string) []emergencyNumber {
	e := extras().Emergency
	if n := e[country]; len(n) > 0 {
		return n
	}
	return e["_"]
}
