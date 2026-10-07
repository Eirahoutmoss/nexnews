package core

import "testing"

func TestWireCopyCountsAsOneOrigin(t *testing.T) {
	a := mkItem("Malatya'da deprem", "hurriyet", 1)
	a.Summary = "Malatya'da 5,4 büyüklüğünde deprem meydana geldi. (AA)"
	b := mkItem("Malatya'da deprem", "milliyet", 2)
	b.Summary = "AFAD açıklamasına göre deprem 7 km derinlikte. Anadolu Ajansı"
	c := mkItem("Malatya'da deprem", "bbc", 3)
	c.Summary = "BBC Türkçe muhabirinin aktardığına göre bölgede hasar tespiti sürüyor."
	st := newStory(a, nil)
	st.add(b)
	st.add(c)
	if n := independentSourceCount(st); n != 2 {
		t.Fatalf("expected 2 origins (AA + BBC), got %d", n)
	}
	n, ag := syndicatedCount(st)
	if n != 2 || len(ag) != 1 || ag[0] != "AA" {
		t.Fatalf("syndicated=%d agencies=%v", n, ag)
	}
}

func TestEchoedBodyTextFolds(t *testing.T) {
	body := "Merkez Bankası Para Politikası Kurulu bugünkü toplantısında politika faizini yüzde kırk beş seviyesinde sabit tutma kararı aldı ve enflasyon görünümüne ilişkin değerlendirmelerini paylaştı"
	a := mkItem("Faiz sabit", "x", 1)
	a.Summary = body
	b := mkItem("Faiz sabit kaldı", "y", 2)
	b.Summary = body + "."
	st := newStory(a, nil)
	st.add(b)
	if n := independentSourceCount(st); n != 1 {
		t.Fatalf("expected echoed text to fold into 1 origin, got %d", n)
	}
}
