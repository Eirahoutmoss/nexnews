package core

import "testing"

func TestCountryRelevanceTurkey(t *testing.T) {
	st := &Story{Title: "Ankara'da yeni eğitim düzenlemesi açıklandı", Category: "Teknoloji"}
	if got := countryRelevance(st, "TR"); got < 0.80 {
		t.Fatalf("Turkey relevance too low: %v", got)
	}
	foreign := &Story{Title: "US election results announced", Category: "Küresel / Uluslararası"}
	if got := countryRelevance(foreign, "TR"); got >= 0.42 {
		t.Fatalf("foreign story passed Turkey threshold: %v", got)
	}
}

func TestCountryRelevanceChad(t *testing.T) {
	cases := []struct {
		title string
		want  float64
	}{
		{"N'Djamena'da hükümet yeni karar açıkladı", 0.80},
		{"Tchad announces new regional policy", 0.80},
		{"Chad Smith yeni albümünü duyurdu", 0.00},
	}
	for _, tc := range cases {
		st := &Story{Title: tc.title, Category: "Küresel / Uluslararası"}
		got := countryRelevance(st, "TD")
		if tc.want > 0 && got < tc.want {
			t.Fatalf("%q: got %v, want >= %v", tc.title, got, tc.want)
		}
		if tc.want == 0 && got >= 0.42 {
			t.Fatalf("%q: false positive %v", tc.title, got)
		}
	}
}
