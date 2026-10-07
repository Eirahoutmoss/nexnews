package core

// Import/export of sources and settings in portable formats so the user can
// back up, move, or reuse their feed list in other apps:
//   - CSV  : Kategori,Ad,URL  (one row per source)
//   - TXT  : one feed URL per line (imported into an "İçe Aktarılan" category)
//   - JSON : full config backup / restore

import (
	"bytes"
	"encoding/csv"
	"strings"
)

func sourcesToCSV(cats []Category) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Kategori", "Ad", "URL"})
	for _, c := range cats {
		for _, s := range c.Sources {
			_ = w.Write([]string{c.Name, s.Name, s.URL})
		}
	}
	w.Flush()
	return buf.Bytes()
}

// csvToCategories parses a CSV (with or without a header row) of Kategori,Ad,URL.
func csvToCategories(data []byte) ([]Category, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // tolerate ragged rows
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	order := []string{}
	byName := map[string]*Category{}
	add := func(cat, name, url string) {
		url = strings.TrimSpace(url)
		if url == "" {
			return
		}
		cat = strings.TrimSpace(cat)
		if cat == "" {
			cat = "İçe Aktarılan"
		}
		c, ok := byName[cat]
		if !ok {
			order = append(order, cat)
			byName[cat] = &Category{Name: cat}
			c = byName[cat]
		}
		c.Sources = append(c.Sources, Source{Name: strings.TrimSpace(name), URL: url})
	}
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		// skip a header row
		if i == 0 && len(row) >= 3 &&
			strings.EqualFold(strings.TrimSpace(row[2]), "URL") {
			continue
		}
		switch {
		case len(row) >= 3:
			add(row[0], row[1], row[2])
		case len(row) == 2:
			add(row[0], "", row[1])
		case len(row) == 1:
			add("", "", row[0])
		}
	}
	out := make([]Category, 0, len(order))
	for _, n := range order {
		out = append(out, *byName[n])
	}
	return out, nil
}

// txtToSources reads one URL per line.
func txtToSources(data []byte) []Source {
	var out []Source
	for _, line := range strings.Split(string(data), "\n") {
		u := strings.TrimSpace(strings.TrimRight(line, "\r"))
		if u == "" || strings.HasPrefix(u, "#") {
			continue
		}
		out = append(out, Source{URL: u})
	}
	return out
}

// mergeCategories adds imported categories into the config, merging into
// existing categories by name and skipping duplicate URLs. Returns how many
// sources were added.
func mergeCategories(c *Config, imported []Category) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	index := map[string]int{}
	existingURL := map[string]bool{}
	for i, cat := range c.Categories {
		index[cat.Name] = i
		for _, s := range cat.Sources {
			existingURL[s.URL] = true
		}
	}
	added := 0
	for _, ic := range imported {
		for _, s := range ic.Sources {
			if existingURL[s.URL] {
				continue
			}
			existingURL[s.URL] = true
			if idx, ok := index[ic.Name]; ok {
				c.Categories[idx].Sources = append(c.Categories[idx].Sources, s)
			} else {
				index[ic.Name] = len(c.Categories)
				c.Categories = append(c.Categories, Category{Name: ic.Name, Sources: []Source{s}})
			}
			added++
		}
	}
	return added
}
