package core

// Yerel haber kategorisi (RC86).
//
// NexNews'in hazır kaynak listesi Türkiye ve büyük bölgeler üzerine kurulu.
// Katar'daki bir kullanıcı uygulamayı açtığında kendi ülkesinin haberlerini
// göremiyordu. Artık kullanıcının ülkesi (Windows "Konum" ayarı ya da
// Ayarlar › Haber bölgesi) okunur ve o ülkenin başlıca haber sitelerinden
// "📍 <Ülke>" kategorisi kurulur. RSS adresleri ezbere yazılmaz: sitelerin ana
// sayfasından uygulamanın kendi otomatik feed keşfiyle bulunur, böylece adres
// değişse de doğru feed yakalanır.

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"
)

// localNewsSites: ülke → o ülkenin başlıca haber siteleri (ana sayfa).
var localNewsSites = map[string][]string{
	"QA": {"aljazeera.net", "gulf-times.com", "thepeninsulaqatar.com", "qna.org.qa", "alsharq.com", "al-watan.com", "raya.com"},
	"AE": {"thenationalnews.com", "gulfnews.com", "khaleejtimes.com", "wam.ae", "alkhaleej.ae", "albayan.ae"},
	"SA": {"arabnews.com", "saudigazette.com.sa", "spa.gov.sa", "okaz.com.sa", "alriyadh.com", "aawsat.com"},
	"KW": {"kuwaittimes.com", "kuna.net.kw", "alqabas.com", "alraimedia.com"},
	"BH": {"gdnonline.com", "bna.bh", "alayam.com"},
	"OM": {"timesofoman.com", "omanobserver.om", "alwatan.com"},
	"JO": {"jordantimes.com", "petra.gov.jo", "alghad.com", "ammonnews.net"},
	"LB": {"naharnet.com", "lorientlejour.com", "annahar.com", "nna-leb.gov.lb"},
	"IQ": {"ina.iq", "rudaw.net", "shafaq.com", "kurdistan24.net"},
	"SY": {"enabbaladi.net", "sana.sy"},
	"PS": {"wafa.ps", "maannews.net"},
	"EG": {"ahram.org.eg", "egyptindependent.com", "youm7.com", "masrawy.com", "dailynewsegypt.com"},
	"MA": {"hespress.com", "le360.ma", "mapexpress.ma", "moroccoworldnews.com"},
	"DZ": {"aps.dz", "tsa-algerie.com", "echoroukonline.com"},
	"TN": {"tap.info.tn", "businessnews.com.tn", "kapitalis.com"},
	"LY": {"libyaobserver.ly", "libyaherald.com"},
	"IR": {"tehrantimes.com", "irna.ir", "isna.ir"},
	"IL": {"timesofisrael.com", "haaretz.com", "jpost.com", "ynetnews.com"},
	"AZ": {"apa.az", "report.az", "azertag.az", "trend.az"},
	"GE": {"civil.ge", "agenda.ge", "interpressnews.ge"},
	"AM": {"armenpress.am", "news.am"},
	"KZ": {"inform.kz", "tengrinews.kz", "astanatimes.com"},
	"UZ": {"kun.uz", "gazeta.uz", "uza.uz"},
	"KG": {"24.kg", "kabar.kg"},
	"TM": {"turkmenportal.com"},
	"DE": {"tagesschau.de", "spiegel.de", "zeit.de", "faz.net", "sueddeutsche.de"},
	"FR": {"lemonde.fr", "francetvinfo.fr", "lefigaro.fr", "liberation.fr"},
	"GB": {"bbc.co.uk", "theguardian.com", "independent.co.uk", "news.sky.com"},
	"IE": {"rte.ie", "irishtimes.com", "independent.ie"},
	"NL": {"nos.nl", "nu.nl", "volkskrant.nl"},
	"BE": {"vrt.be", "lesoir.be", "rtbf.be", "brusselstimes.com"},
	"IT": {"ansa.it", "repubblica.it", "corriere.it", "rainews.it"},
	"ES": {"elpais.com", "rtve.es", "elmundo.es", "lavanguardia.com"},
	"PT": {"publico.pt", "rtp.pt", "observador.pt"},
	"AT": {"orf.at", "derstandard.at", "diepresse.com"},
	"CH": {"srf.ch", "nzz.ch", "swissinfo.ch", "rts.ch"},
	"SE": {"svt.se", "dn.se", "aftonbladet.se"},
	"NO": {"nrk.no", "vg.no", "aftenposten.no"},
	"DK": {"dr.dk", "politiken.dk"},
	"FI": {"yle.fi", "hs.fi"},
	"PL": {"tvn24.pl", "rp.pl", "notesfrompoland.com"},
	"CZ": {"ct24.ceskatelevize.cz", "idnes.cz", "novinky.cz"},
	"HU": {"telex.hu", "hvg.hu", "index.hu"},
	"GR": {"ekathimerini.com", "ertnews.gr", "naftemporiki.gr"},
	"CY": {"cyprus-mail.com", "philenews.com"},
	"RO": {"digi24.ro", "hotnews.ro", "agerpres.ro"},
	"BG": {"novinite.com", "bta.bg"},
	"RS": {"rts.rs", "b92.net"},
	"BA": {"klix.ba", "aa.com.tr/ba"},
	"AL": {"top-channel.tv", "euronews.al"},
	"MK": {"mia.mk", "slobodenpecat.mk"},
	"UA": {"ukrinform.net", "kyivindependent.com", "pravda.com.ua"},
	"RU": {"tass.com", "rbc.ru", "interfax.ru"},
	"US": {"apnews.com", "npr.org", "nytimes.com", "cnn.com"},
	"CA": {"cbc.ca", "globalnews.ca", "theglobeandmail.com"},
	"MX": {"eluniversal.com.mx", "milenio.com", "jornada.com.mx"},
	"BR": {"g1.globo.com", "folha.uol.com.br", "agenciabrasil.ebc.com.br"},
	"AR": {"clarin.com", "lanacion.com.ar", "infobae.com"},
	"CL": {"latercera.com", "biobiochile.cl", "emol.com"},
	"CO": {"eltiempo.com", "elespectador.com", "semana.com"},
	"PE": {"elcomercio.pe", "larepublica.pe", "andina.pe"},
	"VE": {"efectococuyo.com", "elnacional.com"},
	"JP": {"nhk.or.jp", "japantimes.co.jp", "asahi.com", "kyodonews.net"},
	"KR": {"yna.co.kr", "koreaherald.com", "koreatimes.co.kr"},
	"CN": {"xinhuanet.com", "chinadaily.com.cn", "scmp.com"},
	"TW": {"focustaiwan.tw", "taipeitimes.com"},
	"HK": {"scmp.com", "hongkongfp.com"},
	"IN": {"thehindu.com", "indianexpress.com", "ndtv.com"},
	"PK": {"dawn.com", "geo.tv", "tribune.com.pk"},
	"BD": {"thedailystar.net", "bdnews24.com", "prothomalo.com"},
	"AF": {"tolonews.com", "pajhwok.com"},
	"ID": {"antaranews.com", "kompas.com", "thejakartapost.com"},
	"MY": {"thestar.com.my", "malaymail.com", "bernama.com"},
	"SG": {"channelnewsasia.com", "straitstimes.com"},
	"PH": {"rappler.com", "inquirer.net", "philstar.com"},
	"TH": {"bangkokpost.com", "thaipbsworld.com", "nationthailand.com"},
	"VN": {"vnexpress.net", "tuoitre.vn", "vietnamnews.vn"},
	"NG": {"punchng.com", "premiumtimesng.com", "vanguardngr.com", "channelstv.com"},
	"KE": {"nation.africa", "standardmedia.co.ke", "the-star.co.ke"},
	"ZA": {"news24.com", "iol.co.za", "dailymaverick.co.za", "sabcnews.com"},
	"GH": {"myjoyonline.com", "graphic.com.gh", "citinewsroom.com"},
	"ET": {"addisstandard.com", "fanabc.com"},
	"SN": {"aps.sn", "seneweb.com"},
	"SD": {"sudantribune.com", "dabangasudan.org"},
	"SO": {"hiiraan.com", "garoweonline.com"},
	"AU": {"abc.net.au", "smh.com.au", "theguardian.com/au"},
	"NZ": {"rnz.co.nz", "stuff.co.nz", "nzherald.co.nz"},
}

