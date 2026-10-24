  // ---------- gelişmiş arama ----------
  async function doSearch(q){
    q = (q||"").trim();
    if(!q){ backToOverview(); return; }
    let r; try{ r = JSON.parse(await window.nnSearch(q)); }catch(e){ return; }
    document.getElementById("overview").classList.add("hide");
    const p = document.getElementById("catpage");
    p.classList.remove("hide");
    let h = '<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button>'
          + '<h1>Ara: '+esc(r.query)+'</h1>'
          + '<span class="pill">'+r.total+' olay</span>'
          + (r.windowLabel?'<span class="pill">'+esc(r.windowLabel)+'</span>':'')
          + '<span class="pill">'+r.sourceCount+' kaynak</span></div>';
    if(!r.items || !r.items.length){
      h += '<div class="empty">Sonuç bulunamadı. Farklı terim ya da zaman aralığı deneyin.</div>';
    }else{
      indexSummaries(r.items);
      h += '<div class="list">';
      for(const it of r.items) h += itemHTML(it);
      h += '</div>';
    }
    p.innerHTML = h;
    window.scrollTo(0,0);
  }

  async function doDeepSearch(q){
    q=(q||"").trim(); if(!q){doSearch("");return;}
    let r; try{r=JSON.parse(await window.nnDeepSearch(q));}catch(e){return;}
    document.getElementById("overview").classList.add("hide");
    const p=document.getElementById("catpage"); p.classList.remove("hide");
    let h='<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button><h1>🧠 Derin Ara</h1><span class="pill">'+r.total+' olay</span>'+(r.windowLabel?'<span class="pill">'+esc(r.windowLabel)+'</span>':'')+'<span class="pill">'+r.sourceCount+' kaynak</span></div>';
    h+='<section class="deep-card"><div class="deep-head"><h2>'+esc(r.query)+'</h2><span class="pill">Araştırma görünümü</span></div><div class="deep-note">Sonuçlar olay analizi, kaynak mutabakatı ve kişi/kurum ilişkileri üzerinden özetlenmiştir.</div>';
    h+='<div class="deep-grid"><div class="deep-box"><h3>Öne çıkan varlıklar</h3><div class="deep-chips">';
    if(r.entities&&r.entities.length){r.entities.forEach(e=>{h+='<span class="deep-chip '+esc(e.type)+'">'+esc(e.name)+' <b>'+e.count+'</b></span>';});}else h+='<span class="deep-empty">Varlık sinyali bulunamadı.</span>';
    h+='</div></div><div class="deep-box"><h3>Kaynak yoğunluğu</h3>';
    if(r.sources&&r.sources.length){r.sources.forEach(x=>{h+='<div class="deep-source"><span>'+esc(x.name)+'</span><span>'+x.count+' haber</span></div>';});}else h+='<div class="deep-empty">Kaynak verisi yok.</div>';
    h+='</div></div><div class="deep-box deep-events"><h3>Önemli olaylar</h3>';
    if(r.events&&r.events.length){r.events.forEach(e=>{h+='<div class="deep-event" onclick="openEvent(\''+String(e.link||'').replace(/\\/g,'\\\\').replace(/'/g,"\\'")+'\')"><div class="deep-event-title">'+esc(e.title)+'</div><div class="deep-event-meta"><span>Önem '+e.score+'</span><span>Güven '+e.confidence+'%</span><span>'+e.sources+' bağımsız kaynak</span><span>'+esc(e.lifecycle||'')+'</span></div>'+(e.reason?'<div class="deep-event-reason">'+esc(e.reason)+'</div>':'')+'</div>';});}else h+='<div class="deep-empty">Sonuç bulunamadı.</div>';
    h+='</div></section>';
    if(r.events&&r.events.length){indexSummaries(r.events.map(e=>({title:e.title,link:e.link,summary:e.reason||""})));}
    h+='<div class="list">';
    if(r.events&&r.events.length){r.events.forEach(e=>{h+='<div class="item"><div class="item-main"><div class="item-title">'+esc(e.title)+'</div><div class="item-meta">'+esc(e.category||'')+'</div></div></div>';});}
    h+='</div>'; p.innerHTML=h; window.scrollTo(0,0);
  }

  // ---------- open actions ----------
  function openItem(url){ if(CONFIG && CONFIG.openInApp) readItem(url); else browserItem(url); }
  function browserItem(url){ recordHistory(url); window.nnOpen(url); }

  // ---------- reader ----------
  function readItem(url){
    currentReadURL = url;
    currentReadSummary = SUMMARIES[url] || "";
    recordHistory(url);
    document.getElementById("readerOverlay").classList.add("open");
    document.getElementById("reader").classList.add("open");
    updateReadState();
    document.getElementById("rSource").textContent = "Yükleniyor…";
    document.getElementById("rScroll").innerHTML =
      '<div class="r-spin"><div class="ring"></div><div>Haber getiriliyor…</div></div>';
    document.getElementById("rOpenBtn").onclick = ()=>window.nnOpen(url);
    window.nnRead(url);
  }
  function readerTrBar(){
    // Okuyucu başına çeviri kontrolü: tıklanan haberi görünür durumla çevirir.
    return '<div class="r-trbar"><button class="mini" id="rTrBtn" onclick="translateReader()">🌐 Bu haberi çevir</button>'
         + '<span id="rTrInfo" class="r-trinfo"></span></div>';
  }
  window.nnOnArticle = function(a){
    if(!a || a.url !== currentReadURL) return; // ignore stale
    document.getElementById("rSource").textContent = a.title || "Okuyucu";
    const scroll = document.getElementById("rScroll");
    if(a.err && !a.html){
      const sum = currentReadSummary;
      scroll.innerHTML = readerTrBar()
        + (a.title?'<h1 class="art">'+esc(a.title)+'</h1>':'')
        + (sum ? '<div class="article-body"><p>'+esc(sum)+'</p></div>' : '')
        + '<div class="r-note">'+esc(a.err)+'</div>'
        + '<div style="margin-top:14px"><button class="btn primary" data-act="openurl" data-url="'+encodeURIComponent(a.url)+'">Tarayıcıda Aç</button></div>';
      if(CONFIG && CONFIG.translate) translateReader();
      return;
    }
    scroll.innerHTML = readerTrBar()
      + (a.title?'<h1 class="art">'+esc(a.title)+'</h1>':'')
      + '<div class="article-body">'+a.html+'</div>';
    scroll.scrollTop = 0;
    if(CONFIG && CONFIG.translate) translateReader();
  };
  // Tıklanan haberin başlığını ve gövdesini yerinde çevirir; hangi kaynağın
  // çalıştığını ya da tam hatayı okuyucuda gösterir.
  async function translateReader(){
    const info = document.getElementById("rTrInfo");
    const btn = document.getElementById("rTrBtn");
    const target = trTarget();
    if(!target){ if(info) info.textContent = "hedef dil yok"; return; }
    const blocks = Array.from(document.querySelectorAll(
      "#rScroll h1.art, #rScroll .article-body p, #rScroll .article-body li, #rScroll .article-body h2, #rScroll .article-body h3"))
      .filter(n=>!n.hasAttribute("data-trd") && n.textContent.trim().length>1);
    if(!blocks.length){ if(info) info.textContent = "çevrilecek metin yok"; return; }
    blocks.forEach(n=>n.setAttribute("data-trd","1"));
    if(btn) btn.disabled = true;
    if(info) info.textContent = "çevriliyor… ("+target+")";
    let via=null, err=null, done=0, i=0; const CONC=4;
    async function worker(){
      while(i<blocks.length){
        const n=blocks[i++], orig=n.textContent;
        try{ const r = await trFetch(orig, target);
          if(r.out){ n.textContent = r.out; via = r.via; done++; } else { err = r.err || err; }
        }catch(e){ err = (e&&e.message)||err; }
      }
    }
    await Promise.all(Array.from({length:Math.min(CONC,blocks.length)}, worker));
    if(btn) btn.disabled = false;
    if(info) info.textContent = done ? ("✓ çeviri — "+via+" ("+done+" bölüm)") : ("⚠ çeviri başarısız ("+(err||"ağ")+")");
  }
  function updateReadState(){
    const el=document.getElementById("readState");
    if(!el) return;
    const fs=Math.round(((CONFIG&&CONFIG.fontScale)||100));
    const lh=Number((CONFIG&&CONFIG.lineHeight)||1.75).toFixed(2);
    el.textContent="%"+fs+" · satır "+lh;
  }
  function readerScale(delta){
    CONFIG=CONFIG||{}; CONFIG.fontScale=Math.max(80,Math.min(160,(CONFIG.fontScale||100)+delta));
    applyGlass(); updateReadState();
    try{ window.nnSaveConfig(JSON.stringify(CONFIG)); }catch(e){}
  }
  function readerLine(delta){
    CONFIG=CONFIG||{}; CONFIG.lineHeight=Math.max(1.4,Math.min(2.4,Number((CONFIG.lineHeight||1.75))+delta));
    applyGlass(); updateReadState();
    try{ window.nnSaveConfig(JSON.stringify(CONFIG)); }catch(e){}
  }
  function toggleReaderFocus(){
    const r=document.documentElement; r.classList.toggle("reader-focus");
    updateReadState();
  }

  function closeReader(){
    currentReadURL = null;
    document.getElementById("readerOverlay").classList.remove("open");
    document.getElementById("reader").classList.remove("open");
  }

  // ---------- refresh ----------
  function doRefresh(){
    const b = document.getElementById("refreshBtn");
    b.classList.add("loading");
    window.nnRefresh();
    setTimeout(()=>b.classList.remove("loading"), 20000); // safety
  }
  // Go pushes this when a background fetch finishes.
  window.nnOnData = function(){
    document.getElementById("refreshBtn").classList.remove("loading");
    loadOverview();
    refreshVisibleTab(); // yalnızca görünen sağ panel sekmesi
  };

  // ---------- başlık çevirisi (tarayıcı tarafı) ----------
  // Go tarafı bazı ev ISP'lerinde çeviri uçlarına ulaşamıyor; WebView2'nin
  // tarayıcı ağ yığını ise dış hostlara erişebiliyor (radar bunu kanıtladı).
  // Bu yüzden görünen başlıkları doğrudan tarayıcıdan, yedekli kaynaklarla
  // çeviriyoruz. Sonuçlar bellek içinde önbelleğe alınır.
  let TR_SYS = "";
  const TRCACHE = new Map();
  function normalizeLocalLang(v){
    v=String(v||"").trim().toLowerCase().replace('_','-');
    if(!v) return "";
    return v.split('-')[0];
  }
  function localLanguage(){
    const configured=normalizeLocalLang(CONFIG&&CONFIG.translateLang);
    if(configured) return configured;
    const system=normalizeLocalLang(TR_SYS);
    if(system) return system;
    return normalizeLocalLang((navigator&&navigator.language)||"") || "en";
  }
  function trTarget(){ return localLanguage(); }
  function applyDocumentLanguage(){ applyUILang(); }
  // Arayüz dili: ayarda seçilen, yoksa sistem dili. Sözlüğü olmayan diller İngilizce.
  function applyUILang(){
    try{ window.NexI18n && NexI18n.setLang((CONFIG&&CONFIG.uiLang)||TR_SYS||navigator.language||'tr'); }catch(e){}
  }
  function uiLocale(){ try{ return NexI18n.locale(); }catch(e){ return 'tr-TR'; } }
  function trParseGTX(d){ try{ return (d[0]||[]).map(s=>s[0]).join(""); }catch(e){ return null; } }
  async function trFetch(text, target){
    const q = encodeURIComponent(text), tl = encodeURIComponent(target), p = encodeURIComponent(text);
    // Tarayıcı fetch'i CORS'a tabi: ACAO:* gönderen kaynaklar (Lingva,
    // SimplyTranslate) önce denenir; Google uçları CORS göndermediği için
    // genelde burada okunamaz, yine de son çare olarak bırakıldı.
    const tries = [
      // Lingva/SimplyTranslate: kaynak dilini otomatik algılar (her dil için
      // doğru), ama gönüllü aynalar sık sık çöker.
      ["Lingva (lunar)",      "https://lingva.lunar.icu/api/v1/auto/"+tl+"/"+p, "lingva"],
      ["Lingva (plausibility)","https://translate.plausibility.cloud/api/v1/auto/"+tl+"/"+p, "lingva"],
      ["Lingva (garuda)",     "https://lingva.garudalinux.org/api/v1/auto/"+tl+"/"+p, "lingva"],
      ["SimplyTranslate",     "https://simplytranslate.org/api/translate/?engine=google&from=auto&to="+tl+"&text="+q, "simply"],
      ["SimplyTranslate (tokhmi)","https://st.tokhmi.xyz/api/translate/?engine=google&from=auto&to="+tl+"&text="+q, "simply"],
      // MyMemory: ticari destekli, kesintisiz ve CORS'lu — aynalar düşse de
      // ayakta kalır. Kaynak dili şart koşar; uluslararası kaynakların çoğu
      // İngilizce olduğundan en|hedef güçlü bir varsayım.
      ["MyMemory (en→"+target+")", "https://api.mymemory.translated.net/get?q="+q+"&langpair=en%7C"+tl, "mymemory"],
      // Google: anahtarsız ama CORS göndermez; genelde tarayıcıda okunamaz,
      // yine de son çare.
      ["Google (googleapis)", "https://translate.googleapis.com/translate_a/single?client=gtx&sl=auto&tl="+tl+"&dt=t&q="+q, "gtx"],
    ];
    let lastErr = "ağ";
    for(const [name,url,kind] of tries){
      try{
        const r = await fetch(url, {cache:"no-store"});
        if(!r.ok){ lastErr = "HTTP "+r.status; continue; }
        const d = await r.json();
        let out = null;
        if(kind==="gtx") out = trParseGTX(d);
        else if(kind==="lingva") out = d && d.translation;
        else if(kind==="mymemory") out = d && d.responseData && d.responseData.translatedText;
        else out = d && d["translated-text"];
        // MyMemory bazen hata metnini translatedText içinde döndürür
        if(kind==="mymemory" && out && /MYMEMORY WARNING|INVALID|QUERY LENGTH/i.test(out)){ out=null; }
        if(out && out.trim()){ return {out:out, via:name}; }
      }catch(e){ lastErr = (e && e.message) || "engel"; }
    }
    return {out:null, via:null, err:lastErr};
  }
  async function jsTranslateOne(text, target){
    const key = target+"\x00"+text;
    if(TRCACHE.has(key)) return TRCACHE.get(key);
    const r = await trFetch(text, target);
    if(r.out){ TRCACHE.set(key, r.out); return r.out; }
    return null;
  }
  async function translateDOM(){
    // Headline translation is supplied by the Go Translation Manager.
    // Do not call public browser translation endpoints here: they are rate-limited
    // and would duplicate the local LibreTranslate/Argos result.
    return;
    /*
    if(!CONFIG || !CONFIG.translate) return;
    const target = trTarget(); if(!target) return;
    const nodes = Array.from(document.querySelectorAll(
      ".t-tx:not([data-trd]), .tk-ttl:not([data-trd]), .bb-item .t:not([data-trd])"));
    if(!nodes.length) return;
    nodes.forEach(n=>n.setAttribute("data-trd","1"));
    let i=0; const CONC=5;
    async function worker(){
      while(i<nodes.length){
        const n = nodes[i++]; const orig = n.textContent;
        try{ const out = await jsTranslateOne(orig, target); if(out && out!==orig) n.textContent = out; }catch(e){}
      }
    }
    await Promise.all(Array.from({length:Math.min(CONC,nodes.length)}, worker));
    */
  }

