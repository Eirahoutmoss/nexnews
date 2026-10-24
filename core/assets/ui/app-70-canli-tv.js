  // ---------- canlı TV ----------
  // Kanal listesi core/assets/livetv.json dosyasından gelir (Android ile ortak).
  const LIVE_TV_CHANNELS = __NEXLIVETV__;
  let LIVE_HLS = null;
  let LIVE_HLS_LOAD = null;
  function ensureHls(){
    if(window.Hls) return Promise.resolve(window.Hls);
    if(LIVE_HLS_LOAD) return LIVE_HLS_LOAD;
    LIVE_HLS_LOAD = new Promise((resolve,reject)=>{
      const sc=document.createElement('script');
      sc.src='https://cdn.jsdelivr.net/npm/hls.js@1.5.20/dist/hls.min.js';
      sc.onload=()=>window.Hls?resolve(window.Hls):reject(new Error('HLS kütüphanesi yüklenemedi'));
      sc.onerror=()=>reject(new Error('HLS kütüphanesi indirilemedi'));
      document.head.appendChild(sc);
    });
    return LIVE_HLS_LOAD;
  }
  function destroyLivePlayer(){
    if(LIVE_HLS){try{LIVE_HLS.destroy()}catch(e){} LIVE_HLS=null;}
    const v=document.getElementById('liveTVVideo');
    if(v){try{v.pause()}catch(e){} v.removeAttribute('src'); try{v.load()}catch(e){} }
  }
  function hideLiveOverlay(){
    const frame=document.getElementById('liveTVFrame');
    const video=document.getElementById('liveTVVideo');
    const overlay=document.getElementById('liveTVOverlay');
    if(overlay) overlay.remove();
    if(video) video.style.visibility='visible';
    if(frame) frame.classList.remove('has-live-overlay');
  }
  function showLiveOverlay(title, message, buttonText='Resmî yayını aç'){
    const frame=document.getElementById('liveTVFrame');
    const video=document.getElementById('liveTVVideo');
    if(!frame) return;
    const old=document.getElementById('liveTVOverlay');
    if(old) old.remove();
    if(video) video.style.visibility='hidden';
    frame.classList.add('has-live-overlay');
    const overlay=document.createElement('div');
    overlay.id='liveTVOverlay';
    overlay.className='live-tv-error live-tv-overlay';
    overlay.innerHTML='<div><b>'+esc(title)+'</b><span>'+esc(message)+'</span><br><button class="mini" onclick="nextLiveTV()">Sonraki kanala geç ›</button></div>';
    frame.appendChild(overlay);
  }
  let LIVE_RELAY_READY=null;
  function ensureLiveRelay(){
    if(LIVE_RELAY_READY) return LIVE_RELAY_READY;
    LIVE_RELAY_READY=new Promise((resolve)=>{
      const started=Date.now();
      const probe=()=>{
        fetch('http://127.0.0.1:39765/health',{cache:'no-store'}).then(r=>r.ok||r.status===204).then(ok=>{
          if(ok) return resolve(true);
          if(Date.now()-started>3500) return resolve(false);
          setTimeout(probe,120);
        }).catch(()=>{
          if(Date.now()-started>3500) return resolve(false);
          setTimeout(probe,120);
        });
      };
      probe();
    });
    return LIVE_RELAY_READY;
  }
  // RC85: uzun süren Go çağrıları arka planda çalışır; sonuç nnOnAsync ile gelir
  // (WebView2 bağlamaları arayüz iş parçacığında çalıştığından pencere donmaz).
  const NN_PEND={}; let NN_SEQ=0;
  window.nnOnAsync=function(t,ok,v){ const p=NN_PEND[t]; if(!p) return; delete NN_PEND[t]; ok?p.res(v):p.rej(new Error(v)); };
  function nnAsync(fn,...args){
    return new Promise((res,rej)=>{
      const t='a'+(++NN_SEQ); NN_PEND[t]={res,rej};
      const to=setTimeout(()=>{ if(NN_PEND[t]){ delete NN_PEND[t]; rej(new Error('zaman aşımı')); } }, 90000);
      NN_PEND[t]={res:v=>{clearTimeout(to);res(v);},rej:e=>{clearTimeout(to);rej(e);}};
      try{ Promise.resolve(window[fn](t,...args)).catch(e=>{ if(NN_PEND[t]){ delete NN_PEND[t]; clearTimeout(to); rej(e);} }); }catch(e){ delete NN_PEND[t]; clearTimeout(to); rej(e); }
    });
  }
  // ---- RC83: kaynak sağlığı, otomatik onarım ----
  let LIVE_HEALTH={};
  async function loadLiveHealth(){ try{ LIVE_HEALTH=JSON.parse(await window.nnLiveHealth())||{}; }catch(e){ LIVE_HEALTH={}; } }
  function liveCands(c){
    const arr=(c.urls&&c.urls.length?c.urls:(c.url?[c.url]:[])).map(x=>typeof x==='string'?{u:x}:x).filter(x=>x&&x.u);
    const h=LIVE_HEALTH[c.id];
    if(h && h.ok && h.url){
      const i=arr.findIndex(x=>x.u===h.url);
      if(i>0){ arr.unshift(arr.splice(i,1)[0]); }
      else if(i<0 && c._repair && c._repair.u===h.url){ arr.unshift(c._repair); }
    }
    if(c._repair && !arr.some(x=>x.u===c._repair.u)) arr.push(c._repair);
    return arr;
  }
  function relaySrc(x){
    return 'http://127.0.0.1:39765/live?t='+LIVE_RELAY_TOKEN+'&u='+encodeURIComponent(x.u)+(x.r?'&r='+encodeURIComponent(x.r):'')+(x.ua?'&ua='+encodeURIComponent(x.ua):'');
  }
  async function probeChannel(c){
    if(!window.nnProbeLive) return null;
    let r=null;
    try{ r=JSON.parse(await nnAsync('nnProbeLive', c.id, JSON.stringify(liveCands(c)), c.tvg||'', c.rep||'')); }catch(e){ return null; }
    if(r){ if(r.repair){ c._repair=r.repair; } LIVE_HEALTH[c.id]=r; paintLiveHealth(c.id); }
    return r;
  }
  function paintLiveHealth(id){
    document.querySelectorAll('.live-tv-channel[data-tv-id="'+id+'"]').forEach(b=>{
      const h=LIVE_HEALTH[id]; const d=b.querySelector('.live-tv-dot'); if(!d) return;
      d.classList.toggle('ok',!!(h&&h.ok)); d.classList.toggle('bad',!!(h&&!h.ok));
      b.classList.toggle('is-bad',!!(h&&!h.ok));
      const c=LIVE_TV_CHANNELS.find(x=>x.id===id)||{};
      b.title=(h? (h.ok?'Çalışıyor'+(h.ms?' · '+h.ms+' ms':''):'Çalışmıyor: '+(h.reason||'')) +' · denetim: '+new Date(h.ts*1000).toLocaleString(uiLocale()) : 'Henüz denetlenmedi')+(c.geo?'\nBölge kısıtı olabilir: '+c.geo:'')+(c.note?'\n'+c.note:'');
    });
  }
  let LIVE_SCAN=null;
  async function scanLiveChannels(scope){
    if(LIVE_SCAN){ LIVE_SCAN.stop=true; return; }
    const btn=document.getElementById('liveScanBtn');
    let list=LIVE_TV_CHANNELS.filter(c=>!c.officialOnly || c.tvg);
    if(scope!=='all'){ const act=(document.querySelector('.live-tv-continent-tab.active')||{}).dataset; const g=act&&liveGroups().find(x=>x.name===act.continent); if(g) list=g.items.slice(); }
    LIVE_SCAN={stop:false}; let done=0, ok=0;
    const upd=()=>{ if(btn) btn.textContent='⏹ Denetleniyor… '+done+'/'+list.length+' ('+ok+' çalışıyor)'; };
    upd();
    const q=list.slice(); const workers=Array.from({length:4},async()=>{
      while(q.length && !LIVE_SCAN.stop){ const c=q.shift(); const r=await probeChannel(c); done++; if(r&&r.ok) ok++; upd(); }
    });
    await Promise.all(workers);
    const stopped=LIVE_SCAN.stop; LIVE_SCAN=null;
    if(btn) btn.textContent='🩺 Bu bölgeyi denetle'; applyLiveFilter();
    nnToast((stopped?'Denetim durduruldu: ':'Denetim bitti: ')+ok+'/'+done+' kanal çalışıyor');
  }
  async function copyLiveReport(){
    await loadLiveHealth();
    const bad=LIVE_TV_CHANNELS.filter(c=>LIVE_HEALTH[c.id] && !LIVE_HEALTH[c.id].ok);
    if(!bad.length){ nnToast('Denetlenmiş ve çalışmayan kanal yok — önce 🩺 Kanalları denetle.'); return; }
    const lines=['NexNews '+(await (window.nnVersion?window.nnVersion():''))+' · canlı TV sorun raporu · '+new Date().toLocaleString(uiLocale()),''];
    bad.forEach(c=>{ const h=LIVE_HEALTH[c.id]; lines.push('■ '+c.name+' ['+c.id+'] — '+(h.reason||'?')+(h.code?' (HTTP '+h.code+')':'')); liveCands(c).forEach(x=>lines.push('   · '+x.u+(x.r?'  [ref '+x.r+']':''))); });
    const txt=lines.join('\n');
    let ok=false; try{ await navigator.clipboard.writeText(txt); ok=true; }catch(e){}
    if(!ok){ const ta=document.createElement('textarea'); ta.value=txt; document.body.appendChild(ta); ta.select(); try{ ok=document.execCommand('copy'); }catch(e){} ta.remove(); }
    nnToast(ok?(bad.length+' kanalın raporu panoya kopyalandı'):'Kopyalanamadı');
  }
  // ---- RC83: kategori, arama, favori, otomatik gizleme ----
  const LIVE_KINDS=[['all','Tümü'],['haber','Haber'],['ekonomi','Ekonomi'],['spor','Spor'],['belgesel','Belgesel'],['kultur','Kültür'],['genel','Genel'],['muzik','Müzik'],['cocuk','Çocuk / Eğitim']];
  let LIVE_KIND='all', LIVE_ONLY_OK=true, LIVE_Q='', LIVE_FAVS=[], LIVE_HIDDEN=[], LIVE_CUSTOM=[], LIVE_CHECK='ask', LIVE_STATE_READY=false, LIVE_LISTS=[], LIVE_BUILTIN_OFF=false;
  async function loadLiveState(){
    if(LIVE_STATE_READY) return;
    try{
      const st=JSON.parse(await window.nnGetLiveState());
      LIVE_FAVS=Array.isArray(st.favs)?st.favs:[]; window._liveLastId=st.last||'';
      LIVE_HIDDEN=Array.isArray(st.hidden)?st.hidden:[]; LIVE_CHECK=st.check||'ask';
      LIVE_CUSTOM=(Array.isArray(st.custom)?st.custom:[]).filter(c=>c&&c.id&&c.name);
    }catch(e){}
    // Kendi kanallarım ortak listeye katılır (tek oynatıcı / denetim yolu).
    LIVE_CUSTOM.forEach(c=>{ c.custom=true; c.continent='Kendi kanallarım'; if(!LIVE_TV_CHANNELS.some(x=>x.id===c.id)) LIVE_TV_CHANNELS.push(c); });
    await loadLiveLists();
    LIVE_STATE_READY=true;
  }
  // ---- RC87: kullanıcının kendi listeleri ----
  function guessLiveKind(g){
    g=String(g||'').toLowerCase();
    if(/news|haber|info|noticias|nachrichten|أخبار/.test(g)) return 'haber';
    if(/sport|spor|deport/.test(g)) return 'spor';
    if(/doc|belgesel/.test(g)) return 'belgesel';
    if(/music|müzik|musique/.test(g)) return 'muzik';
    if(/kid|child|çocuk|cocuk|anim|educ|eğitim/.test(g)) return 'cocuk';
    if(/business|econom|finans|ekonomi/.test(g)) return 'ekonomi';
    if(/cult|kültür|kultur|relig/.test(g)) return 'kultur';
    return 'genel';
  }
  function liveListGroupName(l){ return '📋 '+l.name; }
  async function loadLiveLists(){
    // önce eski liste kanallarını çıkar (yeniden yüklemede çift olmasın)
    for(let i=LIVE_TV_CHANNELS.length-1;i>=0;i--) if(LIVE_TV_CHANNELS[i].listId) LIVE_TV_CHANNELS.splice(i,1);
    let d={};
    try{ d=JSON.parse(await window.nnLiveLists())||{}; }catch(e){ d={}; }
    LIVE_BUILTIN_OFF=!!d.builtinOff; LIVE_LISTS=Array.isArray(d.lists)?d.lists:[];
    LIVE_LISTS.forEach(l=>(l.channels||[]).forEach(ch=>{
      const kind=guessLiveKind(ch.group);
      LIVE_TV_CHANNELS.push({id:ch.id,name:ch.name,listId:l.id,continent:liveListGroupName(l),country:ch.country||l.name,kind,
        meta:(ch.country||l.name)+' · '+(ch.group||((LIVE_KINDS.find(k=>k[0]===kind)||[])[1])||'Genel'),urls:ch.urls||[]});
    }));
  }
  function openLiveLists(){
    const old=document.getElementById('liveLists'); if(old) old.remove();
    const m=document.createElement('div'); m.className='modal-overlay open'; m.id='liveLists';
    m.onclick=e=>{ if(e.target===m) m.remove(); };
    const fmt=ts=>ts?new Date(ts*1000).toLocaleString(uiLocale(),{day:'2-digit',month:'2-digit',year:'numeric',hour:'2-digit',minute:'2-digit'}):'—';
    const rows=LIVE_LISTS.length?LIVE_LISTS.map(l=>'<div class="ll-row"><div class="ll-info"><b>'+esc(l.name)+'</b><small>'+(l.channels||[]).length+' kanal · son güncelleme: '+fmt(l.updated)+'</small><small class="ll-src">'+(l.url?esc(l.url):'Dosyadan / yapıştırarak eklendi')+'</small></div>'
      +'<div class="ll-act">'+(l.url?'<button class="mini" onclick="refreshLiveList(\''+esc(l.id)+'\',this)" title="Listeyi kaynağından şimdi yeniden indir">⟳ Güncelle</button>':'')
      +'<button class="mini" onclick="renameLiveList(\''+esc(l.id)+'\')">✎</button><button class="mini" onclick="deleteLiveList(\''+esc(l.id)+'\')" title="Listeyi ve kanallarını sil">🗑 Sil</button></div></div>').join('')
      :'<p class="la-note">Henüz liste eklemedin.</p>';
    m.innerHTML='<div class="modal live-add"><div class="modal-head"><h2>📋 Kanal listelerim</h2><button class="btn" onclick="document.getElementById(\'liveLists\').remove()">Kapat</button></div><div class="live-ask-body">'
      +'<label class="ll-builtin"><input type="checkbox" '+(LIVE_BUILTIN_OFF?'':'checked')+' onchange="setLiveBuiltin(this.checked)"> NexNews hazır kanal listesini göster</label>'
      +'<div class="ll-list">'+rows+'</div>'
      +'<h3 class="ll-h">Yeni liste ekle</h3>'
      +'<label>Liste adı<input id="llName" placeholder="Örn. Katar kanalları"></label>'
      +'<label>Liste adresi (M3U / M3U8)<input id="llUrl" placeholder="https://…/liste.m3u"></label>'
      +'<div class="ll-or">ya da <button class="mini" onclick="document.getElementById(\'llFile\').click()">📂 Bilgisayardan dosya seç</button><input type="file" id="llFile" accept=".m3u,.m3u8,.txt" class="hide" onchange="pickLiveListFile(this)"> <span id="llFileName"></span></div>'
      +'<p class="la-note">Listeyi sen seçersin ve yalnızca sen “Güncelle” dediğinde yenilenir; NexNews arka planda liste indirmez. Listedeki yayınların herkese açık ve yasal olmasından liste sahibi ve sen sorumlusunuz. Çalışmayan kanalları 🩺 denetimiyle ayıklayabilir, ✕ ile kaldırabilirsin.</p>'
      +'<div id="llStatus" class="la-status"></div></div>'
      +'<div class="modal-foot"><button class="btn primary" id="llSave" onclick="addLiveList()">Listeyi ekle</button></div></div>';
    document.body.appendChild(m);
  }
  let LL_FILE_TEXT='';
  function pickLiveListFile(inp){
    const f=inp.files&&inp.files[0]; if(!f) return;
    if(f.size>8*1024*1024){ nnToast('Dosya çok büyük (en çok 8 MB)'); return; }
    const r=new FileReader();
    r.onload=()=>{ LL_FILE_TEXT=String(r.result||''); document.getElementById('llFileName').textContent=f.name; const n=document.getElementById('llName'); if(n&&!n.value) n.value=f.name.replace(/\.(m3u8?|txt)$/i,''); };
    r.readAsText(f);
  }
  async function afterLiveListsChange(msg){
    await loadLiveLists(); rerenderLiveGroups(); openLiveLists(); if(msg) nnToast(msg);
  }
  async function addLiveList(){
    const st=document.getElementById('llStatus'), btn=document.getElementById('llSave');
    const name=document.getElementById('llName').value.trim(), url=document.getElementById('llUrl').value.trim();
    if(!url && !LL_FILE_TEXT){ st.textContent='Bir liste adresi yaz ya da dosya seç.'; return; }
    btn.disabled=true; st.textContent=url?'Liste indiriliyor…':'Liste okunuyor…';
    try{
      const l=JSON.parse(await nnAsync('nnLiveListAdd',name,url,url?'':LL_FILE_TEXT));
      LL_FILE_TEXT='';
      await afterLiveListsChange((l.channels||[]).length+' kanallık “'+l.name+'” listesi eklendi');
      selectLiveContinent(liveListGroupName(l));
    }catch(e){ st.textContent=String(e&&e.message||e); btn.disabled=false; }
  }
  async function refreshLiveList(id,b){
    if(b){ b.disabled=true; b.textContent='⟳ …'; }
    try{ const l=JSON.parse(await nnAsync('nnLiveListRefresh',id)); await afterLiveListsChange('“'+l.name+'” güncellendi · '+(l.channels||[]).length+' kanal'); }
    catch(e){ nnToast(String(e&&e.message||e)); if(b){ b.disabled=false; b.textContent='⟳ Güncelle'; } }
  }
  async function deleteLiveList(id){
    const l=LIVE_LISTS.find(x=>x.id===id); if(!l) return;
    if(!confirmInline('“'+l.name+'” listesi ve '+(l.channels||[]).length+' kanalı silinsin mi?')) return;
    try{ await window.nnLiveListDelete(id); }catch(e){}
    await afterLiveListsChange('Liste silindi');
  }
  async function renameLiveList(id){
    const l=LIVE_LISTS.find(x=>x.id===id); if(!l) return;
    const row=[...document.querySelectorAll('#liveLists .ll-row')][LIVE_LISTS.indexOf(l)]; if(!row) return;
    const b=row.querySelector('.ll-info b');
    b.innerHTML='<input class="ll-rename" value="'+esc(l.name)+'" maxlength="40">';
    const inp=b.querySelector('input'); inp.focus(); inp.select();
    const done=async ok=>{ const v=inp.value.trim(); if(ok&&v&&v!==l.name){ try{ await window.nnLiveListRename(id,v); }catch(e){} await afterLiveListsChange('Liste adı değişti'); } else openLiveLists(); };
    inp.onkeydown=e=>{ e.stopPropagation(); if(e.key==='Enter') done(true); if(e.key==='Escape') done(false); };
    inp.onblur=()=>done(true);
  }
  async function setLiveBuiltin(on){
    LIVE_BUILTIN_OFF=!on; try{ await window.nnLiveBuiltin(!!on); }catch(e){}
    rerenderLiveGroups(); nnToast(on?'Hazır kanal listesi gösteriliyor':'Hazır kanal listesi gizlendi — yalnızca senin kanalların ve listelerin görünür');
  }
  // Tarayıcı confirm() diyaloğu WebView'i kilitleyebildiği için iki adımlı onay.
  const CONFIRM_ARM={};
  function confirmInline(msg){
    const now=Date.now();
    if(CONFIRM_ARM[msg] && now-CONFIRM_ARM[msg]<5000){ delete CONFIRM_ARM[msg]; return true; }
    CONFIRM_ARM[msg]=now; nnToast(msg+' Onaylamak için 5 sn içinde yeniden bas.'); return false;
  }
  function saveLiveState(){
    const st={favs:LIVE_FAVS,last:(window._liveTVCurrent||{}).id||window._liveLastId||'',hidden:LIVE_HIDDEN,check:LIVE_CHECK,
      custom:LIVE_CUSTOM.map(c=>({id:c.id,name:c.name,meta:c.meta,country:c.country,kind:c.kind,urls:c.urls}))};
    try{ window.nnSetLiveState(JSON.stringify(st)); }catch(e){}
  }
  try{ const v=localStorage.getItem('nnLiveOnlyOk'); if(v!==null) LIVE_ONLY_OK=v==='1'; }catch(e){}
  function liveKindOf(c){
    if(c.kind) return c.kind==='egitim'?'cocuk':c.kind;
    const m=(c.meta||'').toLowerCase();
    if(/spor|sport|yarış/.test(m)) return 'spor';
    if(/ekonomi|finans|business/.test(m)) return 'ekonomi';
    if(/belgesel|doğa|tarih|bilim|uzay|tarım/.test(m)) return 'belgesel';
    if(/kültür|sanat/.test(m)) return 'kultur';
    if(/haber|news|parlamento|meclis|uluslararası|english|العربية/.test(m)) return 'haber';
    if(/müzik|music/.test(m)) return 'muzik';
    if(/çocuk|eğitim/.test(m)) return 'cocuk';
    return 'genel';
  }
  function liveHidden(c){
    const h=LIVE_HEALTH[c.id];
    if(LIVE_KIND!=='all' && liveKindOf(c)!==LIVE_KIND) return true;
    if(LIVE_ONLY_OK && h && !h.ok) return true;
    if(LIVE_Q){ const t=(c.name+' '+(c.country||'')+' '+(c.meta||'')).toLocaleLowerCase('tr'); if(!t.includes(LIVE_Q)) return true; }
    return false;
  }
  function liveFilter(kind){
    LIVE_KIND=kind;
    document.querySelectorAll('.live-kind').forEach(b=>b.classList.toggle('active',b.dataset.kind===kind));
    applyLiveFilter();
  }
  function liveSearch(q){ LIVE_Q=String(q||'').trim().toLocaleLowerCase('tr'); applyLiveFilter(); }
  function applyLiveFilter(){
    document.querySelectorAll('.live-tv-channel').forEach(b=>{
      const c=LIVE_TV_CHANNELS.find(x=>x.id===b.dataset.tvId)||{};
      b.classList.toggle('hide',liveHidden(c));
    });
    // Sekme ve kategori sayıları görünen kanallara göre.
    const groups=liveGroups();
    document.querySelectorAll('.live-tv-continent-tab').forEach(t=>{
      const g=groups.find(x=>x.name===t.dataset.continent); const sp=t.querySelector('span');
      if(g && sp) sp.textContent=g.items.filter(c=>!liveHidden(c)).length;
    });
    const act=(document.querySelector('.live-tv-continent-tab.active')||{}).dataset||{};
    const g=groups.find(x=>x.name===act.continent);
    if(g){
      const saveK=LIVE_KIND;
      document.querySelectorAll('.live-kind').forEach(b=>{
        LIVE_KIND=b.dataset.kind; const n=g.items.filter(c=>!liveHidden(c)).length; LIVE_KIND=saveK;
        let sp=b.querySelector('i'); if(!sp){ sp=document.createElement('i'); b.appendChild(sp); } sp.textContent=n; b.classList.toggle('zero',n===0 && b.dataset.kind!=='all');
      });
      const hid=g.items.filter(c=>LIVE_HEALTH[c.id] && !LIVE_HEALTH[c.id].ok).length;
      const hn=document.getElementById('liveHiddenNote');
      if(hn) hn.textContent = (LIVE_ONLY_OK && hid) ? (hid+' kanal bu bağlantıdan izlenemediği için gizlendi') : (liveStaleCount(act.continent)?(liveStaleCount(act.continent)+' kanal henüz denetlenmedi'):'');
      document.querySelectorAll('.live-tv-continent-panel.active .live-tv-empty').forEach(e=>e.remove());
      const panel=document.querySelector('.live-tv-continent-panel.active .live-tv-grid');
      if(panel && !panel.querySelector('.live-tv-channel:not(.hide)')){ const e=document.createElement('div'); e.className='live-tv-empty'; e.textContent=act.continent==='Favoriler'?'Henüz favori kanal yok — bir kanalın ☆ işaretine bas.':'Bu filtreye uyan kanal yok.'; panel.appendChild(e); }
    }
  }
  function toggleLiveOnlyOk(on){ LIVE_ONLY_OK=!!on; try{ localStorage.setItem('nnLiveOnlyOk',on?'1':'0'); }catch(e){} applyLiveFilter(); }
  async function toggleLiveFav(id,ev){
    if(ev){ ev.stopPropagation(); ev.preventDefault(); }
    const i=LIVE_FAVS.indexOf(id); if(i>=0) LIVE_FAVS.splice(i,1); else LIVE_FAVS.push(id);
    saveLiveState();
    document.querySelectorAll('.live-fav[data-fav="'+id+'"]').forEach(b=>{ b.classList.toggle('on',LIVE_FAVS.includes(id)); b.textContent=LIVE_FAVS.includes(id)?'★':'☆'; });
    const fp=document.querySelector('.live-tv-continent-panel[data-live-continent="Favoriler"]');
    if(fp){ const g=liveGroups().find(x=>x.name==='Favoriler'); const tmp=document.createElement('div'); tmp.innerHTML=renderLiveTVGroup(g,fp.classList.contains('active')?0:1); fp.replaceWith(tmp.firstChild); }
    applyLiveFilter();
    nnToast(LIVE_FAVS.includes(id)?'Favori kanallara eklendi':'Favorilerden çıkarıldı');
  }
  // Kanalları arka planda sessizce denetle: denetlenmemiş ya da eski sonuçlar
  // (çalışan: 3 gün, çalışmayan: 1 gün). Önce açık sekme, sonra diğerleri.
  let LIVE_AUTO=false;
  async function autoCheckLive(){
    if(LIVE_AUTO || LIVE_SCAN || !window.nnProbeLive) return;
    LIVE_AUTO=true;
    const now=Date.now()/1000;
    const stale=c=>{ const h=LIVE_HEALTH[c.id]; if(!h) return true; return (now-(h.ts||0)) > (h.ok?3*86400:86400); };
    const act=(document.querySelector('.live-tv-continent-tab.active')||{}).dataset||{};
    let list=LIVE_TV_CHANNELS.filter(stale);
    list.sort((a,b)=>(a.continent===act.continent?0:1)-(b.continent===act.continent?0:1));
    if(document.body.classList.contains('data-lite')) list=list.filter(c=>c.continent===act.continent); // sınırlı veri: yalnız açık bölge
    const q=list.slice();
    await Promise.all(Array.from({length:3},async()=>{
      while(q.length && !document.hidden && !LIVE_SCAN){ const c=q.shift(); await probeChannel(c); applyLiveFilter(); }
    }));
    LIVE_AUTO=false;
  }

  async function playLiveTV(c){
    const frame=document.getElementById('liveTVFrame'); if(!frame)return;
    mediaBegin('tv', c.name, c.meta||'Canlı TV');
    const video=document.getElementById('liveTVVideo');
    const status=document.getElementById('liveTVStatus');
    if(!video)return;
    const token=(window._liveTVToken=(window._liveTVToken||0)+1); // eski denemeler yenisini bozmasın
    hideLiveOverlay();
    let sources=liveCands(c);
    if(c.officialOnly && !c.tvg){
      destroyLivePlayer();
      if(status) status.textContent='Resmî web yayını';
      showLiveOverlay(c.name, c.note || 'Bu kanal için NexNews içinde oynatılabilen herkese açık bir yayın adresi yok. Resmî yayın ayrı pencerede açılabilir.');
      return;
    }
    let attempt=0, repaired=false;
    const fallback=async()=>{
      if(token!==window._liveTVToken) return;
      // Bütün adresler düştü: kısa bir denetimle gerçek nedeni bul, gerekirse
      // iptv-org dizininden lisanslı güncel bir adres dene (otomatik onarım).
      if(!repaired){
        repaired=true;
        if(status) status.textContent='Kaynak denetleniyor / onarılıyor…';
        const r=await probeChannel(c);
        if(token!==window._liveTVToken) return;
        if(r && r.ok){
          const x=r.repair || sources.find(s=>s.u===r.url) || {u:r.url};
          sources=[x]; attempt=0;
          if(status) status.textContent=r.repair?'Yeni kaynak bulundu, bağlanıyor…':'Kaynak çalışıyor, yeniden bağlanıyor…';
          return tryNext();
        }
        const why=(r&&r.reason)?r.reason:'Yayın kaynağına ulaşılamadı.';
        const geo=c.geo?(' Bu kanal yalnızca '+c.geo+' içinden izlenebiliyor olabilir.'):'';
        showLiveOverlay('Yayın oynatılamadı', why+'.'+geo+(c.note?' '+c.note:'')+' Diğer kanallar etkilenmez.');
        if(status) status.textContent='Yayın kaynağı kullanılamıyor';
        return;
      }
      showLiveOverlay('Yayın oynatılamadı','Kaynak yanıt veriyor ama oynatıcı akışı çözemedi (kodek ya da DRM). Sonraki kanala geçebilirsin.');
      if(status) status.textContent='Yayın kaynağı kullanılamıyor';
    };
    const tryNext=async()=>{
      if(token!==window._liveTVToken) return;
      destroyLivePlayer();
      if(attempt>=sources.length){ return fallback(); }
      const x=sources[attempt++];
      hideLiveOverlay();
      const relayReady=await ensureLiveRelay();
      if(token!==window._liveTVToken) return;
      const src=relayReady ? relaySrc(x) : x.u;
      if(status) status.textContent=attempt>1?'Alternatif yayın kaynağı deneniyor ('+attempt+'/'+sources.length+')…':(relayReady?'Canlı yayın bağlanıyor…':'Doğrudan yayın deneniyor…');
      let settled=false;
      const fail=()=>{ if(settled||token!==window._liveTVToken) return; settled=true; clearTimeout(guard); setTimeout(()=>tryNext(),150); };
      // 20 sn içinde görüntü gelmezse sonraki adrese geç (asılı kalan CDN'ler).
      const guard=setTimeout(()=>{ if(video.readyState<2) fail(); }, 20000);
      video.onerror=fail;
      try{
        const Hls=await ensureHls();
        if(token!==window._liveTVToken) return;
        if(Hls && Hls.isSupported()){
          const lim=document.body.classList.contains('data-lite');
          const hls=new Hls({
            enableWorker:true,lowLatencyMode:false,backBufferLength:lim?10:30,maxBufferLength:lim?12:30,
            liveSyncDurationCount:3,capLevelToPlayerSize:lim,startLevel:lim?0:-1,abrEwmaDefaultEstimate:lim?5e5:2.5e6,abrBandWidthUpFactor:.8,
            manifestLoadingMaxRetry:2,levelLoadingMaxRetry:3,fragLoadingMaxRetry:3,
            manifestLoadingRetryDelay:700,levelLoadingRetryDelay:700,fragLoadingRetryDelay:700
          });
          LIVE_HLS=hls;
          let mediaRecover=0;
          hls.on(Hls.Events.ERROR,(ev,data)=>{
            if(!data || !data.fatal) return;
            // Kısa kesintide akışı bırakma: önce kurtarmayı dene.
            if(data.type===Hls.ErrorTypes.MEDIA_ERROR && mediaRecover<2){ mediaRecover++; try{ hls.recoverMediaError(); }catch(e){ fail(); } return; }
            fail();
          });
          hls.on(Hls.Events.MANIFEST_PARSED,(ev,d)=>{
            // RC86 "ses boğuklaşıyor": ABR zamanla en düşük varyanta iniyor; bu
            // varyantların çoğunda ses 32–64 kbps HE-AAC (boğuk). Tam modda en
            // düşük katları ABR'nin dışında tutuyoruz (≥ ~360p / 0,8 Mbps).
            try{
              if(!lim && d && d.levels && d.levels.length>1){
                const br=d.levels.map(l=>l.bitrate||0).filter(Boolean).sort((a,b)=>a-b);
                const top=br[br.length-1]||0;
                const floor=Math.min(1.2e6, top*0.5, Math.max(8e5, br[0]));
                hls.config.minAutoBitrate=floor;
              }
            }catch(e){}
            video.play().catch(()=>{});
            if(status) status.textContent='● CANLI'+(lim?' · sınırlı veri':'');
          });
          hls.on(Hls.Events.FRAG_BUFFERED,()=>{ if(!settled){ settled=true; clearTimeout(guard); LIVE_HEALTH[c.id]={ok:true,url:x.u,ts:Math.floor(Date.now()/1000)}; paintLiveHealth(c.id); } });
          hls.loadSource(src); hls.attachMedia(video);
        }else if(video.canPlayType('application/vnd.apple.mpegurl')){
          video.src=src; video.play().catch(()=>{});
        }else{ clearTimeout(guard); fallback(); }
      }catch(e){ fail(); }
    };
    if(!sources.length){ return fallback(); }
    await tryNext();
  }
  const LIVE_TV_REGIONS=[['Türkiye','🇹🇷'],['Avrupa','🇪🇺'],['Orta Doğu','🕌'],['Asya','🌏'],['Afrika','🌍'],['Kuzey Amerika','🇺🇸'],['Güney Amerika','🌎'],['Okyanusya','🇦🇺'],['Rusya','🇷🇺'],['Belgesel & Spor','🎬']];
  function liveGroups(){
    const vis=LIVE_TV_CHANNELS.filter(c=>!LIVE_HIDDEN.includes(c.id) && !(LIVE_BUILTIN_OFF && !c.custom && !c.listId));
    const g=LIVE_TV_REGIONS.map(([n,i])=>({name:n,icon:i,items:vis.filter(c=>c.continent===n)})).filter(x=>x.items.length);
    g.unshift({name:'Favoriler',icon:'⭐',items:LIVE_FAVS.map(id=>vis.find(c=>c.id===id)).filter(Boolean)});
    g.splice(1,0,{name:'Kendi kanallarım',icon:'📌',items:vis.filter(c=>c.custom)});
    LIVE_LISTS.forEach((l,k)=>g.splice(2+k,0,{name:liveListGroupName(l),icon:'',items:vis.filter(c=>c.listId===l.id)}));
    g.push({name:'Tüm dünya',icon:'🌐',items:vis});
    if(LIVE_HIDDEN.length) g.push({name:'Gizlenenler',icon:'🚫',items:LIVE_HIDDEN.map(id=>LIVE_TV_CHANNELS.find(c=>c.id===id)).filter(Boolean)});
    return g;
  }
  function renderLiveTVGroup(grp,i){
    return '<div class="live-tv-continent-panel '+(i===0?'active':'')+'" data-live-continent="'+esc(grp.name)+'"><div class="live-tv-group-title"><span>'+esc(grp.icon)+'</span><b>'+esc(grp.name)+'</b><small>'+grp.items.length+' kanal</small></div><div class="live-tv-grid">'+grp.items.map((c)=>{
      const hh=LIVE_HEALTH[c.id]; const fav=LIVE_FAVS.includes(c.id);
      const dot='live-tv-dot '+(hh?(hh.ok?'ok':'bad'):'');
      const where=grp.name==='Tüm dünya'||grp.name==='Favoriler'||grp.name==='Belgesel & Spor' ? (c.country||c.continent) : '';
      return '<div class="live-tv-channel'+(hh&&!hh.ok?' is-bad':'')+'" role="button" tabindex="0" data-tv-id="'+esc(c.id)+'" onclick="selectLiveTV(\''+esc(c.id)+'\')" onkeydown="if(event.key===\'Enter\')selectLiveTV(\''+esc(c.id)+'\')"><span class="live-tv-name"><span class="'+dot+'"></span>'+esc(c.name)+'<button type="button" class="live-fav'+(fav?' on':'')+'" data-fav="'+esc(c.id)+'" title="Favori kanal" onclick="toggleLiveFav(\''+esc(c.id)+'\',event)">'+(fav?'★':'☆')+'</button>'+(grp.name==='Gizlenenler'?'<button type="button" class="live-x" title="Listeye geri al" onclick="unhideLiveChannel(\''+esc(c.id)+'\',event)">↺</button>':(c.custom?'<button type="button" class="live-x" title="Bu kanalı sil" onclick="deleteCustomChannel(\''+esc(c.id)+'\',event)">🗑</button>':'<button type="button" class="live-x" title="Bu kanalı listeden kaldır" onclick="hideLiveChannel(\''+esc(c.id)+'\',event)">✕</button>'))+(c.geo?'<span class="live-tv-geo" title="Yalnızca kendi ülkesinden izlenebilir">🌐 '+esc(c.geo)+'</span>':'')+'</span><span class="live-tv-meta">'+esc(where?(where+' · '+(c.meta||'').split(' · ').slice(1).join(' · ')):c.meta)+'</span></div>';
    }).join('')+'</div></div>';
  }
  function selectLiveContinent(name){
    document.querySelectorAll('.live-tv-continent-tab').forEach(b=>b.classList.toggle('active',b.dataset.continent===name));
    document.querySelectorAll('.live-tv-continent-panel').forEach(p=>p.classList.toggle('active',p.dataset.liveContinent===name));
    applyLiveFilter();
    autoCheckLive();
  }
  async function openLiveTV(el){
    stopRadar();
    await loadLiveHealth();
    await loadLiveState();
    const groups=liveGroups();
    const startName=LIVE_FAVS.filter(id=>!LIVE_HIDDEN.includes(id)).length?'Favoriler':'Türkiye';
    let startIdx=groups.findIndex(g=>g.name===startName);
    if(startIdx<0) startIdx=Math.max(0,groups.findIndex(g=>g.items.length));
    document.getElementById('overview').classList.add('hide');
    const p=document.getElementById('catpage'); p.classList.remove('hide');
    p.innerHTML='<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button><h1>📺 Canlı TV</h1><span class="pill">'+LIVE_TV_CHANNELS.length+' kanal · '+(groups.length-2)+' bölge</span></div>'+
      '<div class="live-tv-card">'+
      '<div class="live-tv-nav">'+groups.map((g,i)=>'<button class="live-tv-continent-tab '+(i===startIdx?'active':'')+'" data-continent="'+esc(g.name)+'" onclick="selectLiveContinent(\''+esc(g.name)+'\')">'+g.icon+' '+esc(g.name)+' <span>'+g.items.length+'</span></button>').join('')+'</div>'+
      '<div class="live-tv-tools">'+LIVE_KINDS.map(k=>'<button class="live-kind'+(LIVE_KIND===k[0]?' active':'')+'" data-kind="'+k[0]+'" onclick="liveFilter(\''+k[0]+'\')">'+k[1]+'</button>').join('')+
      '<input class="live-search" type="search" placeholder="Kanal veya ülke ara…" value="'+esc(LIVE_Q)+'" oninput="liveSearch(this.value)" onkeydown="event.stopPropagation()">'+
      '</div><div class="live-tv-tools2"><label class="live-okonly"><input type="checkbox" '+(LIVE_ONLY_OK?'checked':'')+' onchange="toggleLiveOnlyOk(this.checked)"> Yalnızca bu bağlantıdan izlenebilenler</label><span id="liveHiddenNote" class="live-hidden-note"></span>'+
      '<button class="mini" onclick="copyLiveReport()" title="Çalışmayan kanalları, nedenlerini ve denenen adresleri panoya kopyalar">📋 Sorun raporu</button><button class="mini" id="liveScanBtn" onclick="scanLiveChannels()" title="Bu bölgedeki kanalları şimdi dene (arka planda)">🩺 Bu bölgeyi denetle</button><button class="mini" onclick="openAddChannel()" title="Kendi yayın adresini ekle">＋ Kanal ekle</button><button class="mini" onclick="openLiveLists()" title="Açık kaynak M3U listelerini adresle ya da dosyadan ekle, güncelle, sil">📋 Listelerim</button></div>'+
      '<div class="live-tv-layout"><aside class="live-tv-side"><div class="live-tv-slot" id="liveTVSlot"></div>'+
      '<div class="live-tv-note">Yalnızca yayıncıların kendi ya da lisanslı ücretsiz platformların (Samsung TV Plus, Pluto, Xumo, Rakuten…) yayınları oynatılır. İstersen NexNews kanalları senin bağlantından kısaca dener ve izlenemeyenleri (ör. yalnızca kendi ülkesinde açık olanlar) gizler. ☆ favori · ✕ listeden kaldır · ＋ kendi kanalını ekle.</div></aside>'+
      '<div id="liveGroupsHost">'+groups.map((g,i)=>renderLiveTVGroup(g,i===startIdx?0:1)).join('')+'</div></div></div>';
    window.scrollTo(0,0);
    applyLiveFilter();
    if(MEDIA.kind==='tv' && window._liveTVCurrent){
      document.querySelectorAll('.live-tv-channel').forEach(b=>b.classList.toggle('active',b.dataset.tvId===window._liveTVCurrent.id));
      syncDock();
    }else{
      let c=LIVE_TV_CHANNELS.find(x=>x.id===window._liveLastId);
      if(!c || (LIVE_HEALTH[c.id] && !LIVE_HEALTH[c.id].ok)) c=LIVE_TV_CHANNELS[0];
      window._liveTVCurrent=c;
      document.querySelectorAll('.live-tv-channel').forEach(b=>b.classList.toggle('active',b.dataset.tvId===c.id));
      playLiveTV(c);
    }
    setTimeout(maybeAskLiveCheck, 900);
  }
  // RC85: kanal denetimi kullanıcıya sorularak yapılır.
  function liveStaleCount(scope){
    const now=Date.now()/1000;
    return LIVE_TV_CHANNELS.filter(c=>!LIVE_HIDDEN.includes(c.id)&&(!scope||c.continent===scope)).filter(c=>{ const h=LIVE_HEALTH[c.id]; return !h || (now-(h.ts||0)) > (h.ok?7*86400:3*86400); }).length;
  }
  function maybeAskLiveCheck(){
    if(LIVE_CHECK==='never') return;
    const n=liveStaleCount();
    if(!n) return;
    if(LIVE_CHECK==='auto'){ autoCheckLive(); return; }
    if(sessionStorageGet('nnLiveAsked')) return;
    sessionStorageSet('nnLiveAsked','1');
    const m=document.createElement('div'); m.className='modal-overlay open'; m.id='liveAsk';
    const kb=Math.round(n*18/1024*10)/10;
    m.innerHTML='<div class="modal live-ask"><div class="modal-head"><h2>📺 Kanal listesini doğrulayalım mı?</h2></div><div class="live-ask-body">'
      +'<p>NexNews, listedeki <b>'+n+' kanalı</b> senin internet bağlantından kısaca deneyip izlenemeyenleri (kapanmış ya da yalnızca kendi ülkesinde açık yayınları) listeden gizleyebilir.</p>'
      +'<ul><li>Süre: yaklaşık <b>'+Math.max(1,Math.round(n/60))+'–'+Math.max(2,Math.round(n/30))+' dakika</b></li><li>Veri: yaklaşık <b>'+(kb<1?'1':kb)+' MB</b></li><li>Bu sırada uygulamayı kullanmaya devam edebilirsin; yavaş bilgisayarlarda kanal listesi kısa süre takılabilir.</li></ul>'
      +'<label class="live-ask-remember"><input type="checkbox" id="liveAskRemember"> Seçimimi hatırla</label></div>'
      +'<div class="modal-foot"><button class="btn" onclick="answerLiveCheck(\'never\')">Doğrulama</button><button class="btn" onclick="answerLiveCheck(\'region\')">Yalnızca açık bölge</button><button class="btn primary" onclick="answerLiveCheck(\'all\')">Tümünü doğrula</button></div></div>';
    document.body.appendChild(m);
  }
  function answerLiveCheck(a){
    const rem=(document.getElementById('liveAskRemember')||{}).checked;
    const m=document.getElementById('liveAsk'); if(m) m.remove();
    if(rem){ LIVE_CHECK=(a==='never')?'never':'auto'; saveLiveState(); }
    if(a==='all') autoCheckLive();
    else if(a==='region') scanLiveChannels();
  }
  function sessionStorageGet(k){ try{ return sessionStorage.getItem(k); }catch(e){ return window['_ss_'+k]; } }
  function sessionStorageSet(k,v){ try{ sessionStorage.setItem(k,v); }catch(e){ window['_ss_'+k]=v; } }

  // ---- RC85: kanal gizleme / elle ekleme ----
  function hideLiveChannel(id,ev){
    if(ev){ ev.stopPropagation(); ev.preventDefault(); }
    if(!LIVE_HIDDEN.includes(id)) LIVE_HIDDEN.push(id);
    LIVE_FAVS=LIVE_FAVS.filter(x=>x!==id); saveLiveState(); rerenderLiveGroups();
    nnToast('Kanal listeden kaldırıldı — 🚫 Gizlenenler sekmesinden geri alabilirsin');
  }
  function unhideLiveChannel(id,ev){
    if(ev){ ev.stopPropagation(); ev.preventDefault(); }
    LIVE_HIDDEN=LIVE_HIDDEN.filter(x=>x!==id); saveLiveState(); rerenderLiveGroups(); nnToast('Kanal listeye geri alındı');
  }
  function deleteCustomChannel(id,ev){
    if(ev){ ev.stopPropagation(); ev.preventDefault(); }
    LIVE_CUSTOM=LIVE_CUSTOM.filter(c=>c.id!==id);
    const i=LIVE_TV_CHANNELS.findIndex(c=>c.id===id); if(i>=0) LIVE_TV_CHANNELS.splice(i,1);
    LIVE_FAVS=LIVE_FAVS.filter(x=>x!==id); saveLiveState(); rerenderLiveGroups(); nnToast('Kanal silindi');
  }
  function rerenderLiveGroups(){
    const act=((document.querySelector('.live-tv-continent-tab.active')||{}).dataset||{}).continent;
    const groups=liveGroups(); const cur=groups.some(g=>g.name===act)?act:((groups.find(g=>g.name==='Türkiye')||groups.find(g=>g.items.length)||groups[0]||{}).name);
    const nav=document.querySelector('.live-tv-nav'); const host=document.getElementById('liveGroupsHost');
    if(!nav||!host) return;
    nav.innerHTML=groups.map(g=>'<button class="live-tv-continent-tab '+(g.name===cur?'active':'')+'" data-continent="'+esc(g.name)+'" onclick="selectLiveContinent(\''+esc(g.name)+'\')">'+g.icon+' '+esc(g.name)+' <span>'+g.items.length+'</span></button>').join('');
    host.innerHTML=groups.map(g=>renderLiveTVGroup(g,g.name===cur?0:1)).join('');
    const c=window._liveTVCurrent; if(c) document.querySelectorAll('.live-tv-channel').forEach(b=>b.classList.toggle('active',b.dataset.tvId===c.id));
    applyLiveFilter();
  }
  function openAddChannel(){
    const m=document.createElement('div'); m.className='modal-overlay open'; m.id='liveAdd';
    m.onclick=e=>{ if(e.target===m) m.remove(); };
    m.innerHTML='<div class="modal live-add"><div class="modal-head"><h2>＋ Kanal ekle</h2><button class="btn" onclick="document.getElementById(\'liveAdd\').remove()">Kapat</button></div><div class="live-ask-body">'
      +'<div class="la-tabs"><button class="la-tab active" onclick="laTab(this,\'one\')">Tek kanal</button><button class="la-tab" onclick="laTab(this,\'m3u\')">M3U listesi yapıştır</button></div>'
      +'<div class="la-pane" data-pane="one"><label>Kanal adı<input id="laName" placeholder="Örn. Yerel Belediye TV"></label>'
      +'<label>Yayın adresi (HLS .m3u8)<input id="laUrl" placeholder="https://…/playlist.m3u8"></label>'
      +'<div class="la-row"><label>Kategori<select id="laKind">'+LIVE_KINDS.filter(k=>k[0]!=='all').map(k=>'<option value="'+k[0]+'">'+k[1]+'</option>').join('')+'</select></label>'
      +'<label>Ülke / not<input id="laCountry" placeholder="Türkiye"></label></div>'
      +'<details><summary>Gelişmiş</summary><label>Referer (gerekirse)<input id="laRef" placeholder="https://kanalin-sitesi.com/"></label></details></div>'
      +'<div class="la-pane hide" data-pane="m3u"><textarea id="laM3u" rows="8" placeholder="#EXTM3U&#10;#EXTINF:-1,Kanal adı&#10;https://…/index.m3u8"></textarea></div>'
      +'<p class="la-note">Yalnızca yayıncısının herkese açık olarak sunduğu yayınları ekle. Eklenen adres önce kısaca denenir; çalışmazsa kaydedilmez.</p>'
      +'<div id="laStatus" class="la-status"></div></div>'
      +'<div class="modal-foot"><button class="btn primary" id="laSave" onclick="saveAddChannel()">Dene ve ekle</button></div></div>';
    document.body.appendChild(m);
    setTimeout(()=>{ const i=document.getElementById('laName'); if(i) i.focus(); },50);
  }
  function laTab(b,k){ document.querySelectorAll('.la-tab').forEach(x=>x.classList.toggle('active',x===b)); document.querySelectorAll('.la-pane').forEach(x=>x.classList.toggle('hide',x.dataset.pane!==k)); }
  function parseM3U(txt){
    const out=[]; let cur=null;
    String(txt||'').split(/\r?\n/).forEach(l=>{ l=l.trim();
      if(l.startsWith('#EXTINF')){ cur={name:(l.split(',').pop()||'').trim()}; const r=l.match(/http-referrer="([^"]+)"/); if(r) cur.r=r[1]; }
      else if(l.startsWith('#EXTVLCOPT:http-referrer=') && cur){ cur.r=l.split('=').slice(1).join('='); }
      else if(/^https?:\/\//i.test(l)){ out.push({name:(cur&&cur.name)||('Kanal '+(out.length+1)),u:l,r:cur&&cur.r}); cur=null; }
    });
    return out;
  }
  async function saveAddChannel(){
    const st=document.getElementById('laStatus'), btn=document.getElementById('laSave');
    const m3uMode=!document.querySelector('.la-pane[data-pane="m3u"]').classList.contains('hide');
    let items=[];
    if(m3uMode){ items=parseM3U(document.getElementById('laM3u').value).slice(0,100); }
    else{
      const name=document.getElementById('laName').value.trim(), u=document.getElementById('laUrl').value.trim(), r=document.getElementById('laRef').value.trim();
      if(!name||!/^https?:\/\//i.test(u)){ st.textContent='Kanal adı ve http(s) ile başlayan bir yayın adresi gerekli.'; return; }
      items=[{name,u,r}];
    }
    if(!items.length){ st.textContent='Listede yayın adresi bulunamadı.'; return; }
    const kind=(document.getElementById('laKind')||{}).value||'genel', country=((document.getElementById('laCountry')||{}).value||'').trim()||'Kendi kanalım';
    btn.disabled=true; let ok=0, bad=0;
    for(const it of items){
      st.textContent='Deneniyor: '+it.name+' ('+(ok+bad+1)+'/'+items.length+')…';
      const c={id:'my-'+Date.now().toString(36)+'-'+Math.random().toString(36).slice(2,6),name:it.name.slice(0,60),meta:country+' · '+((LIVE_KINDS.find(k=>k[0]===kind)||[])[1]||'Genel'),country,kind,custom:true,continent:'Kendi kanallarım',urls:[it.r?{u:it.u,r:it.r}:it.u]};
      const r=await probeChannel(c);
      if(r&&r.ok){ LIVE_CUSTOM.push(c); LIVE_TV_CHANNELS.push(c); ok++; } else bad++;
    }
    btn.disabled=false; saveLiveState();
    if(ok){ document.getElementById('liveAdd').remove(); rerenderLiveGroups(); selectLiveContinent('Kendi kanallarım'); nnToast(ok+' kanal eklendi'+(bad?(' · '+bad+' adres çalışmadığı için eklenmedi'):'')); }
    else st.textContent='Adres çalışmadı: '+((LIVE_HEALTH && Object.values(LIVE_HEALTH).slice(-1)[0]||{}).reason||'yayın açılamadı')+'. Adresi ve gerekiyorsa Referer bilgisini kontrol et.';
  }
  function selectLiveTV(id){
    const c=LIVE_TV_CHANNELS.find(x=>x.id===id)||LIVE_TV_CHANNELS[0]; window._liveTVCurrent=c;
    document.querySelectorAll('.live-tv-channel').forEach(b=>b.classList.toggle('active',b.dataset.tvId===c.id));
    saveLiveState();
    playLiveTV(c);
  }
  // Aynı bölgede sıradaki görünür (çalışan) kanala geç — kullanıcı pencereden çıkmaz.
  function nextLiveTV(){
    applyLiveFilter();
    const btns=Array.from(document.querySelectorAll('.live-tv-continent-panel.active .live-tv-channel:not(.hide)'));
    const cur=(window._liveTVCurrent||{}).id; const i=btns.findIndex(b=>b.dataset.tvId===cur);
    const nx=btns[(i+1)%Math.max(1,btns.length)]; if(nx && nx.dataset.tvId!==cur) selectLiveTV(nx.dataset.tvId);
  }
  function openCurrentLiveTV(){ const c=window._liveTVCurrent||LIVE_TV_CHANNELS[0]; const f=(liveCands(c)[0]||{}).u; const u=c.official||f; if(!u){ nnToast('Bu kanalın resmî sayfası kayıtlı değil.'); return; } try{window.nnOpen(u);}catch(e){window.open(u,'_blank');} }

  function openWorldRadio(){
    stopRadar();
    document.getElementById('overview').classList.add('hide');
    const p=document.getElementById('catpage');
    p.classList.remove('hide');
    p.innerHTML='<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button><h1>📻 Dünya Radyoları</h1><span class="pill">Canlı radyo istasyonları</span></div>'+
      '<div class="radio-card" style="margin-top:0"><div class="radio-head"><h3>📻 Dünya Radyoları</h3><span id="radioUpdated">hazırlanıyor…</span></div>'+
      '<div class="radio-sub" id="radioSub">Yerel istasyonlar + dünyanın öne çıkan haber radyoları</div>'+
      '<div id="radioList" class="radio-list"></div>'+
      '<div class="radio-foot">Seçtiğin istasyon alttaki oynatıcıda çalar; başka sayfalara geçsen de çalmaya devam eder.</div>'+
      '<div class="radio-foot">Canlı yayınlar istasyonların herkese açık akışlarından oynatılır. İstasyon dizini: Radio Browser.</div></div>';
    window.scrollTo(0,0);
    loadRadio();
  }

