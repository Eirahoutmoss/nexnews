package core

// Configuration: categories, their sources, and a few display options. Stored as
// JSON under the OS user-config dir (%APPDATA%\NexNews\config.json on Windows)
// so user-added sources survive restarts. The file is created with a sensible
// multilingual starter set on first run to prove "dil/kaynak bağımsız" at once.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Source struct {
	Name string `json:"name"` // optional friendly name; feed title used if empty
	URL  string `json:"url"`
}

type Category struct {
	Name    string   `json:"name"`
	Sources []Source `json:"sources"`
}

type Config struct {
	Categories     []Category `json:"categories"`
	MaxPerCategory int        `json:"maxPerCategory"` // headlines shown before "Tümünü Göster"
	OpenInApp      bool       `json:"openInApp"`      // true: in-app reader; false: default browser
	RefreshMinutes int        `json:"refreshMinutes"` // background auto-refresh; 0 = manual only
	Glass          bool       `json:"glass"`          // translucent (acrylic) window
	GlassOpacity   int        `json:"glassOpacity"`   // panel opacity 40–100 (%) when glass on
	Premium        bool       `json:"premium"`        // (legacy) cosmic background
	Theme          string     `json:"theme"`          // premium / koyu / acik / kontrast
	Translate      bool       `json:"translate"`      // translate titles to local language
	TranslateLang  string     `json:"translateLang"`  // target code; empty = system default
	UILang         string     `json:"uiLang"`         // arayüz dili; boş = sistem dili
	HomeView       string     `json:"homeView"`       // "sakin" (varsayılan) ya da "tam"
	CalmHeadlines  bool       `json:"calmHeadlines"`  // RC97: abartılı başlık yerine olayın en sakin başlığı
	CrisisMode     string     `json:"crisisMode"`     // RC97: "" = otomatik, "off" = kapalı
	FactCheckOff   bool       `json:"factCheckOff"`   // RC97: doğrulama sitelerini hiç indirme
	EasyRead       bool       `json:"easyRead"`       // RC97: kolay okuma görünümü
	Weather        bool       `json:"weather"`        // show the weather panel
	WeatherPlace   string     `json:"weatherPlace"`   // manual city; empty = auto (IP)
	Interests      []string   `json:"interests"`      // keywords for the "SENİN İÇİN" section
	HideWords      []string   `json:"hideWords"`      // negative filter: hide matching headlines
	Notify         bool       `json:"notify"`         // desktop notifications for important new events
	NotifyMinScore int        `json:"notifyMinScore"` // min importance score to notify (non-breaking)
	WidgetTopmost  bool       `json:"widgetTopmost"`  // remembered "üstte kal" choice
	WidgetCats     []string   `json:"widgetCats"`     // categories shown in widget; empty = all
	WidgetImpOnly  bool       `json:"widgetImpOnly"`  // widget shows only breaking/important
	WidgetSlideSec int        `json:"widgetSlideSec"` // seconds each headline stays before sliding
	WidgetOpacity  int        `json:"widgetOpacity"`  // widget surface opacity 15–100 (%)
	WidgetWidth    int        `json:"widgetWidth"`    // widget width in px
	WidgetHeight   int        `json:"widgetHeight"`   // widget height in px
	FontScale      int        `json:"fontScale"`      // accessibility scale 80–160 (%)
	LineHeight     float64    `json:"lineHeight"`     // reading line height
	ReaderWidth    int        `json:"readerWidth"`    // reader text measure in px
	DyslexiaFont   bool       `json:"dyslexiaFont"`
	MarketWatch    []string   `json:"marketWatch"`            // BIST/FX/commodity symbols, e.g. THYAO, USDTRY=X, GC=F
	SportsWatch    []string   `json:"sportsWatch"`            // sports leagues: futbol, premier, ucl, nba, euroleague, nfl
	SportsTeams    []string   `json:"sportsTeams"`            // favorite teams, matched against returned team names
	NewsCountry    string     `json:"newsCountry"`            // ISO-3166 country code; empty = all world
	CountryScope   string     `json:"countryScope"`           // strict or related
	LocalCountry   string     `json:"localCountry,omitempty"` // country the "📍" local category was built for
	SchemaV        int        `json:"schemaV"`                // config migration version

	// v11 (RC81)
	WidgetMode   string `json:"widgetMode"`   // opak / cam (acrylic) / bulanik (blur) / saydam (clear)
	WidgetTint   int    `json:"widgetTint"`   // backdrop tint strength 0–100 (%) for cam/bulanık
	WidgetSingle bool   `json:"widgetSingle"` // one-line ticker (no separate breaking row)
	Density      string `json:"density"`      // rahat / siki (list density)
	NetMeter     bool   `json:"netMeter"`     // show network rate + 5-min data in the top bar
	QuietHours   bool   `json:"quietHours"`   // no desktop toasts between QuietFrom–QuietTo
	QuietFrom    string `json:"quietFrom"`    // "23:00"
	QuietTo      string `json:"quietTo"`      // "07:00"
	RightTab     string `json:"rightTab"`     // remembered right-panel tab

	// v12 (RC83): veri modu
	DataMode      string `json:"dataMode"`      // tam / sinirli
	DataLimitKBps int    `json:"dataLimitKBps"` // sınırlı modda en yüksek indirme hızı (KB/s)
	DataQuotaMB   int    `json:"dataQuotaMB"`   // aylık kota (MB); 0 = yok

	// Canlı TV: favori kanallar ve son izlenen kanal.
	LiveFavs   []string          `json:"liveFavs"`
	LiveLast   string            `json:"liveLast"`
	LiveHidden []string          `json:"liveHidden"` // kullanıcının listeden kaldırdığı kanallar
	LiveCustom []json.RawMessage `json:"liveCustom"` // kullanıcının elle eklediği kanallar
	LiveCheck  string            `json:"liveCheck"`  // kanal denetimi: ask / auto / never

	mu   sync.RWMutex `json:"-"`
	path string       `json:"-"`
}

const currentSchema = 11

