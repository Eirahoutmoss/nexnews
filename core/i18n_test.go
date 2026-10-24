package core

import "testing"

func TestGoSideTranslation(t *testing.T) {
	old := cfg
	defer func() { cfg = old }()
	cfg = &Config{UILang: "en"}
	if got := uiTK("hava:Açık", "Açık"); got != "Clear" {
		t.Fatalf("hava:Açık → %q", got)
	}
	if got := uiT("Önemli"); got != "Important" {
		t.Fatalf("Önemli → %q", got)
	}
	cfg = &Config{UILang: "de"}
	if got := uiTK("gün:Çar", "Çar"); got != "Mi" {
		t.Fatalf("gün:Çar → %q", got)
	}
	cfg = &Config{UILang: "ja"} // sözlüğü yok → İngilizce
	if got := uiTK("hava:Açık", "Açık"); got != "Clear" {
		t.Fatalf("ja → %q", got)
	}
	cfg = &Config{UILang: "tr"}
	if got := uiTK("hava:Açık", "Açık"); got != "Açık" {
		t.Fatalf("tr → %q", got)
	}
}
