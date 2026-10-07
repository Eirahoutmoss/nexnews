package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceFromPersistsGlobalNewsCountry(t *testing.T) {
	dir := t.TempDir()
	c := defaultConfig()
	c.path = filepath.Join(dir, "config.json")

	raw, err := json.Marshal(map[string]any{
		"categories":     c.Categories,
		"maxPerCategory": 5,
		"newsCountry":    "",
		"countryScope":   "related",
		"schemaV":        currentSchema,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.replaceFrom(raw); err != nil {
		t.Fatal(err)
	}
	if c.NewsCountry != "" {
		t.Fatalf("global selection must persist as empty country, got %q", c.NewsCountry)
	}

	saved, err := os.ReadFile(c.path)
	if err != nil {
		t.Fatal(err)
	}
	var got Config
	if err := json.Unmarshal(saved, &got); err != nil {
		t.Fatal(err)
	}
	if got.NewsCountry != "" {
		t.Fatalf("saved global selection must remain empty, got %q", got.NewsCountry)
	}
}

func TestReplaceFromPersistsSelectedNewsCountry(t *testing.T) {
	dir := t.TempDir()
	c := defaultConfig()
	c.path = filepath.Join(dir, "config.json")

	raw := []byte(`{"categories":[],"maxPerCategory":5,"newsCountry":"td","countryScope":"strict","schemaV":10}`)
	if err := c.replaceFrom(raw); err != nil {
		t.Fatal(err)
	}
	if c.NewsCountry != "TD" {
		t.Fatalf("selected country not normalized/persisted: %q", c.NewsCountry)
	}
	if c.CountryScope != "strict" {
		t.Fatalf("country scope not persisted: %q", c.CountryScope)
	}
}
