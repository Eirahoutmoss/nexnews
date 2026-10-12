package main

const researchWorkspaceHTML = `<!doctype html>
<html lang="tr">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>NexNews Intelligence · Araştırma Masası</title>
<style>
:root{color-scheme:dark;--bg:#09111b;--surface:#101b29;--surface2:#152337;--line:#293b51;--text:#eaf2fb;--muted:#96a9bf;--cyan:#7ad9e7;--gold:#e8c98b;--green:#8fd8b0;--red:#f3a4a4}
*{box-sizing:border-box}body{margin:0;background:radial-gradient(ellipse at 80% 0%,#19304a 0,transparent 36%),var(--bg);color:var(--text);font:14px/1.55 system-ui,-apple-system,Segoe UI,sans-serif}
header{padding:25px clamp(16px,3vw,38px) 20px;border-bottom:1px solid var(--line);display:flex;justify-content:space-between;gap:20px;align-items:flex-start;background:rgba(6,12,20,.7)}
.eyebrow{color:var(--cyan);text-transform:uppercase;letter-spacing:.16em;font-weight:700;font-size:11px}h1{font-size:clamp(25px,3vw,34px);letter-spacing:-.04em;margin:5px 0}.sub{max-width:800px;color:var(--muted)}.status{border:1px solid #355f53;border-radius:40px;color:var(--green);padding:7px 10px;white-space:nowrap;font-size:12px}
a{color:var(--cyan);text-decoration:none}a:hover{text-decoration:underline}
.layout{display:grid;grid-template-columns:320px minmax(0,1fr);gap:18px;padding:20px clamp(12px,2vw,28px) 48px;max-width:1800px;margin:auto}.side,.main{min-width:0}.panel{background:linear-gradient(145deg,rgba(20,33,50,.96),rgba(12,21,33,.96));border:1px solid var(--line);border-radius:16px;padding:17px;margin-bottom:14px}.panel h2,.panel h3{margin:0 0 12px;letter-spacing:-.02em}.panel h2{font-size:19px}.panel h3{font-size:16px}.muted{color:var(--muted)}.small{font-size:12px}.field{display:grid;gap:6px;margin:10px 0}.field label{font-size:12px;color:#b4c7db}.field input,.field textarea,.field select{width:100%;background:#09131f;border:1px solid #34475f;border-radius:9px;padding:10px 11px;color:var(--text);font:inherit;min-width:0}.field textarea{resize:vertical;min-height:75px}.field input:focus,.field textarea:focus,.field select:focus{outline:1px solid var(--cyan);border-color:var(--cyan)}.grid2{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px}.button{border:1px solid #426781;background:#1a3348;color:#eaf6ff;border-radius:9px;padding:10px 13px;font:inherit;cursor:pointer}.button:hover{background:#23445d}.button.secondary{background:#111e2e;border-color:#344a63}.button.danger{background:#30202b;border-color:#67404e}.button:disabled{opacity:.55;cursor:wait}.stack{display:grid;gap:10px}.case-button{width:100%;background:#0c1725;border:1px solid #2a3d53;border-radius:11px;color:var(--text);padding:12px;text-align:left;cursor:pointer}.case-button.active{border-color:#6ec8d8;background:#14293a}.case-button strong{display:block;font-size:13px;margin-bottom:4px}.case-button small{color:var(--muted)}.case-button .meta{display:flex;gap:5px;flex-wrap:wrap;margin-top:7px}.pill{font-size:11px;padding:3px 7px;border:1px solid #344b63;border-radius:24px;color:#bcd0e3}.pill.gold{border-color:#615238;color:var(--gold)}.pill.green{border-color:#356850;color:var(--green)}.pill.red{border-color:#70424c;color:var(--red)}.topline{display:flex;align-items:flex-start;justify-content:space-between;gap:14px}.topline h2{margin:0;font-size:23px}.metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:9px;margin:15px 0}.metric{border:1px solid var(--line);background:#0c1724;padding:12px;border-radius:11px}.metric strong{display:block;font-size:22px;letter-spacing:-.04em}.metric span{color:var(--muted);font-size:11px}.notice{border-left:3px solid var(--gold);background:#211f1a;color:#d2c2a4;padding:11px 13px;border-radius:0 8px 8px 0;margin:12px 0}.section{margin-top:16px}.section-title{display:flex;justify-content:space-between;align-items:center;gap:10px;margin-bottom:9px}.section-title h3{margin:0}.item{border:1px solid var(--line);border-radius:12px;padding:13px;background:rgba(8,16,26,.58);min-width:0}.item p{margin:7px 0;overflow-wrap:anywhere}.item h4{margin:0 0 6px;overflow-wrap:anywhere}.item .meta{display:flex;gap:7px;flex-wrap:wrap;color:var(--muted);font-size:11px;margin-top:8px}.empty{border:1px dashed #354960;padding:22px;border-radius:12px;text-align:center;color:var(--muted)}.hidden{display:none!important}.formgrid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.formgrid>.panel{margin:0}.message{border-radius:9px;padding:10px 12px;margin-bottom:12px;background:#132b32;border:1px solid #2e605d;color:#afe7cf}.message.error{background:#301c24;border-color:#703c49;color:#ffc1c1}.row{display:flex;gap:8px;align-items:center;flex-wrap:wrap}.right{margin-left:auto}.footer{color:var(--muted);font-size:11px;padding-top:14px}.rule{border:0;border-top:1px solid var(--line);margin:14px 0}
@media(max-width:1100px){.layout{grid-template-columns:280px minmax(0,1fr)}.metrics{grid-template-columns:repeat(2,minmax(0,1fr))}.formgrid{grid-template-columns:1fr}}
@media(max-width:780px){header{display:block}.status{display:inline-block;margin-top:12px}.layout{grid-template-columns:1fr}.side{display:grid;grid-template-columns:1fr}.formgrid{grid-template-columns:1fr}}
@media(max-width:460px){.grid2{grid-template-columns:1fr}.metrics{grid-template-columns:repeat(2,minmax(0,1fr))}.panel{padding:13px}}
</style>
</head>
<body>
<header>
  <div><div class="eyebrow">NexNews Intelligence / Research Core</div><h1>Araştırma Masası</h1><div class="sub">Bir sorudan başla. Kaynakları ve kökenlerini kaydet, hipotezleri karşı kanıtlarla sınayıp belirsizlikleri görünür tut. Bu çalışma alanı kendi araştırma kayıtlarını NexNews haber önbelleğinden ayrı saklar.</div></div>
  <div><div class="status">● Yerel · Kaynak izli çalışma</div><p class="small"><a href="/">← Intelligence ana ekranı</a></p></div>
</header>
<div class="layout">
  <aside class="side">
    <section class="panel">
      <h2>Yeni araştırma</h2>
      <p class="muted small">Araştırılabilir bir soru gir. Başlangıçta sonuç veya niyet varsayma.</p>
      <form id="caseForm">
        <div class="field"><label for="caseTitle">Araştırma başlığı</label><input id="caseTitle" maxlength="200" required placeholder="Örn. Enerji tesislerinin hedef seçimi"></div>
        <div class="field"><label for="caseQuestion">Ana araştırma sorusu</label><textarea id="caseQuestion" maxlength="2000" required placeholder="Hangi örüntüyü, hangi dönem ve kapsamda sınayacağız?"></textarea></div>
        <div class="field"><label for="caseContext">Başlangıç bağlamı / kapsam (isteğe bağlı)</label><textarea id="caseContext" maxlength="10000" placeholder="Dönem, coğrafya, kapsam ve ilk gözlemler"></textarea></div>
        <button class="button" type="submit" id="createCaseButton">Araştırma dosyası oluştur</button>
      </form>
    </section>
    <section class="panel">
      <div class="section-title"><h3>Kurumsal hafıza</h3><span class="pill" id="caseCount">0 dosya</span></div>
      <div class="stack" id="caseList"><div class="empty">Araştırma dosyaları yükleniyor…</div></div>
      <p class="footer">Dosyalar yerel Intelligence veritabanına kalıcı kaydedilir.</p>
    </section>
  </aside>
  <main class="main">
    <div id="message" class="message hidden" role="status"></div>
    <div id="welcome" class="panel">
      <h2>Haber toplamanın ötesine geç</h2>
      <p class="muted">Bir araştırma dosyası seç veya yenisini oluştur. Her kanıt doğrudan bir kaynağa bağlanır. Aynı ajans metninin tekrar yayımlanması bağımsız teyit gibi gösterilmez.</p>
      <div class="notice">İlk sürüm, araştırma sürecinin kayıt ve kanıt disiplinini sağlar. İnterneti otomatik tarama, olay kayıtlarını bağımsız biçimde doğrulama veya stratejik niyeti otomatik çıkarma henüz bu çekirdeğin parçası değildir.</div>
    </div>
    <div id="workspace" class="hidden">
      <section class="panel">
        <div class="topline"><div><div class="eyebrow">Aktif araştırma</div><h2 id="activeTitle"></h2></div><div class="row"><span id="activeStatus" class="pill gold">Taslak</span><button id="refreshSources" class="button secondary hidden" type="button">NexNews kaynaklarını yenile</button></div></div>
        <p id="activeQuestion"></p><p id="activeContext" class="muted"></p>
        <div class="metrics" id="metrics"></div>
        <div id="limitations"></div>
      </section>
      <section class="panel section">
        <div class="section-title"><h3>Kanıt kapsama analizi</h3><button class="button secondary" id="analysisRefresh" type="button">Analizi yenile</button></div>
        <p class="muted small">Bu rapor, hipotez başına bağlanan destekleyici, karşıt ve bağlamsal kayıtları sayar. Yayıncı alan adlarını bağımsız kanıt olarak varsaymaz ve olayın doğruluğu ya da aktörlerin niyeti hakkında otomatik hüküm vermez.</p>
        <div class="metrics" id="analysisOverview"><div class="empty">Analiz raporu hazırlanıyor…</div></div>
        <div class="stack" id="hypothesisCoverage"></div>
        <div id="analysisLimitations"></div>
      </section>
      <div class="formgrid">
        <section class="panel">
          <h3>Hipotezler</h3>
          <p class="muted small">Hipotez sonuç değil, sınanabilir açıklamadır. Sonuç statüsü için bağlı ve yönü açık kanıt gerekir.</p>
          <form id="hypothesisForm">
            <div class="field"><label for="hypothesisText">Yeni hipotez</label><textarea id="hypothesisText" required maxlength="3000" placeholder="Bu örüntü şu nedenle ortaya çıkıyor olabilir…"></textarea></div>
            <button class="button" type="submit">Hipotez ekle</button>
          </form>
          <hr class="rule">
          <div class="stack" id="hypotheses"></div>
        </section>
        <section class="panel">
          <h3>Açık sorular</h3>
          <p class="muted small">Eksik veri ve çözülmemiş soruları hafızada tut. Bunlar analizin sonraki araştırma adımlarını belirler.</p>
          <form id="questionForm">
            <div class="field"><label for="questionText">Yeni araştırma sorusu</label><textarea id="questionText" required maxlength="2000" placeholder="Hangi veri eksik? Hangi alternatif açıklama sınanmalı?"></textarea></div>
            <button class="button" type="submit">Açık soru ekle</button>
          </form>
          <hr class="rule">
          <div class="stack" id="questions"></div>
        </section>
      </div>
      <section class="panel section">
        <h3>Kaynaklı kanıt ekle</h3>
        <p class="muted small">Sadece gerçekten erişilen başlık, özet, tam metin veya veri açıklamasını kaydet. Kaynakta olmayan alıntı yazma. Bir kaynak bir hipotezi destekleyebilir, ona karşı çıkabilir ya da yalnızca bağlam sağlayabilir.</p>
        <form id="evidenceForm">
          <div class="grid2">
            <div class="field"><label for="evidenceTitle">Kaynak başlığı / belge adı</label><input id="evidenceTitle" required maxlength="500" placeholder="Kaynağın gerçek başlığı"></div>
            <div class="field"><label for="evidenceURL">Kaynak URL</label><input id="evidenceURL" required type="url" placeholder="https://…"></div>
            <div class="field"><label for="publisher">Yayıncı / veri sağlayıcı</label><input id="publisher" required placeholder="Kurum, yayıncı veya muhabirlik ekibi"></div>
            <div class="field"><label for="author">Yazar / muhabir (isteğe bağlı)</label><input id="author" placeholder="Ad veya ekip"></div>
            <div class="field"><label for="publishedAt">Yayın tarihi (isteğe bağlı)</label><input id="publishedAt" type="datetime-local"></div>
            <div class="field"><label for="sourceKind">Kaynak türü</label><select id="sourceKind"><option value="primary_source">Birincil kaynak</option><option value="local_reporting">Yerel habercilik / saha bilgisi</option><option value="international_media">Uluslararası medya</option><option value="official_document">Resmî belge</option><option value="dataset">Yapılandırılmış veri kümesi</option><option value="academic_research">Akademik araştırma</option><option value="other">Diğer</option></select></div>
            <div class="field"><label for="contentLevel">Gerçekte erişilen içerik</label><select id="contentLevel"><option value="title_only">Yalnızca başlık</option><option value="title_summary">Başlık ve özet</option><option value="full_text">Tam metin</option><option value="structured_data">Yapılandırılmış veri</option></select></div>
            <div class="field"><label for="sourceOrigin">Kaynak kökeni</label><select id="sourceOrigin"><option value="unknown">Henüz bilinmiyor</option><option value="original_reporting">Özgün haber / doğrudan araştırma</option><option value="official_statement">Resmî açıklama</option><option value="syndicated_or_republished">Ajans metni / yeniden yayımlanmış</option></select></div>
            <div class="field"><label for="evidenceKind">Kanıt türü</label><select id="evidenceKind"><option value="observation">Gözlem / doğrudan bulgu</option><option value="attributed_claim">Bir tarafa atfedilen iddia</option><option value="data_point">Veri noktası</option><option value="background">Bağlam bilgisi</option><option value="correction_or_update">Düzeltme / güncelleme</option></select></div>
            <div class="field"><label for="syndicationGroup">Ajans / kopya zinciri etiketi</label><input id="syndicationGroup" placeholder="Biliniyorsa ortak köken (isteğe bağlı)"></div>
          </div>
          <div class="field"><label for="excerpt">Gerçek alıntı veya veri açıklaması</label><textarea id="excerpt" required maxlength="12000" placeholder="Bu kaynağın gerçekten söylediği metin veya veri tanımı"></textarea></div>
          <div class="field"><label for="methodNote">Yöntem ve kapsam notu (isteğe bağlı)</label><textarea id="methodNote" placeholder="Nereden alındı, neyi kapsıyor, hangi sınırlılıkları var?"></textarea></div>
          <hr class="rule">
          <div class="grid2">
            <div class="field"><label for="assessedHypothesis">İlişkilendirilecek hipotez (isteğe bağlı)</label><select id="assessedHypothesis"><option value="">Yalnızca kaynak olarak kaydet</option></select></div>
            <div class="field"><label for="assessmentDirection">Kanıtın bu hipoteze yönü</label><select id="assessmentDirection"><option value="context">Bağlam sağlar</option><option value="supports">Destekler</option><option value="contradicts">Karşı kanıt</option></select></div>
          </div>
          <div class="field"><label for="assessmentRationale">İlişkilendirme gerekçesi</label><textarea id="assessmentRationale" placeholder="Hipotez seçersen bu kaynağın onu neden desteklediğini, zayıflattığını veya yalnızca bağlam sunduğunu açıkla."></textarea></div>
          <button class="button" type="submit" id="saveEvidenceButton">Kaynaklı kanıtı kaydet</button>
        </form>
      </section>
      <section class="panel section">
        <div class="section-title"><h3>Kaynak ve kanıt günlüğü</h3><span class="pill" id="evidenceCount">0 kayıt</span></div>
        <div class="stack" id="evidenceList"></div>
      </section>
      <section class="panel section" id="targetSelectionPanel">
        <div class="section-title"><h3>Hedef seçimi ve örüntü analizi</h3><span class="pill">Betimsel karşılaştırma</span></div>
        <p class="muted small">Bir olay örüntüsünü değerlendirmek için aynı ölçütlerle tanımlanmış tesis evreni, araştırma dönemi ve kaynaklı sahiplik kayıtları gir. “Hedef iddiası” fiilî saldırı olarak sayılmaz. Bu araç interneti otomatik taramaz veya saldırganın niyetini çıkarmaz.</p>
        <div id="targetStudyForm">
          <div class="grid2">
            <div class="field"><label for="targetStudyTitle">Çalışma başlığı</label><input id="targetStudyTitle" required maxlength="200" placeholder="Örn. Rafinerilerde hedef seçimi"></div>
            <div class="field"><label for="targetStudyQuestion">Sınanacak soru</label><input id="targetStudyQuestion" required maxlength="1000" placeholder="Sahiplik kategorileri arasında kayıtlı hedefleme oranı farklı mı?"></div>
            <div class="field"><label for="targetPeriodStart">Araştırma başlangıcı</label><input id="targetPeriodStart" type="date" required></div>
            <div class="field"><label for="targetPeriodEnd">Araştırma bitişi</label><input id="targetPeriodEnd" type="date" required></div>
          </div>
          <div class="field"><label for="targetUniverseDefinition">Karşılaştırma evreni tanımı</label><textarea id="targetUniverseDefinition" required placeholder="Coğrafya, tesis türü, dâhil etme ölçütü ve dönemin nasıl seçildiği"></textarea></div>
          <label class="row small"><input id="targetUniverseComplete" type="checkbox"> Bu tesis evreninin tanımlı kapsama göre eksiksiz olduğunu kaynaklarla kontrol ettim</label>
          <div class="notice">Evrenin eksiksizliği işaretlenmediyse sistem iki sahiplik grubu arasındaki farkı örüntü sonucu olarak yayımlamaz. İşaretleme araştırmacının beyanıdır, otomatik doğrulama değildir.</div>
          <hr class="rule">
          <h3>Tesis envanteri</h3>
          <p class="muted small">Her tesis için dâhil edilme gerekçesini ve sahiplik dönemini destekleyen, bu araştırma dosyasına daha önce kaydedilmiş kanıtları seç.</p>
          <form id="targetAssetForm">
            <div class="grid2">
              <div class="field"><label for="targetAssetId">Tesis kimliği</label><input id="targetAssetId" required placeholder="Örn. refinery-01"></div>
              <div class="field"><label for="targetAssetName">Tesis adı</label><input id="targetAssetName" required placeholder="Tesisin belgelerdeki adı"></div>
              <div class="field"><label for="targetAssetType">Tesis türü</label><input id="targetAssetType" required value="refinery"></div>
              <div class="field"><label for="targetAssetCategory">Araştırma dönemi sahiplik sınıfı</label><select id="targetAssetCategory"><option value="foreign_participation">Yabancı ortak/katılım var</option><option value="no_foreign_participation">Yabancı ortak/katılım yok</option><option value="unknown">Bilinmiyor / sınıflandırılamıyor</option></select></div>
            </div>
            <div class="field"><label for="targetAssetInclusion">Bu tesis neden karşılaştırma evrenine dâhil?</label><textarea id="targetAssetInclusion" required placeholder="Ölçütleri diğer tesislere de aynı biçimde uygulayan gerekçe"></textarea></div>
            <div class="grid2">
              <div class="field"><label for="targetAssetEligibilityEvidence">Tesisin uygunluğunu gösteren kaynak</label><select id="targetAssetEligibilityEvidence" required></select></div>
              <div class="field"><label for="targetAssetOwnershipEvidence">Sahiplik sınıfını gösteren kaynak</label><select id="targetAssetOwnershipEvidence" required></select></div>
              <div class="field"><label for="targetOwnershipValidFrom">Sahiplik kaydı geçerlilik başlangıcı</label><input id="targetOwnershipValidFrom" type="date" required></div>
              <div class="field"><label for="targetOwnershipValidTo">Sahiplik kaydı geçerlilik bitişi (boşsa hâlen geçerli)</label><input id="targetOwnershipValidTo" type="date"></div>
            </div>
            <div class="field"><label for="targetAssetOwnershipDetails">Ortaklık, pay oranları ve kaynak notu</label><textarea id="targetAssetOwnershipDetails" required placeholder="Kaydın tam olarak neyi gösterdiği; paylar ve varsa belirsizlikler"></textarea></div>
            <button class="button secondary" type="submit">Tesis envanterine ekle</button>
          </form>
          <div class="stack section" id="targetAssetList"><div class="empty">Henüz tesis eklenmedi.</div></div>
          <hr class="rule">
          <h3>Hedefleme / saldırı kayıtları</h3>
          <p class="muted small">Yalnızca bu araştırma aralığı içindeki kayıtları ekle. İddiayı, bildirilen girişimi, önlenen saldırıyı, isabeti ve hasarı birbirine karıştırma.</p>
          <form id="targetIncidentForm">
            <div class="grid2">
              <div class="field"><label for="targetIncidentAsset">İlgili tesis</label><select id="targetIncidentAsset" required></select></div>
              <div class="field"><label for="targetIncidentActor">İsnat edilen aktör</label><input id="targetIncidentActor" required placeholder="Kaynakta adı geçen aktör"></div>
              <div class="field"><label for="targetIncidentDate">Olay zamanı</label><input id="targetIncidentDate" type="date" required></div>
              <div class="field"><label for="targetIncidentStatus">Kayıt durumu</label><select id="targetIncidentStatus"><option value="claim_only">Yalnızca hedefleme iddiası</option><option value="attempt_reported">Saldırı girişimi bildirildi</option><option value="intercepted">Önleme / havada imha bildirildi</option><option value="impact_reported">İsabet bildirildi</option><option value="damage_reported">Hasar bildirildi</option></select></div>
            </div>
            <div class="field"><label for="targetIncidentDescription">Kaydın tam açıklaması</label><textarea id="targetIncidentDescription" required placeholder="Kaynağın bildirdiği durum, neyin bağımsız doğrulanmadığı dâhil"></textarea></div>
            <div class="field"><label for="targetIncidentEvidence">Olay / iddia kaynağı</label><select id="targetIncidentEvidence" required></select></div>
            <button class="button secondary" type="submit">Olay kaydına ekle</button>
          </form>
          <div class="stack section" id="targetIncidentList"><div class="empty">Henüz olay kaydı yok.</div></div>
          <div class="row section"><button class="button" id="saveTargetStudy" type="button">Çalışmayı kaydet ve analiz et</button><span class="muted small" id="targetStudyReadiness">Kaynak kanıtı ve en az bir tesis gereklidir.</span></div>
        </div>
        <hr class="rule">
        <div class="section-title"><h3>Kaydedilmiş karşılaştırmalar</h3><button class="button secondary" id="refreshTargetStudies" type="button">Yenile</button></div>
        <div class="stack" id="targetStudyList"><div class="empty">Araştırma dosyası seçildiğinde kayıtlar gösterilecek.</div></div>
        <div id="targetStudyReport" class="section"></div>
      </section>
    </div>
    <div class="footer">NexNews Intelligence Research Core · Kaynak kökeni kaybolmaz. Kaynak sayısı tek başına doğrulama değildir. Sonuçlar ve yorumlar, kaynaklardan ayrı tutulur.</div>
  </main>
</div>
<script>
(function(){
  "use strict";
  var selectedCaseId = "";
  var selectedCase = null;
  var caseSummaries = [];
  var linkedEventId = new URLSearchParams(window.location.search).get("eventId") || "";
  var messageTimer = null;
  var targetAssets = [];
  var targetIncidents = [];
  function byId(id){return document.getElementById(id);}
  function node(tag, cls, text){
    var n=document.createElement(tag);if(cls)n.className=cls;if(text!==undefined&&text!==null)n.textContent=String(text);return n;
  }
  function add(parent,tag,cls,text){var n=node(tag,cls,text);parent.appendChild(n);return n;}
  function clear(n){n.replaceChildren();}
  function setMessage(text,isError){
    var m=byId("message");clear(m);m.textContent=text;m.className="message"+(isError?" error":"");
    if(messageTimer)window.clearTimeout(messageTimer);
    messageTimer=window.setTimeout(function(){m.className="message hidden";},6500);
  }
  async function api(url, options){
    options=options||{};options.headers=Object.assign({"Accept":"application/json"},options.headers||{});
    if(options.body!==undefined)options.headers["Content-Type"]="application/json";
    var response=await fetch(url,options);var data=null;
    try{data=await response.json();}catch(_){}
    if(!response.ok)throw new Error(data&&data.error?data.error:"İstek başarısız ("+response.status+").");
    return data;
  }
  function pill(parent,text,kind){add(parent,"span","pill "+(kind||""),text);}
  function date(v){
    if(!v)return "Tarih belirtilmemiş";
    var d=new Date(v);return isNaN(d.getTime())?"Tarih belirtilmemiş":d.toLocaleString("tr-TR");
  }
  function safeLink(raw,label){
    try{var u=new URL(raw);if(u.protocol!=="https:"&&u.protocol!=="http:")return null;var a=node("a","",label||raw);a.href=u.href;a.target="_blank";a.rel="noopener noreferrer";return a;}catch(_){return null;}
  }
  function sourceKind(value){
    return ({primary_source:"Birincil kaynak",local_reporting:"Yerel habercilik",international_media:"Uluslararası medya",official_document:"Resmî belge",dataset:"Veri kümesi",academic_research:"Akademik araştırma",other:"Diğer"})[value]||value;
  }
  function originText(value){
    return ({original_reporting:"Özgün haber / doğrudan araştırma",official_statement:"Resmî açıklama",syndicated_or_republished:"Ajans / yeniden yayımlama",unknown:"Köken bilinmiyor"})[value]||value;
  }
  function levelText(value){
    return ({title_only:"Yalnızca başlık",title_summary:"Başlık ve özet",full_text:"Tam metin",structured_data:"Yapılandırılmış veri"})[value]||value;
  }
  function evidenceKindText(value){
    return ({source_article:"NexNews haber kaydı",observation:"Gözlem",attributed_claim:"Atfedilen iddia",data_point:"Veri noktası",background:"Bağlam",correction_or_update:"Düzeltme / güncelleme"})[value]||value;
  }
  function statusText(value){
    return ({draft:"Taslak",active:"Aktif",monitoring:"İzlemede",closed:"Kapatıldı",open:"Açık",under_review:"İncelemede",supported:"Destek buldu",weakened:"Zayıfladı",rejected:"Reddedildi",inconclusive:"Sonuçsuz",supports:"Destekler",contradicts:"Karşı kanıt",context:"Bağlam"})[value]||value;
  }
  function refreshCaseList(){
    return api("/api/research-cases").then(function(result){
      caseSummaries=result.items||[];
      renderCaseList();
      if(selectedCaseId)return loadCase(selectedCaseId);
      byId("welcome").classList.remove("hidden");byId("workspace").classList.add("hidden");
      return null;
    });
  }
  function renderCaseList(){
    var host=byId("caseList");clear(host);byId("caseCount").textContent=caseSummaries.length+" dosya";
    if(!caseSummaries.length){host.appendChild(node("div","empty","Henüz araştırma yok. İlk dosyanı oluştur."));return;}
    caseSummaries.forEach(function(summary){
      var c=summary.case;var b=node("button","case-button"+(c.id===selectedCaseId?" active":""));
      b.type="button";add(b,"strong","",c.title);add(b,"small","",c.question);
      var meta=add(b,"div","meta","");pill(meta,(summary.evidenceCount||0)+" kanıt");pill(meta,(summary.hypothesisCount||0)+" hipotez");pill(meta,(summary.openQuestionCount||0)+" açık soru");
      b.addEventListener("click",function(){loadCase(c.id).catch(function(e){setMessage(e.message,true);});});
      host.appendChild(b);
    });
  }
  function renderMetrics(summary){
    var host=byId("metrics");clear(host);
    [["Kanıt",summary.evidenceCount],["Hipotez",summary.hypothesisCount],["Yayıncı alan adı",summary.publisherDomainCount],["Açık soru",summary.openQuestionCount]].forEach(function(pair){
      var m=add(host,"div","metric","");add(m,"strong","",pair[1]);add(m,"span","",pair[0]);
    });
    var limitations=byId("limitations");clear(limitations);
    (summary.limitations||[]).forEach(function(note){add(limitations,"div","notice",note);});
  }
  function renderHypothesisOptions(){
    var select=byId("assessedHypothesis");var chosen=select.value;clear(select);
    var empty=add(select,"option","","Yalnızca kaynak olarak kaydet");empty.value="";
    (selectedCase.hypotheses||[]).forEach(function(h){
      var option=add(select,"option","",h.statement);option.value=h.id;
    });
    if((selectedCase.hypotheses||[]).some(function(h){return h.id===chosen;}))select.value=chosen;
  }
  function renderAnalysis(report){
    var overview=byId("analysisOverview");clear(overview);
    [["Kanıt kaydı",report.evidenceCount],["Yayıncı alan adı",report.publisherDomainCount],["Tam metin / veri",report.fullTextOrStructuredCount],["Kökeni bilinmeyen",report.unknownSourceOriginCount],["Atfedilmiş iddia",report.attributedClaimCount],["Ortak köken etiketi",report.knownSyndicationGroupCount],["Açık soru",report.openQuestionCount]].forEach(function(pair){
      var m=add(overview,"div","metric","");add(m,"strong","",pair[1]);add(m,"span","",pair[0]);
    });
    var host=byId("hypothesisCoverage");clear(host);
    if(!report.hypotheses||!report.hypotheses.length){host.appendChild(node("div","empty","Henüz hipotez yok. Hipotez eklediğinde ona bağlı kanıtların kapsamı burada çözümlenecek."));}
    (report.hypotheses||[]).forEach(function(h){
      var item=add(host,"article","item","");add(item,"h4","",h.statement);
      var row=add(item,"div","meta","");pill(row,statusText(h.status));pill(row,h.coverageLabel.replace(/_/g," "));
      var countrow=add(item,"div","meta","");pill(countrow,"Destek: "+h.supportingEvidenceCount);pill(countrow,"Karşı kanıt: "+h.contradictingEvidenceCount);pill(countrow,"Bağlam: "+h.contextEvidenceCount);
      add(item,"p","muted small","Destekleyen yayıncı alan adları: "+((h.supportingPublisherDomains||[]).join(", ")||"yok")+" · Karşı kanıt alan adları: "+((h.contradictingPublisherDomains||[]).join(", ")||"yok"));
      (h.limitations||[]).forEach(function(note){add(item,"div","notice",note);});
      host.appendChild(item);
    });
    var limits=byId("analysisLimitations");clear(limits);
    (report.limitations||[]).forEach(function(note){add(limits,"div","notice",note);});
    var stamp=add(limits,"p","muted small","Rapor zamanı: "+date(report.generatedAt)+" · Alan adı sayısı bağımsız doğrulama sayısı değildir.");
  }
  async function loadAnalysis(){
    if(!selectedCase)return;
    var id=selectedCase.id;
    try{
      var report=await api("/api/research-analysis?caseId="+encodeURIComponent(id));
      if(selectedCase&&selectedCase.id===id)renderAnalysis(report);
    }catch(err){
      if(selectedCase&&selectedCase.id===id){byId("analysisOverview").replaceChildren(node("div","empty","Kanıt kapsama analizi alınamadı: "+err.message));}
    }
  }
  function renderHypotheses(){
    var host=byId("hypotheses");clear(host);
    var items=selectedCase.hypotheses||[];
    if(!items.length){host.appendChild(node("div","empty","Henüz hipotez eklenmedi. Önce araştırılabilir bir açıklama tanımla."));}
    items.forEach(function(h){
      var box=add(host,"article","item","");add(box,"h4","",h.statement);pill(box,statusText(h.status),h.status==="supported"?"green":h.status==="rejected"?"red":"gold");
      if(h.assessmentNote)add(box,"p","muted",h.assessmentNote);
      var form=add(box,"form","stack","");form.style.marginTop="10px";
      var select=add(form,"select","");select.setAttribute("aria-label","Hipotez durumu");
      ["open","under_review","supported","weakened","rejected","inconclusive"].forEach(function(s){var o=add(select,"option","",statusText(s));o.value=s;});
      select.value=h.status;
      var note=add(form,"textarea","");note.placeholder="Değerlendirme notu. Sonuç statüsü için gerekçeli not zorunlu.";note.value=h.assessmentNote||"";note.rows=2;
      var button=add(form,"button","button secondary","Durumu güncelle");button.type="submit";
      form.addEventListener("submit",async function(event){
        event.preventDefault();button.disabled=true;
        try{var item=await api("/api/research-hypothesis-status?caseId="+encodeURIComponent(selectedCase.id)+"&hypothesisId="+encodeURIComponent(h.id),{method:"POST",body:JSON.stringify({status:select.value,note:note.value})});selectedCase=item;renderCurrentCase();setMessage("Hipotez değerlendirmesi kaydedildi.",false);}
        catch(err){setMessage(err.message,true);}finally{button.disabled=false;}
      });
      host.appendChild(box);
    });
    renderHypothesisOptions();
  }
  function renderQuestions(){
    var host=byId("questions");clear(host);var items=selectedCase.openQuestions||[];
    if(!items.length){host.appendChild(node("div","empty","Açık soru bulunmuyor. Veri boşluklarını burada kaydet."));return;}
    items.forEach(function(q){
      var c=add(host,"article","item","");add(c,"p","",q.text);
      var row=add(c,"div","meta","");pill(row,statusText(q.status),q.status==="resolved"?"green":"gold");add(row,"span","","Eklenme: "+date(q.createdAt));
      if(q.resolutionNote)add(c,"p","muted small","Çözüm notu: "+q.resolutionNote);
      var form=add(c,"form","stack","");form.style.marginTop="10px";
      var select=add(form,"select","");select.setAttribute("aria-label","Araştırma sorusunun durumu");
      [["open","Açık"],["resolved","Çözüldü"]].forEach(function(pair){var option=add(select,"option","",pair[1]);option.value=pair[0];});
      select.value=q.status;
      var note=add(form,"textarea","");note.rows=2;note.placeholder="Çözüldüyse yanıtı ve dayandığı kanıtı not et.";note.value=q.resolutionNote||"";
      var button=add(form,"button","button secondary","Soru durumunu güncelle");button.type="submit";
      form.addEventListener("submit",async function(event){
        event.preventDefault();button.disabled=true;
        try{
          var item=await api("/api/research-question-status?caseId="+encodeURIComponent(selectedCase.id)+"&questionId="+encodeURIComponent(q.id),{method:"POST",body:JSON.stringify({status:select.value,note:note.value})});
          selectedCase=item;await refreshCaseList();renderCurrentCase();setMessage("Araştırma sorusunun durumu kaydedildi.",false);
        }catch(err){setMessage(err.message,true);}finally{button.disabled=false;}
      });
    });
  }
  function renderEvidence(){
    var host=byId("evidenceList");clear(host);var items=selectedCase.evidence||[];byId("evidenceCount").textContent=items.length+" kayıt";
    if(!items.length){host.appendChild(node("div","empty","Henüz kaynaklı kanıt yok. İlk kaynak kaydı olmadan hipotezler sonuçlandırılamaz."));return;}
    items.slice().reverse().forEach(function(ev){
      var c=add(host,"article","item","");add(c,"h4","",ev.title);
      var row=add(c,"div","meta","");pill(row,evidenceKindText(ev.kind));pill(row,sourceKind(ev.sourceKind));pill(row,levelText(ev.contentLevel));pill(row,originText(ev.sourceOrigin),ev.sourceOrigin==="unknown"?"gold":"");
      add(c,"p","",ev.excerpt);
      add(c,"p","muted small","Kanıt kayıt kimliği: "+ev.id);
      var meta=add(c,"p","muted small","Yayıncı: "+ev.publisher+" ("+ev.publisherDomain+") · Yayın tarihi: "+date(ev.publishedAt)+" · Kaydedilme: "+date(ev.retrievedAt));
      if(ev.author)add(c,"p","muted small","Yazar / ekip: "+ev.author);
      if(ev.syndicationGroup)add(c,"p","muted small","Paylaşılan köken etiketi: "+ev.syndicationGroup);
      if(ev.methodNote)add(c,"p","muted small","Yöntem notu: "+ev.methodNote);
      var linkrow=add(c,"p","","");var link=safeLink(ev.sourceUrl,"Orijinal kaynağa git");if(link)linkrow.appendChild(link);
      (ev.hypothesisAssessments||[]).forEach(function(a){
        var h=(selectedCase.hypotheses||[]).find(function(x){return x.id===a.hypothesisId;});
        var assessment=add(c,"div","notice","");add(assessment,"strong","",statusText(a.direction)+(h?" · "+h.statement:""));add(assessment,"p","",a.rationale);
      });
    });
  }
  function targetDateISO(id) {
    var value=byId(id).value;
    if(!value)return null;
    var endOfDay=(id==="targetPeriodEnd"||id==="targetOwnershipValidTo");
    var date=new Date(value+(endOfDay?"T23:59:59.999Z":"T00:00:00.000Z"));
    return isNaN(date.getTime())?null:date.toISOString();
  }
  function fillEvidenceSelect(id, placeholder) {
    var select=byId(id);if(!select)return;
    var previous=select.value;clear(select);
    var first=add(select,"option","",placeholder);first.value="";
    (selectedCase&&selectedCase.evidence||[]).forEach(function(ev){
      var option=add(select,"option","",ev.title+" · "+ev.publisherDomain+" · "+ev.id);
      option.value=ev.id;
    });
    if((selectedCase&&selectedCase.evidence||[]).some(function(ev){return ev.id===previous;}))select.value=previous;
  }
  function renderTargetSelectors() {
    fillEvidenceSelect("targetAssetEligibilityEvidence","Tesis uygunluk kanıtını seç");
    fillEvidenceSelect("targetAssetOwnershipEvidence","Sahiplik kanıtını seç");
    fillEvidenceSelect("targetIncidentEvidence","Olay kanıtını seç");
    var select=byId("targetIncidentAsset"),prior=select.value;clear(select);
    var empty=add(select,"option","","Önce tesis envanteri oluştur");empty.value="";
    targetAssets.forEach(function(asset){var option=add(select,"option","",asset.name+" ("+asset.id+")");option.value=asset.id;});
    if(targetAssets.some(function(asset){return asset.id===prior;}))select.value=prior;
  }
  function renderTargetAssets() {
    var host=byId("targetAssetList");clear(host);
    if(!targetAssets.length){host.appendChild(node("div","empty","Henüz tesis eklenmedi."));}
    targetAssets.forEach(function(asset,index){
      var c=add(host,"article","item","");
      add(c,"h4","",asset.name+" · "+asset.id);
      pill(c,asset.ownershipHistory[0].category==="foreign_participation"?"Yabancı katılım var":asset.ownershipHistory[0].category==="no_foreign_participation"?"Yabancı katılım yok":"Sınıf bilinmiyor");
      add(c,"p","muted small",asset.assetType+" · Uygunluk kanıtı: "+asset.eligibilityEvidenceIds.join(", ")+" · Sahiplik kanıtı: "+asset.ownershipHistory[0].evidenceIds.join(", "));
      var remove=add(c,"button","button secondary","Envanterden çıkar");remove.type="button";
      remove.addEventListener("click",function(){
        var removed=targetAssets[index].id;targetAssets.splice(index,1);
        targetIncidents=targetIncidents.filter(function(incident){return incident.assetId!==removed;});
        renderTargetAssets();renderTargetIncidents();renderTargetSelectors();updateTargetReadiness();
      });
    });
  }
  function renderTargetIncidents() {
    var host=byId("targetIncidentList");clear(host);
    if(!targetIncidents.length){host.appendChild(node("div","empty","Henüz olay kaydı yok."));}
    targetIncidents.forEach(function(incident,index){
      var asset=targetAssets.find(function(item){return item.id===incident.assetId;});
      var c=add(host,"article","item","");
      add(c,"h4","",date(incident.occurredAt)+" · "+(asset?asset.name:incident.assetId));
      pill(c,({claim_only:"Yalnızca iddia",attempt_reported:"Girişim bildirildi",intercepted:"Önleme bildirildi",impact_reported:"İsabet bildirildi",damage_reported:"Hasar bildirildi"})[incident.status]||incident.status);
      add(c,"p","","Aktör: "+incident.actor+" · "+incident.description);
      add(c,"p","muted small","Kaynak kanıtı: "+incident.evidenceIds.join(", "));
      var remove=add(c,"button","button secondary","Olay kaydını çıkar");remove.type="button";
      remove.addEventListener("click",function(){targetIncidents.splice(index,1);renderTargetIncidents();updateTargetReadiness();});
    });
  }
  function updateTargetReadiness() {
    var evidenceCount=(selectedCase&&selectedCase.evidence||[]).length;
    var valid=targetAssets.length>0 && evidenceCount>0 && byId("targetPeriodStart").value && byId("targetPeriodEnd").value;
    byId("saveTargetStudy").disabled=!valid;
    byId("targetStudyReadiness").textContent=valid
      ? targetAssets.length+" tesis · "+targetIncidents.length+" olay kaydı · kaynaklar bağlı; kapsamı kaydetmeden önce kontrol et."
      : "Kaynak kanıtı, geçerli dönem ve en az bir tesis gereklidir.";
  }
  function categoryText(value) {
    return ({foreign_participation:"Yabancı katılım var",no_foreign_participation:"Yabancı katılım yok",unknown:"Sınıflandırılamıyor"})[value]||value;
  }
  async function loadTargetStudyAnalysis(studyID) {
    if(!selectedCase)return;
    var host=byId("targetStudyReport");clear(host);host.appendChild(node("div","empty","Karşılaştırma raporu hesaplanıyor…"));
    try {
      var report=await api("/api/target-selection-analysis?caseId="+encodeURIComponent(selectedCase.id)+"&studyId="+encodeURIComponent(studyID));
      clear(host);
      var heading=add(host,"div","section-title","");add(heading,"h3","","Analiz raporu");
      var status=add(host,"p","muted small","Durum: "+report.comparisonStatus.replace(/_/g," "));
      if(report.descriptiveDifferencePercentagePoints!==undefined && report.descriptiveDifferencePercentagePoints!==null) {
        add(host,"div","notice","Betimsel oran farkı: "+Number(report.descriptiveDifferencePercentagePoints).toFixed(1)+" yüzde puan (yabancı katılım var eksi yabancı katılım yok). Bu, niyet veya nedensellik kanıtı değildir.");
      } else { add(host,"div","notice","Ana gruplar için karşılaştırmalı oran farkı hesaplanmadı. Durum ve uyarıları incele."); }
      var metrics=add(host,"div","metrics","");
      [[ "Evren tesisi",report.universeAssetCount ],[ "Sınıflandırılan",report.classifiedAssetCount ],[ "Sınıflandırılamayan",report.unclassifiedAssetCount ],[ "Olay kaydı",report.incidentRecordCount ],[ "Yalnızca iddia",report.claimOnlyRecordCount ]].forEach(function(pair){
        var m=add(metrics,"div","metric","");add(m,"strong","",pair[1]);add(m,"span","",pair[0]);
      });
      (report.groups||[]).forEach(function(group){
        var box=add(host,"article","item","");
        add(box,"h4","",categoryText(group.category));
        add(box,"p","","Tesisler: "+group.eligibleAssetCount+" · Kayıtlı hedefleme görülen tesis: "+group.assetsWithRecordedTargeting+" · Yalnızca iddia bulunan tesis: "+group.assetsWithClaimOnly);
        add(box,"p","","Gözlem kaydı: "+group.recordedTargetingIncidents+" · Yalnızca iddia kaydı: "+group.claimOnlyRecords+" · Kayıtlı hedefleme görülen tesis oranı: "+(group.rateAvailable?Number(group.targetedAssetRatePercent).toFixed(1)+"%":"hesaplanamadı"));
        if(group.smallDenominator)add(box,"p","muted small","Küçük payda: tek kayıt oranı ciddi biçimde değiştirebilir.");
      });
      (report.warnings||[]).forEach(function(note){add(host,"div","notice",note);});
      (report.interpretationLimitations||[]).forEach(function(note){add(host,"p","muted small",note);});
    } catch(err) { clear(host);host.appendChild(node("div","notice","Analiz alınamadı: "+err.message)); }
  }
  async function loadTargetStudies() {
    if(!selectedCase)return;
    var host=byId("targetStudyList");clear(host);
    try {
      var result=await api("/api/target-selection-studies?caseId="+encodeURIComponent(selectedCase.id));
      var items=result.items||[];
      if(!items.length){host.appendChild(node("div","empty","Bu araştırma dosyasında henüz hedef seçimi çalışması yok."));return;}
      items.slice().reverse().forEach(function(study){
        var c=add(host,"article","item","");
        add(c,"h4","",study.title);
        add(c,"p","muted small",study.question);
        add(c,"p","muted small",date(study.periodStart)+" – "+date(study.periodEnd)+" · "+(study.assets||[]).length+" tesis · "+(study.incidents||[]).length+" olay kaydı");
        var row=add(c,"div","row","");
        var analyze=add(row,"button","button","Analizi aç");analyze.type="button";analyze.addEventListener("click",function(){loadTargetStudyAnalysis(study.id);});
      });
    } catch(err) {host.appendChild(node("div","notice","Çalışmalar yüklenemedi: "+err.message));}
  }
  function renderTargetWorkspace() {
    if(!selectedCase)return;
    renderTargetSelectors();renderTargetAssets();renderTargetIncidents();updateTargetReadiness();loadTargetStudies();
  }
  function renderCurrentCase(){
    if(!selectedCase)return;
    byId("welcome").classList.add("hidden");byId("workspace").classList.remove("hidden");
    byId("activeTitle").textContent=selectedCase.title;byId("activeQuestion").textContent=selectedCase.question;
    byId("activeContext").textContent=selectedCase.context||"Başlangıç bağlamı ayrıca belirtilmedi.";
    byId("activeStatus").textContent=statusText(selectedCase.status);
    byId("refreshSources").classList.toggle("hidden",!selectedCase.relatedEventId);
    var summary=caseSummaries.find(function(s){return s.case.id===selectedCase.id;});
    renderMetrics(summary||{evidenceCount:(selectedCase.evidence||[]).length,hypothesisCount:(selectedCase.hypotheses||[]).length,openQuestionCount:(selectedCase.openQuestions||[]).filter(function(q){return q.status!=="resolved";}).length,publisherDomainCount:new Set((selectedCase.evidence||[]).map(function(e){return e.publisherDomain;})).size,limitations:[]});
    renderHypotheses();renderQuestions();renderEvidence();loadAnalysis();renderTargetWorkspace();
    renderCaseList();
  }
  async function loadCase(id){
    if(selectedCaseId!==id){targetAssets=[];targetIncidents=[];}
    var item=await api("/api/research-case?caseId="+encodeURIComponent(id));selectedCaseId=id;selectedCase=item;
    await api("/api/research-cases").then(function(result){caseSummaries=result.items||[];});
    renderCurrentCase();return item;
  }
  byId("targetAssetForm").addEventListener("submit",function(event){
    event.preventDefault();if(!selectedCase)return;
    var id=byId("targetAssetId").value.trim();
    if(targetAssets.some(function(asset){return asset.id===id;})){setMessage("Bu tesis kimliği zaten kullanılıyor.",true);return;}
    var validFrom=targetDateISO("targetOwnershipValidFrom");
    var validTo=targetDateISO("targetOwnershipValidTo");
    if(!validFrom){setMessage("Sahiplik dönemi başlangıcını gir.",true);return;}
    if(validTo && validTo<validFrom){setMessage("Sahiplik bitişi başlangıçtan önce olamaz.",true);return;}
    var eligibility=byId("targetAssetEligibilityEvidence").value;
    var ownershipEvidence=byId("targetAssetOwnershipEvidence").value;
    if(!eligibility||!ownershipEvidence){setMessage("Tesis ve sahiplik için kaynak kanıtlarını seç.",true);return;}
    targetAssets.push({
      id:id,name:byId("targetAssetName").value.trim(),assetType:byId("targetAssetType").value.trim(),
      inclusionRationale:byId("targetAssetInclusion").value.trim(),eligibilityEvidenceIds:[eligibility],
      ownershipHistory:[{
        validFrom:validFrom,validTo:validTo,category:byId("targetAssetCategory").value,
        details:byId("targetAssetOwnershipDetails").value.trim(),evidenceIds:[ownershipEvidence]
      }]
    });
    byId("targetAssetForm").reset();
    byId("targetAssetType").value="refinery";
    renderTargetAssets();renderTargetSelectors();updateTargetReadiness();
    setMessage("Tesis, uygunluk ve sahiplik kaynağıyla geçici envantere eklendi.",false);
  });
  byId("targetIncidentForm").addEventListener("submit",function(event){
    event.preventDefault();if(!selectedCase)return;
    var assetID=byId("targetIncidentAsset").value;
    var occurredAt=targetDateISO("targetIncidentDate");
    var evidenceID=byId("targetIncidentEvidence").value;
    if(!assetID||!occurredAt||!evidenceID){setMessage("Olay için tesis, tarih ve kanıt seçimi gereklidir.",true);return;}
    targetIncidents.push({
      id:"incident-"+Date.now()+"-"+(targetIncidents.length+1),assetId:assetID,
      actor:byId("targetIncidentActor").value.trim(),occurredAt:occurredAt,
      status:byId("targetIncidentStatus").value,
      description:byId("targetIncidentDescription").value.trim(),evidenceIds:[evidenceID]
    });
    byId("targetIncidentForm").reset();
    renderTargetIncidents();updateTargetReadiness();
    setMessage("Olay / iddia kaydı eklendi. Durum ayrı tutulacak.",false);
  });
  byId("saveTargetStudy").addEventListener("click",async function(event){
    event.preventDefault();if(!selectedCase)return;
    var button=byId("saveTargetStudy");button.disabled=true;
    try {
      var periodStart=targetDateISO("targetPeriodStart"),periodEnd=targetDateISO("targetPeriodEnd");
      if(!periodStart||!periodEnd||periodEnd<periodStart)throw new Error("Araştırma dönemi geçersiz.");
      if(!targetAssets.length)throw new Error("En az bir kaynakla desteklenmiş tesis ekle.");
      var input={
        title:byId("targetStudyTitle").value.trim(),question:byId("targetStudyQuestion").value.trim(),
        periodStart:periodStart,periodEnd:periodEnd,
        universeDefinition:byId("targetUniverseDefinition").value.trim(),
        universeComplete:byId("targetUniverseComplete").checked,
        assets:targetAssets,incidents:targetIncidents
      };
      var saved=await api("/api/target-selection-studies?caseId="+encodeURIComponent(selectedCase.id),{method:"POST",body:JSON.stringify(input)});
      selectedCase=await api("/api/research-case?caseId="+encodeURIComponent(selectedCase.id));
      targetAssets=[];targetIncidents=[];
      renderCurrentCase();
      await loadTargetStudyAnalysis(saved.id);
      setMessage("Hedef seçimi çalışması kaydedildi. Rapor, evren ve kanıt kapsamı sınırlarını koruyarak oluşturuldu.",false);
    } catch(err) {setMessage(err.message,true);}
    finally {button.disabled=false;updateTargetReadiness();}
  });
  byId("refreshTargetStudies").addEventListener("click",function(){loadTargetStudies();});
  ["targetPeriodStart","targetPeriodEnd","targetUniverseComplete"].forEach(function(id){
    byId(id).addEventListener("change",updateTargetReadiness);
  });
  byId("analysisRefresh").addEventListener("click",function(){loadAnalysis();});
  byId("refreshSources").addEventListener("click",async function(){
    if(!selectedCase||!selectedCase.relatedEventId)return;
    var button=byId("refreshSources");button.disabled=true;
    try{
      var result=await api("/api/research-import?caseId="+encodeURIComponent(selectedCase.id),{method:"POST"});
      var caseID=selectedCase.id;selectedCase=result.case;await refreshCaseList();await loadCase(caseID);
      setMessage("NexNews kaynak eşitlemesi: "+result.stats.evidenceImported+" yeni, "+result.stats.unchangedSkipped+" aynı sürüm, "+result.stats.revisionsAdded+" yeni sürüm, "+result.stats.articlesSkipped+" atlandı.",false);
    }catch(err){setMessage(err.message,true);}finally{button.disabled=false;}
  });
  byId("caseForm").addEventListener("submit",async function(event){
    event.preventDefault();var button=byId("createCaseButton");button.disabled=true;
    try{
      var item=await api("/api/research-cases",{method:"POST",body:JSON.stringify({title:byId("caseTitle").value,question:byId("caseQuestion").value,context:byId("caseContext").value,relatedEventId:linkedEventId})});
      linkedEventId="";window.history.replaceState(null,"",window.location.pathname);
      byId("caseForm").reset();selectedCaseId=item.id;selectedCase=item;
      var importMessage="";
      if(item.relatedEventId){
        try{
          var imported=await api("/api/research-import?caseId="+encodeURIComponent(item.id),{method:"POST"});
          item=imported.case;selectedCase=item;
          importMessage=" NexNews haber kaydı: "+imported.stats.evidenceImported+" yeni, "+imported.stats.unchangedSkipped+" aynı sürüm, "+imported.stats.revisionsAdded+" yeni sürüm.";
        }catch(importErr){importMessage=" NexNews kaynak aktarımı tamamlanamadı: "+importErr.message;}
      }
      await refreshCaseList();await loadCase(item.id);setMessage("Araştırma dosyası oluşturuldu."+importMessage, importMessage.indexOf("tamamlanamadı")>=0);
    }catch(err){setMessage(err.message,true);}finally{button.disabled=false;}
  });
  byId("hypothesisForm").addEventListener("submit",async function(event){
    event.preventDefault();if(!selectedCase)return;
    var button=event.submitter;button.disabled=true;
    try{var item=await api("/api/research-hypothesis?caseId="+encodeURIComponent(selectedCase.id),{method:"POST",body:JSON.stringify({statement:byId("hypothesisText").value})});selectedCase=item;byId("hypothesisText").value="";await refreshCaseList();renderCurrentCase();setMessage("Hipotez kaydedildi.",false);}
    catch(err){setMessage(err.message,true);}finally{button.disabled=false;}
  });
  byId("questionForm").addEventListener("submit",async function(event){
    event.preventDefault();if(!selectedCase)return;
    var button=event.submitter;button.disabled=true;
    try{var item=await api("/api/research-question?caseId="+encodeURIComponent(selectedCase.id),{method:"POST",body:JSON.stringify({question:byId("questionText").value})});selectedCase=item;byId("questionText").value="";await refreshCaseList();renderCurrentCase();setMessage("Açık soru hafızaya kaydedildi.",false);}
    catch(err){setMessage(err.message,true);}finally{button.disabled=false;}
  });
  byId("evidenceForm").addEventListener("submit",async function(event){
    event.preventDefault();if(!selectedCase)return;
    var button=byId("saveEvidenceButton");button.disabled=true;
    try{
      var hypothesisId=byId("assessedHypothesis").value;
      var assessments=[];
      if(hypothesisId){
        var rationale=byId("assessmentRationale").value.trim();
        if(!rationale)throw new Error("Hipotezle ilişki kurmak için gerekçe yaz.");
        assessments.push({hypothesisId:hypothesisId,direction:byId("assessmentDirection").value,rationale:rationale});
      }
      var rawDate=byId("publishedAt").value;
      var publishedAt=rawDate?new Date(rawDate).toISOString():null;
      var input={
        kind:byId("evidenceKind").value,title:byId("evidenceTitle").value,sourceUrl:byId("evidenceURL").value,
        publisher:byId("publisher").value,author:byId("author").value,publishedAt:publishedAt,
        excerpt:byId("excerpt").value,contentLevel:byId("contentLevel").value,sourceKind:byId("sourceKind").value,
        sourceOrigin:byId("sourceOrigin").value,syndicationGroup:byId("syndicationGroup").value,
        methodNote:byId("methodNote").value,hypothesisAssessments:assessments
      };
      var item=await api("/api/research-evidence?caseId="+encodeURIComponent(selectedCase.id),{method:"POST",body:JSON.stringify(input)});
      selectedCase=item;byId("evidenceForm").reset();await refreshCaseList();renderCurrentCase();setMessage("Kaynak ve kanıt araştırma dosyasına eklendi.",false);
    }catch(err){setMessage(err.message,true);}finally{button.disabled=false;}
  });
  byId("assessedHypothesis").addEventListener("change",function(){
    byId("assessmentRationale").required=Boolean(this.value);
  });
  if(linkedEventId){
    api("/api/event-dossier?eventId="+encodeURIComponent(linkedEventId)).then(function(d){
      byId("caseTitle").value="Araştırma: "+d.event.title;
      byId("caseQuestion").value="Bu olayda hangi iddialar destekleniyor, anlatılar nerede ayrışıyor ve hangi alternatif açıklamalar verilerle sınanabilir?";
      byId("caseContext").value="NexNews olay adayı: "+d.event.title+"\nBağlı haber sayısı: "+d.articleCount+"\nNot: Olay kümesi doğruluk hükmü değildir. Olayın tarihi, kaynak kökeni, kapsamı ve karşı kanıtlar ayrıca değerlendirilmelidir.";
      setMessage("NexNews olay dosyası araştırma başlangıcı olarak yüklendi. Araştırma dosyası oluşturduğunda olay bağlantısı korunacak.",false);
    }).catch(function(){
      linkedEventId="";
      setMessage("İstenen NexNews olay dosyası okunamadı. Yeni araştırma yine bağımsız oluşturulabilir.",true);
    });
  }
  refreshCaseList().catch(function(err){byId("caseList").replaceChildren(node("div","empty","Araştırma veritabanı okunamadı: "+err.message));setMessage(err.message,true);});
})();
</script>
</body></html>`