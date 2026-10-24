package core

// Sesli brifing (RC97).
//
// Günün özeti, arabada ya da yürürken dinlenebilecek 3–4 dakikalık bir
// bülten metnine dönüştürülür. Metni masaüstünde Windows'un, Android'de
// telefonun kendi konuşma motoru okur; NexNews ses dosyası indirmez.

import (
	"encoding/json"
	"strconv"
	"time"
)

type audioBriefItem struct {
	Title  string `json:"title"`
	Link   string `json:"link"`
	Source string `json:"source"`
}

type audioBrief struct {
	Text  string           `json:"text"`
	Items []audioBriefItem `json:"items"`
}

func (s *Store) audioBriefJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	greet := uiT("İyi akşamlar.")
	switch h := now.Hour(); {
	case h >= 5 && h < 12:
		greet = uiT("Günaydın.")
	case h >= 12 && h < 18:
		greet = uiT("İyi günler.")
	}
	b := audioBrief{Items: []audioBriefItem{}}
	seen := map[string]bool{}
	pick := func(st *Story) {
		if len(b.Items) >= 6 || seen[st.Link] || !s.countryAllowed(st) || s.isHidden(st) {
			return
		}
		seen[st.Link] = true
		o := s.storyOut(st) // sakin başlık ayarı burada da geçerli
		b.Items = append(b.Items, audioBriefItem{Title: o.Title, Link: st.Link, Source: o.Source})
	}
	if c := s.crisisFor(); c != nil {
		b.Text += uiT("Önemli uyarı:") + " " + c.Title + ". " + uiT("Yalnızca resmî kaynaklardan gelen bilgilere güven.") + " "
	}
	for _, st := range s.important {
		if st.Breaking {
			pick(st)
		}
	}
	for _, st := range s.important {
		if st.Score >= 40 {
			pick(st)
		}
	}
	text := greet + " " + uiT("NexNews sesli brifingi.") + " "
	if len(b.Items) == 0 {
		text += uiT("Şu anda öne çıkan önemli bir gelişme yok.")
		b.Text = text + b.Text
		return json.Marshal(b)
	}
	text += b.Text + uiT("Öne çıkan gelişmeler") + ": " + strconv.Itoa(len(b.Items)) + ". "
	b.Text = text
	for i, it := range b.Items {
		b.Text += strconv.Itoa(i+1) + ". " + it.Title + ". "
		if it.Source != "" {
			b.Text += uiT("Kaynak") + ": " + it.Source + ". "
		}
	}
	if follows != nil {
		if n := follows.count(); n > 0 {
			b.Text += uiT("Takip ettiğin olayların ayrıntıları uygulamada.") + " "
		}
	}
	b.Text += uiT("Bugünlük bu kadar. İyi bir gün geçir.")
	return json.Marshal(b)
}
