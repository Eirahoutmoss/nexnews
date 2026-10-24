package core

// Resmî tatiller ve bayramlar (RC98).
//
// İnsanların gündemi yalnızca faiz kararları değildir: bayramlar, anma
// günleri ve ulusal günler toplumun ortak takvimidir. Ülke bazındaki resmî
// tatiller python-holidays verisinden (tools/gen_holidays.py) üretilip
// assets/holidays.json olarak gömülür: 240'ı aşkın ülke ve bölge, Moritanya'nın
// bağımsızlık gününden Antarktika'nın kış ortası gününe kadar. Ağ gerekmez;
// yeni yılların verisi imzalı uzaktan güncellemeyle gelir.

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed assets/holidays.json
var holidaysRaw []byte

type holidayCountry struct {
	N []map[string]string  `json:"n"` // ad tablosu: "_" varsayılan, dil kodu → ad
	D [][2]json.RawMessage `json:"d"` // [tarih, ad sırası]
}

type holidayFile struct {
	Years []int                     `json:"years"`
	C     map[string]holidayCountry `json:"c"`
}

var (
	holMu   sync.Mutex
	holData *holidayFile
	holSrc  int
)

func holidaySource() []byte {
	if b := updatedAsset("holidays.json"); b != nil {
		return b
	}
	return holidaysRaw
}

func holidaysData() *holidayFile {
	src := holidaySource()
	holMu.Lock()
	defer holMu.Unlock()
	if holData != nil && holSrc == len(src) {
		return holData
	}
	d := &holidayFile{}
	if err := json.Unmarshal(src, d); err != nil {
		_ = json.Unmarshal(holidaysRaw, d)
	}
	holData, holSrc = d, len(src)
	return d
}

// HolidaysAsset: imzalama aracı için.
func HolidaysAsset() []byte { return holidaysRaw }

type holidayDay struct {
	Date string
	Name string
}

// countryHolidays: ülkenin resmî tatilleri, arayüz dilinde (yoksa İngilizce,
// o da yoksa ülkenin kendi dilinde).
func countryHolidays(code string) []holidayDay {
	c, ok := holidaysData().C[strings.ToUpper(code)]
	if !ok {
		return nil
	}
	lang := uiLangCode()
	out := make([]holidayDay, 0, len(c.D))
	for _, d := range c.D {
		var date string
		var i int
		if json.Unmarshal(d[0], &date) != nil || json.Unmarshal(d[1], &i) != nil || i < 0 || i >= len(c.N) {
			continue
		}
		names := c.N[i]
		n := names[lang]
		if n == "" {
			n = names["en"]
		}
		if n == "" {
			n = names["_"]
		}
		out = append(out, holidayDay{Date: date, Name: n})
	}
	return out
}

// flagEmoji: ISO ülke kodundan bayrak emojisi (AQ gibi bölgeler dahil).
func flagEmoji(code string) string {
	code = strings.ToUpper(code)
	if len(code) != 2 {
		return ""
	}
	r := []rune{rune(code[0]) - 'A' + 0x1F1E6, rune(code[1]) - 'A' + 0x1F1E6}
	return string(r)
}