func defaultConfig() *Config {
	cats := []Category{
		{Name: "⚡ Küresel Uyarılar", Sources: []Source{
			{Name: "GDACS Afet Uyarıları", URL: "https://www.gdacs.org/xml/rss.xml"},
			{Name: "USGS Önemli Depremler", URL: "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/significant_week.atom"},
		}},
		{Name: "Teknoloji", Sources: []Source{
			{Name: "The Verge", URL: "https://www.theverge.com/rss/index.xml"},
			{Name: "Ars Technica", URL: "https://feeds.arstechnica.com/arstechnica/index"},
			{Name: "Hacker News", URL: "https://hnrss.org/frontpage"},
		}},
		{Name: "Bilim", Sources: []Source{
			{Name: "NASA", URL: "https://www.nasa.gov/rss/dyn/breaking_news.rss"},
			{Name: "ScienceDaily", URL: "https://www.sciencedaily.com/rss/all.xml"},
		}},
	}
	cats = append(cats, regionCategories()...)
	cats = mergeBundledSources(cats, expandedSourceCategories())
	return &Config{
		MaxPerCategory: 5,
		OpenInApp:      true,
		RefreshMinutes: 20,
		Glass:          true,
		GlassOpacity:   72,
		Premium:        true,
		Theme:          "premium",
		Translate:      true,
		Weather:        true,
		WidgetSlideSec: 3,
		WidgetOpacity:  78,
		WidgetWidth:    460,
		WidgetHeight:   168,
		FontScale:      100,
		LineHeight:     1.75,
		ReaderWidth:    760,
		MarketWatch:    []string{"^XU100", "^XU030", "THYAO.IS", "ASELS.IS", "GARAN.IS", "BIMAS.IS", "USDTRY=X", "EURTRY=X", "GC=F"},
		SportsWatch:    []string{"futbol", "ucl", "premier", "nba"},
		SportsTeams:    []string{"Galatasaray", "Fenerbahçe", "Beşiktaş"},
		Notify:         true,
		NotifyMinScore: 75,
		NewsCountry:    "",
		CountryScope:   "related",
		SchemaV:        currentSchema,
		WidgetMode:     "cam",
		WidgetTint:     45,
		WidgetSingle:   true,
		Density:        "rahat",
		NetMeter:       true,
		QuietFrom:      "23:00",
		QuietTo:        "07:00",
		RightTab:       "ozet",
		Categories:     cats,
	}
}

