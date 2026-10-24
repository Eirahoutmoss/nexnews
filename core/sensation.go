package core

// Duygu yükü / sansasyon ölçer (RC97).
//
// Medya çalışmalarındaki "çerçeveleme" yaklaşımına göre aynı olgu farklı
// sözcüklerle anlatıldığında okurun olayı algılayışı da değişir. Abartılı,
// korku ya da öfke uyandıran ve merak tuzağı kuran başlıklar tıklanmayı
// artırır ama haber yorgunluğunu da besler. Bu ölçer bir doğruluk yargısı
// değildir; yalnızca başlığın DİLİNİ ölçer: duygu yüklü sözcükler, merak
// tuzağı kalıpları, büyük harf ve ünlem yoğunluğu.

import (
	"strings"
	"unicode"
)

// Ağırlıklı sözcük/kalıp listesi (küçük harf). Çok dilli; yalnızca sık ve
// açıkça duygu yüklü olanlar. "son dakika" bilerek yok: o bir etiket.
var sensationTerms = map[string]int{
	// Türkçe — korku / şok
	"şok": 22, "şoke": 22, "dehşet": 24, "vahşet": 22, "korkunç": 18, "kahreden": 18, "kan donduran": 26,
	"yürek burkan": 18, "yürekleri ağza getiren": 24, "felaket": 10, "kabus": 16, "panik": 12, "çılgın": 14,
	"skandal": 18, "rezalet": 20, "olay yaratan": 18, "ortalık karıştı": 18, "sosyal medya yıkıldı": 24,
	"bomba": 14, "flaş": 14, "inanılmaz": 16, "akılalmaz": 18, "akıl almaz": 18, "görenler şaştı": 22,
	"herkesi şaşırttı": 20, "şaşırtan": 12, "ağızları açık bıraktı": 24, "nefes kesen": 14, "tüyler ürperten": 22,
	"bakın ne": 22, "bakın kim": 22, "işte o": 14, "ortaya çıktı": 6, "tepki yağdı": 16, "yerden yere vurdu": 20,
	"sert çıktı": 8, "fena yakalandı": 18, "itiraf etti": 6, "gözyaşlarına boğuldu": 14,
	"çıldırttı": 20, "kriz çıktı": 12, "ifşa": 12, "olay sözler": 18, "olay paylaşım": 16, "duyanlar": 14,
	// English
	"shocking": 22, "shock": 14, "horrifying": 22, "horror": 14, "terrifying": 20, "chilling": 16, "bombshell": 20,
	"outrage": 16, "outraged": 16, "slams": 16, "blasts": 14, "destroys": 18, "obliterates": 20, "chaos": 12,
	"meltdown": 16, "nightmare": 14, "stunning": 12, "stunned": 12, "jaw-dropping": 22, "unbelievable": 18,
	"you won't believe": 30, "you will not believe": 30, "what happened next": 26, "here's why": 10, "insane": 18,
	"furious": 14, "fury": 14, "brutal": 12, "explosive": 12, "devastating": 10, "epic": 12, "must see": 18,
	// Deutsch / Français / Español / Русский / العربية (sık olanlar)
	"schock": 20, "unfassbar": 18, "horreur": 18, "choc": 16, "incroyable": 16, "scandale": 16,
	"impactante": 20, "escándalo": 16, "increíble": 16, "шок": 22, "скандал": 16, "ужас": 20,
	"сенсация": 18, "صادم": 20, "فضيحة": 18, "كارثة": 10,
}

// sensationScore: 0 (sakin) – 100 (çok abartılı).
func sensationScore(title string) int {
	t := strings.TrimSpace(title)
	if t == "" {
		return 0
	}
	low := strings.ToLower(t)
	score := 0
	for term, w := range sensationTerms {
		if w > 0 && strings.Contains(low, term) {
			score += w
		}
	}
	// Ünlem ve soru (merak tuzağı) işaretleri
	ex := strings.Count(t, "!")
	if ex > 0 {
		score += 10 + 6*min(ex-1, 3)
	}
	if strings.Contains(t, "?!") || strings.Contains(t, "!?") {
		score += 8
	}
	if strings.HasSuffix(t, "...") || strings.HasSuffix(t, "…") {
		score += 8 // yarım bırakılan cümle
	}
	// BÜYÜK HARF oranı (en az 3 harfli sözcükler; kısaltmalar tek başına sayılmaz)
	caps, words := 0, 0
	for _, w := range strings.Fields(t) {
		letters, upper := 0, 0
		for _, r := range w {
			if unicode.IsLetter(r) {
				letters++
				if unicode.IsUpper(r) {
					upper++
				}
			}
		}
		if letters >= 3 {
			words++
			if upper == letters {
				caps++
			}
		}
	}
	if words >= 4 && caps*100/words >= 60 {
		score += 22 // başlığın tamamı büyük harf
	} else if caps >= 2 {
		score += 8 * min(caps-1, 3)
	}
	// Emoji ve süs işaretleri
	for _, r := range t {
		if r >= 0x1F300 && r <= 0x1FAFF || r == '🔴' || r == '‼' {
			score += 8
			break
		}
	}
	if score > 100 {
		score = 100
	}
	return score
}

const sensationHigh = 40 // bu puandan itibaren arayüz "abartılı dil" işareti gösterir

// calmestTitle: olaydaki başlıklar arasından en sakin dilli olanı seçer.
// Temsilci başlık yeterince sakinse ya da belirgin biçimde daha sakin bir
// başlık yoksa boş döner. Çağıran s.mu kilidini tutuyor olmalı.
func (s *Store) calmestTitle(st *Story) (title, source string, repScore int) {
	repScore = sensationScore(s.trTitle(st.Title))
	if repScore < sensationHigh || len(st.items) < 2 {
		return "", "", repScore
	}
	best, bestScore, bestSrc := "", repScore, ""
	for _, it := range st.items {
		t := s.trTitle(it.Title)
		if sc := sensationScore(t); sc+15 <= bestScore && len([]rune(t)) >= 20 {
			best, bestScore, bestSrc = t, sc, it.Source
		}
	}
	return best, bestSrc, repScore
}
