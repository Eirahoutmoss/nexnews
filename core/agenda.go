package core

// Gündem takvimi (RC97): "Yarın ne konuşulacak?"
//
// Önceden bilinen ve gündemi belirleyen olaylar (merkez bankası faiz
// kararları, enflasyon verileri, istihdam raporu…) tek listede. Tarihler
// kurumların yayımladığı resmî takvimlerden derlenip assets/extras.json'a
// yazılır; yeni yılın takvimi imzalı uzaktan güncellemeyle gelir.

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // Windows'ta saat dilimi veritabanı olmayabilir
)

type agendaOut struct {
	TS      int64  `json:"ts"`
	AllDay  bool   `json:"allDay,omitempty"`
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	Country string `json:"country,omitempty"`
	Link    string `json:"link,omitempty"`
	Note    string `json:"note,omitempty"`
	Flag    string `json:"flag,omitempty"`  // RC98: ülke bayrağı (resmî tatiller)
	Local   bool   `json:"local,omitempty"` // RC98: ad zaten arayüz dilinde (çevrilmez)
}

func agendaTime(date, hm, tz string) (time.Time, bool, bool) {
	loc, err := time.LoadLocation(tz)
	if err != nil || tz == "" {
		loc = time.UTC
	}
	d, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return time.Time{}, false, false
	}
	if hm == "" {
		return d.Add(12 * time.Hour), true, true // saat bilinmiyor: gün ortası
	}
	p := strings.SplitN(hm, ":", 2)
	h, _ := strconv.Atoi(p[0])
	m := 0
	if len(p) == 2 {
		m, _ = strconv.Atoi(p[1])
	}
	return time.Date(d.Year(), d.Month(), d.Day(), h, m, 0, 0, loc), false, true
}

func agendaJSON(days int) ([]byte, error) {
	if days <= 0 {
		days = 14
	}
	now := time.Now()
	from := now.Add(-3 * time.Hour) // az önce açıklanan da görünsün
	to := now.Add(time.Duration(days) * 24 * time.Hour)
	country := ""
	if cfg != nil {
		country = localNewsCountry()
	}
	ex := extras()
	var out []agendaOut
	add := func(t time.Time, allDay bool, title, kind, c, link, note string) {
		if t.Before(from) || t.After(to) {
			return
		}
		if c != "" && c != country {
			return
		}
		out = append(out, agendaOut{TS: t.Unix(), AllDay: allDay, Title: title, Kind: kind, Country: c, Link: link, Note: note})
	}
	// RC98: kullanıcının ülkesinin resmî tatilleri ve bayramları (gün boyu, yerel saat).
	if country != "" {
		for _, h := range countryHolidays(country) {
			d, err := time.ParseInLocation("2006-01-02", h.Date, time.Local)
			if err != nil {
				continue
			}
			t := d.Add(12 * time.Hour)
			if d.Add(24*time.Hour).Before(now) || t.After(to) {
				continue
			}
			out = append(out, agendaOut{TS: t.Unix(), AllDay: true, Title: h.Name, Kind: "bayram", Country: country, Flag: flagEmoji(country), Local: true})
		}
	}
	// RC98: anma günleri (ülkeye özel) ve BM dünya günleri.
	for _, r := range ex.Observances {
		for _, date := range ruleDates(r.Rule, now) {
			d, err := time.ParseInLocation("2006-01-02", date, time.Local)
			if err != nil || d.Add(24*time.Hour).Before(now) {
				continue
			}
			t := d.Add(12 * time.Hour)
			if t.After(to) || (r.Country != "" && r.Country != country) {
				continue
			}
			out = append(out, agendaOut{TS: t.Unix(), AllDay: true, Title: r.Title, Kind: r.Kind, Country: r.Country, Link: r.Link, Note: r.Note})
		}
	}
	for _, a := range ex.Agenda {
		if t, allDay, ok := agendaTime(a.Date, a.Time, a.TZ); ok {
			add(t, allDay, a.Title, a.Kind, a.Country, a.Link, a.Note)
		}
	}
	for _, r := range ex.AgendaRules {
		kind, arg, _ := strings.Cut(r.Rule, ":")
		n, err := strconv.Atoi(arg)
		if err != nil {
			continue
		}
		for mo := 0; mo < 3; mo++ {
			base := time.Date(now.Year(), now.Month()+time.Month(mo), 1, 0, 0, 0, 0, time.UTC)
			var day int
			switch kind {
			case "monthly":
				day = n
			case "firstweekday": // ayın ilk <n> günü (0 = pazar)
				day = 1 + (n-int(base.Weekday())+7)%7
			default:
				continue
			}
			date := time.Date(base.Year(), base.Month(), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			if t, allDay, ok := agendaTime(date, r.Time, r.TZ); ok {
				add(t, allDay, r.Title, r.Kind, r.Country, r.Link, r.Note)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TS < out[j].TS })
	if out == nil {
		out = []agendaOut{}
	}
	return json.Marshal(out)
}

// ruleDates: "yearly:MM-DD" ya da "nth:MM:<haftanın günü 0=pazar>:<kaçıncı>"
// kuralının bu yıl ve gelecek yıl için tarihleri.
func ruleDates(rule string, now time.Time) []string {
	kind, arg, _ := strings.Cut(rule, ":")
	var out []string
	for _, y := range []int{now.Year(), now.Year() + 1} {
		switch kind {
		case "yearly":
			out = append(out, strconv.Itoa(y)+"-"+arg)
		case "nth":
			p := strings.Split(arg, ":")
			if len(p) != 3 {
				continue
			}
			mo, e1 := strconv.Atoi(p[0])
			wd, e2 := strconv.Atoi(p[1])
			n, e3 := strconv.Atoi(p[2])
			if e1 != nil || e2 != nil || e3 != nil || n < 1 {
				continue
			}
			first := time.Date(y, time.Month(mo), 1, 0, 0, 0, 0, time.UTC)
			day := 1 + (wd-int(first.Weekday())+7)%7 + 7*(n-1)
			out = append(out, time.Date(y, time.Month(mo), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
		}
	}
	return out
}

// todayDays: bugünün bayram / anma / dünya günleri (akışın üstündeki şerit için).
func todayDays() []agendaOut {
	b, err := agendaJSON(1)
	if err != nil {
		return nil
	}
	var all []agendaOut
	if json.Unmarshal(b, &all) != nil {
		return nil
	}
	today := time.Now().Format("2006-01-02")
	var out []agendaOut
	for _, a := range all {
		if a.AllDay && (a.Kind == "bayram" || a.Kind == "anma" || a.Kind == "dunya") && time.Unix(a.TS, 0).Format("2006-01-02") == today {
			out = append(out, a)
		}
	}
	return out
}