// countryNameTR: kategori başlığı için Türkçe ülke adı.
var countryNameTR = map[string]string{
	"QA": "Katar", "AE": "BAE", "SA": "Suudi Arabistan", "KW": "Kuveyt", "BH": "Bahreyn", "OM": "Umman", "JO": "Ürdün",
	"LB": "Lübnan", "IQ": "Irak", "SY": "Suriye", "PS": "Filistin", "EG": "Mısır", "MA": "Fas", "DZ": "Cezayir",
	"TN": "Tunus", "LY": "Libya", "IR": "İran", "IL": "İsrail", "AZ": "Azerbaycan", "GE": "Gürcistan", "AM": "Ermenistan",
	"KZ": "Kazakistan", "UZ": "Özbekistan", "KG": "Kırgızistan", "TM": "Türkmenistan", "DE": "Almanya", "FR": "Fransa",
	"GB": "Birleşik Krallık", "IE": "İrlanda", "NL": "Hollanda", "BE": "Belçika", "IT": "İtalya", "ES": "İspanya",
	"PT": "Portekiz", "AT": "Avusturya", "CH": "İsviçre", "SE": "İsveç", "NO": "Norveç", "DK": "Danimarka",
	"FI": "Finlandiya", "PL": "Polonya", "CZ": "Çekya", "HU": "Macaristan", "GR": "Yunanistan", "CY": "Kıbrıs",
	"RO": "Romanya", "BG": "Bulgaristan", "RS": "Sırbistan", "BA": "Bosna-Hersek", "AL": "Arnavutluk",
	"MK": "K. Makedonya", "UA": "Ukrayna", "RU": "Rusya", "US": "ABD", "CA": "Kanada", "MX": "Meksika",
	"BR": "Brezilya", "AR": "Arjantin", "CL": "Şili", "CO": "Kolombiya", "PE": "Peru", "VE": "Venezuela",
	"JP": "Japonya", "KR": "Güney Kore", "CN": "Çin", "TW": "Tayvan", "HK": "Hong Kong", "IN": "Hindistan",
	"PK": "Pakistan", "BD": "Bangladeş", "AF": "Afganistan", "ID": "Endonezya", "MY": "Malezya", "SG": "Singapur",
	"PH": "Filipinler", "TH": "Tayland", "VN": "Vietnam", "NG": "Nijerya", "KE": "Kenya", "ZA": "Güney Afrika",
	"GH": "Gana", "ET": "Etiyopya", "SN": "Senegal", "SD": "Sudan", "SO": "Somali", "AU": "Avustralya",
	"NZ": "Yeni Zelanda",
}

