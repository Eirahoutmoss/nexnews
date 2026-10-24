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
