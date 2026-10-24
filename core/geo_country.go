package core

import "strings"

// CountryProfile describes the geographic identity of a news region. The
// classifier deliberately uses title/summary/category context, not source
// country, because a Turkish source can publish a US story and vice versa.
type CountryProfile struct {
	Code     string
	Name     string
	Aliases  []string
	Capitals []string
}

var countryProfiles = []CountryProfile{
	{Code: "TR", Name: "Türkiye", Aliases: []string{"turkiye", "türkiye", "turkey", "republic of türkiye", "republic of turkey"}, Capitals: []string{"ankara"}},
	{Code: "TD", Name: "Çad", Aliases: []string{"çad", "tchad", "chad"}, Capitals: []string{"n'djamena", "ndjamena", "n’djamena"}},
	{Code: "DE", Name: "Almanya", Aliases: []string{"almanya", "germany", "deutschland"}, Capitals: []string{"berlin"}},
	{Code: "FR", Name: "Fransa", Aliases: []string{"fransa", "france"}, Capitals: []string{"paris"}},
	{Code: "GB", Name: "Birleşik Krallık", Aliases: []string{"birleşik krallık", "ingiltere", "united kingdom", "great britain", "england", "uk"}, Capitals: []string{"london"}},
	{Code: "US", Name: "Amerika Birleşik Devletleri", Aliases: []string{"amerika birleşik devletleri", "amerika", "united states", "united states of america", "usa", "u.s.", "u.s.a."}, Capitals: []string{"washington"}},
	{Code: "CA", Name: "Kanada", Aliases: []string{"kanada", "canada"}, Capitals: []string{"ottawa"}},
	{Code: "MX", Name: "Meksika", Aliases: []string{"meksika", "mexico"}, Capitals: []string{"mexico city", "ciudad de méxico"}},
	{Code: "BR", Name: "Brezilya", Aliases: []string{"brezilya", "brazil"}, Capitals: []string{"brasilia"}},
	{Code: "AR", Name: "Arjantin", Aliases: []string{"arjantin", "argentina"}, Capitals: []string{"buenos aires"}},
	{Code: "CL", Name: "Şili", Aliases: []string{"şili", "chile"}, Capitals: []string{"santiago"}},
	{Code: "CO", Name: "Kolombiya", Aliases: []string{"kolombiya", "colombia"}, Capitals: []string{"bogota", "bogotá"}},
	{Code: "PE", Name: "Peru", Aliases: []string{"peru"}, Capitals: []string{"lima"}},
	{Code: "AU", Name: "Avustralya", Aliases: []string{"avustralya", "australia"}, Capitals: []string{"canberra"}},
	{Code: "NZ", Name: "Yeni Zelanda", Aliases: []string{"yeni zelanda", "new zealand"}, Capitals: []string{"wellington"}},
	{Code: "JP", Name: "Japonya", Aliases: []string{"japonya", "japan", "日本"}, Capitals: []string{"tokyo", "東京"}},
	{Code: "KR", Name: "Güney Kore", Aliases: []string{"güney kore", "south korea", "republic of korea", "대한민국", "한국"}, Capitals: []string{"seoul", "seul", "서울"}},
	{Code: "CN", Name: "Çin", Aliases: []string{"çin", "china", "people's republic of china", "中国"}, Capitals: []string{"beijing", "peking", "pekin", "北京"}},
	{Code: "IN", Name: "Hindistan", Aliases: []string{"hindistan", "india"}, Capitals: []string{"new delhi", "yeni delhi"}},
	{Code: "PK", Name: "Pakistan", Aliases: []string{"pakistan"}, Capitals: []string{"islamabad"}},
	{Code: "ID", Name: "Endonezya", Aliases: []string{"endonezya", "indonesia"}, Capitals: []string{"jakarta"}},
	{Code: "MY", Name: "Malezya", Aliases: []string{"malezya", "malaysia"}, Capitals: []string{"kuala lumpur"}},
	{Code: "SG", Name: "Singapur", Aliases: []string{"singapur", "singapore"}, Capitals: []string{"singapore"}},
	{Code: "TH", Name: "Tayland", Aliases: []string{"tayland", "thailand"}, Capitals: []string{"bangkok"}},
	{Code: "VN", Name: "Vietnam", Aliases: []string{"vietnam", "viet nam"}, Capitals: []string{"hanoi"}},
	{Code: "PH", Name: "Filipinler", Aliases: []string{"filipinler", "philippines"}, Capitals: []string{"manila"}},
	{Code: "RU", Name: "Rusya", Aliases: []string{"rusya", "russia", "russian federation", "россия"}, Capitals: []string{"moscow", "moskova", "москва"}},
	{Code: "UA", Name: "Ukrayna", Aliases: []string{"ukrayna", "ukraine", "україна"}, Capitals: []string{"kyiv", "kiev", "київ"}},
	{Code: "PL", Name: "Polonya", Aliases: []string{"polonya", "poland", "polska"}, Capitals: []string{"warsaw", "varşova"}},
	{Code: "IT", Name: "İtalya", Aliases: []string{"italya", "italy", "italia"}, Capitals: []string{"rome", "roma"}},
	{Code: "ES", Name: "İspanya", Aliases: []string{"ispanya", "spain", "españa"}, Capitals: []string{"madrid"}},
	{Code: "NL", Name: "Hollanda", Aliases: []string{"hollanda", "netherlands", "the netherlands"}, Capitals: []string{"amsterdam"}},
	{Code: "BE", Name: "Belçika", Aliases: []string{"belçika", "belgium"}, Capitals: []string{"brussels", "brüksel"}},
	{Code: "SE", Name: "İsveç", Aliases: []string{"isveç", "sweden"}, Capitals: []string{"stockholm"}},
	{Code: "NO", Name: "Norveç", Aliases: []string{"norveç", "norway"}, Capitals: []string{"oslo"}},
	{Code: "DK", Name: "Danimarka", Aliases: []string{"danimarka", "denmark"}, Capitals: []string{"copenhagen", "kopenhag"}},
	{Code: "FI", Name: "Finlandiya", Aliases: []string{"finlandiya", "finland"}, Capitals: []string{"helsinki"}},
	{Code: "GR", Name: "Yunanistan", Aliases: []string{"yunanistan", "greece", "hellas"}, Capitals: []string{"athens", "atina"}},
	{Code: "BG", Name: "Bulgaristan", Aliases: []string{"bulgaristan", "bulgaria"}, Capitals: []string{"sofia"}},
	{Code: "RO", Name: "Romanya", Aliases: []string{"romanya", "romania"}, Capitals: []string{"bucharest", "bükreş"}},
	{Code: "RS", Name: "Sırbistan", Aliases: []string{"sırbistan", "serbia"}, Capitals: []string{"belgrade", "belgrad"}},
	{Code: "IL", Name: "İsrail", Aliases: []string{"israil", "israel"}, Capitals: []string{"jerusalem", "kudüs"}},
	{Code: "PS", Name: "Filistin", Aliases: []string{"filistin", "palestine", "palestinian territories"}, Capitals: []string{"ramallah"}},
	{Code: "SA", Name: "Suudi Arabistan", Aliases: []string{"suudi arabistan", "saudi arabia"}, Capitals: []string{"riyadh", "riyad"}},
	{Code: "AE", Name: "Birleşik Arap Emirlikleri", Aliases: []string{"birleşik arap emirlikleri", "united arab emirates", "uae"}, Capitals: []string{"abu dhabi"}},
	{Code: "IR", Name: "İran", Aliases: []string{"iran", "iran"}, Capitals: []string{"tehran", "tahran"}},
	{Code: "IQ", Name: "Irak", Aliases: []string{"ırak", "irak", "iraq"}, Capitals: []string{"baghdad", "bağdat"}},
	{Code: "SY", Name: "Suriye", Aliases: []string{"suriye", "syria"}, Capitals: []string{"damascus", "şam"}},
	{Code: "EG", Name: "Mısır", Aliases: []string{"mısır", "misir", "egypt"}, Capitals: []string{"cairo", "kahire"}},
	{Code: "DZ", Name: "Cezayir", Aliases: []string{"cezayir", "cezayir", "algeria"}, Capitals: []string{"algiers"}},
	{Code: "MA", Name: "Fas", Aliases: []string{"fas", "morocco"}, Capitals: []string{"rabat"}},
	{Code: "TN", Name: "Tunus", Aliases: []string{"tunus", "tunisia"}, Capitals: []string{"tunis"}},
	{Code: "LY", Name: "Libya", Aliases: []string{"libya", "libya"}, Capitals: []string{"tripoli"}},
	{Code: "ET", Name: "Etiyopya", Aliases: []string{"etiyopya", "ethiopia"}, Capitals: []string{"addis ababa"}},
	{Code: "KE", Name: "Kenya", Aliases: []string{"kenya"}, Capitals: []string{"nairobi"}},
	{Code: "NG", Name: "Nijerya", Aliases: []string{"nijerya", "nigeria"}, Capitals: []string{"abuja"}},
	{Code: "ZA", Name: "Güney Afrika", Aliases: []string{"güney afrika", "south africa"}, Capitals: []string{"pretoria", "cape town", "cape town"}},
	{Code: "GH", Name: "Gana", Aliases: []string{"gana", "ghana"}, Capitals: []string{"accra"}},
	{Code: "SN", Name: "Senegal", Aliases: []string{"senegal"}, Capitals: []string{"dakar"}},
	{Code: "CM", Name: "Kamerun", Aliases: []string{"kamerun", "cameroon"}, Capitals: []string{"yaounde", "yaoundé"}},
	{Code: "SD", Name: "Sudan", Aliases: []string{"sudan"}, Capitals: []string{"khartoum", "hartum"}},
	{Code: "TZ", Name: "Tanzanya", Aliases: []string{"tanzanya", "tanzania"}, Capitals: []string{"dodoma"}},
	{Code: "UG", Name: "Uganda", Aliases: []string{"uganda"}, Capitals: []string{"kampala"}},
	{Code: "RW", Name: "Ruanda", Aliases: []string{"ruanda", "rwanda"}, Capitals: []string{"kigali"}},
	{Code: "QA", Name: "Katar", Aliases: []string{"katar", "qatar", "قطر"}, Capitals: []string{"doha", "doha", "الدوحة"}},
	{Code: "KW", Name: "Kuveyt", Aliases: []string{"kuveyt", "kuwait", "الكويت"}, Capitals: []string{"kuwait city"}},
	{Code: "BH", Name: "Bahreyn", Aliases: []string{"bahreyn", "bahrain", "البحرين"}, Capitals: []string{"manama"}},
	{Code: "OM", Name: "Umman", Aliases: []string{"umman", "oman", "عمان"}, Capitals: []string{"muscat", "maskat"}},
	{Code: "JO", Name: "Ürdün", Aliases: []string{"ürdün", "jordan", "الأردن"}, Capitals: []string{"amman"}},
	{Code: "LB", Name: "Lübnan", Aliases: []string{"lübnan", "lebanon", "liban", "لبنان"}, Capitals: []string{"beirut", "beyrut"}},
	{Code: "AZ", Name: "Azerbaycan", Aliases: []string{"azerbaycan", "azerbaijan"}, Capitals: []string{"baku", "bakü"}},
	{Code: "GE", Name: "Gürcistan", Aliases: []string{"gürcistan", "georgia"}, Capitals: []string{"tbilisi", "tiflis"}},
	{Code: "AM", Name: "Ermenistan", Aliases: []string{"ermenistan", "armenia"}, Capitals: []string{"yerevan", "erivan"}},
	{Code: "KZ", Name: "Kazakistan", Aliases: []string{"kazakistan", "kazakhstan"}, Capitals: []string{"astana", "almaty"}},
	{Code: "UZ", Name: "Özbekistan", Aliases: []string{"özbekistan", "uzbekistan"}, Capitals: []string{"tashkent", "taşkent"}},
	{Code: "KG", Name: "Kırgızistan", Aliases: []string{"kırgızistan", "kyrgyzstan"}, Capitals: []string{"bishkek", "bişkek"}},
	{Code: "TM", Name: "Türkmenistan", Aliases: []string{"türkmenistan", "turkmenistan"}, Capitals: []string{"ashgabat", "aşkabat"}},
	{Code: "IE", Name: "İrlanda", Aliases: []string{"irlanda", "ireland"}, Capitals: []string{"dublin"}},
	{Code: "PT", Name: "Portekiz", Aliases: []string{"portekiz", "portugal"}, Capitals: []string{"lisbon", "lizbon"}},
	{Code: "AT", Name: "Avusturya", Aliases: []string{"avusturya", "austria", "österreich"}, Capitals: []string{"vienna", "viyana", "wien"}},
	{Code: "CH", Name: "İsviçre", Aliases: []string{"isviçre", "switzerland", "schweiz", "suisse"}, Capitals: []string{"bern", "geneva", "cenevre", "zurich"}},
	{Code: "CZ", Name: "Çekya", Aliases: []string{"çekya", "czech republic", "czechia"}, Capitals: []string{"prague", "prag"}},
	{Code: "HU", Name: "Macaristan", Aliases: []string{"macaristan", "hungary"}, Capitals: []string{"budapest", "budapeşte"}},
	{Code: "CY", Name: "Kıbrıs", Aliases: []string{"kıbrıs", "cyprus"}, Capitals: []string{"nicosia", "lefkoşa"}},
	{Code: "BA", Name: "Bosna-Hersek", Aliases: []string{"bosna", "bosnia"}, Capitals: []string{"sarajevo", "saraybosna"}},
	{Code: "AL", Name: "Arnavutluk", Aliases: []string{"arnavutluk", "albania"}, Capitals: []string{"tirana", "tiran"}},
	{Code: "MK", Name: "Kuzey Makedonya", Aliases: []string{"makedonya", "north macedonia"}, Capitals: []string{"skopje", "üsküp"}},
	{Code: "TW", Name: "Tayvan", Aliases: []string{"tayvan", "taiwan"}, Capitals: []string{"taipei"}},
	{Code: "HK", Name: "Hong Kong", Aliases: []string{"hong kong"}, Capitals: []string{}},
	{Code: "BD", Name: "Bangladeş", Aliases: []string{"bangladeş", "bangladesh"}, Capitals: []string{"dhaka", "dakka"}},
	{Code: "AF", Name: "Afganistan", Aliases: []string{"afganistan", "afghanistan"}, Capitals: []string{"kabul", "kâbil"}},
	{Code: "SO", Name: "Somali", Aliases: []string{"somali", "somalia"}, Capitals: []string{"mogadishu", "mogadişu"}},
	{Code: "VE", Name: "Venezuela", Aliases: []string{"venezuela"}, Capitals: []string{"caracas", "karakas"}},
}

