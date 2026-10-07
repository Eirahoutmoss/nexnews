package core

import "testing"

func TestHideAndInterest(t *testing.T) {
	s := newStore(&Config{})
	s.setPersonal([]string{"deprem", "uzay"}, []string{"magazin"})
	hide := &Story{Title: "Magazin dünyasından haberler"}
	keep := &Story{Title: "Van'da deprem oldu"}
	if !s.isHidden(hide) {
		t.Error("magazin should be hidden")
	}
	if s.isHidden(keep) {
		t.Error("deprem should not be hidden")
	}
	if !s.matchesInterest(keep) {
		t.Error("deprem should match interest")
	}
	if s.matchesInterest(&Story{Title: "Hava güzel"}) {
		t.Error("no match expected")
	}
}

func TestReliability(t *testing.T) {
	if l, _ := reliability(4); l != "yüksek" {
		t.Errorf("4 src => %s", l)
	}
	if l, _ := reliability(2); l != "orta" {
		t.Errorf("2 src => %s", l)
	}
	if l, _ := reliability(1); l != "düşük" {
		t.Errorf("1 src => %s", l)
	}
}
