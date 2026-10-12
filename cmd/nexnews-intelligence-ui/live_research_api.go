package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"nexnews/intelligence"
)

const maxLiveResearchRequestBytes = 16 << 10

func registerLiveResearchRoutes(mux *http.ServeMux) {
	engine := intelligence.NewLiveResearchEngine()
	mux.HandleFunc("/analyst", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analyst" {
			http.NotFound(w, r)
			return
		}
		if !allowGET(w, r) {
			return
		}
		setSecurityHeaders(w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(liveResearchHTML))
	})
	mux.HandleFunc("/api/live-research", func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, `{"error":"Yalnızca POST destekleniyor."}`, http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxLiveResearchRequestBytes)
		defer r.Body.Close()
		var input intelligence.LiveResearchRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeLiveResearchAPIError(w, http.StatusBadRequest, "Geçerli bir JSON isteği gerekli.")
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			writeLiveResearchAPIError(w, http.StatusBadRequest, "İstek gövdesinde fazladan veri var.")
			return
		}
		if strings.TrimSpace(input.Query) == "" {
			writeLiveResearchAPIError(w, http.StatusBadRequest, "Araştırmak istediğin konuyu tek cümleyle yaz.")
			return
		}
		report, err := engine.Run(r.Context(), input)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "yapılandırılmamış") {
				status = http.StatusServiceUnavailable
			}
			writeLiveResearchAPIError(w, status, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(report)
	})
}

func writeLiveResearchAPIError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

