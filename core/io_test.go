package core

import "testing"

func TestCSVRoundTrip(t *testing.T) {
	cats := []Category{
		{Name: "Teknoloji", Sources: []Source{{Name: "Verge", URL: "https://a/x"}}},
		{Name: "Dünya", Sources: []Source{{Name: "", URL: "https://b/y"}}},
	}
	csv := sourcesToCSV(cats)
	got, err := csvToCategories(csv)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Teknoloji" || got[0].Sources[0].URL != "https://a/x" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestTxtImport(t *testing.T) {
	srcs := txtToSources([]byte("https://a/1\n# comment\n\nhttps://b/2\r\n"))
	if len(srcs) != 2 || srcs[1].URL != "https://b/2" {
		t.Fatalf("txt parse: %+v", srcs)
	}
}

func TestMergeDedup(t *testing.T) {
	c := &Config{Categories: []Category{{Name: "Teknoloji", Sources: []Source{{URL: "https://a/x"}}}}}
	n := mergeCategories(c, []Category{
		{Name: "Teknoloji", Sources: []Source{{URL: "https://a/x"}, {URL: "https://a/new"}}},
		{Name: "Yeni", Sources: []Source{{URL: "https://z/1"}}},
	})
	if n != 2 {
		t.Errorf("added=%d want 2", n)
	}
	if len(c.Categories) != 2 {
		t.Errorf("categories=%d want 2", len(c.Categories))
	}
}