const localCatPrefix = "📍 "

var localMu sync.Mutex

// sysCountryOverride: Android tarafının bildirdiği cihaz ülkesi.
var sysCountryOverride string

// localNewsCountry: kullanıcı haber bölgesi seçtiyse o, yoksa sistemin ülkesi.
func localNewsCountry() string {
	cfg.mu.RLock()
	c := strings.ToUpper(strings.TrimSpace(cfg.NewsCountry))
	cfg.mu.RUnlock()
	if c == "" {
		c = sysCountryOverride
	}
	if c == "" {
		c = systemCountry()
	}
	return c
}

// ensureLocalCategory builds (once per country) the "📍 <Ülke>" category from
// the country's main news sites and returns true when the config changed.
// Türkiye için hazır kategori zaten olduğundan bir şey yapılmaz.
func ensureLocalCategory(ctx context.Context) bool {
	localMu.Lock()
	defer localMu.Unlock()
	cc := localNewsCountry()
	cfg.mu.RLock()
	done := cfg.LocalCountry
	cfg.mu.RUnlock()
	if cc == "" || cc == "TR" || cc == done {
		return false
	}
	sites := localNewsSites[cc]
	if len(sites) == 0 {
		cfg.mu.Lock()
		cfg.LocalCountry = cc
		cfg.mu.Unlock()
		_ = cfg.save()
		return false
	}
	type hit struct {
		i   int
		src Source
	}
	var mu sync.Mutex
	var hits []hit
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, site := range sites {
		wg.Add(1)
		go func(i int, site string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			c2, cancel := context.WithTimeout(ctx, 25*time.Second)
			defer cancel()
			var res discoverResult
			if json.Unmarshal(discoverJSON(c2, site), &res) != nil || len(res.Feeds) == 0 {
				return
			}
			f := res.Feeds[0]
			for _, x := range res.Feeds { // yorum/arama beslemelerini atla
				l := strings.ToLower(x.URL)
				if !strings.Contains(l, "comment") && !strings.Contains(l, "search") {
					f = x
					break
				}
			}
			name := strings.TrimSpace(f.Title)
			if name == "" || len([]rune(name)) > 40 {
				name = site
			}
			mu.Lock()
			hits = append(hits, hit{i, Source{Name: name, URL: f.URL}})
			mu.Unlock()
		}(i, site)
	}
	wg.Wait()
	if len(hits) == 0 {
		log.Printf("yerel kategori: %s için feed bulunamadı", cc)
		return false // ağ yoksa bir dahaki açılışta yeniden denenir
	}
	srcs := make([]Source, len(sites))
	for _, h := range hits {
		srcs[h.i] = h.src
	}
	var list []Source
	for _, s := range srcs {
		if s.URL != "" {
			list = append(list, s)
		}
	}
	name := countryNameTR[cc]
	if name == "" {
		name = cc
	}
	cat := Category{Name: localCatPrefix + name, Sources: list}
	cfg.mu.Lock()
	// Eski yerel kategoriyi (ülke değiştiyse) kaldır, yenisini uyarılardan sonra koy.
	var cats []Category
	for _, c := range cfg.Categories {
		if !strings.HasPrefix(c.Name, localCatPrefix) {
			cats = append(cats, c)
		}
	}
	pos := 0
	if len(cats) > 0 && strings.HasPrefix(cats[0].Name, "⚡") {
		pos = 1
	}
	cats = append(cats[:pos], append([]Category{cat}, cats[pos:]...)...)
	cfg.Categories = cats
	cfg.LocalCountry = cc
	cfg.mu.Unlock()
	_ = cfg.save()
	log.Printf("yerel kategori: %s (%d kaynak)", cat.Name, len(list))
	return true
}
