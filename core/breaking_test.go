package core

import (
	"testing"
	"time"
)

func TestClassifyBreaking(t *testing.T) {
	now := time.Now()
	cases := []struct {
		title   string
		pub     time.Time
		hasDate bool
		want    bool
	}{
		{"Son dakika: Ankara'da gelişme", time.Time{}, false, true},                             // strong marker, no date
		{"Japan issues tsunami warning after quake", now.Add(-30 * time.Minute), true, true},    // hazard + recent
		{"速報：地震発生", now.Add(-10 * time.Minute), true, true},                                     // JP marker/hazard + recent
		{"Allerta vulcano: evacuazione in corso", now.Add(-1 * time.Hour), true, true},          // IT hazard + recent
		{"New smartphone released with better camera", now.Add(-20 * time.Minute), true, false}, // normal
		{"Earthquake retrospective from last year", now.Add(-10 * time.Hour), true, false},      // hazard but stale
	}
	for _, c := range cases {
		got := classifyBreaking(c.title, c.pub, c.hasDate)
		if got != c.want {
			t.Errorf("classifyBreaking(%q)=%v want %v", c.title, got, c.want)
		}
	}
}