// regionCategories are the geographic feeds (Türkiye + continents), backed mostly
// by BBC's reliable regional RSS plus one extra source each.
func regionCategories() []Category {
	return []Category{
		{Name: "Türkiye", Sources: []Source{
			{Name: "BBC Türkçe", URL: "https://feeds.bbci.co.uk/turkce/rss.xml"},
			{Name: "NTV Son Dakika", URL: "https://www.ntv.com.tr/son-dakika.rss"},
			{Name: "T24", URL: "https://t24.com.tr/rss"},
			{Name: "Cumhuriyet Son Dakika", URL: "https://www.cumhuriyet.com.tr/rss/son_dakika.xml"},
		}},
		{Name: "Avrupa", Sources: []Source{
			{Name: "BBC Europe", URL: "https://feeds.bbci.co.uk/news/world/europe/rss.xml"},
			{Name: "Euronews", URL: "https://www.euronews.com/rss?level=theme&name=news"},
			{Name: "France 24 Avrupa", URL: "https://www.france24.com/en/europe/rss"},
			{Name: "Guardian Avrupa", URL: "https://www.theguardian.com/world/europe-news/rss"},
			{Name: "Politico Europe", URL: "https://www.politico.eu/feed/"},
			{Name: "Le Monde (FR)", URL: "https://www.lemonde.fr/rss/une.xml"},
			{Name: "Der Spiegel (DE)", URL: "https://www.spiegel.de/schlagzeilen/tops/index.rss"},
			{Name: "El País (ES)", URL: "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/portada"},
			{Name: "RFI Europe (FR)", URL: "https://www.rfi.fr/en/europe/rss"},
			{Name: "SVT Nyheter (SE)", URL: "https://www.svt.se/nyheter/rss.xml"},
			{Name: "Euractiv", URL: "https://www.euractiv.com/feed/"},
			{Name: "Irish Times", URL: "https://www.irishtimes.com/cmlink/news-1.1910142"},
		}},
		{Name: "Asya", Sources: []Source{
			{Name: "BBC Asia", URL: "https://feeds.bbci.co.uk/news/world/asia/rss.xml"},
			{Name: "NHK World", URL: "https://www3.nhk.or.jp/nhkworld/en/news/all.xml"},
			{Name: "Guardian Asya", URL: "https://www.theguardian.com/world/asia/rss"},
			{Name: "The Hindu Dünya", URL: "https://www.thehindu.com/news/international/feeder/default.rss"},
			{Name: "Channel NewsAsia", URL: "https://www.channelnewsasia.com/rssfeeds/8395986"},
			{Name: "NHK (日本語)", URL: "https://www.nhk.or.jp/rss/news/cat0.xml"},
			{Name: "Yonhap (한국어)", URL: "https://www.yna.co.kr/rss/news.xml"},
			{Name: "CGTN", URL: "https://www.cgtn.com/subscribe/rss/section/world.xml"},
			{Name: "China Daily", URL: "https://www.chinadaily.com.cn/rss/world_rss.xml"},
			{Name: "The Japan Times", URL: "https://www.japantimes.co.jp/feed/"},
			{Name: "The Hindu", URL: "https://www.thehindu.com/news/national/feeder/default.rss"},
			{Name: "Indian Express", URL: "https://indianexpress.com/feed/"},
			{Name: "Focus Taiwan", URL: "https://focustaiwan.tw/rss"},
			{Name: "Rappler (PH)", URL: "https://www.rappler.com/feed/"},
			{Name: "Jakarta Post", URL: "https://www.thejakartapost.com/rss"},
		}},
		{Name: "Ortadoğu", Sources: []Source{
			{Name: "BBC Middle East", URL: "https://feeds.bbci.co.uk/news/world/middle_east/rss.xml"},
			{Name: "Al Jazeera", URL: "https://www.aljazeera.com/xml/rss/all.xml"},
			{Name: "Guardian Ortadoğu", URL: "https://www.theguardian.com/world/middleeast/rss"},
			{Name: "France 24 Ortadoğu", URL: "https://www.france24.com/en/middle-east/rss"},
			{Name: "Middle East Eye", URL: "https://www.middleeasteye.net/rss"},
			{Name: "Jerusalem Post", URL: "https://www.jpost.com/rss/rssfeedsmiddleeastnews.aspx"},
			{Name: "BBC Arapça (العربية)", URL: "https://feeds.bbci.co.uk/arabic/rss.xml"},
			{Name: "Al Arabiya (العربية)", URL: "https://www.alarabiya.net/.mrss/ar.xml"},
		}},
		{Name: "Afrika", Sources: []Source{
			{Name: "BBC Africa", URL: "https://feeds.bbci.co.uk/news/world/africa/rss.xml"},
			{Name: "AllAfrica", URL: "https://allafrica.com/tools/headlines/rdf/latest/headlines.rdf"},
			{Name: "Guardian Afrika", URL: "https://www.theguardian.com/world/africa/rss"},
			{Name: "France 24 Afrika", URL: "https://www.france24.com/en/africa/rss"},
			{Name: "Africanews", URL: "https://www.africanews.com/feed/rss"},
			{Name: "RFI Afrique (FR)", URL: "https://www.rfi.fr/fr/afrique/rss"},
			{Name: "Jeune Afrique (FR)", URL: "https://www.jeuneafrique.com/feed/"},
			{Name: "Premium Times (NG)", URL: "https://www.premiumtimesng.com/feed"},
			{Name: "Daily Maverick (ZA)", URL: "https://www.dailymaverick.co.za/feed/"},
			{Name: "News24 (ZA)", URL: "https://feeds.24.com/articles/news24/topstories/rss"},
			{Name: "The EastAfrican", URL: "https://www.theeastafrican.co.ke/tea/rss"},
			{Name: "Mail & Guardian (ZA)", URL: "https://mg.co.za/feed/"},
			{Name: "The Punch (NG)", URL: "https://punchng.com/feed/"},
		}},
		{Name: "Küresel / Uluslararası", Sources: []Source{
			{Name: "UN News", URL: "https://news.un.org/feed/subscribe/en/news/all/rss.xml"},
			{Name: "UN News Türkçe", URL: "https://news.un.org/feed/subscribe/tr/news/all/rss.xml"},
			{Name: "WHO News", URL: "https://www.who.int/feeds/entity/mediacentre/news/en/rss.xml"},
			{Name: "Euronews", URL: "https://www.euronews.com/rss?level=theme&name=news"},
			{Name: "France 24 World", URL: "https://www.france24.com/en/rss"},
			{Name: "RFI World", URL: "https://www.rfi.fr/en/rss"},
			{Name: "Al Jazeera", URL: "https://www.aljazeera.com/xml/rss/all.xml"},
		}},
		{Name: "Okyanusya", Sources: []Source{
			{Name: "ABC News Australia", URL: "https://www.abc.net.au/news/feed/51120/rss.xml"},
			{Name: "SBS News Australia", URL: "https://www.sbs.com.au/news/feed"},
			{Name: "RNZ", URL: "https://www.rnz.co.nz/rss/national.xml"},
			{Name: "The Sydney Morning Herald", URL: "https://www.smh.com.au/rss/feed.xml"},
			{Name: "The Age", URL: "https://www.theage.com.au/rss/feed.xml"},
			{Name: "Stuff NZ", URL: "https://www.stuff.co.nz/rss"},
		}},
		{Name: "Amerika", Sources: []Source{
			{Name: "BBC US & Canada", URL: "https://feeds.bbci.co.uk/news/world/us_and_canada/rss.xml"},
			{Name: "BBC Latin America", URL: "https://feeds.bbci.co.uk/news/world/latin_america/rss.xml"},
			{Name: "Guardian ABD", URL: "https://www.theguardian.com/us-news/rss"},
			{Name: "NPR", URL: "https://feeds.npr.org/1001/rss.xml"},
			{Name: "France 24 Amerika", URL: "https://www.france24.com/en/americas/rss"},
			{Name: "CBC (Kanada)", URL: "https://www.cbc.ca/webfeed/rss/rss-topstories"},
			{Name: "Clarín (ES)", URL: "https://www.clarin.com/rss/lo-ultimo/"},
			{Name: "G1 Brasil (PT)", URL: "https://g1.globo.com/rss/g1/"},
			{Name: "Associated Press", URL: "https://feeds.apnews.com/rss/apf-topnews"},
			{Name: "PBS NewsHour", URL: "https://www.pbs.org/newshour/feeds/rss/headlines"},
			{Name: "El Universal México", URL: "https://www.eluniversal.com.mx/rss.xml"},
			{Name: "Infobae", URL: "https://www.infobae.com/feeds/rss/"},
			{Name: "Folha de S.Paulo", URL: "https://feeds.folha.uol.com.br/emcimadahora/rss091.xml"},
		}},
	}
}