func countryProfile(code string) (CountryProfile, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, p := range countryProfiles {
		if p.Code == code {
			return p, true
		}
	}
	return CountryProfile{}, false
}

func countryOptions() []CountryProfile { return append([]CountryProfile(nil), countryProfiles...) }

// countryRelevance returns 0..1. Title is intentionally dominant. A source's
// origin is never used as evidence because source-country and story-country are
// different concepts.
func countryRelevance(st *Story, code string) float64 {
	p, ok := countryProfile(code)
	if !ok || st == nil {
		return 1
	}
	text := normalizeGeo(st.Title)
	summary := normalizeGeo(st.Summary)
	cat := normalizeGeo(st.Category)
	if p.Code == "TD" && (strings.Contains(text, "chad smith") || strings.Contains(text, "chadwick")) {
		return 0
	}
	best := 0.0
	for _, a := range append(append(append([]string{}, p.Aliases...), p.Name), p.Capitals...) {
		a = normalizeGeo(a)
		if a == "" {
			continue
		}
		if geoHas(text, a) {
			score := 0.94
			if containsCapital(p, a) {
				score = 0.86
			}
			if score > best {
				best = score
			}
		}
		if geoHas(summary, a) && 0.62 > best {
			best = 0.62
		}
		if geoHas(cat, a) && 0.82 > best {
			best = 0.82
		}
	}
	// Regional category is supporting evidence only, never enough for country-only mode.
	if best == 0 && geoCategorySupportsCountry(cat, p.Code) {
		best = 0.28
	}
	return best
}

