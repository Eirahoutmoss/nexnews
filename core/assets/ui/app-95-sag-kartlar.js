  // ---------- right cards: compact + collapsible ----------
  function initRightCardToggles(){
    const rp=document.getElementById('rightpanel'); if(!rp) return;
    const ids=['followPanel','standingsPanel','marketPanel','smartAlertPanel','personalPanel','briefingPanel','contentIntelPanel','sourceHealthPanel','officialPanel','worldPanel','sportPanel','intel2Panel','predictPanel','entityPanel','eventGraphPanel'];
    ids.forEach(id=>{
      const p=document.getElementById(id); if(!p || p.dataset.rcToggleReady==='1') return;
      const head=p.querySelector(':scope > .market-head, :scope > .smart-alert-head, :scope > .personal-head, :scope > .briefing-head, :scope > .content-intel-head, :scope > .source-health-head, :scope > .official-head, :scope > .world-head, :scope > .sport-head, :scope > .intel2-head, :scope > .predict-head, :scope > .entity-head, :scope > .eg-head, :scope > .follow-card-head');
      if(!head) return;
      p.dataset.rcToggleReady='1';
      head.classList.add('right-card-head');
      const oldBody=p.querySelector(':scope > .market-collapsible, :scope > .smart-alert-collapsible');
      let body=oldBody;
      if(!body){
        body=document.createElement('div'); body.className='right-card-body';
        const nodes=[]; let n=head.nextElementSibling; while(n){ nodes.push(n); n=n.nextElementSibling; }
        nodes.forEach(x=>body.appendChild(x)); p.appendChild(body);
      }else{
        body.classList.add('right-card-body');
      }
      const key='nexnews.rightcard.'+id;
      let saved=null; try{ saved=localStorage.getItem(key); }catch(e){}
      let collapsed=(saved==='1');
      if(saved===null){ collapsed=(id in CARD_DEFAULT_CLOSED); }
      p.classList.toggle('right-card-collapsed',collapsed);
      head.setAttribute('role','button'); head.setAttribute('tabindex','0');
      head.setAttribute('aria-expanded',collapsed?'false':'true');
      const toggle=()=>{
        const c=!p.classList.contains('right-card-collapsed');
        p.classList.toggle('right-card-collapsed',c);
        head.setAttribute('aria-expanded',c?'false':'true');
        try{ localStorage.setItem(key,c?'1':'0'); }catch(e){}
        if(!c && CARD_LOADERS[id]) CARD_LOADERS[id](); // açılınca verisini topla
      };
      head.addEventListener('click',toggle);
      head.addEventListener('keydown',e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();toggle();}});
    });
  }

  // =====================================================================
  // RC81: sekmeli sağ panel + görünür sekmeye göre yoklama
  // =====================================================================
  const RIGHT_GROUPS = {
    ozet:   ['followPanel','smartAlertPanel','briefingPanel','personalPanel','worldPanel','officialPanel'],
    borsa:  ['marketPanel'],
    spor:   ['standingsPanel','sportPanel'],
    analiz: ['intel2Panel','predictPanel','entityPanel','eventGraphPanel','contentIntelPanel','sourceHealthPanel'],
    olay:   ['eventPanel']
  };
  // RC85: kapalı kartların verisi toplanmaz; kart açıldığında yüklenir.
  const CARD_DEFAULT_CLOSED={smartAlertPanel:1};
  const CARD_LOADERS={smartAlertPanel:()=>loadSmartAlerts(),briefingPanel:()=>loadBriefing(),personalPanel:()=>loadPersonalIntelligence(),worldPanel:()=>loadWorldPulse(),officialPanel:()=>loadOfficialPublications(),followPanel:()=>loadFollowCard(),
    marketPanel:()=>loadFinance(),standingsPanel:()=>loadStandings(),sportPanel:()=>loadSports(),intel2Panel:()=>loadEventIntelligence(),predictPanel:()=>loadPredictive(),entityPanel:()=>loadEntityIntelligence(),eventGraphPanel:()=>loadEventGraph(),contentIntelPanel:()=>loadContentIntelligence(),sourceHealthPanel:()=>loadSourceHealth()};
  function cardOpen(id){ const p=document.getElementById(id); return !!p && !p.classList.contains('right-card-collapsed'); }
  function loadOpenCards(tab){ (RIGHT_GROUPS[tab]||[]).forEach(id=>{ if(CARD_LOADERS[id] && cardOpen(id)) CARD_LOADERS[id](); }); }
  const RIGHT_LOADERS = {
    ozet:   ()=>loadOpenCards('ozet'),
    borsa:  ()=>loadOpenCards('borsa'),
    spor:   ()=>loadOpenCards('spor'),
    analiz: ()=>loadOpenCards('analiz'),
    olay:   ()=>{}
  };
  let RIGHT_TAB = 'ozet', RIGHT_PREV = 'ozet';
  const RIGHT_LOADED = {};
  function initRightTabs(){
    const rp=document.getElementById('rightpanel'); if(!rp) return;
    Object.keys(RIGHT_GROUPS).forEach(g=>RIGHT_GROUPS[g].forEach(id=>{ const el=document.getElementById(id); if(el) el.setAttribute('data-rgroup',g); }));
    const saved=(CONFIG&&CONFIG.rightTab)||'ozet';
    setRightTab(RIGHT_GROUPS[saved]?saved:'ozet', true);
  }
  function setRightTab(tab, quiet){
    if(!RIGHT_GROUPS[tab]) tab='ozet';
    if(tab==='olay'){ if(RIGHT_TAB!=='olay') RIGHT_PREV=RIGHT_TAB; } else RIGHT_PREV=tab;
    RIGHT_TAB=tab;
    const rp=document.getElementById('rightpanel'); if(rp) rp.setAttribute('data-tab',tab);
    document.querySelectorAll('.rp-tab').forEach(b=>{ const on=b.dataset.rtab===tab; b.classList.toggle('active',on); b.setAttribute('aria-selected',on?'true':'false'); });
    if(rightVisible()){ RIGHT_LOADERS[tab](); RIGHT_LOADED[tab]=Date.now(); }
    if(!quiet && tab!=='olay' && CONFIG && CONFIG.rightTab!==tab){
      CONFIG.rightTab=tab;
      try{ window.nnSetRightTab(tab); }catch(e){}
    }
  }
  // Yalnızca görünen sekmenin verisi, pencere görünürken yenilenir.
  function refreshVisibleTab(){
    if(document.hidden || document.body.classList.contains('widget') || !rightVisible()) return;
    RIGHT_LOADERS[RIGHT_TAB](); RIGHT_LOADED[RIGHT_TAB]=Date.now();
  }
  function everyVisible(fn, ms, tab){
    setInterval(()=>{
      if(document.hidden || document.body.classList.contains('widget')) return;
      if(tab && (RIGHT_TAB!==tab || !rightVisible())) return;
      // Sınırlı veri: sağ panel verileri üç kat seyrek yenilenir.
      if(document.body.classList.contains('data-lite')){ const k=(fn._lite=(fn._lite||0)+1); if(k%3) return; }
      fn();
    }, ms);
  }
  document.addEventListener('visibilitychange',()=>{ if(!document.hidden) refreshVisibleTab(); });

  // =====================================================================
  // RC81: ağ göstergesi (bağlantı hızı + son 5 dk veri)
  // =====================================================================
  let NET_SRC = 'pc';
  try{ NET_SRC = localStorage.getItem('nexnews.netsrc') || 'pc'; }catch(e){}
  function fmtBytes(n){
    n=Number(n||0);
    const u=['B','KB','MB','GB']; let i=0;
    while(n>=1024 && i<u.length-1){ n/=1024; i++; }
    return (i===0?Math.round(n):n.toLocaleString(uiLocale(),{maximumFractionDigits:n<10?1:0}))+' '+u[i];
  }
  function fmtRate(bytesPerSec){
    if(bytesPerSec==null || bytesPerSec<0) return '—';
    const bits=bytesPerSec*8;
    if(bits<1000) return Math.round(bits)+' bps';
    if(bits<1e6) return (bits/1e3).toLocaleString(uiLocale(),{maximumFractionDigits:bits<1e4?1:0})+' Kbps';
    if(bits<1e9) return (bits/1e6).toLocaleString(uiLocale(),{maximumFractionDigits:bits<1e7?1:0})+' Mbps';
    return (bits/1e9).toLocaleString(uiLocale(),{maximumFractionDigits:2})+' Gbps';
  }
  function toggleNetSource(){
    NET_SRC = NET_SRC==='pc' ? 'app' : 'pc';
    try{ localStorage.setItem('nexnews.netsrc',NET_SRC); }catch(e){}
    pollNet();
  }
  async function pollNet(){
    const el=document.getElementById('netMeter'); if(!el) return;
    if(!(CONFIG && CONFIG.netMeter!==false) || document.body.classList.contains('widget')){ el.classList.add('hide'); return; }
    let st; try{ st=JSON.parse(await window.nnNetStats()); }catch(e){ el.classList.add('hide'); return; }
    const pcKnown = st.sysDownBps!=null && st.sysDownBps>=0;
    const usePc = NET_SRC==='pc' && pcKnown;
    const down = usePc ? st.sysDownBps : st.appDownBps;
    const up = usePc ? st.sysUpBps : st.appUpBps;
    el.classList.remove('hide');
    el.classList.toggle('offline', !st.online && pcKnown===false && NET_SRC==='pc');
    el.innerHTML = '<span class="nm-rate">⇣ '+fmtRate(down)+'</span><span class="nm-sep">|</span><span>5 dk: '+fmtBytes(st.down5m)+'</span><span class="nm-src">'+(usePc?'PC':'App')+'</span>';
    const lines = [
      usePc ? 'Anlık hız: bilgisayarın tamamı (etkin bağlantı)' : 'Anlık hız: yalnızca NexNews',
      '⇣ '+fmtRate(down)+'   ⇡ '+fmtRate(up),
      '',
      'NexNews son 5 dakikada: ⇣ '+fmtBytes(st.down5m)+'  ⇡ '+fmtBytes(st.up5m)+(st.web5m?('  (sayfa içi: '+fmtBytes(st.web5m)+')'):''),
      'Bu oturumda: ⇣ '+fmtBytes(st.sessionDown)+'  ⇡ '+fmtBytes(st.sessionUp),
    ];
    if(st.adapter) lines.push('Bağlantı: '+st.adapter+(st.linkMbps?(' · '+st.linkMbps.toLocaleString(uiLocale())+' Mbps bağlantı hızı'):''));
    lines.push('', 'Tıkla: '+(usePc?'NexNews':'bilgisayar')+' hızını göster');
    el.title = lines.join('\n');
  }
  // WebView'in kendi yüklediği kaynaklar (görseller, harita, tarayıcı çevirisi).
  (function(){
    let pending=0;
    try{
      new PerformanceObserver(list=>{ list.getEntries().forEach(e=>{ if(/^https?:\/\/(127\.0\.0\.1|localhost)/.test(e.name||'')) return; /* canlı TV aktarıcısı Go tarafında zaten sayılıyor */ const n=e.transferSize||0; if(n>0) pending+=n; }); })
        .observe({type:'resource',buffered:true});
    }catch(e){}
    setInterval(()=>{ if(pending>0 && window.nnAddWebTraffic){ const n=pending; pending=0; try{ window.nnAddWebTraffic(Math.round(n)); }catch(e){} } }, 4000);
  })();

  // =====================================================================
  // RC83: veri modu (Tam / Sınırlı) ve kullanım defteri
  // =====================================================================
  let DATA_USAGE=null;
  function fmtDur(sec){ sec=Math.max(0,Math.round(sec||0)); const d=Math.floor(sec/86400), h=Math.floor(sec%86400/3600), m=Math.floor(sec%3600/60); return d?(d+' g '+h+' sa'):(h?(h+' sa '+m+' dk'):(m+' dk')); }
  function paintDataMode(){
    const u=DATA_USAGE||{}; const lim=u.mode==='sinirli';
    document.body.classList.toggle('data-lite',lim);
    const b=document.getElementById('dataModeBtn'); if(!b) return;
    b.classList.toggle('lim',lim);
    b.textContent=(lim?'🐢 Sınırlı':'⚡ Tam')+' · bugün '+fmtBytes((u.today&&u.today.down)||0);
    b.title=(lim?'Sınırlı veri akışı (en çok '+(u.limitKBps||256)+' KB/s)':'Tam veri akışı')+'\nBu oturum: ⇣ '+fmtBytes((u.session&&u.session.down)||0)+'\nBu ay: ⇣ '+fmtBytes((u.month&&u.month.down)||0)+(u.quotaMB?(' / '+u.quotaMB.toLocaleString(uiLocale())+' MB kota'):'')+'\nTıkla: ayrıntılar ve mod seçimi';
  }
  async function loadDataUsage(){ try{ DATA_USAGE=JSON.parse(await window.nnUsage()); }catch(e){ return; } paintDataMode(); const pn=document.getElementById('dataPanel'); if(pn && !pn.classList.contains('hide')) renderDataPanel(); }
  window.nnOnDataMode=loadDataUsage;
  function usageRow(label,p,extra){ p=p||{}; return '<tr><td>'+label+'</td><td>⇣ '+fmtBytes(p.down||0)+'</td><td>⇡ '+fmtBytes(p.up||0)+'</td><td>'+(p.secs!=null?fmtDur(p.secs):'')+'</td>'+(extra||'')+'</tr>'; }
  function renderDataPanel(){
    const pn=document.getElementById('dataPanel'); if(!pn) return;
    const u=DATA_USAGE||{}; const lim=u.mode==='sinirli';
    const days=(u.days||[]); const mx=Math.max(1,...days.map(d=>d.down||0));
    const bars=days.map(d=>'<i title="'+esc(d.day)+': '+fmtBytes(d.down)+(d.secs?' · '+fmtDur(d.secs)+' açık':'')+'" style="height:'+Math.max(2,Math.round((d.down||0)/mx*46))+'px"></i>').join('');
    const q=u.quotaMB?('<div class="dp-quota"><div class="dp-qbar"><b style="width:'+Math.min(100,Math.round((u.quotaUsed||0)*100))+'%"></b></div><span>Bu ay kotanın %'+Math.round((u.quotaUsed||0)*100)+' kullanıldı'+(u.autoLimited?' · kota dolduğu için sınırlı moda geçildi':'')+'</span></div>'):'';
    pn.innerHTML='<div class="dp-head"><b>Veri akışı</b><button class="mini" onclick="toggleDataPanel()">✕</button></div>'
      +'<div class="dp-modes"><button class="dp-mode'+(!lim?' active':'')+'" onclick="setDataMode(\'tam\')"><b>⚡ Tam veri akışı</b><small>Her şey normal hızda; canlı TV en iyi kalitede.</small></button>'
      +'<button class="dp-mode'+(lim?' active':'')+'" onclick="setDataMode(\'sinirli\')"><b>🐢 Sınırlı veri akışı</b><small>İndirme hızı sınırlanır, canlı TV en çok 480p, küçük görseller yüklenmez, otomatik yenileme seyrekleşir. Canlı TV için en az 256 KB/s önerilir.</small></button></div>'
      +'<div class="dp-opts"><label>Sınırlı modda en yüksek hız <select id="dpLimit" onchange="setDataMode(null)">'+[64,128,256,512,1024,2048].map(k=>'<option value="'+k+'"'+(k===(u.limitKBps||256)?' selected':'')+'>'+(k>=1024?(k/1024)+' MB/s':k+' KB/s')+'</option>').join('')+'</select></label>'
      +'<label>Aylık kota <input id="dpQuota" type="number" min="0" step="100" value="'+(u.quotaMB||0)+'" onchange="setDataMode(null)"> MB <small>(0 = yok; dolunca sınırlı moda geçer)</small></label></div>'
      +'<table class="dp-table"><tr><th></th><th>İndirme</th><th>Yükleme</th><th>Açık kalma</th></tr>'
      +usageRow('Bu oturum',u.session)+usageRow('Bugün',u.today)+usageRow('Bu hafta',u.week)+usageRow('Bu ay',u.month)+usageRow('Son 30 gün',u.last30)+usageRow('Toplam',u.all)+'</table>'
      +q
      +'<div class="dp-chart" title="Son 30 günün günlük indirmesi">'+bars+'</div>'
      +'<div class="dp-foot">Toplam, '+(u.since?new Date(u.since*1000).toLocaleDateString(uiLocale()):'—')+' tarihinden bu yana · açık kalınan gün başına ortalama ⇣ '+fmtBytes(u.dailyAvg||0)+'<button class="mini" onclick="resetDataUsage()">Sayaçları sıfırla</button></div>';
  }
  function toggleDataPanel(ev){
    if(ev) ev.stopPropagation();
    let pn=document.getElementById('dataPanel');
    if(!pn){ pn=document.createElement('div'); pn.id='dataPanel'; pn.className='data-panel hide'; pn.addEventListener('click',e=>e.stopPropagation()); document.body.appendChild(pn); document.addEventListener('click',()=>pn.classList.add('hide')); }
    const show=pn.classList.contains('hide');
    pn.classList.toggle('hide',!show);
    if(show){ renderDataPanel(); loadDataUsage(); }
  }
  async function setDataMode(mode){
    const u=DATA_USAGE||{};
    if(!mode) mode=u.mode||'tam';
    const l=document.getElementById('dpLimit'), q=document.getElementById('dpQuota');
    const kbps=l?parseInt(l.value,10):(u.limitKBps||256), quota=q?Math.max(0,parseInt(q.value,10)||0):(u.quotaMB||0);
    try{ await window.nnSetDataMode(mode,kbps,quota); }catch(e){ nnToast('Veri modu kaydedilemedi'); return; }
    if(CONFIG){ CONFIG.dataMode=mode; CONFIG.dataLimitKBps=kbps; CONFIG.dataQuotaMB=quota; }
    await loadDataUsage();
    nnToast(mode==='sinirli'?'Sınırlı veri akışı açık — en çok '+(kbps>=1024?(kbps/1024)+' MB/s':kbps+' KB/s'):'Tam veri akışı açık');
    if(MEDIA && MEDIA.kind==='tv' && window._liveTVCurrent) playLiveTV(window._liveTVCurrent); // yeni kalite sınırı
  }
  async function resetDataUsage(){ try{ await window.nnResetUsage(); }catch(e){} loadDataUsage(); nnToast('Veri sayaçları sıfırlandı'); }

  // =====================================================================
  // RC81: klavye ile gezinme
  // =====================================================================
  let KBD_IDX = -1;
  function kbdItems(){
    const root = !document.getElementById('catpage').classList.contains('hide') ? document.getElementById('catpage') : document.getElementById('overview');
    return Array.from(root.querySelectorAll('.item')).filter(n=>n.offsetParent!==null);
  }
  function kbdMove(d){
    const items=kbdItems(); if(!items.length) return;
    document.querySelectorAll('.item.kbd-focus').forEach(n=>n.classList.remove('kbd-focus'));
    KBD_IDX = Math.max(0, Math.min(items.length-1, (KBD_IDX<0 && d<0) ? 0 : KBD_IDX+d));
    const it=items[KBD_IDX]; it.classList.add('kbd-focus'); it.scrollIntoView({block:'nearest'});
  }
  function kbdCurrentURL(){
    const n=document.querySelector('.item.kbd-focus'); return n ? decodeURIComponent(n.dataset.url||'') : '';
  }
  document.addEventListener('keydown',e=>{
    if(document.body.classList.contains('widget')) return;
    const t=e.target, tag=(t&&t.tagName)||'';
    if(e.key==='F5'){ e.preventDefault(); doRefresh(); return; }
    if(tag==='INPUT'||tag==='TEXTAREA'||tag==='SELECT'||(t&&t.isContentEditable)) return;
    if(e.ctrlKey||e.metaKey||e.altKey) return;
    if(document.getElementById('settingsOverlay').classList.contains('open')) return;
    if(document.getElementById('reader').classList.contains('open') && e.key!=='Escape') return;
    const k=e.key.toLowerCase();
    if(k==='j'){ e.preventDefault(); kbdMove(1); }
    else if(k==='k'){ e.preventDefault(); kbdMove(-1); }
    else if(k==='/'){ e.preventDefault(); const sb=document.getElementById('searchBox'); sb.focus(); sb.select(); }
    else if(k==='?'){ document.getElementById('keysOverlay').classList.add('open'); }
    else if(k==='g'){ const h=document.querySelector('.navi[data-nav="home"]'); navHome(h); window.scrollTo(0,0); }
    else if(k==='escape'){ document.getElementById('keysOverlay').classList.remove('open'); }
    else {
      const url=kbdCurrentURL(); if(!url) return;
      if(k==='o'||k==='enter'){ e.preventDefault(); const n=document.querySelector('.item.kbd-focus .title'); if(n&&n.dataset.act==='event') openEvent(url); else openItem(url); }
      else if(k==='r') readItem(url);
      else if(k==='b') browserItem(url);
      else if(k==='e') openEvent(url);
      else if(k==='s') toggleFav(url);
    }
  });

  // =====================================================================
  // RC81: widget görünümü (opak / cam / bulanık / saydam)
  // =====================================================================
  function applyWidgetLookCSS(){
    const b=document.body;
    const mode=(CONFIG&&CONFIG.widgetMode)||'cam';
    b.setAttribute('data-wmode',mode);
    b.classList.toggle('w-single', !!(CONFIG&&CONFIG.widgetSingle));
    const op=Math.max(0,Math.min(100,Number(CONFIG&&CONFIG.widgetOpacity!=null?CONFIG.widgetOpacity:78)));
    // RC86: modlar gerçekten farklı görünsün. "Yüzey" kaydırıcısı her modda
    // farklı bir aralıkta çalışır: cam → buzlu panel, bulanık → hafif koyu
    // bulanık, saydam → neredeyse çıplak (yalnızca yazı gölgesi).
    const k = mode==='opak' ? 0 : mode==='cam' ? .55 : mode==='bulanik' ? .45 : .28;
    const tn=Math.max(0,Math.min(100,Number(CONFIG&&CONFIG.widgetTint!=null?CONFIG.widgetTint:45)))/100;
    // Saydam modda Windows tonu kullanılmıyor; Ton kaydırıcısı CSS yüzeyine eklenir.
    const wa = mode==='opak' ? 1 : mode==='saydam' ? Math.min(.85, op/100*k + tn*.3) : op/100*k;
    document.documentElement.style.setProperty('--wa', wa.toFixed(3));
    updateWidgetPreview();
  }
  function updateWidgetPreview(){
    const pv=document.getElementById('wPreview'); if(!pv || !CONFIG) return;
    const mode=CONFIG.widgetMode||'cam';
    const op=Math.max(0,Math.min(100,Number(CONFIG.widgetOpacity!=null?CONFIG.widgetOpacity:78)))/100;
    const tint=Math.max(0,Math.min(100,Number(CONFIG.widgetTint!=null?CONFIG.widgetTint:45)))/100;
    pv.className='wprev '+mode;
    // Önizleme: cam/bulanık için CSS bulanıklığı, yüzey + ton birlikte opaklığı belirler.
    let a=op, blur=0;
    if(mode==='opak'){ a=1; }
    else if(mode==='cam'){ blur=14; a=Math.min(1, op*0.55 + tint*0.35); }
    else if(mode==='bulanik'){ blur=10; a=op*0.45; }
    else { a=op*0.28 + tint*0.3; }
    pv.style.setProperty('--pv-a', a.toFixed(2));
    pv.style.setProperty('--pv-blur', blur+'px');
  }
  let WLOOK_T=null;
  function pushWidgetLook(){
    applyWidgetLookCSS();
    if(WLOOK_T) clearTimeout(WLOOK_T);
    WLOOK_T=setTimeout(()=>{ try{ window.nnSetWidgetLook(CONFIG.widgetMode||'cam', Number(CONFIG.widgetTint||0), Number(CONFIG.widgetOpacity!=null?CONFIG.widgetOpacity:78)); }catch(e){} }, 120);
  }
  function setWidgetMode(m){ CONFIG.widgetMode=m; syncSeg('optWMode',m); pushWidgetLook(); if(!document.getElementById('wmenu').classList.contains('hide')) showWMenu(); }
  function syncSeg(id,v){ const el=document.getElementById(id); if(el) el.querySelectorAll('button').forEach(b=>b.classList.toggle('active',b.dataset.v===v)); }
  function bindSeg(id,cb){ const el=document.getElementById(id); if(!el||el.dataset.bound) return; el.dataset.bound='1'; el.addEventListener('click',e=>{ const b=e.target.closest('button[data-v]'); if(!b) return; syncSeg(id,b.dataset.v); cb(b.dataset.v); }); }

  // ayarlar: bölüm gezintisi
  function showSetSec(sec){
    document.querySelectorAll('.set-sec').forEach(x=>x.classList.toggle('active',x.dataset.sec===sec));
    document.querySelectorAll('#setNav button').forEach(x=>x.classList.toggle('active',x.dataset.sec===sec));
    const mb=document.querySelector('.settings-modal .modal-body'); if(mb) mb.scrollTop=0;
  }
  // kaynak sağlığı (ayarlar → kaynaklar)
  let SRC_HEALTH = {};
  function normFeedURL(u){ return String(u||'').trim().toLowerCase().replace(/^https?:\/\//,'').replace(/^www\./,'').replace(/\/+$/,''); }
  async function loadSrcHealthMap(){
    SRC_HEALTH={};
    try{ (JSON.parse(await window.nnSourceHealth())||[]).forEach(h=>{ SRC_HEALTH[normFeedURL(h.url)]=h; }); }catch(e){}
  }
  function healthDot(url){
    const h=SRC_HEALTH[normFeedURL(url)];
    if(!h) return '<i class="hdot" title="Henüz denenmedi"></i>';
    const cls = h.status==='healthy'?'ok':(h.status==='stale'?'stale':'err');
    const tip = (h.status==='healthy'?'Çalışıyor':(h.status==='stale'?'Yavaş/eski — önbellekten gösteriliyor':'Hata'))
      + (h.itemCount?(' · '+h.itemCount+' başlık'):'') + (h.latencyMs?(' · '+h.latencyMs+' ms'):'') + (h.lastError?(' · '+h.lastError):'');
    return '<i class="hdot '+cls+'" title="'+esc(tip)+'"></i>';
  }

  // =====================================================================
  // Kalıcı oynatıcı (radyo + canlı TV). Medya öğeleri hiçbir zaman sayfa
  // içeriğiyle birlikte silinmez; TV sayfası açıkken oynatıcı sayfadaki
  // yuvanın üzerine oturur, başka sayfadayken sağ altta küçük pencere olur.
  // =====================================================================
  const MEDIA = { kind:null };
  function mediaBegin(kind, title, sub){
    if(MEDIA.kind && MEDIA.kind!==kind) mediaHalt(MEDIA.kind);
    MEDIA.kind=kind;
    document.getElementById('mdTitle').textContent=title||'';
    document.getElementById('mdSub').textContent=sub||'';
    mediaVolume(document.getElementById('mdVol').value);
    syncDock();
  }
  function mediaHalt(kind){
    if(kind==='tv'){ destroyLivePlayer(); }
    if(kind==='radio'){ const a=document.getElementById('radioAudio'); try{a.pause();}catch(e){} a.removeAttribute('src'); try{a.load();}catch(e){} }
  }
  function playRadio(st){
    mediaBegin('radio', st.name||'Radyo', [st.country, st.language].filter(Boolean).join(' · ')||'Canlı radyo');
    const a=document.getElementById('radioAudio');
    a.src=st.stream; a.play().catch(()=>{});
  }
  function mediaEl(){ return MEDIA.kind==='tv' ? document.getElementById('liveTVVideo') : (MEDIA.kind==='radio' ? document.getElementById('radioAudio') : null); }
  function mediaToggle(){ const m=mediaEl(); if(!m) return; if(m.paused) m.play().catch(()=>{}); else m.pause(); }
  function mediaVolume(v){ const x=Math.max(0,Math.min(100,Number(v)||0))/100; ['liveTVVideo','radioAudio'].forEach(id=>{ const el=document.getElementById(id); if(el) el.volume=x; }); }
  function mediaStop(){ if(MEDIA.kind) mediaHalt(MEDIA.kind); MEDIA.kind=null; syncDock(); }
  function mediaOpenPage(){ if(MEDIA.kind==='tv') openLiveTV(); else if(MEDIA.kind==='radio') openWorldRadio(); }
  function syncPlayIcon(){ const m=mediaEl(); const b=document.getElementById('mdPlay'); if(b) b.textContent=(m && !m.paused) ? '❚❚' : '▶'; }
  function syncDock(){
    const dock=document.getElementById('mediaDock'); if(!dock) return;
    const tvArea=document.getElementById('mdTV');
    if(!MEDIA.kind){ dock.className='media-dock hide'; dock.removeAttribute('style'); return; }
    tvArea.classList.toggle('hide', MEDIA.kind!=='tv');
    const slot=document.getElementById('liveTVSlot');
    const onTVPage = MEDIA.kind==='tv' && slot && slot.isConnected && slot.offsetParent!==null;
    if(onTVPage){
      const r=slot.getBoundingClientRect();
      dock.className='media-dock docked';
      dock.style.left=(r.left+window.scrollX)+'px'; dock.style.top=(r.top+window.scrollY)+'px';
      dock.style.width=r.width+'px'; dock.style.height=r.height+'px';
    }else{
      dock.className='media-dock mini'+(MEDIA.kind==='tv'?' has-tv':'');
      dock.removeAttribute('style');
    }
    syncPlayIcon();
  }
  (function(){
    let raf=0; const later=()=>{ if(raf) return; raf=requestAnimationFrame(()=>{ raf=0; syncDock(); }); };
    const mo=new MutationObserver(later);
    ['catpage','overview'].forEach(id=>{ const el=document.getElementById(id); if(el) mo.observe(el,{attributes:true,attributeFilter:['class'],childList:true}); });
    window.addEventListener('resize',later);
    try{ new ResizeObserver(later).observe(document.getElementById('catpage')); }catch(e){}
    ['liveTVVideo','radioAudio'].forEach(id=>{ const el=document.getElementById(id); if(el){ el.addEventListener('play',syncPlayIcon); el.addEventListener('pause',syncPlayIcon); } });
  })();