// expandedSourceCategories adds the large, multilingual source pack.
// These are bundled feed records; official publications remain a separate layer.
func expandedSourceCategories() []Category {
	return []Category{
		{Name: "Türkiye", Sources: []Source{
			{Name: "Anadolu Ajansı", URL: "https://www.aa.com.tr/tr/rss/default?cat=guncel"},
			{Name: "TRT Haber", URL: "https://www.trthaber.com/sondakika_articles.rss"},
			{Name: "Hürriyet Gündem", URL: "https://www.hurriyet.com.tr/rss/gundem"},
			{Name: "Milliyet Son Dakika", URL: "https://www.milliyet.com.tr/rss/rssnew/sondakika.xml"},
			{Name: "Habertürk", URL: "https://www.haberturk.com/rss/manset.xml"},
			{Name: "Yeni Şafak", URL: "https://www.yenisafak.com/Rss"},
			{Name: "Karar", URL: "https://www.karar.com/rss"},
			{Name: "Diken", URL: "https://www.diken.com.tr/feed/"},
			{Name: "BirGün", URL: "https://www.birgun.net/rss"},
			{Name: "Independent Türkçe", URL: "https://www.indyturk.com/rss.xml"},
			{Name: "T24 Gündem", URL: "https://t24.com.tr/rss/haberler"},
			{Name: "VOA Türkçe", URL: "https://www.voaturkce.com/api/z-pqpe-qpp"},
		}},
		{Name: "Avrupa", Sources: []Source{
			{Name: "ZEIT ONLINE", URL: "https://newsfeed.zeit.de/index"},
			{Name: "FAZ", URL: "https://www.faz.net/rss/aktuell/"},
			{Name: "tagesschau", URL: "https://www.tagesschau.de/xml/rss2"},
			{Name: "FOCUS Online", URL: "https://rss.focus.de/fol/XML/rss_folnews.xml"},
			{Name: "ANSA", URL: "https://www.ansa.it/sito/notizie/topnews/topnews_rss.xml"},
			{Name: "Corriere della Sera", URL: "https://www.corriere.it/rss/homepage.xml"},
			{Name: "la Repubblica", URL: "https://www.repubblica.it/rss/homepage/rss2.0.xml"},
			{Name: "NOS Nieuws", URL: "https://feeds.nos.nl/nosnieuwsalgemeen"},
			{Name: "NRC", URL: "https://www.nrc.nl/rss/"},
			{Name: "Yle News", URL: "https://yle.fi/rss/news"},
			{Name: "NRK", URL: "https://www.nrk.no/nyheter/siste.rss"},
			{Name: "Aftenposten", URL: "https://www.aftenposten.no/rss"},
			{Name: "DR Nyheder", URL: "https://www.dr.dk/nyheder/service/feeds/allenyheder"},
			{Name: "Dagens Nyheter", URL: "https://www.dn.se/rss/"},
			{Name: "Svenska Dagbladet", URL: "https://www.svd.se/feed/articles.rss"},
			{Name: "Franceinfo", URL: "https://www.francetvinfo.fr/titres.rss"},
			{Name: "Ouest-France", URL: "https://www.ouest-france.fr/rss-en-continu.xml"},
			{Name: "L’Obs", URL: "https://www.nouvelobs.com/a-la-une/rss.xml"},
			{Name: "ElDiario.es", URL: "https://www.eldiario.es/rss/"},
			{Name: "El Confidencial", URL: "https://rss.elconfidencial.com/espana/"},
			{Name: "Expansion España", URL: "https://e00-expansion.uecdn.es/rss/portada.xml"},
			{Name: "Agencia EFE", URL: "https://www.efe.com/efe/english/4/rss"},
			{Name: "Polskie Radio", URL: "https://www.polskieradio.pl/rss"},
			{Name: "The First News Poland", URL: "https://www.thefirstnews.com/rss"},
			{Name: "Dennik N", URL: "https://dennikn.sk/feed/"},
			{Name: "The Local EU", URL: "https://feeds.thelocal.com/rss"},
			{Name: "EUobserver", URL: "https://euobserver.com/rss"},
			{Name: "The Parliament Magazine", URL: "https://www.theparliamentmagazine.eu/rss"},
		}},
		{Name: "Asya", Sources: []Source{
			{Name: "Times of India", URL: "https://timesofindia.indiatimes.com/rssfeedstopstories.cms"},
			{Name: "The Hindu", URL: "https://www.thehindu.com/feeder/default.rss"},
			{Name: "NDTV", URL: "https://feeds.feedburner.com/ndtvnews-top-stories"},
			{Name: "India Today", URL: "https://www.indiatoday.in/rss/home"},
			{Name: "News18 World", URL: "https://www.news18.com/rss/world.xml"},
			{Name: "Firstpost India", URL: "https://www.firstpost.com/rss/india.xml"},
			{Name: "Business Standard", URL: "https://www.business-standard.com/rss/home_page_top_stories.rss"},
			{Name: "Moneycontrol", URL: "http://www.moneycontrol.com/rss/latestnews.xml"},
			{Name: "Economic Times", URL: "https://economictimes.indiatimes.com/rssfeedsdefault.cms"},
			{Name: "Financial Express India", URL: "https://www.financialexpress.com/feed/"},
			{Name: "ThePrint", URL: "https://theprint.in/feed/"},
			{Name: "Deccan Chronicle", URL: "https://www.deccanchronicle.com/rss_feed/"},
			{Name: "Scroll.in", URL: "http://feeds.feedburner.com/ScrollinArticles.rss"},
			{Name: "Loksatta", URL: "https://www.loksatta.com/desh-videsh/feed/"},
			{Name: "Amar Ujala", URL: "https://www.amarujala.com/rss/breaking-news.xml"},
			{Name: "Navbharat Times", URL: "https://navbharattimes.indiatimes.com/rssfeedsdefault.cms"},
			{Name: "Jansatta", URL: "https://www.jansatta.com/feed/"},
			{Name: "Live Hindustan", URL: "https://feed.livehindustan.com/rss/3127"},
			{Name: "Japan Times", URL: "https://www.japantimes.co.jp/feed/topstories/"},
			{Name: "Japan Today", URL: "https://japantoday.com/feed"},
			{Name: "Kyodo News", URL: "https://english.kyodonews.net/rss/all.xml"},
			{Name: "Asahi Shimbun", URL: "http://rss.asahi.com/rss/asahi/newsheadlines.rdf"},
			{Name: "Livedoor News", URL: "https://news.livedoor.com/topics/rss/top.xml"},
			{Name: "News On Japan", URL: "http://www.newsonjapan.com/rss/top.xml"},
			{Name: "Korea Times", URL: "https://www.koreatimes.co.kr/www/rss/nation.xml"},
			{Name: "Korea Herald", URL: "https://www.koreaherald.com/common/rss_xml.php?ct=102"},
			{Name: "CNA Asia", URL: "https://www.channelnewsasia.com/api/v1/rss-outbound-feed?_format=xml&category=6511"},
			{Name: "CNA World", URL: "https://www.channelnewsasia.com/api/v1/rss-outbound-feed?_format=xml&category=6311"},
			{Name: "CNA Business", URL: "https://www.channelnewsasia.com/api/v1/rss-outbound-feed?_format=xml&category=6936"},
			{Name: "CNA Singapore", URL: "https://www.channelnewsasia.com/api/v1/rss-outbound-feed?_format=xml&category=10416"},
			{Name: "Republika Online", URL: "https://www.republika.co.id/rss/"},
			{Name: "Tribunnews", URL: "https://www.tribunnews.com/rss"},
			{Name: "Merdeka.com", URL: "https://www.merdeka.com/feed/"},
			{Name: "Suara.com", URL: "https://www.suara.com/rss"},
			{Name: "Inquirer Philippines", URL: "https://www.inquirer.net/fullfeed"},
			{Name: "Philippine Information Agency", URL: "https://pia.gov.ph/news/feed.rss"},
			{Name: "Manila Times", URL: "https://www.manilatimes.net/rssfeed/"},
			{Name: "Philstar", URL: "https://www.philstar.com/rss/headlines"},
			{Name: "BusinessWorld Philippines", URL: "https://www.bworldonline.com/feed/"},
			{Name: "GMA News", URL: "https://data.gmanews.tv/gno/rss/news/feed.xml"},
			{Name: "Philippine News Agency", URL: "https://www.pna.gov.ph/latest.rss"},
			{Name: "Express Tribune Pakistan", URL: "https://tribune.com.pk/feed/home"},
			{Name: "The Nation Pakistan", URL: "https://nation.com.pk/rss/top-stories"},
			{Name: "The News International", URL: "https://www.thenews.com.pk/rss/1/1"},
			{Name: "UrduPoint", URL: "https://www.urdupoint.com/rss/urdupoint.rss"},
			{Name: "Express Urdu", URL: "https://www.express.pk/feed/"},
			{Name: "Hong Kong Free Press", URL: "https://www.hongkongfp.com/feed/"},
			{Name: "The Standard Hong Kong", URL: "https://www.thestandard.com.hk/newsfeed/latest/news.xml"},
			{Name: "HK Economic Journal", URL: "https://www.hket.com/rss/hongkong"},
			{Name: "The Daily Star Bangladesh", URL: "https://www.thedailystar.net/frontpage/rss.xml"},
			{Name: "bdnews24", URL: "https://bdnews24.com/?widgetName=rssfeed&widgetId=1150&getXmlFeed=true"},
			{Name: "Prothom Alo", URL: "https://www.prothomalo.com/feed/"},
			{Name: "The Kathmandu Post", URL: "https://kathmandupost.com/rss"},
			{Name: "Dawn Pakistan", URL: "https://www.dawn.com/feeds/home"},
			{Name: "Taipei Times", URL: "https://www.taipeitimes.com/xml/index.rss"},
			{Name: "Bangkok Post", URL: "https://www.bangkokpost.com/rss/data/topstories.xml"},
			{Name: "The Straits Times", URL: "https://www.straitstimes.com/news/asia/rss.xml"},
		}},
		{Name: "Ortadoğu", Sources: []Source{
			{Name: "Asharq Al-Awsat", URL: "https://aawsat.com/feed"},
			{Name: "Arab News", URL: "https://www.arabnews.com/rss.xml"},
			{Name: "Middle East Monitor", URL: "https://www.middleeastmonitor.com/feed/"},
			{Name: "The National UAE", URL: "https://www.thenationalnews.com/arc/outboundfeeds/rss/"},
			{Name: "Gulf News", URL: "https://gulfnews.com/rss"},
			{Name: "Saudi Gazette", URL: "https://saudigazette.com.sa/rss"},
			{Name: "Tehran Times", URL: "https://www.tehrantimes.com/rss"},
			{Name: "Press TV", URL: "https://www.presstv.ir/rss"},
			{Name: "Al Mayadeen", URL: "https://english.almayadeen.net/rss"},
			{Name: "Daily Sabah", URL: "https://www.dailysabah.com/rss"},
			{Name: "Jerusalem Post World", URL: "https://www.jpost.com/rss/rssfeedsworld.aspx"},
			{Name: "Times of Israel", URL: "https://www.timesofisrael.com/feed/"},
			{Name: "Jordan Times", URL: "https://jordantimes.com/rss.xml"},
		}},
		{Name: "Afrika", Sources: []Source{
			{Name: "The Guardian Nigeria", URL: "https://guardian.ng/feed/"},
			{Name: "Daily Post Nigeria", URL: "https://dailypost.ng/feed"},
			{Name: "The Nation Nigeria", URL: "https://thenationonlineng.net/feed/"},
			{Name: "Legit.ng", URL: "https://www.legit.ng/rss/all.rss"},
			{Name: "Information Nigeria", URL: "https://www.informationng.com/feed"},
			{Name: "Tribune Online Nigeria", URL: "http://tribuneonlineng.com/feed/"},
			{Name: "IOL South Africa", URL: "https://www.iol.co.za/rss"},
			{Name: "BusinessTech South Africa", URL: "https://businesstech.co.za/news/feed/"},
			{Name: "TimesLIVE South Africa", URL: "https://www.timeslive.co.za/rss/"},
			{Name: "Daily Nation Kenya", URL: "https://nation.africa/kenya/rss"},
			{Name: "The Star Kenya", URL: "https://www.the-star.co.ke/rss"},
			{Name: "Capital Ethiopia", URL: "https://capitalethiopia.com/feed/"},
			{Name: "Morocco World News", URL: "https://www.moroccoworldnews.com/feed/"},
			{Name: "Egypt Today", URL: "https://www.egypttoday.com/RSS"},
			{Name: "Ahram Online", URL: "https://english.ahram.org.eg/rss.aspx"},
		}},
		{Name: "Amerika", Sources: []Source{
			{Name: "NYT Top Stories", URL: "https://rss.nytimes.com/services/xml/rss/nyt/HomePage.xml"},
			{Name: "CNN International", URL: "http://rss.cnn.com/rss/edition.rss"},
			{Name: "HuffPost World", URL: "https://www.huffpost.com/section/world-news/feed"},
			{Name: "Washington Post World", URL: "http://feeds.washingtonpost.com/rss/world"},
			{Name: "Los Angeles Times World", URL: "https://www.latimes.com/world-nation/rss2.0.xml"},
			{Name: "FOX News", URL: "http://feeds.foxnews.com/foxnews/latest"},
			{Name: "Yahoo News", URL: "https://news.yahoo.com/rss/mostviewed"},
			{Name: "CNBC", URL: "https://www.cnbc.com/id/100003114/device/rss/rss.html"},
			{Name: "Politico Playbook", URL: "https://rss.politico.com/playbook.xml"},
			{Name: "CTV News Canada", URL: "https://www.ctvnews.ca/rss/ctvnews-ca-top-stories-public-rss-1.822009"},
			{Name: "Global News Canada", URL: "https://globalnews.ca/feed/"},
			{Name: "Financial Post", URL: "https://business.financialpost.com/feed/"},
			{Name: "National Post", URL: "https://nationalpost.com/feed/"},
			{Name: "La Presse Canada", URL: "https://www.lapresse.ca/actualites/rss"},
			{Name: "Toronto Star", URL: "https://www.thestar.com/content/thestar/feed.RSSManagerServlet.articles.topstories.rss"},
			{Name: "Toronto Sun", URL: "https://torontosun.com/category/news/feed"},
			{Name: "Excelsior Mexico", URL: "https://www.excelsior.com.mx/rss.xml"},
			{Name: "Vanguardia Mexico", URL: "https://vanguardia.com.mx/rss.xml"},
			{Name: "El Financiero Mexico", URL: "https://www.elfinanciero.com.mx/arc/outboundfeeds/rss/?outputType=xml"},
			{Name: "Mexico News Daily", URL: "https://mexiconewsdaily.com/feed/"},
			{Name: "El Informador Mexico", URL: "https://www.informador.mx/rss/ultimas-noticias.xml"},
			{Name: "24 Horas Mexico", URL: "https://www.24-horas.mx/feed/"},
			{Name: "La Nación Argentina", URL: "https://www.lanacion.com.ar/arc/outboundfeeds/rss/"},
			{Name: "R7 Brasil", URL: "https://noticias.r7.com/feed.xml"},
			{Name: "UOL Brasil", URL: "http://rss.home.uol.com.br/index.xml"},
			{Name: "Portal EBC Brasil", URL: "http://www.ebc.com.br/rss/feed.xml"},
			{Name: "The Rio Times", URL: "https://riotimesonline.com/feed/"},
			{Name: "Jornal de Brasília", URL: "https://jornaldebrasilia.com.br/feed/"},
			{Name: "El Comercio Peru", URL: "https://elcomercio.pe/feed/"},
			{Name: "La Tercera Chile", URL: "https://www.latercera.com/feed/"},
			{Name: "BioBioChile", URL: "https://www.biobiochile.cl/feed/"},
			{Name: "El Tiempo Colombia", URL: "https://www.eltiempo.com/rss/colombia.xml"},
			{Name: "El Espectador Colombia", URL: "https://www.elespectador.com/rss/"},
			{Name: "MercoPress", URL: "https://en.mercopress.com/rss/"},
			{Name: "Telesur English", URL: "https://www.telesurenglish.net/rss/"},
		}},
		{Name: "Okyanusya", Sources: []Source{
			{Name: "ABC News Australia", URL: "https://www.abc.net.au/news/feed/1948/rss.xml"},
			{Name: "Daily Telegraph Australia", URL: "https://www.dailytelegraph.com.au/news/breaking-news/rss"},
			{Name: "Herald Sun Australia", URL: "https://www.heraldsun.com.au/news/breaking-news/rss"},
			{Name: "Canberra Times", URL: "https://www.canberratimes.com.au/rss.xml"},
			{Name: "Brisbane Times", URL: "https://www.brisbanetimes.com.au/rss/feed.xml"},
			{Name: "PerthNow", URL: "https://www.perthnow.com.au/news/feed"},
			{Name: "Business News Australia", URL: "https://www.businessnews.com.au/rssfeed/latest.rss"},
			{Name: "New Zealand Herald", URL: "https://www.nzherald.co.nz/arc/outboundfeeds/rss/"},
		}},
		{Name: "Küresel / Uluslararası", Sources: []Source{
			{Name: "BBC World", URL: "https://feeds.bbci.co.uk/news/world/rss.xml"},
			{Name: "NATO News", URL: "https://www.nato.int/cps/en/natohq/news.rss"},
			{Name: "NIST News", URL: "https://www.nist.gov/news-events/news/rss.xml"},
			{Name: "ESA News", URL: "https://www.esa.int/rssfeed/Our_Activities"},
			{Name: "UNICEF News", URL: "https://www.unicef.org/press-releases/rss.xml"},
			{Name: "UNESCO News", URL: "https://www.unesco.org/en/rss.xml"},
			{Name: "IMF News", URL: "https://www.imf.org/en/News/RSS"},
			{Name: "World Bank News", URL: "https://www.worldbank.org/en/news/all?format=rss"},
			{Name: "OECD News", URL: "https://www.oecd.org/newsroom/rss.xml"},
			{Name: "WTO News", URL: "https://www.wto.org/english/news_e/news_e.rss"},
			{Name: "IAEA News", URL: "https://www.iaea.org/feeds/news"},
			{Name: "ICRC News", URL: "https://www.icrc.org/en/rss"},
			{Name: "INTERPOL News", URL: "https://www.interpol.int/rss/news"},
			{Name: "Europol News", URL: "https://www.europol.europa.eu/rss.xml"},
			{Name: "UNHCR News", URL: "https://www.unhcr.org/rss/news"},
		}},
		{Name: "Teknoloji", Sources: []Source{
			{Name: "TechCrunch", URL: "https://techcrunch.com/feed/"},
			{Name: "Wired", URL: "https://www.wired.com/feed/rss"},
			{Name: "Engadget", URL: "https://www.engadget.com/rss.xml"},
			{Name: "TechRadar", URL: "https://www.techradar.com/rss"},
			{Name: "ZDNET", URL: "https://www.zdnet.com/news/rss.xml"},
			{Name: "The Register", URL: "https://www.theregister.com/headlines.atom"},
			{Name: "VentureBeat", URL: "https://venturebeat.com/feed/"},
			{Name: "MIT Technology Review", URL: "https://www.technologyreview.com/feed/"},
			{Name: "IEEE Spectrum", URL: "https://spectrum.ieee.org/feeds/feed.rss"},
			{Name: "NIST Cybersecurity", URL: "https://www.nist.gov/blogs/cybersecurity-insights/rss.xml"},
			{Name: "Krebs on Security", URL: "https://krebsonsecurity.com/feed/"},
			{Name: "BleepingComputer", URL: "https://www.bleepingcomputer.com/feed/"},
			{Name: "Dark Reading", URL: "https://www.darkreading.com/rss.xml"},
		}},
		{Name: "Bilim", Sources: []Source{
			{Name: "Nature News", URL: "https://www.nature.com/nature.rss"},
			{Name: "New Scientist", URL: "https://www.newscientist.com/feed/home/"},
			{Name: "Phys.org", URL: "https://phys.org/rss-feed/"},
			{Name: "Scientific American", URL: "https://www.scientificamerican.com/rss/"},
			{Name: "EurekAlert", URL: "https://www.eurekalert.org/rss.xml"},
			{Name: "NOAA Research News", URL: "https://research.noaa.gov/feed/"},
			{Name: "US DOE Science", URL: "https://science.osti.gov/RSS"},
			{Name: "ESA Human Spaceflight", URL: "https://www.esa.int/rssfeed/Human_Spaceflight"},
			{Name: "ESA Exploration", URL: "https://www.esa.int/rssfeed/Science_Exploration"},
		}},
		{Name: "⚡ Küresel Uyarılar", Sources: []Source{
			{Name: "EMSC Earthquakes", URL: "https://www.emsc-csem.org/service/rss/rss.php"},
			{Name: "NOAA Weather Alerts", URL: "https://www.weather.gov/rss_page.php"},
			{Name: "WHO Emergencies", URL: "https://www.who.int/feeds/entity/emergencies/en/rss.xml"},
			{Name: "WHO Africa Emergencies", URL: "https://www.afro.who.int/rss-feeds"},
		}},
	}
}

