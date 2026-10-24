package core

// Kullanıcının kendi kanal listeleri (RC87).
//
// Kullanıcı açık kaynaklı bir M3U listesini adresiyle, dosyadan ya da
// yapıştırarak ekler. Liste ayrı bir dosyada (livelists.json) tutulur;
// güncelleme yalnızca kullanıcı "Güncelle" dediğinde yapılır. Böylece
// NexNews arka planda kimseye haber vermeden liste çekmez, kullanıcı da
// listesinin nereden geldiğini ve ne zaman yenilendiğini hep görür.
// NexNews'in hazır listesi de istenirse tamamen gizlenebilir.

import (
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxUserLists     = 30
	maxListChannels  = 3000
	listFetchTimeout = 25 * time.Second
)

type userListChannel struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Group   string            `json:"group,omitempty"`
	Country string            `json:"country,omitempty"`
	URLs    []json.RawMessage `json:"urls"`
}

type userList struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	URL      string            `json:"url,omitempty"` // boşsa dosyadan/yapıştırılarak eklendi
	Updated  int64             `json:"updated"`
	Channels []userListChannel `json:"channels"`
}

type userListsFile struct {
	BuiltinOff bool       `json:"builtinOff"`
	Lists      []userList `json:"lists"`
}

var (
	ulMu   sync.Mutex
	ulData *userListsFile
)

func userListsPath() string { return filepath.Join(appDataDir(), "livelists.json") }

func ulLoad() *userListsFile {
	if ulData != nil {
		return ulData
	}
	ulData = &userListsFile{}
	if b, err := os.ReadFile(userListsPath()); err == nil {
		_ = json.Unmarshal(b, ulData)
	}
	return ulData
}

func ulSave() error {
	b, err := json.Marshal(ulData)
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(userListsPath()), 0o755)
	return atomicWriteFile(userListsPath(), b)
}

func liveListsJSON() string {
	ulMu.Lock()
	defer ulMu.Unlock()
	b, _ := json.Marshal(ulLoad())
	return string(b)
}

var (
	reAttr = regexp.MustCompile(`([a-zA-Z-]+)="([^"]*)"`)
)

func shortHash(s string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return strconv.FormatUint(uint64(h.Sum32()), 36)
}

// parseM3UList: #EXTINF (tvg-id, group-title, tvg-country, http-referrer,
// http-user-agent) ve #EXTVLCOPT satırlarını okur; yalnızca http(s) adresleri alır.
// Aynı adla gelen yayınlar tek kanalda yedek adres olarak toplanır.
func parseM3UList(listID, text string) []userListChannel {
	type cur struct{ name, group, country, ref, ua string }
	var c *cur
	var out []userListChannel
	byName := map[string]int{}
	for _, raw := range strings.Split(text, "\n") {
		l := strings.TrimSpace(strings.TrimPrefix(raw, "\uFEFF"))
		switch {
		case strings.HasPrefix(l, "#EXTINF"):
			c = &cur{}
			if i := strings.LastIndex(l, ","); i >= 0 {
				c.name = strings.TrimSpace(l[i+1:])
			}
			for _, m := range reAttr.FindAllStringSubmatch(l, -1) {
				switch strings.ToLower(m[1]) {
				case "group-title":
					c.group = m[2]
				case "tvg-country":
					c.country = m[2]
				case "http-referrer", "referrer":
					c.ref = m[2]
				case "http-user-agent", "user-agent":
					c.ua = m[2]
				case "tvg-name":
					if c.name == "" {
						c.name = m[2]
					}
				}
			}
		case strings.HasPrefix(strings.ToLower(l), "#extvlcopt:http-referrer=") && c != nil:
			c.ref = l[strings.Index(l, "=")+1:]
		case strings.HasPrefix(strings.ToLower(l), "#extvlcopt:http-user-agent=") && c != nil:
			c.ua = l[strings.Index(l, "=")+1:]
		case strings.HasPrefix(strings.ToLower(l), "http://") || strings.HasPrefix(strings.ToLower(l), "https://"):
			name := ""
			if c != nil {
				name = c.name
			}
			if name == "" {
				name = "Kanal " + strconv.Itoa(len(out)+1)
			}
			if len([]rune(name)) > 80 {
				name = string([]rune(name)[:80])
			}
			var u json.RawMessage
			if c != nil && (c.ref != "" || c.ua != "") {
				m := map[string]string{"u": l}
				if c.ref != "" {
					m["r"] = c.ref
				}
				if c.ua != "" {
					m["ua"] = c.ua
				}
				u, _ = json.Marshal(m)
			} else {
				u, _ = json.Marshal(l)
			}
			key := strings.ToLower(name)
			if i, ok := byName[key]; ok {
				if len(out[i].URLs) < 4 {
					out[i].URLs = append(out[i].URLs, u)
				}
			} else if len(out) < maxListChannels {
				ch := userListChannel{ID: "ul" + listID + "-" + shortHash(key), Name: name, URLs: []json.RawMessage{u}}
				if c != nil {
					ch.Group, ch.Country = c.group, c.country
				}
				byName[key] = len(out)
				out = append(out, ch)
			}
			c = nil
		}
	}
	return out
}

