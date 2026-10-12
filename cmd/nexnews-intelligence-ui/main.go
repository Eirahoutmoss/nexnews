package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"nexnews/intelligence"
)

type summary struct {
	Sources   int `json:"sources"`
	Articles  int `json:"articles"`
	Events    int `json:"events"`
	Claims    int `json:"claims"`
	Evidence  int `json:"evidence"`
	Entities  int `json:"entities"`
	Mentions  int `json:"mentions"`
	Relations int `json:"relations"`
	RepeatedWordingGroups int `json:"repeatedWordingGroups"`
	ExplicitOppositionCases int `json:"explicitOppositionCases"`
	QualityErrors int `json:"qualityErrors"`
	QualityWarnings int `json:"qualityWarnings"`
}

type viewState struct {
	Schema    string                 `json:"schema"`
	SavedAt   time.Time              `json:"savedAt"`
	LastSyncAt time.Time             `json:"lastSyncAt,omitempty"`
	Summary   summary                `json:"summary"`
	Sources   []intelligence.Source  `json:"sources"`
	Articles  []intelligence.Article `json:"articles"`
	Events    []intelligence.Event   `json:"events"`
	Claims    []intelligence.Claim   `json:"claims"`
	Evidence  []intelligence.Evidence `json:"evidence"`
	Entities  []intelligence.Entity `json:"entities"`
	Mentions  []intelligence.EntityMention `json:"mentions"`
	Relations []intelligence.Relation `json:"relations"`
	SourceComparisons intelligence.SourceComparisonReport `json:"sourceComparisons"`
	ClaimAssessment intelligence.ClaimAssessmentReport `json:"claimAssessment"`
	DataQuality intelligence.DataQualityReport `json:"dataQuality"`
}

func main() {
	var dbPath, addr string
	flag.StringVar(&dbPath, "db", "nexnews-intelligence.json", "Intelligence database file")
	flag.StringVar(&addr, "addr", "127.0.0.1:8787", "HTTP listen address (default is local-only)")
	flag.Parse()
	dbPath = strings.TrimSpace(dbPath)
	addr = strings.TrimSpace(addr)
	if dbPath == "" || addr == "" {
		log.Fatal("both -db and -addr must be non-empty")
	}

	repo, err := intelligence.NewJSONRepository(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	handler := newHandler(repo)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen on %s: %v", addr, err)
	}
	log.Printf("NexNews Intelligence UI: http://%s", listener.Addr().String())
	log.Printf("read-only database: %s", dbPath)
	server := &http.Server{
		Handler: handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newHandler(repo *intelligence.JSONRepository) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if !allowGET(w, r) {
			return
		}
		setSecurityHeaders(w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(indexHTML))
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		if !allowGET(w, r) {
			return
		}
		setSecurityHeaders(w)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		state, err := loadState(repo)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Intelligence veritabanı okunamadı."})
			return
		}
		_ = json.NewEncoder(w).Encode(state)
	})
	mux.HandleFunc("/api/event-dossier", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/event-dossier" {
			http.NotFound(w, r)
			return
		}
		if !allowGET(w, r) {
			return
		}
		setSecurityHeaders(w)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		eventID := strings.TrimSpace(r.URL.Query().Get("eventId"))
		if eventID == "" {
			http.Error(w, "eventId is required", http.StatusBadRequest)
			return
		}
		db, err := repo.Load()
		if err != nil {
			http.Error(w, "Intelligence veritabanı okunamadı.", http.StatusInternalServerError)
			return
		}
		dossier, err := intelligence.BuildEventDossier(db, eventID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, "Olay adayı bulunamadı.", http.StatusNotFound)
			} else {
				http.Error(w, "Olay dosyası oluşturulamadı.", http.StatusInternalServerError)
			}
			return
		}
		_ = json.NewEncoder(w).Encode(dossier)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if !allowGET(w, r) {
			return
		}
		setSecurityHeaders(w)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "mode": "read-only"})
	})
	registerResearchRoutes(mux, repo)
	registerLiveResearchRoutes(mux)
	return mux
}

func allowGET(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; frame-ancestors 'none'")
}

func loadState(repo *intelligence.JSONRepository) (viewState, error) {
	db, err := repo.Load()
	if err != nil {
		return viewState{}, err
	}
	out := viewState{
		Schema: db.Schema, SavedAt: db.SavedAt, LastSyncAt: db.LastSyncAt,
		Sources: make([]intelligence.Source, 0, len(db.Sources)),
		Articles: make([]intelligence.Article, 0, len(db.Articles)),
		Events: make([]intelligence.Event, 0, len(db.Events)),
		Claims: make([]intelligence.Claim, 0, len(db.Claims)),
		Evidence: make([]intelligence.Evidence, 0, len(db.Evidence)),
		Entities: make([]intelligence.Entity, 0, len(db.Entities)),
		Mentions: make([]intelligence.EntityMention, 0, len(db.EntityMentions)),
		Relations: make([]intelligence.Relation, 0, len(db.Relations)),
		SourceComparisons: intelligence.BuildSourceComparisonReport(db),
		ClaimAssessment: intelligence.BuildClaimAssessmentReport(db),
		DataQuality: intelligence.BuildDataQualityReport(db),
	}
	for _, v := range db.Sources { out.Sources = append(out.Sources, v) }
	for _, v := range db.Articles { out.Articles = append(out.Articles, v) }
	for _, v := range db.Events { out.Events = append(out.Events, v) }
	for _, v := range db.Claims { out.Claims = append(out.Claims, v) }
	for _, v := range db.Evidence { out.Evidence = append(out.Evidence, v) }
	for _, v := range db.Entities { out.Entities = append(out.Entities, v) }
	for _, v := range db.EntityMentions { out.Mentions = append(out.Mentions, v) }
	for _, v := range db.Relations { out.Relations = append(out.Relations, v) }

	out.Summary = summary{
		Sources: len(out.Sources), Articles: len(out.Articles), Events: len(out.Events),
		Claims: len(out.Claims), Evidence: len(out.Evidence), Entities: len(out.Entities),
		Mentions: len(out.Mentions), Relations: len(out.Relations),
		RepeatedWordingGroups: out.SourceComparisons.RepeatedWordingGroups,
		ExplicitOppositionCases: out.ClaimAssessment.ExplicitOppositionCases,
		QualityErrors: out.DataQuality.ErrorCount,
		QualityWarnings: out.DataQuality.WarningCount,
	}
	return out, nil
}

