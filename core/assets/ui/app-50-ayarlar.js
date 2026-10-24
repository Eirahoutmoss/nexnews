  // ---------- settings ----------
  const NEWS_COUNTRIES = [
    ["TR","🇹🇷 Türkiye"],["TD","🇹🇩 Çad"],["DE","🇩🇪 Almanya"],["FR","🇫🇷 Fransa"],["GB","🇬🇧 Birleşik Krallık"],["US","🇺🇸 Amerika Birleşik Devletleri"],["CA","🇨🇦 Kanada"],["MX","🇲🇽 Meksika"],["BR","🇧🇷 Brezilya"],["AR","🇦🇷 Arjantin"],["CL","🇨🇱 Şili"],["CO","🇨🇴 Kolombiya"],["PE","🇵🇪 Peru"],["AU","🇦🇺 Avustralya"],["NZ","🇳🇿 Yeni Zelanda"],["JP","🇯🇵 Japonya"],["KR","🇰🇷 Güney Kore"],["CN","🇨🇳 Çin"],["IN","🇮🇳 Hindistan"],["PK","🇵🇰 Pakistan"],["ID","🇮🇩 Endonezya"],["MY","🇲🇾 Malezya"],["SG","🇸🇬 Singapur"],["TH","🇹🇭 Tayland"],["VN","🇻🇳 Vietnam"],["PH","🇵🇭 Filipinler"],["RU","🇷🇺 Rusya"],["UA","🇺🇦 Ukrayna"],["PL","🇵🇱 Polonya"],["IT","🇮🇹 İtalya"],["ES","🇪🇸 İspanya"],["NL","🇳🇱 Hollanda"],["BE","🇧🇪 Belçika"],["SE","🇸🇪 İsveç"],["NO","🇳🇴 Norveç"],["DK","🇩🇰 Danimarka"],["FI","🇫🇮 Finlandiya"],["GR","🇬🇷 Yunanistan"],["BG","🇧🇬 Bulgaristan"],["RO","🇷🇴 Romanya"],["RS","🇷🇸 Sırbistan"],["IL","🇮🇱 İsrail"],["PS","🇵🇸 Filistin"],["SA","🇸🇦 Suudi Arabistan"],["AE","🇦🇪 Birleşik Arap Emirlikleri"],["IR","🇮🇷 İran"],["IQ","🇮🇶 Irak"],["SY","🇸🇾 Suriye"],["EG","🇪🇬 Mısır"],["DZ","🇩🇿 Cezayir"],["MA","🇲🇦 Fas"],["TN","🇹🇳 Tunus"],["LY","🇱🇾 Libya"],["ET","🇪🇹 Etiyopya"],["KE","🇰🇪 Kenya"],["NG","🇳🇬 Nijerya"],["ZA","🇿🇦 Güney Afrika"],["GH","🇬🇭 Gana"],["SN","🇸🇳 Senegal"],["CM","🇨🇲 Kamerun"],["SD","🇸🇩 Sudan"],["TZ","🇹🇿 Tanzanya"],["UG","🇺🇬 Uganda"],["RW","🇷🇼 Ruanda"],["QA","🇶🇦 Katar"],["KW","🇰🇼 Kuveyt"],["BH","🇧🇭 Bahreyn"],["OM","🇴🇲 Umman"],["JO","🇯🇴 Ürdün"],["LB","🇱🇧 Lübnan"],["AZ","🇦🇿 Azerbaycan"],["GE","🇬🇪 Gürcistan"],["AM","🇦🇲 Ermenistan"],["KZ","🇰🇿 Kazakistan"],["UZ","🇺🇿 Özbekistan"],["KG","🇰🇬 Kırgızistan"],["TM","🇹🇲 Türkmenistan"],["IE","🇮🇪 İrlanda"],["PT","🇵🇹 Portekiz"],["AT","🇦🇹 Avusturya"],["CH","🇨🇭 İsviçre"],["CZ","🇨🇿 Çekya"],["HU","🇭🇺 Macaristan"],["CY","🇨🇾 Kıbrıs"],["BA","🇧🇦 Bosna-Hersek"],["AL","🇦🇱 Arnavutluk"],["MK","🇲🇰 Kuzey Makedonya"],["TW","🇹🇼 Tayvan"],["HK","🇭🇰 Hong Kong"],["BD","🇧🇩 Bangladeş"],["AF","🇦🇫 Afganistan"],["SO","🇸🇴 Somali"],["VE","🇻🇪 Venezuela"]
  ];
  function renderNewsCountryOptions(){
    const el=document.getElementById("optNewsCountry"); if(!el) return;
    el.innerHTML='<option value="">🌍 Tüm Dünya</option>'+NEWS_COUNTRIES.map(x=>'<option value="'+x[0]+'">'+esc(x[1])+'</option>').join('');
    el.value=(CONFIG&&CONFIG.newsCountry)||"";
  }

  async function openSettings(sec){
    try{ CONFIG = JSON.parse(await window.nnGetConfig()); }catch(e){ CONFIG = CONFIG||{categories:[]}; }
    showSetSec(typeof sec==="string" ? sec : "gorunum");
    // RC81 alanları
    syncSeg('optDensity', CONFIG.density||'rahat');
    bindSeg('optDensity', v=>{ CONFIG.density=v; applyGlass(); });
    document.getElementById('optNetMeter').checked = CONFIG.netMeter!==false;
    document.getElementById('optQuiet').checked = !!CONFIG.quietHours;
    document.getElementById('optQuietFrom').value = CONFIG.quietFrom||'23:00';
    document.getElementById('optQuietTo').value = CONFIG.quietTo||'07:00';
    document.getElementById('optWSingle').checked = !!CONFIG.widgetSingle;
    syncSeg('optWMode', CONFIG.widgetMode||'cam');
    bindSeg('optWMode', v=>{ CONFIG.widgetMode=v; syncTintUI(); pushWidgetLook(); });
    const wt=document.getElementById('optWTint'), wtv=document.getElementById('optWTintVal');
    wt.value = CONFIG.widgetTint!=null ? CONFIG.widgetTint : 45; wtv.textContent='%'+wt.value;
    wt.oninput=()=>{ CONFIG.widgetTint=+wt.value; wtv.textContent='%'+wt.value; pushWidgetLook(); };
    const syncTintUI=()=>{ const m=CONFIG.widgetMode||'cam'; const on=(m==='cam'||m==='saydam'); wt.disabled=!on; document.getElementById('optWTintLbl').style.opacity=on?1:.45; document.getElementById('optWTintLbl').textContent = m==='saydam'?'Gölge tonu':'Cam tonu'; };
    syncTintUI();
    document.getElementById('optWSingle').onchange=e=>{ CONFIG.widgetSingle=e.target.checked; pushWidgetLook(); };
    updateWidgetPreview();
    loadSrcHealthMap().then(()=>{ if(document.getElementById("settingsOverlay").classList.contains("open")) renderCatsEditor(); });
    document.getElementById("optMax").value = CONFIG.maxPerCategory||5;
    renderNewsCountryOptions();
    document.getElementById("optCountryScope").value = CONFIG.countryScope || "related";
    document.getElementById("optInApp").checked = !!CONFIG.openInApp;
    document.getElementById("optRefresh").value = CONFIG.refreshMinutes||0;
    // cam kontrolleri
    const ul=document.getElementById('optUILang'); if(ul){ ul.value=CONFIG.uiLang||''; ul.onchange=()=>{ CONFIG.uiLang=ul.value; applyUILang(); }; }
    const th = document.getElementById("optTheme");
    const go = document.getElementById("optGlassOp");
    const gv = document.getElementById("optGlassVal");
    const gw = document.getElementById("optGlassWrap");
    th.value = (CONFIG.theme || "premium");
    go.value = CONFIG.glassOpacity||72;
    gv.textContent = "%"+go.value;
    const syncGW = ()=>{ gw.style.display = (th.value==="premium") ? "" : "none"; };
    syncGW();
    th.onchange = ()=>{ CONFIG.theme = th.value; applyGlass(); syncGW(); };
    go.oninput = ()=>{ CONFIG.glassOpacity = +go.value; gv.textContent="%"+go.value; applyGlass(); };
    const fs=document.getElementById("optFontScale"), fsv=document.getElementById("optFontScaleVal");
    const lh=document.getElementById("optLineHeight"), lhv=document.getElementById("optLineHeightVal");
    const rw=document.getElementById("optReaderWidth"), dys=document.getElementById("optDyslexia");
    const wo=document.getElementById("optWidgetOpacity"), wov=document.getElementById("optWidgetOpacityVal");
    fs.value=CONFIG.fontScale||100; fsv.textContent="%"+fs.value;
    lh.value=CONFIG.lineHeight||1.75; lhv.textContent=Number(lh.value).toFixed(2);
    rw.value=CONFIG.readerWidth||760; dys.checked=!!CONFIG.dyslexiaFont;
    wo.value=(CONFIG.widgetOpacity!=null?CONFIG.widgetOpacity:78); wov.textContent="%"+wo.value;
    fs.oninput=()=>{ CONFIG.fontScale=+fs.value; fsv.textContent="%"+fs.value; applyGlass(); updateReadState(); };
    lh.oninput=()=>{ CONFIG.lineHeight=+lh.value; lhv.textContent=Number(lh.value).toFixed(2); applyGlass(); updateReadState(); };
    rw.onchange=()=>{ CONFIG.readerWidth=+rw.value; applyGlass(); };
    dys.onchange=()=>{ CONFIG.dyslexiaFont=dys.checked; applyGlass(); };
    wo.oninput=()=>{ CONFIG.widgetOpacity=+wo.value; wov.textContent="%"+wo.value; pushWidgetLook(); };
    // çeviri kontrolleri
    const tr = document.getElementById("optTr");
    const trl = document.getElementById("optTrLang");
    const tri = document.getElementById("optTrInfo");
    tr.checked = !!CONFIG.translate;
    trl.value = CONFIG.translateLang || "";
    let sys = "";
    try{ sys = await window.nnSysLang(); }catch(e){}
    const showTrInfo = (msg)=>{ tri.textContent = msg || ("etkin dil: " + (normalizeLocalLang(trl.value)||normalizeLocalLang(sys)||normalizeLocalLang((navigator&&navigator.language)||"")||"en")); };
    showTrInfo();
    const applyTr = async ()=>{ CONFIG.translate=tr.checked; CONFIG.translateLang=trl.value.trim();
      // Ayarı Go tarafına da bildir; başlıklar gerektiğinde WebView2'den
      // kontrollü olarak çevrilebilir.
      try{ await window.nnSetTranslate(tr.checked, trl.value.trim()); }catch(e){}
      if(!tr.checked){ showTrInfo("çeviri kapalı"); return; }
      const target = trTarget();
      showTrInfo("çeviri sınanıyor… ("+(target||"?")+")");
      const r = await trFetch("Breaking news", target);
      if(r.out){ showTrInfo("✓ çalışıyor — "+r.via+" · “Breaking news” → “"+r.out+"”"); }
      else { showTrInfo("⚠ çeviri kaynaklarına ulaşılamadı ("+(r.err||"ağ")+") — başlıklar özgün dilde kalır"); }
      // Açık sayfadaki başlıkları hemen çevir (yeniden yüklemeye gerek yok)
      document.querySelectorAll("[data-trd]").forEach(n=>n.removeAttribute("data-trd"));
      translateDOM();
    };
    tr.onchange = applyTr;
    trl.onchange = ()=>{ applyDocumentLanguage(); applyTr(); };
    // hava durumu
    document.getElementById("optWx").checked = (CONFIG.weather !== false);
    document.getElementById("optInterests").value = (CONFIG.interests||[]).join(", ");
    document.getElementById("optHide").value = (CONFIG.hideWords||[]).join(", ");
    document.getElementById("optMarketWatch").value = (CONFIG.marketWatch||[]).join(", ");
    document.getElementById("optSportsWatch").value = (CONFIG.sportsWatch||[]).join(", ");
    document.getElementById("optSportsTeams").value = (CONFIG.sportsTeams||[]).join(", ");
    document.getElementById("optNotify").checked = (CONFIG.notify !== false);
    document.getElementById("optNotifyMin").value = CONFIG.notifyMinScore || 75;
    // widget içeriği
    document.getElementById("optWImp").checked = !!CONFIG.widgetImpOnly;
    document.getElementById("optWSlide").value = String(CONFIG.widgetSlideSec || 3);
    const wsel = document.getElementById("widgetCatSel");
    const chosen = CONFIG.widgetCats || [];
    wsel.innerHTML = (CONFIG.categories||[]).map(c=>
      '<label style="font-size:13px;color:var(--txt-dim)"><input type="checkbox" class="wcat" value="'+esc(c.name)+'"'
      + (chosen.indexOf(c.name)>=0?' checked':'') + '> '+esc(c.name)+'</label>').join("");
    renderCatsEditor();
    document.getElementById("settingsOverlay").classList.add("open");
  }
  function closeSettings(){
    const ov=document.getElementById("settingsOverlay");
    if(!ov.classList.contains("open")) return;
    ov.classList.remove("open");
    // Kaydedilmeden kapatıldıysa önizleme amaçlı canlı değişiklikleri geri al.
    window.nnGetConfig().then(j=>{ try{ CONFIG=JSON.parse(j); applyGlass(); applyUILang(); }catch(e){} }).catch(()=>{});
  }

  let openSourceCat = -1;
  function toggleCatEditor(ci){
    openSourceCat = (openSourceCat === ci) ? -1 : ci;
    renderCatsEditor();
  }
  function renderCatsEditor(){
    const box = document.getElementById("catsEditor");
    box.innerHTML = "";
    (CONFIG.categories||[]).forEach((cat,ci)=>{
      const d = document.createElement("div");
      const open = openSourceCat === ci;
      d.className = "seccat" + (open ? " open" : "");
      const count = (cat.sources||[]).length;
      let srcs = open ? (cat.sources||[]).map((s,si)=>
        '<div class="srcrow">'
        + healthDot(s.url)
        + '<input class="nm" type="text" placeholder="Kaynak adı" value="'+esc(s.name||"")+'" data-ci="'+ci+'" data-si="'+si+'" data-f="name">'
        + '<input class="url" type="text" placeholder="https://… .rss / feed URL" value="'+esc(s.url||"")+'" data-ci="'+ci+'" data-si="'+si+'" data-f="url">'
        + '<button class="mini" onclick="removeSource('+ci+','+si+')">Sil</button>'
        + '</div>').join("") : "";
      d.innerHTML =
        '<div class="seccat-head">'
        + '<button class="mini seccat-toggle" onclick="toggleCatEditor('+ci+')" title="Kaynakları aç/kapat">›</button>'
        + '<input class="catname" type="text" value="'+esc(cat.name)+'" data-ci="'+ci+'" data-f="catname" onclick="event.stopPropagation()">'
        + '<span class="seccat-count">'+count+' kaynak</span>'
        + (function(){ const bad=(cat.sources||[]).filter(x=>{ const h=SRC_HEALTH[normFeedURL(x.url)]; return h && h.status!=='healthy'; }).length; return bad?'<span class="seccat-health" title="Sorunlu kaynak sayısı"><i class="hdot err"></i>'+bad+'</span>':''; })()
        + '<button class="mini" onclick="moveCat('+ci+',-1)" title="Yukarı taşı">↑</button>'
        + '<button class="mini" onclick="moveCat('+ci+',1)" title="Aşağı taşı">↓</button>'
        + '<button class="mini" onclick="addSource('+ci+')">+ Kaynak</button>'
        + '<button class="mini" onclick="removeCategory('+ci+')">Sil</button>'
        + '</div>'
        + (open ? '<div class="seccat-body"><div class="srclist">'+(srcs||'<div class="access-note">Henüz kaynak yok.</div>')+'</div>'
          + '<div class="discover">'
          + '<input type="text" class="disc-url" placeholder="site adresinden bul: ör. ntv.com.tr">'
          + '<button class="mini" onclick="discoverFor('+ci+',this)">🔎 Bul</button>'
          + '<div class="disc-results" id="disc-'+ci+'"></div>'
          + '</div></div>' : '');
      box.appendChild(d);
    });
    box.querySelectorAll("input").forEach(inp=>{
      inp.addEventListener("input",()=>{
        const ci=+inp.dataset.ci, f=inp.dataset.f;
        if(f==="catname"){ CONFIG.categories[ci].name = inp.value; }
        else { const si=+inp.dataset.si; if(CONFIG.categories[ci] && CONFIG.categories[ci].sources[si]) CONFIG.categories[ci].sources[si][f] = inp.value; }
      });
    });
  }
  // --- autodiscovery (site adresinden feed bul) ---
  let discTarget = -1;
  function discoverFor(ci, btn){
    const inp = btn.parentElement.querySelector(".disc-url");
    const val = (inp && inp.value || "").trim();
    const out = document.getElementById("disc-"+ci);
    if(!val){ if(out) out.innerHTML = '<span class="disc-note">Adres girin.</span>'; return; }
    discTarget = ci;
    if(out) out.innerHTML = '<span class="disc-note">Aranıyor… (birkaç saniye)</span>';
    window.nnDiscover(val);
  }
  window.nnOnDiscover = function(res){
    const ci = discTarget;
    const out = document.getElementById("disc-"+ci);
    if(!out) return;
    if(!res || (res.err && (!res.feeds || !res.feeds.length))){
      out.innerHTML = '<span class="disc-note">'+esc(res && res.err ? res.err : "Bulunamadı.")+'</span>';
      return;
    }
    let html = '<div class="disc-note">'+res.feeds.length+' akış bulundu:</div>';
    res.feeds.forEach(f=>{
      html += '<div class="disc-row">'
        + '<span class="disc-name">'+esc(f.title||f.url)+'</span>'
        + '<button class="mini add-found" data-url="'+encodeURIComponent(f.url)+'" data-title="'+encodeURIComponent(f.title||"")+'">+ Ekle</button>'
        + '</div>';
    });
    out.innerHTML = html;
    out.querySelectorAll(".add-found").forEach(b=>{
      b.addEventListener("click",()=>{
        addFound(ci, decodeURIComponent(b.dataset.url), decodeURIComponent(b.dataset.title||""));
      });
    });
  };
  function addFound(ci, url, title){
    if(ci<0 || !CONFIG.categories[ci]) return;
    CONFIG.categories[ci].sources = CONFIG.categories[ci].sources||[];
    if(CONFIG.categories[ci].sources.some(s=>s.url===url)){ return; } // zaten var
    CONFIG.categories[ci].sources.push({name:title, url:url});
    renderCatsEditor();
  }

  function addCategory(){ CONFIG.categories.push({name:"Yeni Kategori",sources:[]}); openSourceCat=CONFIG.categories.length-1; renderCatsEditor(); }
  function removeCategory(ci){ CONFIG.categories.splice(ci,1); renderCatsEditor(); }
  function moveCat(ci,dir){ const a=CONFIG.categories; const j=ci+dir; if(j<0||j>=a.length) return; const t=a[ci]; a[ci]=a[j]; a[j]=t; renderCatsEditor(); }
  function addSource(ci){ CONFIG.categories[ci].sources = CONFIG.categories[ci].sources||[]; CONFIG.categories[ci].sources.push({name:"",url:""}); openSourceCat=ci; renderCatsEditor(); }
  function removeSource(ci,si){ CONFIG.categories[ci].sources.splice(si,1); openSourceCat=ci; renderCatsEditor(); }

  async function saveSettings(){
    CONFIG.maxPerCategory = Math.max(1, +document.getElementById("optMax").value||5);
    CONFIG.newsCountry = document.getElementById("optNewsCountry").value || "";
    CONFIG.countryScope = document.getElementById("optCountryScope").value || "related";
    CONFIG.openInApp = document.getElementById("optInApp").checked;
    CONFIG.refreshMinutes = Math.max(0, +document.getElementById("optRefresh").value||0);
    CONFIG.theme = document.getElementById("optTheme").value;
    CONFIG.glassOpacity = Math.max(40, Math.min(100, +document.getElementById("optGlassOp").value||72));
    CONFIG.fontScale = Math.max(80, Math.min(160, +document.getElementById("optFontScale").value||100));
    CONFIG.lineHeight = Math.max(1.4, Math.min(2.4, +document.getElementById("optLineHeight").value||1.75));
    CONFIG.readerWidth = Math.max(520, Math.min(980, +document.getElementById("optReaderWidth").value||760));
    CONFIG.dyslexiaFont = document.getElementById("optDyslexia").checked;
    { const v=parseInt(document.getElementById("optWidgetOpacity").value,10); CONFIG.widgetOpacity = isFinite(v)?Math.max(0,Math.min(100,v)):78; }
    CONFIG.widgetTint = Math.max(0, Math.min(100, parseInt(document.getElementById("optWTint").value,10)||0));
    CONFIG.widgetSingle = document.getElementById("optWSingle").checked;
    CONFIG.netMeter = document.getElementById("optNetMeter").checked;
    CONFIG.quietHours = document.getElementById("optQuiet").checked;
    CONFIG.quietFrom = document.getElementById("optQuietFrom").value || "23:00";
    CONFIG.quietTo = document.getElementById("optQuietTo").value || "07:00";
    CONFIG.translate = document.getElementById("optTr").checked;
    CONFIG.translateLang = document.getElementById("optTrLang").value.trim();
    CONFIG.weather = document.getElementById("optWx").checked;
    const splitCsv = s => (s||"").split(",").map(x=>x.trim()).filter(Boolean);
    CONFIG.interests = splitCsv(document.getElementById("optInterests").value);
    CONFIG.hideWords = splitCsv(document.getElementById("optHide").value);
    CONFIG.marketWatch = splitCsv(document.getElementById("optMarketWatch").value);
    CONFIG.sportsWatch = splitCsv(document.getElementById("optSportsWatch").value);
    CONFIG.sportsTeams = splitCsv(document.getElementById("optSportsTeams").value);
    CONFIG.notify = document.getElementById("optNotify").checked;
    CONFIG.notifyMinScore = Math.max(40, Math.min(100, parseInt(document.getElementById("optNotifyMin").value,10)||75));
    CONFIG.widgetImpOnly = document.getElementById("optWImp").checked;
    CONFIG.widgetSlideSec = parseInt(document.getElementById("optWSlide").value,10) || 3;
    CONFIG.widgetCats = Array.from(document.querySelectorAll(".wcat:checked")).map(x=>x.value);
    // prune empty sources
    CONFIG.categories.forEach(c=>{ c.sources=(c.sources||[]).filter(s=>s.url && s.url.trim()); });
    await window.nnSaveConfig(JSON.stringify(CONFIG));
    closeSettings();
    applyGlass();
    document.getElementById("refreshBtn").classList.add("loading");
    loadWeather();
    renderNav();
    pollNet();
  }

  async function exportCfg(kind){
    const info = document.getElementById("ioInfo");
    info.textContent = "Kaydediliyor…";
    try{
      const path = await window.nnExport(kind);
      info.textContent = path ? ("Kaydedildi: " + path) : "İptal edildi.";
    }catch(e){ info.textContent = "Hata: kaydedilemedi."; }
  }
  async function importCfg(){
    const info = document.getElementById("ioInfo");
    info.textContent = "İçe aktarılıyor…";
    try{
      const msg = await window.nnImport();
      info.textContent = msg || "Tamam.";
      CONFIG = JSON.parse(await window.nnGetConfig());
      document.getElementById("optMax").value = CONFIG.maxPerCategory||5;
    renderNewsCountryOptions();
    document.getElementById("optCountryScope").value = CONFIG.countryScope || "related";
      renderCatsEditor();
    }catch(e){ info.textContent = "Hata: içe aktarılamadı."; }
  }