const liveResearchHTML = `<!doctype html>
<html lang="tr">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>NexNews · Analizci</title>
<style>
:root{color-scheme:dark;--bg:#080e17;--panel:#101b29;--panel2:#142337;--line:#293c52;--text:#eef5fc;--muted:#96a9be;--cyan:#7ce0ed;--gold:#e8cb8e;--green:#8be0b7;--red:#ffadad}
*{box-sizing:border-box}body{margin:0;background:radial-gradient(ellipse at 70% -5%,#244663 0,transparent 42%),var(--bg);color:var(--text);font:14px/1.6 system-ui,-apple-system,Segoe UI,sans-serif}a{color:var(--cyan);overflow-wrap:anywhere}.shell{max-width:1500px;margin:auto;padding:24px clamp(14px,3vw,36px) 60px}.top{display:flex;justify-content:space-between;gap:20px;align-items:flex-start;border-bottom:1px solid var(--line);padding:0 0 22px}.brand{color:var(--cyan);font-size:11px;letter-spacing:.18em;font-weight:800;text-transform:uppercase}.top h1{font-size:clamp(28px,4vw,44px);letter-spacing:-.05em;margin:5px 0}.sub{color:var(--muted);max-width:760px}.pill{border:1px solid #36594f;color:var(--green);border-radius:30px;padding:7px 11px;font-size:12px;white-space:nowrap}.search{margin:24px 0 16px;padding:20px;border:1px solid #38536d;border-radius:18px;background:linear-gradient(130deg,rgba(24,46,66,.94),rgba(13,22,34,.95));box-shadow:0 20px 60px #0002}.search label{display:block;font-size:13px;color:#c5d7e8;margin-bottom:8px}.queryrow{display:flex;gap:10px}.queryrow input{flex:1;min-width:0;background:#08121e;color:var(--text);border:1px solid #3d536b;border-radius:11px;padding:15px;font:inherit;font-size:16px;outline:none}.queryrow input:focus{border-color:var(--cyan)}button{border:1px solid #5da6ba;background:linear-gradient(135deg,#2e7186,#234d68);color:white;padding:12px 17px;border-radius:11px;font:inherit;font-weight:750;cursor:pointer}button:disabled{opacity:.5;cursor:wait}.hints{display:flex;flex-wrap:wrap;gap:8px;margin-top:12px}.hint{background:#101e2d;border:1px solid #344b63;border-radius:30px;color:#c7d8e8;font-size:12px;padding:6px 10px;cursor:pointer}.hint:hover{border-color:var(--cyan)}.message{margin:12px 0;color:var(--muted)}.message.error{color:var(--red)}.metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px;margin:16px 0}.metric,.panel{border:1px solid var(--line);border-radius:15px;background:linear-gradient(145deg,rgba(17,28,42,.97),rgba(11,19,30,.97))}.metric{padding:14px}.metric strong{display:block;font-size:27px;letter-spacing:-.05em}.metric span{font-size:11px;color:var(--muted)}.grid{display:grid;grid-template-columns:minmax(0,1.25fr) minmax(300px,.75fr);gap:14px}.panel{padding:18px;margin-bottom:14px;min-width:0}.panel h2{font-size:18px;margin:0 0 12px;letter-spacing:-.02em}.panel h3{font-size:15px;margin:0 0 5px}.muted{color:var(--muted)}.cards{display:grid;gap:10px}.finding,.article,.theme{padding:13px;border:1px solid #2a3e53;background:#0b1521;border-radius:11px;min-width:0}.finding p,.article p{margin:6px 0}.kind{font-size:10px;letter-spacing:.1em;text-transform:uppercase;color:var(--gold);font-weight:800}.article .headline{font-weight:700;line-height:1.4;font-size:15px}.article .meta{font-size:11px;color:var(--muted);display:flex;gap:8px;flex-wrap:wrap;margin:7px 0}.summary{font-size:16px}.sectionhead{display:flex;justify-content:space-between;gap:10px;align-items:center}.mini{font-size:11px;color:var(--muted)}.themes{display:grid;gap:9px}.theme{display:flex;justify-content:space-between;gap:10px;align-items:center}.theme strong{display:block}.tag{font-size:11px;border:1px solid #344c62;color:#bdd5e7;border-radius:20px;padding:3px 7px;white-space:nowrap}.statusrow{display:grid;gap:8px}.provider{display:flex;justify-content:space-between;gap:12px;border-bottom:1px solid #243448;padding:7px 0}.provider:last-child{border:0}.ok{color:var(--green)}.fail{color:var(--red)}.limitlist{padding-left:18px;color:var(--muted)}.limitlist li{margin:8px 0}.empty{border:1px dashed #354a60;border-radius:12px;padding:22px;color:var(--muted);text-align:center}.hidden{display:none!important}.foot{font-size:11px;color:var(--muted);margin-top:20px}@media(max-width:850px){.grid{grid-template-columns:1fr}.metrics{grid-template-columns:repeat(2,minmax(0,1fr))}.top{display:block}.pill{display:inline-block;margin-top:10px}.queryrow{flex-direction:column}}@media(max-width:460px){.shell{padding-left:12px;padding-right:12px}.panel,.search{padding:14px}.metric{padding:11px}.metric strong{font-size:23px}}
</style>
</head>
<body><div class="shell">
<header class="top"><div><div class="brand">NexNews Intelligence · Research Core</div><h1>Analizci</h1><div class="sub">Soruyu ver. Açık haber kaynakları taransın, sonuçlar birleştirilsin, tekrar eden temalar ve kaynak boşlukları çıkarılsın. Bulguların dayandığı bağlantılar raporun içinde kalsın.</div></div><div class="pill">● Canlı kaynak keşfi</div></header>
<section class="search"><form id="researchForm"><label for="query">Neyi araştırıyoruz?</label><div class="queryrow"><input id="query" maxlength="500" required placeholder="Örn. Karadeniz enerji altyapısındaki son gelişmeler" autocomplete="off"><button id="run" type="submit">Araştırmayı başlat ↗</button></div><div class="hints"><button class="hint" type="button" data-q="Türkiye ve Avrupa'da son enerji güvenliği gelişmeleri">Enerji güvenliği</button><button class="hint" type="button" data-q="Dünyada son yapay zekâ düzenlemeleri">Yapay zekâ düzenlemeleri</button><button class="hint" type="button" data-q="Son 7 günde Kızıldeniz deniz taşımacılığı ve güvenliği">Kızıldeniz</button></div></form><div id="message" class="message">Tek bir konu yeterli. İlk tarama birkaç saniye sürebilir.</div></section>
<div id="report" class="hidden">
<div class="metrics"><div class="metric"><strong id="articleCount">0</strong><span>benzersiz haber bağlantısı</span></div><div class="metric"><strong id="publisherCount">0</strong><span>yayıncı alan adı</span></div><div class="metric"><strong id="themeCount">0</strong><span>tekrarlanan tema sinyali</span></div><div class="metric"><strong id="providerCount">0/3</strong><span>yanıt veren arama sağlayıcısı</span></div></div>
<section class="panel"><div class="sectionhead"><h2>İlk durum değerlendirmesi</h2><span id="reportStatus" class="tag"></span></div><p id="summary" class="summary"></p></section>
<div class="grid"><div>
<section class="panel"><h2>Analiz bulguları</h2><div id="findings" class="cards"></div></section>
<section class="panel"><div class="sectionhead"><h2>Kaynak akışı</h2><span id="articleLabel" class="mini"></span></div><div id="articles" class="cards"></div></section>
</div><div>
<section class="panel"><h2>Tekrarlanan temalar</h2><div id="themes" class="themes"></div></section>
<section class="panel"><h2>Kaynak sağlayıcıları</h2><div id="providers" class="statusrow"></div></section>
<section class="panel"><h2>Takip edilmesi gereken sorular</h2><div id="questions"></div></section>
<section class="panel"><h2>Yorumlama sınırları</h2><ul id="limitations" class="limitlist"></ul></section>
</div></div></div>
<div id="initial" class="empty">Henüz analiz çalıştırılmadı. Bir soru yazıp araştırmayı başlat.</div>
<div class="foot">NexNews Intelligence · Canlı arama, URL tekilleştirme ve kural tabanlı kapsama analizi. Başlık benzerliği doğrulama değildir; kaynak bağlantıları her bulgudan ayrıştırılabilir.</div>
</div>
<script>
const $ = id => document.getElementById(id);
document.querySelectorAll('[data-q]').forEach(b=>b.addEventListener('click',()=>{$('query').value=b.dataset.q;$('query').focus()}));
$('researchForm').addEventListener('submit',async e=>{
 e.preventDefault();
 const query=$('query').value.trim(); if(!query)return;
 $('run').disabled=true;$('run').textContent='Kaynaklar taranıyor…';$('message').className='message';$('message').textContent='Açık haber dizinleri paralel sorgulanıyor. Yavaş veya yanıt vermeyen kaynaklar diğerlerini bekletmemeli.';
 $('initial').classList.add('hidden');$('report').classList.add('hidden');
 try{
  const res=await fetch('/api/live-research',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({query,maxResults:30})});
  const data=await res.json();if(!res.ok)throw new Error(data.error||'Araştırma isteği başarısız.');
  render(data);$('message').textContent=data.status==='no_results'?'Arama tamamlandı ancak eşleşme bulunamadı.':data.status==='partial'?'Araştırma kısmen tamamlandı. Bazı sağlayıcılar yanıt vermedi.':'İlk tarama tamamlandı. Bulgular kaynak bağlantılarıyla birlikte gösteriliyor.';
 }catch(err){$('message').className='message error';$('message').textContent=err.message||'Araştırma sırasında beklenmeyen bir hata oluştu.';$('initial').classList.remove('hidden');}
 finally{$('run').disabled=false;$('run').textContent='Araştırmayı başlat ↗'}
});
function node(tag,cls,text){const n=document.createElement(tag);if(cls)n.className=cls;if(text!==undefined)n.textContent=text;return n}
function render(d){
 $('report').classList.remove('hidden');$('articleCount').textContent=d.articleCount;$('publisherCount').textContent=d.publisherCount;$('themeCount').textContent=d.themes.length;
 const ok=d.providerStatuses.filter(x=>x.status==='ok').length;$('providerCount').textContent=ok+'/'+d.providerStatuses.length;
 $('summary').textContent=d.summary;$('reportStatus').textContent=d.status==='completed'?'Tamamlandı':d.status==='partial'?'Kısmi sonuç':d.status==='no_results'?'Sonuç yok':d.status;
 $('findings').replaceChildren();if(!d.findings.length)$('findings').append(node('div','empty','Bu taramada ayrıca bulgu üretilemedi.'));
 d.findings.forEach(f=>{const card=node('article','finding');card.append(node('div','kind',f.kind.replaceAll('-',' ')),node('h3','',f.title),node('p','muted',f.detail));if(f.sourceUrls?.length){const row=node('p','');f.sourceUrls.forEach((u,i)=>{if(i)row.append(document.createTextNode(' · '));const a=node('a','', 'Kaynak '+(i+1));a.href=u;a.target='_blank';a.rel='noopener noreferrer';row.append(a)});card.append(row)}$('findings').append(card)});
 $('articles').replaceChildren();$('articleLabel').textContent=d.articleCount+' kayıt';
 if(!d.articles.length)$('articles').append(node('div','empty','Bu sorguda erişilebilir haber kaydı gelmedi.'));
 d.articles.forEach(a=>{const card=node('article','article');const link=node('a','headline',a.title);link.href=a.url;link.target='_blank';link.rel='noopener noreferrer';card.append(link);const meta=node('div','meta');meta.append(node('span','',a.publisher||a.domain||'Yayıncı bilinmiyor'),node('span','',a.publishedAt?new Date(a.publishedAt).toLocaleString('tr-TR'):'Yayın tarihi bilinmiyor'),node('span','',a.providers.join(', ')));card.append(meta);if(a.summary)card.append(node('p','muted',a.summary));$('articles').append(card)});
 $('themes').replaceChildren();if(!d.themes.length)$('themes').append(node('div','empty','Tekrarlanan tema için yeterli tekrar bulunmadı.'));
 d.themes.forEach(t=>{const row=node('div','theme');const left=node('div','');left.append(node('strong','',t.term),node('span','mini',t.examples.slice(0,1).join('')));row.append(left,node('span','tag',t.articleCount+' haber · '+t.sourceCount+' alan adı'));$('themes').append(row)});
 $('providers').replaceChildren();d.providerStatuses.forEach(p=>{const row=node('div','provider');row.append(node('span','',p.name));const st=node('span',p.status==='ok'?'ok':'fail',p.status==='ok'?p.resultCount+' sonuç':'Yanıt yok');row.append(st);if(p.message)row.title=p.message;$('providers').append(row)});
 $('questions').replaceChildren();d.openQuestions.forEach(q=>$('questions').append(node('p','muted','• '+q)));
 $('limitations').replaceChildren();d.limitations.forEach(q=>$('limitations').append(node('li','',q)));
}
</script></body></html>`