const indexHTML = `<!doctype html>
<html lang="tr">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>NexNews Intelligence</title>
<style>
:root{color-scheme:dark;--bg:#0a0f18;--panel:#111a27;--panel2:#151f2e;--line:#263447;--text:#eaf1fb;--muted:#91a2b9;--cyan:#78dce8;--gold:#e7c98a;--green:#91d6b2}
*{box-sizing:border-box}body{margin:0;background:radial-gradient(ellipse at 75% 0%,#142338 0,transparent 40%),var(--bg);color:var(--text);font:14px/1.5 system-ui,-apple-system,Segoe UI,sans-serif}
.layout{display:grid;grid-template-columns:235px minmax(0,1fr);min-height:100vh}.side{padding:24px 16px;border-right:1px solid var(--line);background:#090e16}.brand{display:flex;align-items:center;gap:12px;padding:4px 8px 26px}.logo{width:38px;height:38px;border-radius:12px;background:linear-gradient(145deg,#89e0e9,#397b9b);color:#07111a;display:grid;place-items:center;font-weight:900;font-size:18px}.brand strong{display:block;font-size:15px;letter-spacing:.02em}.brand small{color:var(--muted)}.nav{display:grid;gap:6px}.nav button{border:1px solid transparent;background:transparent;color:var(--muted);text-align:left;padding:11px 12px;border-radius:10px;cursor:pointer;font:inherit}.nav button.active,.nav button:hover{background:var(--panel2);border-color:var(--line);color:var(--text)}.nav .symbol{display:inline-block;width:25px;color:var(--cyan)}.side-note{margin:26px 6px 0;padding:12px;border:1px solid var(--line);border-radius:12px;color:var(--muted);font-size:12px}
main{min-width:0;padding:30px clamp(18px,3vw,42px) 50px}.top{display:flex;align-items:flex-start;justify-content:space-between;gap:18px;margin-bottom:28px}.eyebrow{color:var(--cyan);font-size:11px;letter-spacing:.16em;text-transform:uppercase;font-weight:700}.top h1{margin:6px 0 6px;font-size:clamp(25px,3vw,36px);letter-spacing:-.04em}.sub{color:var(--muted);max-width:720px}.status{white-space:nowrap;border:1px solid #3c6957;color:var(--green);padding:7px 10px;border-radius:30px;font-size:12px}.metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin-bottom:22px}.metric,.panel{background:linear-gradient(145deg,rgba(21,31,46,.94),rgba(13,21,32,.94));border:1px solid var(--line);border-radius:16px}.metric{padding:17px}.metric label{display:block;color:var(--muted);font-size:12px}.metric strong{display:block;font-size:27px;margin-top:8px;font-weight:650;letter-spacing:-.04em}.metric small{color:var(--muted)}.toolbar{display:flex;gap:10px;align-items:center;flex-wrap:wrap;margin:0 0 16px}.open-dossier{display:inline-flex;align-items:center;gap:8px;margin-top:12px;padding:9px 12px;border:1px solid #42627b;border-radius:9px;background:#172b3d;color:#dcebf7;cursor:pointer;font:inherit}.open-dossier:hover{background:#203a50}.search{flex:1;min-width:220px;background:#0b131f;border:1px solid var(--line);border-radius:10px;padding:11px 13px;color:var(--text);font:inherit;outline:none}.search:focus{border-color:#5794a8}.refresh{background:#18283a;border:1px solid #35506a;border-radius:10px;padding:10px 13px;color:var(--text);font:inherit;cursor:pointer}.section-head{display:flex;align-items:center;justify-content:space-between;gap:10px;margin:20px 0 12px}.section-head h2{margin:0;font-size:18px}.count{color:var(--muted);font-size:12px}.cards{display:grid;gap:12px}.item{background:var(--panel);border:1px solid var(--line);padding:17px 18px;border-radius:14px;min-width:0}.item h3{font-size:15px;line-height:1.45;margin:0 0 8px}.item p{margin:6px 0;color:#b4c1d3;overflow-wrap:anywhere}.meta{display:flex;align-items:center;gap:8px;flex-wrap:wrap;color:var(--muted);font-size:11px;margin-top:10px}.pill{display:inline-block;border:1px solid #35465b;color:#b1c6da;padding:3px 7px;border-radius:20px;font-size:11px}.pill.green{border-color:#376d59;color:var(--green)}.pill.red{border-color:#804348;color:#ffb0b0}.pill.gold{border-color:#65563c;color:var(--gold)}a{color:var(--cyan);text-decoration:none}a:hover{text-decoration:underline}.empty,.error{padding:30px;text-align:center;border:1px dashed #334156;border-radius:14px;color:var(--muted)}.error{color:#f0a9a9}.foot{margin-top:28px;padding-top:14px;border-top:1px solid var(--line);color:var(--muted);font-size:12px}.warning{border-left:3px solid var(--gold);padding:10px 13px;background:#171a20;color:#cfbea0;border-radius:0 9px 9px 0;margin:0 0 16px}
@media(max-width:900px){.layout{grid-template-columns:1fr}.side{border-right:0;border-bottom:1px solid var(--line);padding:12px 16px}.brand{padding:4px 4px 12px}.nav{grid-template-columns:repeat(3,minmax(0,1fr))}.nav button{padding:8px;font-size:12px}.nav .symbol{width:18px}.side-note{display:none}main{padding-top:20px}.metrics{grid-template-columns:repeat(2,minmax(0,1fr))}}
@media(max-width:500px){.top{display:block}.status{display:inline-block;margin-top:10px}.metrics{gap:8px}.metric{padding:12px}.metric strong{font-size:23px}.nav{grid-template-columns:repeat(2,minmax(0,1fr))}}
</style>
</head>
<body>
<div class="layout">
<aside class="side">
  <div class="brand"><div class="logo">N</div><div><strong>NexNews</strong><small>INTELLIGENCE</small></div></div>
  <nav class="nav" aria-label="Intelligence bölümleri">
    <button class="active" data-view="overview"><span class="symbol">◈</span> Genel Bakış</button>
    <button data-view="events"><span class="symbol">▣</span> Olay Dosyaları</button>
    <button data-view="claims"><span class="symbol">≋</span> Kaynak İfadeleri</button>
    <button data-view="comparison"><span class="symbol">⇄</span> Kaynak Karşılaştırma</button>
    <button data-view="assessment"><span class="symbol">⚖</span> İddia Değerlendirme</button>
    <button data-view="quality"><span class="symbol">⌘</span> Veri Kalitesi</button>
    <button data-view="evidence"><span class="symbol">⌕</span> Kanıt Alıntıları</button>
    <button data-view="entities"><span class="symbol">◎</span> Varlık Anılmaları</button>
    <button data-view="relations"><span class="symbol">⌁</span> İlişkiler</button>
    <button type="button" onclick="window.location.href='/research-workspace'"><span class="symbol">⌖</span> Araştırma Masası</button>
    <button type="button" onclick="window.location.href='/analyst'"><span class="symbol">✳</span> Analizci</button>
  </nav>
  <div class="side-note">Bu ilk arayüz salt okunurdur. Kaynağa bağlanmayan bir iddia veya kanıtsız ilişki oluşturmaz.</div>
</aside>
<main>
  <header class="top">
    <div><div class="eyebrow">Evidence-first analysis</div><h1 id="pageTitle">Genel Bakış</h1><div class="sub" id="pageSub">Kaynakların gerçekten söylediği şeyler, açık sınırları ve izlenebilir kanıtlarıyla.</div></div>
    <div class="status">● Yerel · Salt okunur</div>
  </header>
  <div class="metrics" id="metrics"><div class="metric"><label>Veri yükleniyor</label><strong>…</strong></div></div>
  <div class="warning">Kaynak ifadesi, iddianın doğru olduğunun kanıtı değildir. Bu sürüm ifadeleri “bildirildi” olarak tutar; otomatik ilişki çıkarmaz.</div>
  <div class="toolbar"><input id="search" class="search" type="search" placeholder="Bu görünümde ara…" aria-label="Ara"><button id="reload" class="refresh">↻ Yenile</button></div>
  <div class="section-head"><h2 id="sectionTitle">Son kayıtlar</h2><span id="resultCount" class="count"></span></div>
  <section id="content" class="cards" aria-live="polite"><div class="empty">Intelligence veritabanı okunuyor…</div></section>
  <footer class="foot"><span id="dbMeta">Veritabanı bilgisi bekleniyor.</span><br>Analiz kayıtları kaynak metniyle birlikte saklanır. İlişki haritası ancak kanıt şartlarını sağlayan kayıtlar olduğunda anlamlıdır.</footer>
</main>
</div>
<script>
(function(){
  "use strict";
  var state=null, view="overview", selectedDossier=null;
  var titles={
    overview:["Genel Bakış","Kaynakların gerçekten söylediği şeyler, açık sınırları ve izlenebilir kanıtlarıyla.","Son kayıtlar"],
    events:["Olay Dosyaları","NexNews grupları yalnızca olay adayıdır; Intelligence doğrulaması sayılmaz.","Olay adayları"],
    claims:["Kaynak İfadeleri","Birebir kaynak cümleleri. Tümü bildirildi statüsünde; doğruluk kararı verilmedi.","İfadeler"],
    evidence:["Kanıt Alıntıları","Her alıntı orijinal haberin başlığına veya özetine geri bağlanır.","Kaynak alıntıları"],
    entities:["Varlık Anılmaları","Sınırlı sözlükte birebir eşleşen metin parçaları. Kimlik eşlemesi kesin değildir.","Varlık anılmaları"],
    comparison:["Kaynak Karşılaştırma","Kaynak ifadelerinin birebir tekrarlarını ve kaynak dağılımını gösterir. Aynı ifade, bağımsız teyit anlamına gelmez.","Tekrarlanan ifadeler"],
    assessment:["İddia Değerlendirme","Sınırlı açık karşıt ifade kurallarıyla bulunan inceleme adayları. Hiçbir taraf otomatik olarak doğru sayılmaz.","Olası çelişki sinyalleri"],
    quality:["Veri Kalitesi","Kaynak, olay, iddia, kanıt ve ilişki kayıtlarının yapısal tutarlılığı. Bu rapor haberlerin doğruluğuna karar vermez.","Denetim bulguları"],
    dossier:["Olay Dosyası","Seçilen olay adayına bağlı haberler, kaynak ifadeleri, karşılaştırmalar ve açık sorular.","Olay incelemesi"],
    relations:["İlişkiler","Yalnızca kanıt ve tür doğrulamasından geçmiş ilişkiler bu bölümde gösterilir.","Kayıtlı ilişkiler"]
  };
  function el(tag,cls,text){
    var node=document.createElement(tag);
    if(cls) node.className=cls;
    if(text!==undefined && text!==null) node.textContent=String(text);
    return node;
  }
  function append(parent,tag,cls,text){
    var n=el(tag,cls,text);parent.appendChild(n);return n;
  }
  function safeLink(raw,label){
    if(!raw)return null;
    try{
      var u=new URL(raw,window.location.href);
      if(u.protocol!=="https:" && u.protocol!=="http:")return null;
      var a=el("a","",label||raw);a.href=u.href;a.target="_blank";a.rel="noopener noreferrer";return a;
    }catch(_){return null;}
  }
  function pill(parent,label,cls){append(parent,"span","pill "+(cls||""),label);}
  function date(v){
    if(!v)return "Tarih belirtilmemiş";
    var d=new Date(v);return isNaN(d.getTime())?"Tarih belirtilmemiş":d.toLocaleString("tr-TR");
  }
  function addSource(parent,url,label){
    var a=safeLink(url,label||"Kaynağı aç");
    if(a)parent.appendChild(a);else append(parent,"span","",label||"Kaynak bağlantısı yok");
  }
  function metric(parent,label,value,caption){
    var box=append(parent,"div","metric");
    append(box,"label","",label);append(box,"strong","",value);append(box,"small","",caption||"");
  }
  function renderMetrics(){
    var m=document.getElementById("metrics");m.replaceChildren();
    var s=state.summary;
    metric(m,"Haber belgeleri",s.articles,"Kaynak URL'siyle tekilleştirilmiş");
    metric(m,"Olay adayları",s.events,"Bağımsız doğrulanmış olay sayısı değildir");
    metric(m,"Kaynak ifadeleri",s.claims,"Bildirildi statüsünde");
    metric(m,"Kanıt alıntıları",s.evidence,"Orijinal başlık/özet metnine bağlı");
    metric(m,"Varlıklar",s.entities,"Sözlük eşleşmesi adayları");
    metric(m,"Anılmalar",s.mentions,"Tam metin aralıklarına bağlı");
    metric(m,"Kaynaklar",s.sources,"Yayıncı/kaynak kayıtları");
    metric(m,"İlişkiler",s.relations,"Yalnızca kayıtlı ve doğrulanabilir bağlar");
    metric(m,"Tekrarlanan ifade grupları",s.repeatedWordingGroups,"Birebir metin eşleşmesi; bağımsız teyit değildir");
    metric(m,"Olası çelişki sinyalleri",s.explicitOppositionCases,"İnceleme adayıdır; otomatik hüküm değildir");
    metric(m,"Veri bütünlüğü hataları",s.qualityErrors,"Kopuk veya uyuşmayan kayıtlar");
    metric(m,"Veri kalite uyarıları",s.qualityWarnings,"İnsan incelemesi önerilir");
  }
  function card(title,desc){
    var box=el("article","item");
    append(box,"h3","",title||"Başlıksız kayıt");
    if(desc)append(box,"p","",desc);
    return box;
  }
  function renderEvent(item){
    var c=card(item.title,item.scope||"Kapsam belirtilmemiş");
    pill(c,item.status||"durum bilinmiyor","gold");
    pill(c,(item.articleIds||[]).length+" haber");
    append(c,"p","","İlk gözlem: "+date(item.firstObservedAt)+" · Son gözlem: "+date(item.lastObservedAt));
    if(item.coverageLimitations && item.coverageLimitations.length){
      append(c,"p","","Sınırlar: "+item.coverageLimitations.join(" "));
    }
    var open=append(c,"button","open-dossier","Olay dosyasını aç");
    open.type="button";
    open.addEventListener("click",function(){openDossier(item.id);});
    (item.articleIds||[]).slice(0,5).forEach(function(id){
      var article=state.articles.find(function(a){return a.id===id;});
      if(article){var row=append(c,"p","","");addSource(row,article.canonicalUrl,article.title);}
    });
    return c;
  }
  function renderClaim(item){
    var c=card(item.claimText,item.statusReason);
    pill(c,item.status||"unknown","gold");
    pill(c,"Kaynak ifadesi");
    append(c,"p","","Atıf: "+(item.attributedTo||"Kaynak belirtilmemiş")+" · İlk görülme: "+date(item.firstSeenAt));
    (item.evidenceIds||[]).forEach(function(id){
      var ev=state.evidence.find(function(e){return e.id===id;});
      if(ev){var row=append(c,"p","","Alıntı: “"+ev.quote+"” ");addSource(row,ev.sourceUrl,"Orijinal kaynak");}
    });
    return c;
  }
  function renderAssessment(item){
    var c=card(item.context,item.assessment);
    pill(c,"İnceleme gerekli","gold");
    pill(c,item.assertionType);
    pill(c,item.articleCount+" haber");
    pill(c,item.domainCount+" yayıncı alan adı");
    var sides=[
      {label:"Olumlu biçimde bildirilen ifade",items:item.positiveSources},
      {label:"Olumsuz biçimde bildirilen ifade",items:item.negativeSources}
    ];
    sides.forEach(function(side){
      append(c,"h4","",side.label);
      side.items.forEach(function(o){
        var row=append(c,"p","","");
        addSource(row,o.canonicalUrl,(o.publisherName||o.publisherDomain||"Kaynak")+" · "+(o.publisherDomain||"alan adı yok")+" · "+date(o.publishedAt));
        append(c,"p","","“"+o.quote+"”");
      });
    });
    (item.limitations||[]).forEach(function(note){append(c,"p","","Sınır: "+note);});
    return c;
  }
  function renderQuality(item){
    var severity=item.severity||"info";
    var c=card(item.code||"Kalite bulgusu",item.message||"");
    pill(c,severity,severity==="error"?"red":severity==="warning"?"gold":"green");
    pill(c,item.recordType||"record");
    pill(c,item.recordId||"kimlik yok");
    if(item.sourceUrl){var row=append(c,"p","","");addSource(row,item.sourceUrl,"İlgili kaynağı aç");}
    return c;
  }
  function renderDossierArticle(item){
    var c=card(item.title,item.summary||"Bu haberde özet kayıtlı değil.");
    pill(c,item.contentAvailability||"İçerik düzeyi bilinmiyor","gold");
    pill(c,item.publisherDomain||"Yayıncı bilinmiyor");
    append(c,"p","","Yayın zamanı: "+date(item.publishedAt)+" · Sistemin gözlemi: "+date(item.retrievedAt));
    var row=append(c,"p","","");addSource(row,item.canonicalUrl,"Orijinal haberi aç");
    return c;
  }
  function renderDossierStatement(item){
    var c=card("“"+item.quote+"”",item.publisherName+" · "+item.publisherDomain);
    pill(c,item.claimStatus||"durum bilinmiyor","gold");
    pill(c,item.excerptField==="title"?"Başlık":item.excerptField==="summary"?"Özet":"Metin");
    if(item.wordingStatus==="first_seen_in_dossier"){
      pill(c,"Dosyadaki en erken eşleşme","green");
    }else if(item.wordingStatus==="repeated_verbatim"){
      pill(c,"Birebir tekrar","gold");
    }
    append(c,"p","","Haberde yayın tarihi: "+date(item.publishedAt));
    var row=append(c,"p","","");addSource(row,item.sourceUrl,"İfadenin kaynağına git");
    if(item.wordingStatus==="repeated_verbatim" && item.firstOccurrence){
      var origin=append(c,"p","","İlk eşleşen kayıt: "+(item.firstOccurrence.publisherName||item.firstOccurrence.publisherDomain)+" · "+date(item.firstOccurrence.publishedAt)+" · ");
      addSource(origin,item.firstOccurrence.sourceUrl,item.firstOccurrence.articleTitle||"İlk eşleşen haberi aç");
    }
    return c;
  }
  function renderDossierComparison(item){
    var c=card(item.statement,item.assessment);
    pill(c,item.articleCount+" haber","gold");
    pill(c,item.publisherDomainCount+" yayıncı alan adı");
    (item.occurrences||[]).forEach(function(o){
      var row=append(c,"p","","");
      addSource(row,o.canonicalUrl,(o.publisherName||o.publisherDomain)+" · "+o.publisherDomain);
      append(c,"p","","“"+o.quote+"”");
    });
    append(c,"p","","Bu tekrar bağımsız doğrulama anlamına gelmez.");
    return c;
  }
  function renderDossierConflict(item){
    var c=card(item.context,item.assessment);
    pill(c,"İnsan incelemesi gerekli","gold");
    pill(c,item.assertionType);
    var sides=[{label:"Olumlu biçimde bildirilen",items:item.positiveSources||[]},{label:"Olumsuz biçimde bildirilen",items:item.negativeSources||[]}];
    sides.forEach(function(side){
      append(c,"h4","",side.label);
      side.items.forEach(function(o){
        var row=append(c,"p","","");
        addSource(row,o.canonicalUrl,(o.publisherName||o.publisherDomain)+" · "+date(o.publishedAt));
        append(c,"p","","“"+o.quote+"”");
      });
    });
    (item.limitations||[]).forEach(function(note){append(c,"p","","Sınır: "+note);});
    return c;
  }
  function renderDossier(d){
    var host=document.getElementById("content");
    host.replaceChildren();
    var back=append(host,"button","open-dossier","← Olay listesine dön");
    back.type="button";
    back.addEventListener("click",function(){selectedDossier=null;view="events";syncNav();draw();});
    var summary=card(d.event.title,"Bu, NexNews'ten alınan bir olay adayıdır. Dosya yalnızca bu adaya bağlı haber ve kanıt kayıtlarını içerir; olayın doğruluğu bağımsız olarak doğrulanmış değildir.");
    pill(summary,d.event.status||"durum bilinmiyor","gold");
    pill(summary,d.articleCount+" haber");
    pill(summary,d.publisherDomainCount+" yayıncı alan adı");
    pill(summary,d.statementCount+" kaynak ifadesi");
    pill(summary,(d.conflictCandidates||[]).length+" olası çelişki sinyali");
    append(host,"div","cards","").appendChild(summary);
    var researchLink=append(host,"a","open-dossier","Bu olay için araştırma başlat");
    researchLink.href="/research-workspace?eventId="+encodeURIComponent(d.event.id);
    var metricsBox=append(host,"div","metrics","");
    metric(metricsBox,"Haber",d.articleCount,"Bu olay adayına bağlı");
    metric(metricsBox,"Yayıncı alan adı",d.publisherDomainCount,"Bağımsız kaynak sayısı değildir");
    metric(metricsBox,"Yayın zamanı eksik",d.articlesWithoutDate,"Kronolojik kapsamı sınırlar");
    metric(metricsBox,"Yalnızca başlık",d.titleOnlyArticles,"Özet/tam metin kapsamı sınırlı");
    metric(metricsBox,"Kaynak ifadesi",d.statementCount,"Alıntısı saklanan ifadeler");
    metric(metricsBox,"Farklı ifade kalıbı",d.distinctWordingCount||0,"Yazım normalizasyonuna göre");
    metric(metricsBox,"Birebir tekrar",d.exactRepeatOccurrences||0,"Semantik yenilik ölçümü değildir");
    metric(metricsBox,"Tekrar grubu",(d.repeatedWordingGroups||[]).length,"Bağımsız teyit anlamına gelmez");
    (d.limitations||[]).forEach(function(note){host.appendChild(el("div","warning",note));});

    function heading(text,count){
      var row=append(host,"div","section-head","");
      append(row,"h2","",text);append(row,"span","count",count+" kayıt");
    }
    function filtered(list,textOf){
      var q=document.getElementById("search").value.trim().toLocaleLowerCase("tr-TR");
      return (list||[]).filter(function(item){return !q||textOf(item).toLocaleLowerCase("tr-TR").indexOf(q)>=0;});
    }
    var latest=filtered(d.latestArticles,function(a){return [a.title,a.summary,a.publisherName,a.publisherDomain,a.canonicalUrl].join(" ");});
    heading("Son gelişmeler · en yeni kayıtlar",latest.length);
    if(!latest.length)host.appendChild(el("div","empty","Yayın zamanı bilinen güncel haber bulunamadı."));
    latest.forEach(function(a){host.appendChild(renderDossierArticle(a));});

    var articles=filtered(d.articles,function(a){return [a.title,a.summary,a.publisherName,a.publisherDomain,a.canonicalUrl].join(" ");});
    heading("Tüm haberler · yayın sırasına göre",articles.length);
    if(!articles.length)host.appendChild(el("div","empty","Aramayla eşleşen haber yok."));
    articles.forEach(function(a){host.appendChild(renderDossierArticle(a));});

    var statements=filtered(d.statements,function(x){return [x.quote,x.publisherName,x.publisherDomain,x.claimStatus,x.excerptField].join(" ");});
    heading("Kaynak ifadeleri",statements.length);
    if(!statements.length)host.appendChild(el("div","empty","Bu olay adayında aramayla eşleşen kaynak ifadesi yok."));
    statements.forEach(function(x){host.appendChild(renderDossierStatement(x));});

    var repeats=filtered(d.repeatedWordingGroups,function(x){return [x.statement,x.assessment,(x.occurrences||[]).map(function(o){return o.publisherName+" "+o.publisherDomain+" "+o.quote;}).join(" ")].join(" ");});
    heading("Birebir tekrarlanan ifadeler",repeats.length);
    if(!repeats.length)host.appendChild(el("div","empty","Aynı ifade tekrarına rastlanmadı. Farklı anlatımlar burada semantik olarak karşılaştırılmıyor."));
    repeats.forEach(function(x){host.appendChild(renderDossierComparison(x));});

    var conflicts=filtered(d.conflictCandidates,function(x){return [x.context,x.assertionType,x.assessment,(x.positiveSources||[]).concat(x.negativeSources||[]).map(function(o){return o.quote+" "+o.publisherDomain;}).join(" ")].join(" ");});
    heading("Olası çelişkiler · inceleme adayları",conflicts.length);
    if(!conflicts.length)host.appendChild(el("div","empty","Mevcut sınırlı kurallarla çelişki adayı bulunmadı. Bu, çelişki olmadığı anlamına gelmez."));
    conflicts.forEach(function(x){host.appendChild(renderDossierConflict(x));});

    var entities=filtered(d.entities,function(x){return [x.canonicalName,x.entityType,(x.mentions||[]).map(function(m){return m.quote+" "+m.publisherDomain+" "+m.articleTitle;}).join(" ")].join(" ");});
    heading("Varlık anılmaları · kimlik kesin değil",entities.length);
    if(!entities.length)host.appendChild(el("div","empty","Bu dosya için varlık anılması kaydı bulunmuyor."));
    entities.forEach(function(x){
      var c=card(x.canonicalName,x.entityType+" · "+x.identityStatus);
      pill(c,x.articleCount+" haber");pill(c,x.mentionCount+" anılma");
      (x.mentions||[]).forEach(function(m){
        var row=append(c,"p","","“"+m.quote+"” · "+m.publisherDomain+" · "+m.field);
        addSource(row,m.canonicalUrl,"Haberi aç");
      });
      host.appendChild(c);
    });

    heading("Açık sorular ve eksikler",(d.openChecks||[]).length);
    (d.openChecks||[]).forEach(function(note){host.appendChild(el("div","warning",note));});
  }
  function renderComparison(item){
    var c=card(item.statement,item.assessment);
    pill(c,"Birebir ifade tekrarı","gold");
    pill(c,item.articleCount+" haber");
    pill(c,item.publisherDomainCount+" yayıncı alan adı");
    append(c,"p","","Bu eşleşme bağımsız doğrulama sayısı değildir. Ortak ajans/kopya kaynağı ayrıca belirlenmedi.");
    (item.occurrences||[]).forEach(function(o){
      var row=append(c,"p","","");
      var label=(o.publisherName||o.publisherDomain||"Kaynak")+" · "+(o.publisherDomain||"alan adı bilinmiyor")+" · "+date(o.publishedAt)+" · "+o.excerptField;
      addSource(row,o.canonicalUrl,label);
      append(c,"p","","“"+o.quote+"”");
    });
    (item.limitations||[]).forEach(function(note){append(c,"p","","Sınır: "+note);});
    return c;
  }
  function renderEvidence(item){
    var c=card("“"+item.quote+"”",item.excerptField==="title"?"Başlıktan alındı":item.excerptField==="summary"?"Özetten alındı":"Tam metinden alındı");
    pill(c,item.stance==="reports"?"Kaynak bunu bildirdi":item.stance);
    pill(c,item.contentLevel);
    append(c,"p","","Kaydedilme: "+date(item.capturedAt)+" · Alan: "+item.excerptField);
    var row=append(c,"p","","");addSource(row,item.sourceUrl,"Haberi aç");
    return c;
  }
  function renderEntity(item){
    var c=card(item.canonicalName,item.type);
    pill(c,item.identityStatus||"candidate","gold");
    var related=state.mentions.filter(function(m){return m.entityId===item.id;});
    append(c,"p","","Kaydedilmiş anılma: "+related.length);
    related.slice(0,4).forEach(function(m){
      var row=append(c,"p","","“"+m.quote+"” · "+m.field+" · byte ["+m.startOffset+","+m.endOffset+")");
      var a=state.articles.find(function(a){return a.id===m.articleId;});
      if(a)addSource(row,a.canonicalUrl,"Kaynak");
    });
    return c;
  }
  function renderRelation(item){
    var c=card(item.subject.type+":"+item.subject.id+" → "+item.object.type+":"+item.object.id,item.rationale);
    pill(c,item.predicate);pill(c,item.status,item.status==="supported"?"green":"gold");
    (item.evidenceIds||[]).forEach(function(id){
      var ev=state.evidence.find(function(e){return e.id===id;});
      if(ev){var row=append(c,"p","","Kanıt: “"+ev.quote+"” ");addSource(row,ev.sourceUrl,"Kaynak");}
    });
    return c;
  }
  function renderArticle(item){
    var c=card(item.title,item.summary||"Bu kayıt için özet bulunmuyor.");
    pill(c,item.contentAvailability);
    append(c,"p","","Kaynak: "+(item.publisherSourceId||"belirtilmemiş")+" · Yayın: "+date(item.publishedAt));
    var row=append(c,"p","","");addSource(row,item.canonicalUrl,"Haberi aç");
    return c;
  }
  function entries(){
    if(!state)return [];
    if(view==="events")return state.events.map(function(x){return {kind:"event",data:x,search:[x.title,x.scope,x.status,(x.coverageLimitations||[]).join(" ")].join(" ")};});
    if(view==="claims")return state.claims.map(function(x){return {kind:"claim",data:x,search:[x.claimText,x.attributedTo,x.statusReason,x.status].join(" ")};});
    if(view==="comparison")return (state.sourceComparisons.groups||[]).map(function(x){return {kind:"comparison",data:x,search:[x.statement,x.assessment,(x.occurrences||[]).map(function(o){return o.publisherName+" "+o.publisherDomain+" "+o.articleTitle+" "+o.quote;}).join(" ")].join(" ")};});
    if(view==="assessment")return (state.claimAssessment.cases||[]).map(function(x){return {kind:"assessment",data:x,search:[x.context,x.assertionType,x.assessment,(x.positiveSources||[]).concat(x.negativeSources||[]).map(function(o){return o.publisherName+" "+o.publisherDomain+" "+o.quote;}).join(" ")].join(" ")};});
    if(view==="quality")return (state.dataQuality.issues||[]).map(function(x){return {kind:"quality",data:x,search:[x.severity,x.code,x.recordType,x.recordId,x.message,x.sourceUrl].join(" ")};});
    if(view==="evidence")return state.evidence.map(function(x){return {kind:"evidence",data:x,search:[x.quote,x.sourceUrl,x.excerptField,x.stance].join(" ")};});
    if(view==="entities")return state.entities.map(function(x){return {kind:"entity",data:x,search:[x.canonicalName,x.type,x.identityStatus].join(" ")};});
    if(view==="relations")return state.relations.map(function(x){return {kind:"relation",data:x,search:[x.predicate,x.rationale,x.status].join(" ")};});
    return state.articles.map(function(x){return {kind:"article",data:x,search:[x.title,x.summary,x.sourceName,x.canonicalUrl].join(" ")};});
  }
  function syncNav(){
    document.querySelectorAll("[data-view]").forEach(function(b){
      var buttonView=b.getAttribute("data-view");
      b.classList.toggle("active",buttonView===(view==="dossier"?"events":view));
    });
  }
  function openDossier(eventID){
    var host=document.getElementById("content");
    host.replaceChildren(el("div","empty","Olay dosyası oluşturuluyor…"));
    document.getElementById("search").value="";
    fetch("/api/event-dossier?eventId="+encodeURIComponent(eventID),{cache:"no-store"}).then(function(r){
      if(!r.ok)throw new Error("HTTP "+r.status);
      return r.json();
    }).then(function(data){
      selectedDossier=data;view="dossier";syncNav();draw();
    }).catch(function(){
      host.replaceChildren(el("div","error","Olay dosyası oluşturulamadı. Olay kaydını ve Intelligence veritabanını kontrol edin."));
    });
  }
  function draw(){
    if(!state)return;
    var info=titles[view];
    document.getElementById("metrics").style.display=view==="dossier"?"none":"grid";
    document.getElementById("pageTitle").textContent=info[0];
    document.getElementById("pageSub").textContent=info[1];
    document.getElementById("sectionTitle").textContent=info[2];
    if(view==="dossier"){
      document.getElementById("resultCount").textContent=selectedDossier?selectedDossier.articleCount+" haber · "+selectedDossier.statementCount+" ifade":"yükleniyor";
      renderDossier(selectedDossier||{event:{title:"Yükleniyor"},articles:[],statements:[],repeatedWordingGroups:[],conflictCandidates:[],entities:[],openChecks:[],limitations:[]});
      return;
    }
    document.getElementById("pageTitle").textContent=info[0];
    document.getElementById("pageSub").textContent=info[1];
    document.getElementById("sectionTitle").textContent=info[2];
    var query=document.getElementById("search").value.trim().toLocaleLowerCase("tr-TR");
    var filtered=entries().filter(function(x){return !query||x.search.toLocaleLowerCase("tr-TR").indexOf(query)>=0;});
    document.getElementById("resultCount").textContent=filtered.length+" kayıt";
    var host=document.getElementById("content");host.replaceChildren();
    if(!filtered.length){
      var msg=view==="relations"?"Henüz gösterilecek ilişki yok. Bu beklenen bir sonuçtur; ilişki oluşturmak için bağlantı uydurulmayacak.":view==="comparison"?"Birebir aynı kaynak ifadesini birden fazla haber belgesinde tekrarlayan grup bulunamadı. Farklı ifadelerle anlatılan haberler bu sürümde semantik olarak karşılaştırılmaz.":view==="assessment"?"İzinli açık karşıt ifade kalıplarıyla eşleşen inceleme adayı bulunamadı. Bu, haberlerde çelişki olmadığı anlamına gelmez.":view==="quality"?"Kalite denetiminde bulgu yok. Tarih veya tam metin eksikliği gibi kapsama sınırlamaları, yapısal hata olmadan da bulunabilir.":"Bu bölümde eşleşen kayıt bulunamadı.";
      host.appendChild(el("div","empty",msg));return;
    }
    filtered.slice(0,250).forEach(function(entry){
      var node;
      if(entry.kind==="event")node=renderEvent(entry.data);
      else if(entry.kind==="claim")node=renderClaim(entry.data);
      else if(entry.kind==="comparison")node=renderComparison(entry.data);
      else if(entry.kind==="assessment")node=renderAssessment(entry.data);
      else if(entry.kind==="quality")node=renderQuality(entry.data);
      else if(entry.kind==="evidence")node=renderEvidence(entry.data);
      else if(entry.kind==="entity")node=renderEntity(entry.data);
      else if(entry.kind==="relation")node=renderRelation(entry.data);
      else node=renderArticle(entry.data);
      host.appendChild(node);
    });
    if(filtered.length>250)host.appendChild(el("div","empty","İlk 250 kayıt gösteriliyor. Aramayı daraltın."));
  }
  function load(){
    var host=document.getElementById("content");host.replaceChildren(el("div","empty","Veriler okunuyor…"));
    fetch("/api/state",{cache:"no-store"}).then(function(r){
      if(!r.ok)throw new Error("API "+r.status);
      return r.json();
    }).then(function(data){
      if(data.error)throw new Error(data.error);
      state=data;renderMetrics();draw();
      var text="Şema "+(data.schema||"bilinmiyor")+" · Son kayıt: "+date(data.savedAt)+" · Son içe aktarma: "+date(data.lastSyncAt);
      document.getElementById("dbMeta").textContent=text;
    }).catch(function(){
      host.replaceChildren(el("div","error","Veritabanı okunamadı. Önce CLI ile bir snapshot içe aktarın ve veritabanı yolunu kontrol edin."));
    });
  }
  document.querySelectorAll("[data-view]").forEach(function(button){
    button.addEventListener("click",function(){
      view=button.getAttribute("data-view");
      selectedDossier=null;
      syncNav();
      draw();
    });
  });
  document.getElementById("search").addEventListener("input",draw);
  document.getElementById("reload").addEventListener("click",load);
  load();
})();
</script>
</body>
</html>
`