func mergeBundledSources(cats []Category, extras []Category) []Category {
	idx := map[string]int{}
	for i, cat := range cats {
		idx[cat.Name] = i
	}
	for _, ec := range extras {
		i, ok := idx[ec.Name]
		if !ok {
			cats = append(cats, ec)
			idx[ec.Name] = len(cats) - 1
			continue
		}
		have := map[string]bool{}
		for _, src := range cats[i].Sources {
			have[src.URL] = true
		}
		for _, src := range ec.Sources {
			if !have[src.URL] {
				cats[i].Sources = append(cats[i].Sources, src)
				have[src.URL] = true
			}
		}
	}
	return cats
}

// migrate brings an older config up to the current schema. v1 adds the regional
// categories (only those the user doesn't already have, by name).
func (c *Config) migrate() bool {
	if c.SchemaV >= currentSchema {
		return false
	}
	idx := map[string]int{}
	for i, cat := range c.Categories {
		idx[cat.Name] = i
	}
	for _, rc := range regionCategories() {
		i, ok := idx[rc.Name]
		if !ok {
			// v1: user doesn't have this regional category yet — add it whole.
			c.Categories = append(c.Categories, rc)
			continue
		}
		// v2: category exists — add any newly-bundled sources it is missing,
		// matched by URL, without touching the user's own additions/ordering.
		have := map[string]bool{}
		for _, s := range c.Categories[i].Sources {
			have[s.URL] = true
		}
		for _, s := range rc.Sources {
			if !have[s.URL] {
				c.Categories[i].Sources = append(c.Categories[i].Sources, s)
			}
		}
	}
	c.Categories = mergeBundledSources(c.Categories, expandedSourceCategories())
	// v5: restore the intended system-language headline translation default.
	// Only once — a user who later switched translation off keeps that choice.
	if c.SchemaV < 5 {
		c.Translate = true
	}
	if c.SportsWatch == nil {
		c.SportsWatch = []string{"futbol", "ucl", "premier", "nba"}
	}
	if c.SportsTeams == nil {
		c.SportsTeams = []string{"Galatasaray", "Fenerbahçe", "Beşiktaş"}
	}
	if c.MarketWatch == nil {
		c.MarketWatch = []string{"^XU100", "^XU030", "THYAO.IS", "ASELS.IS", "GARAN.IS", "BIMAS.IS", "USDTRY=X", "EURTRY=X", "GC=F"}
	}
	// v7: add BIST 100/30 benchmarks without overwriting a user watchlist.
	if c.SchemaV < 7 {
		have := map[string]bool{}
		for _, s := range c.MarketWatch {
			have[s] = true
		}
		if !have["^XU100"] {
			c.MarketWatch = append([]string{"^XU100"}, c.MarketWatch...)
		}
		if !have["^XU030"] {
			c.MarketWatch = append([]string{"^XU030"}, c.MarketWatch...)
		}
	}
	if strings.TrimSpace(c.NewsCountry) != "" {
		c.NewsCountry = strings.ToUpper(strings.TrimSpace(c.NewsCountry))
	}
	if c.CountryScope != "strict" && c.CountryScope != "related" {
		c.CountryScope = "related"
	}
	// v11: widget glass modes, density, network meter, quiet hours.
	if c.SchemaV < 11 {
		if c.WidgetMode == "" {
			c.WidgetMode = "cam"
		}
		if c.WidgetTint == 0 {
			c.WidgetTint = 45
		}
		c.WidgetSingle = true
		if c.Density == "" {
			c.Density = "rahat"
		}
		c.NetMeter = true
		if c.QuietFrom == "" {
			c.QuietFrom = "23:00"
		}
		if c.QuietTo == "" {
			c.QuietTo = "07:00"
		}
		if c.RightTab == "" {
			c.RightTab = "ozet"
		}
	}
	c.normalizeV11()
	c.SchemaV = currentSchema
	return true
}

