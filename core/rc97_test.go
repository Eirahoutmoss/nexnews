package core

import (
	"strings"
	"testing"
	"time"
)

func TestSensationScore(t *testing.T) {
	calm := sensationScore("Merkez Bankası politika faizini yüzde 40'ta sabit tuttu")
	hot := sensationScore("ŞOK GELİŞME! Görenler şaştı, sosyal medya yıkıldı!!")
	if calm >= sensationHigh {
		t.Fatalf("sakin başlık yüksek puan aldı: %d", calm)
	}
	if hot < sensationHigh {
		t.Fatalf("abartılı başlık düşük puan aldı: %d", hot)
	}
	if s := sensationScore("You won't believe what happened next"); s < sensationHigh {
		t.Fatalf("merak tuzağı yakalanmadı: %d", s)
	}
	if s := sensationScore("NATO ve AB liderleri Brüksel'de bir araya geldi"); s >= 20 {
		t.Fatalf("kısaltmalar abartı sayıldı: %d", s)
	}
}

func TestFactVerdict(t *testing.T) {
	cases := map[string]string{
		"Depremde barajın çöktüğü iddiası asılsız":                "yanlis",
		"Fotoğraf 2019'a ait, bağlamından koparılarak paylaşıldı": "yaniltici",
		"Viral video is misleading":                               "yaniltici",
		"Claim about vaccines is false":                           "yanlis",
		"Bakanlığın açıklaması üzerine":                           "",
	}
	for in, want := range cases {
		if got := factVerdict(in); got != want {
			t.Errorf("%q: %q beklendi, %q geldi", in, want, got)
		}
	}
}

func TestParseMag(t *testing.T) {
	cases := map[string]float64{
		"M 6.1 - 12 km SW of Sindirgi, Turkey":                           6.1,
		"Malatya'da 5,6 büyüklüğünde deprem":                             5.6,
		"Orange earthquake alert (Magnitude 6.3M, Depth:10km) in Turkey": 6.3,
		"Ekonomide 2.5 puanlık büyüme":                                   0,
	}
	for in, want := range cases {
		if got := parseMag(in); got != want {
			t.Errorf("%q: %v beklendi, %v geldi", in, want, got)
		}
	}
	if crisisKindOf("İzmir'de sel felaketi") != "sel" || crisisKindOf("Seller rekor kırdı") != "" {
		t.Error("sel türü yanlış algılandı")
	}
	if !reGDACS.MatchString("Red earthquake alert (Magnitude 7.8M, Depth:10km) in Turkey 06/02/2023") {
		t.Error("GDACS kırmızı uyarı eşleşmedi")
	}
}

func TestContextCandidates(t *testing.T) {
	titles := []string{
		"Dışişleri Bakanı Hakan Fidan Brüksel'de NATO toplantısına katıldı",
		"Hakan Fidan NATO zirvesinde konuştu",
		"NATO'da Hakan Fidan ile görüşme",
	}
	c := contextCandidates(titles)
	j := strings.Join(c, "|")
	if !strings.Contains(j, "Hakan Fidan") || !strings.Contains(j, "NATO") {
		t.Fatalf("beklenen adlar yok: %v", c)
	}
	if got := trimSentences("Bir. İki. Üç. Dört.", 2, 100); got != "Bir. İki." {
		t.Fatalf("trimSentences: %q", got)
	}
}

func TestAgendaRules(t *testing.T) {
	tm, allDay, ok := agendaTime("2026-10-28", "14:00", "America/New_York")
	if !ok || allDay || tm.UTC().Hour() != 18 {
		t.Fatalf("Fed saati yanlış: %v %v %v", tm.UTC(), allDay, ok)
	}
	b, err := agendaJSON(400)
	if err != nil || !strings.HasPrefix(string(b), "[") {
		t.Fatalf("agendaJSON: %v %s", err, b)
	}
	// İlk cuma kuralı: 2026 Kasım'ın ilk cuması 6 Kasım.
	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if day := 1 + (5-int(base.Weekday())+7)%7; day != 6 {
		t.Fatalf("ilk cuma %d", day)
	}
}

func TestVisitsTouch(t *testing.T) {
	v := &visitStore{m: map[string]*visitRec{}, path: t.TempDir() + "/v.json"}
	if p := v.touch("e1"); p != 0 {
		t.Fatalf("ilk ziyaret %d", p)
	}
	v.m["e1"].Cur -= 3600 // bir saat önce açılmış gibi
	first := v.m["e1"].Cur
	if p := v.touch("e1"); p != first {
		t.Fatalf("önceki ziyaret %d, beklenen %d", p, first)
	}
	if p := v.touch("e1"); p != first { // hemen yeniden açılış aynı ziyaret
		t.Fatalf("aynı ziyaret tabanı değişti: %d", p)
	}
}

func TestCrisisCountryMatch(t *testing.T) {
	if !titleMentionsCountry("M 6.1 - 12 km SW of Sindirgi, Turkey", "TR") {
		t.Error("USGS Turkey eşleşmedi")
	}
	if !titleMentionsCountry("Orange earthquake alert (Magnitude 6.3M) in Türkiye", "TR") {
		t.Error("Türkiye eşleşmedi")
	}
	if titleMentionsCountry("M 6.5 - 40 km S of Kokopo, Papua New Guinea", "TR") {
		t.Error("başka ülke eşleşti")
	}
}

func TestHolidaysAndRules(t *testing.T) {
	found := false
	for _, h := range countryHolidays("MR") {
		if h.Date == "2026-11-28" {
			found = true
		}
	}
	if !found {
		t.Error("Moritanya bağımsızlık günü yok")
	}
	if len(countryHolidays("AQ")) == 0 {
		t.Error("Antarktika günleri yok")
	}
	tr := countryHolidays("TR")
	ok := false
	for _, h := range tr {
		if h.Date == "2026-10-29" {
			ok = true
		}
	}
	if !ok {
		t.Error("29 Ekim yok")
	}
	if d := ruleDates("nth:05:0:2", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); d[0] != "2026-05-10" {
		t.Errorf("Anneler Günü %v", d)
	}
	if flagEmoji("TR") != "🇹🇷" {
		t.Error("bayrak")
	}
}