func containsCapital(p CountryProfile, x string) bool {
	for _, c := range p.Capitals {
		if normalizeGeo(c) == x {
			return true
		}
	}
	return false
}
func geoHas(text, term string) bool {
	text = normalizeGeo(text)
	term = normalizeGeo(term)
	if text == "" || term == "" {
		return false
	}
	// Match normal word boundaries, plus Turkish/other inflected forms such as
	// "Ankara'da" and "N'Djamena'da". The latter are common in headlines and
	// must still count as a geographic mention without turning arbitrary
	// substrings into matches.
	for _, raw := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == ',' || r == '.' || r == ':' || r == ';' || r == '!' || r == '?' || r == '(' || r == ')' || r == '[' || r == ']' || r == '"'
	}) {
		tok := strings.Trim(raw, "-–—/\\")
		if tok == term || strings.HasPrefix(tok, term+"'") || strings.HasPrefix(tok, term+"’") {
			return true
		}
	}
	return false
}
func normalizeGeo(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
}
func geoCategorySupportsCountry(cat, code string) bool {
	if strings.Contains(cat, "türkiye") && code == "TR" {
		return true
	}
	if strings.Contains(cat, "afrika") && code != "TR" {
		return true
	}
	if strings.Contains(cat, "avrupa") && code != "TR" {
		return true
	}
	if strings.Contains(cat, "asya") && code != "TR" {
		return true
	}
	if strings.Contains(cat, "ortadoğu") && code != "TR" {
		return true
	}
	return false
}