// normalizeV11 clamps the RC81 settings to valid values.
func (c *Config) normalizeV11() {
	if c.DataMode != dataModeLimited {
		c.DataMode = dataModeFull
	}
	if c.DataLimitKBps < 32 || c.DataLimitKBps > 100000 {
		c.DataLimitKBps = defaultLimitKBs
	}
	if c.DataQuotaMB < 0 {
		c.DataQuotaMB = 0
	}
	switch c.WidgetMode {
	case "opak", "cam", "bulanik", "saydam":
	default:
		c.WidgetMode = "cam"
	}
	if c.WidgetTint < 0 || c.WidgetTint > 100 {
		c.WidgetTint = 45
	}
	if c.Density != "siki" {
		c.Density = "rahat"
	}
	if _, ok := parseClock(c.QuietFrom); !ok {
		c.QuietFrom = "23:00"
	}
	if _, ok := parseClock(c.QuietTo); !ok {
		c.QuietTo = "07:00"
	}
	switch c.RightTab {
	case "ozet", "borsa", "spor", "analiz":
	case "piyasa": // RC82 öncesi birleşik sekme
		c.RightTab = "borsa"
	default:
		c.RightTab = "ozet"
	}
}

// parseClock parses "HH:MM" into minutes after midnight.
func parseClock(s string) (int, bool) {
	var h, m int
	if n, err := fmt.Sscanf(strings.TrimSpace(s), "%d:%d", &h, &m); err != nil || n != 2 || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// inQuietHours reports whether t falls in the [QuietFrom, QuietTo) window
// (which may wrap past midnight).
func (c *Config) inQuietHours(t time.Time) bool {
	c.mu.RLock()
	on, from, to := c.QuietHours, c.QuietFrom, c.QuietTo
	c.mu.RUnlock()
	if !on {
		return false
	}
	f, ok1 := parseClock(from)
	e, ok2 := parseClock(to)
	if !ok1 || !ok2 || f == e {
		return false
	}
	now := t.Hour()*60 + t.Minute()
	if f < e {
		return now >= f && now < e
	}
	return now >= f || now < e
}

// dataDirOverride, when set (Android: the app's private files dir), replaces
// the OS user-config directory for every NexNews data file.
var dataDirOverride string

// appDataDir is the folder holding config, caches, favourites and history:
// %APPDATA%\NexNews on Windows, <filesDir>/NexNews on Android.
func appDataDir() string {
	dir := dataDirOverride
	if dir == "" {
		d, err := os.UserConfigDir()
		if err != nil || d == "" {
			d, _ = os.Getwd()
		}
		dir = d
	}
	return filepath.Join(dir, "NexNews")
}

func configPath() string { return filepath.Join(appDataDir(), "config.json") }

// loadConfig reads the config file, creating it with defaults if absent.
func loadConfig() *Config {
	p := configPath()
	data, err := os.ReadFile(p)
	if err != nil {
		c := defaultConfig()
		c.path = p
		_ = c.save()
		return c
	}
	c := &Config{}
	if err := json.Unmarshal(data, c); err != nil {
		c = defaultConfig()
	}
	c.path = p
	if c.MaxPerCategory <= 0 {
		c.MaxPerCategory = 5
	}
	changed := c.migrate()
	if strings.TrimSpace(c.NewsCountry) != "" {
		norm := strings.ToUpper(strings.TrimSpace(c.NewsCountry))
		if c.NewsCountry != norm {
			c.NewsCountry = norm
			changed = true
		}
	}
	if c.CountryScope != "strict" && c.CountryScope != "related" {
		c.CountryScope = "related"
		changed = true
	}
	if changed {
		_ = c.save()
	}
	return c
}

func (c *Config) save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.path == "" {
		c.path = configPath()
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(c.path, data)
}

// snapshot returns a deep-ish copy safe to hand to the fetcher without holding
// the lock during network I/O.
func (c *Config) snapshot() ([]Category, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cats := make([]Category, len(c.Categories))
	for i, cat := range c.Categories {
		src := make([]Source, len(cat.Sources))
		copy(src, cat.Sources)
		cats[i] = Category{Name: cat.Name, Sources: src}
	}
	return cats, c.MaxPerCategory
}

// replaceFrom overwrites the editable fields from a JSON payload sent by the UI
// and persists the result.
func (c *Config) replaceFrom(raw []byte) error {
	tmp := &Config{}
	if err := json.Unmarshal(raw, tmp); err != nil {
		return err
	}
	c.mu.Lock()
	c.Categories = tmp.Categories
	if tmp.MaxPerCategory > 0 {
		c.MaxPerCategory = tmp.MaxPerCategory
	}
	c.OpenInApp = tmp.OpenInApp
	c.RefreshMinutes = tmp.RefreshMinutes
	c.Glass = tmp.Glass
	c.GlassOpacity = tmp.GlassOpacity
	c.Premium = tmp.Premium
	c.Theme = tmp.Theme
	c.Translate = tmp.Translate
	c.TranslateLang = tmp.TranslateLang
	c.UILang = strings.ToLower(strings.TrimSpace(tmp.UILang))
	c.HomeView = tmp.HomeView
	c.CalmHeadlines = tmp.CalmHeadlines
	c.CrisisMode = tmp.CrisisMode
	c.FactCheckOff = tmp.FactCheckOff
	c.EasyRead = tmp.EasyRead
	c.Weather = tmp.Weather
	c.WeatherPlace = tmp.WeatherPlace
	c.Interests = tmp.Interests
	c.HideWords = tmp.HideWords
	c.Notify = tmp.Notify
	c.NotifyMinScore = tmp.NotifyMinScore
	c.WidgetTopmost = tmp.WidgetTopmost
	c.WidgetCats = tmp.WidgetCats
	c.WidgetImpOnly = tmp.WidgetImpOnly
	c.WidgetSlideSec = tmp.WidgetSlideSec
	c.WidgetOpacity = tmp.WidgetOpacity
	c.WidgetWidth = tmp.WidgetWidth
	c.WidgetHeight = tmp.WidgetHeight
	c.FontScale = tmp.FontScale
	c.LineHeight = tmp.LineHeight
	c.ReaderWidth = tmp.ReaderWidth
	c.DyslexiaFont = tmp.DyslexiaFont
	if tmp.MarketWatch != nil {
		c.MarketWatch = append([]string(nil), tmp.MarketWatch...)
	}
	if tmp.SportsTeams != nil {
		c.SportsTeams = append([]string(nil), tmp.SportsTeams...)
	}
	if tmp.SportsWatch != nil {
		c.SportsWatch = append([]string(nil), tmp.SportsWatch...)
	}
	c.WidgetMode = tmp.WidgetMode
	c.WidgetTint = tmp.WidgetTint
	c.WidgetSingle = tmp.WidgetSingle
	c.Density = tmp.Density
	c.NetMeter = tmp.NetMeter
	c.QuietHours = tmp.QuietHours
	c.QuietFrom = tmp.QuietFrom
	c.QuietTo = tmp.QuietTo
	if tmp.RightTab != "" {
		c.RightTab = tmp.RightTab
	}
	c.normalizeV11()
	// v10: persist geographic personalization settings. Empty NewsCountry is
	// intentional and means "Tüm Dünya". Never replace it with a default.
	c.NewsCountry = strings.ToUpper(strings.TrimSpace(tmp.NewsCountry))
	c.CountryScope = strings.ToLower(strings.TrimSpace(tmp.CountryScope))
	if c.CountryScope != "strict" && c.CountryScope != "related" {
		c.CountryScope = "related"
	}
	if c.WidgetOpacity < 0 || c.WidgetOpacity > 100 {
		c.WidgetOpacity = 78
	}
	if c.WidgetWidth < 300 || c.WidgetWidth > 1000 {
		c.WidgetWidth = 460
	}
	if c.WidgetHeight < 60 || c.WidgetHeight > 500 {
		c.WidgetHeight = 168
	}
	if c.FontScale < 80 || c.FontScale > 160 {
		c.FontScale = 100
	}
	if c.LineHeight < 1.4 || c.LineHeight > 2.4 {
		c.LineHeight = 1.75
	}
	if c.ReaderWidth < 520 || c.ReaderWidth > 980 {
		c.ReaderWidth = 760
	}
	if tmp.SchemaV > c.SchemaV {
		c.SchemaV = tmp.SchemaV
	}
	c.mu.Unlock()
	return c.save()
}

func (c *Config) toJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.Marshal(c)
}
