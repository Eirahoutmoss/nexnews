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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed assets/holidays.json
var holidaysRaw []byte

type holidayCountry struct {
	N []map[string]string `json:"n"` // ad tablosu: "_" varsayılan, dil kodu → ad
	D [][]json.RawMessage `json:"d"` // [tarih, ad sırası, (1 = ulusal gün)]
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
	Date     string
	Name     string
	National bool
}

func pickName(names map[string]string, lang string) string {
	if n := names[lang]; n != "" {
		return n
	}
	en := names["en"]
	if en == "" {
		en = names["_"]
	}
	// Ülkenin verisinde arayüz dili yoksa sık geçen ulusal gün adlarını çevir.
	if m, ok := commonDayNames[en]; ok {
		if v := m[lang]; v != "" {
			return v
		}
	}
	return en
}

// commonDayNames: en sık görülen ulusal gün adlarının çevirileri.
var commonDayNames = map[string]map[string]string{
	"Independence Day":                {"tr": "Bağımsızlık Günü", "de": "Unabhängigkeitstag", "fr": "Fête de l'indépendance", "es": "Día de la Independencia", "ru": "День независимости", "ar": "عيد الاستقلال"},
	"National Day":                    {"tr": "Ulusal Gün", "de": "Nationalfeiertag", "fr": "Fête nationale", "es": "Día Nacional", "ru": "Национальный день", "ar": "اليوم الوطني"},
	"Constitution Day":                {"tr": "Anayasa Günü", "de": "Tag der Verfassung", "fr": "Jour de la Constitution", "es": "Día de la Constitución", "ru": "День Конституции", "ar": "يوم الدستور"},
	"Victory Day":                     {"tr": "Zafer Günü", "de": "Tag des Sieges", "fr": "Jour de la Victoire", "es": "Día de la Victoria", "ru": "День Победы", "ar": "يوم النصر"},
	"Republic Day":                    {"tr": "Cumhuriyet Günü", "de": "Tag der Republik", "fr": "Fête de la République", "es": "Día de la República", "ru": "День Республики", "ar": "يوم الجمهورية"},
	"Liberation Day":                  {"tr": "Kurtuluş Günü", "de": "Tag der Befreiung", "fr": "Jour de la Libération", "es": "Día de la Liberación", "ru": "День освобождения", "ar": "يوم التحرير"},
	"Revolution Day":                  {"tr": "Devrim Günü", "de": "Tag der Revolution", "fr": "Jour de la Révolution", "es": "Día de la Revolución", "ru": "День революции", "ar": "يوم الثورة"},
	"National Independence Day":       {"tr": "Ulusal Bağımsızlık Günü", "de": "Nationaler Unabhängigkeitstag", "fr": "Fête de l'indépendance nationale", "es": "Día de la Independencia Nacional", "ru": "День национальной независимости", "ar": "يوم الاستقلال الوطني"},
	"Statehood Day":                   {"tr": "Devlet Günü", "de": "Tag der Staatlichkeit", "fr": "Jour de l'État", "es": "Día de la Estatalidad", "ru": "День государственности", "ar": "يوم الدولة"},
	"Freedom Day":                     {"tr": "Özgürlük Günü", "de": "Tag der Freiheit", "fr": "Jour de la Liberté", "es": "Día de la Libertad", "ru": "День свободы", "ar": "يوم الحرية"},
	"Restoration of Independence Day": {"tr": "Bağımsızlığın Yeniden Kazanılması Günü", "de": "Tag der Wiederherstellung der Unabhängigkeit", "fr": "Jour de la restauration de l'indépendance", "es": "Día de la Restauración de la Independencia", "ru": "День восстановления независимости", "ar": "يوم استعادة الاستقلال"},
}

// worldDay: "Dünyada bugün" — bir ülkenin ulusal günü.
type worldDay struct {
	Country string `json:"country"` // ISO kodu; ad arayüzde yerelleştirilir
	Flag    string `json:"flag"`
	Title   string `json:"title"`
}

// worldToday: bugün ulusal günü (bağımsızlık, cumhuriyet, kuruluş…) olan ülkeler.
var (
	worldMu  sync.Mutex
	worldKey string
	worldVal []worldDay
)

func worldToday() []worldDay {
	today := time.Now().Format("2006-01-02")
	lang := uiLangCode()
	key := today + "|" + lang + "|" + strconv.Itoa(len(holidaySource()))
	worldMu.Lock()
	defer worldMu.Unlock()
	if key == worldKey {
		return worldVal
	}
	var out []worldDay
	for code, c := range holidaysData().C {
		for _, d := range c.D {
			if len(d) < 3 || string(d[2]) != "1" {
				continue
			}
			var date string
			var i int
			if json.Unmarshal(d[0], &date) != nil || date != today || json.Unmarshal(d[1], &i) != nil || i < 0 || i >= len(c.N) {
				continue
			}
			out = append(out, worldDay{Country: code, Flag: flagEmoji(code), Title: pickName(c.N[i], lang)})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Country < out[b].Country })
	worldKey, worldVal = key, out
	return out
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
		if len(d) < 2 || json.Unmarshal(d[0], &date) != nil || json.Unmarshal(d[1], &i) != nil || i < 0 || i >= len(c.N) {
			continue
		}
		nat := len(d) > 2 && string(d[2]) == "1"
		out = append(out, holidayDay{Date: date, Name: pickName(c.N[i], lang), National: nat})
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