func fetchListText(ctx context.Context, u string) (string, error) {
	low := strings.ToLower(strings.TrimSpace(u))
	if !strings.HasPrefix(low, "http://") && !strings.HasPrefix(low, "https://") {
		return "", errors.New("Liste adresi http(s) ile başlamalı.")
	}
	ctx, cancel := context.WithTimeout(ctx, listFetchTimeout)
	defer cancel()
	b, err := httpGet(ctx, &http.Client{Timeout: listFetchTimeout}, strings.TrimSpace(u))
	if err != nil {
		return "", errors.New("Liste indirilemedi. Adresi kontrol et.")
	}
	return string(b), nil
}

// liveListAdd: url doluysa indirir, değilse text'i (dosya/yapıştırma) kullanır.
func liveListAdd(ctx context.Context, name, url, text string) ([]byte, error) {
	if strings.TrimSpace(url) != "" {
		t, err := fetchListText(ctx, url)
		if err != nil {
			return nil, err
		}
		text = t
	}
	ulMu.Lock()
	defer ulMu.Unlock()
	d := ulLoad()
	if len(d.Lists) >= maxUserLists {
		return nil, errors.New("En çok " + strconv.Itoa(maxUserLists) + " liste eklenebilir.")
	}
	id := strconv.FormatInt(time.Now().UnixNano()%1e9, 36)
	chs := parseM3UList(id, text)
	if len(chs) == 0 {
		return nil, errors.New("Bu listede yayın adresi bulunamadı (M3U biçimi bekleniyor).")
	}
	name = cleanListName(name)
	if name == "" {
		name = "Listem " + strconv.Itoa(len(d.Lists)+1)
	}
	if len([]rune(name)) > 40 {
		name = string([]rune(name)[:40])
	}
	l := userList{ID: id, Name: name, URL: strings.TrimSpace(url), Updated: time.Now().Unix(), Channels: chs}
	d.Lists = append(d.Lists, l)
	if err := ulSave(); err != nil {
		return nil, err
	}
	return json.Marshal(l)
}

// liveListRefresh: yalnızca kullanıcı isteyince listeyi kaynağından yeniden indirir.
// Kanal kimlikleri ada göre üretildiği için favoriler ve gizlenenler korunur.
func liveListRefresh(ctx context.Context, id string) ([]byte, error) {
	ulMu.Lock()
	var src string
	for _, l := range ulLoad().Lists {
		if l.ID == id {
			src = l.URL
		}
	}
	ulMu.Unlock()
	if src == "" {
		return nil, errors.New("Bu liste dosyadan eklendi; güncellemek için silip yeni dosyayla yeniden ekle.")
	}
	text, err := fetchListText(ctx, src)
	if err != nil {
		return nil, err
	}
	chs := parseM3UList(id, text)
	if len(chs) == 0 {
		return nil, errors.New("Güncel listede yayın adresi bulunamadı; eski liste korundu.")
	}
	ulMu.Lock()
	defer ulMu.Unlock()
	d := ulLoad()
	for i := range d.Lists {
		if d.Lists[i].ID == id {
			d.Lists[i].Channels = chs
			d.Lists[i].Updated = time.Now().Unix()
			if err := ulSave(); err != nil {
				return nil, err
			}
			return json.Marshal(d.Lists[i])
		}
	}
	return nil, errors.New("Liste bulunamadı.")
}

func liveListDelete(id string) error {
	ulMu.Lock()
	defer ulMu.Unlock()
	d := ulLoad()
	kept := d.Lists[:0]
	for _, l := range d.Lists {
		if l.ID != id {
			kept = append(kept, l)
		}
	}
	d.Lists = kept
	return ulSave()
}

func liveListRename(id, name string) error {
	name = cleanListName(name)
	if name == "" {
		return nil
	}
	ulMu.Lock()
	defer ulMu.Unlock()
	d := ulLoad()
	for i := range d.Lists {
		if d.Lists[i].ID == id {
			d.Lists[i].Name = name
		}
	}
	return ulSave()
}

func liveSetBuiltin(on bool) error {
	ulMu.Lock()
	defer ulMu.Unlock()
	ulLoad().BuiltinOff = !on
	return ulSave()
}

// cleanListName: ad, sayfada onclick içinde kullanıldığı için tırnak/ters bölü içermez.
func cleanListName(s string) string {
	s = strings.NewReplacer("'", "’", "\"", "”", "\\", "/", "<", "", ">", "").Replace(strings.TrimSpace(s))
	if r := []rune(s); len(r) > 40 {
		s = string(r[:40])
	}
	return s
}
