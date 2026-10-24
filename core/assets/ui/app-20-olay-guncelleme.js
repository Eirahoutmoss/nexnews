  // ---------- olay dosyası (event file) + zaman çizelgesi ----------
  function clockTime(ts){
    if(!ts) return "—";
    const d = new Date(ts*1000);
    const hhmm = d.toLocaleTimeString(uiLocale(),{hour:"2-digit",minute:"2-digit"});
    const ageDays = (Date.now()/1000 - ts)/86400;
    if(ageDays >= 1){ return d.toLocaleDateString(uiLocale(),{day:"numeric",month:"short"})+" "+hhmm; }
    return hhmm;
  }
  function eventDuration(a,b){
    if(!a||!b||b<=a) return "";
    const mins = Math.round((b-a)/60);
    if(mins < 60) return mins+" dk";
    if(mins < 1440) return Math.round(mins/60)+" sa";
    return Math.round(mins/1440)+" gün";
  }
  function timelineIntelHTML(e){
    const tl=Array.isArray(e.timeline)?e.timeline.filter(x=>x&&x.ts):[];
    if(!tl.length) return '';
    let gaps=[];
    for(let i=1;i<tl.length;i++) gaps.push(Math.max(0,tl[i].ts-tl[i-1].ts));
    const maxGap=gaps.length?Math.max(...gaps):0;
    const total=(e.firstTs&&e.lastTs)?Math.max(0,e.lastTs-e.firstTs):0;
    const sources=new Set(tl.map(x=>String(x.source||'').trim()).filter(Boolean));
    let phase='İlk sinyal';
    if(tl.length>=4) phase='Hızlı gelişim';
    else if(tl.length>=2) phase='Doğrulama aşaması';
    if(e.lifecycle==='stabilizing') phase='Stabilizasyon';
    if(e.lifecycle==='archived') phase='Arşivleniyor';
    let h='<div class="tl-intel"><div class="tl-intel-head"><span>Zaman çizelgesi özeti</span><b>'+esc(phase)+'</b></div>';
    h+='<div class="tl-intel-grid"><div><small>Gelişim</small><strong>'+tl.length+' sinyal</strong></div><div><small>Kaynak</small><strong>'+sources.size+' farklı</strong></div><div><small>Yayılım</small><strong>'+esc(eventDuration(0,total)||'anlık')+'</strong></div>';
    if(maxGap>0) h+='<div><small>En uzun sessizlik</small><strong>'+esc(eventDuration(0,maxGap))+'</strong></div>';
    h+='</div>';
    if(maxGap>=6*3600) h+='<div class="tl-intel-note">Uzun bir sessizliğin ardından yeni sinyal gelmiş. Olayın yeniden hızlanıp hızlanmadığı izlenmeli.</div>';
    else if(tl.length>=4 && total<=24*3600) h+='<div class="tl-intel-note">Kısa zaman aralığında çoklu sinyal oluşmuş. Olayın yayılım hızı yükseliyor olabilir.</div>';
    else h+='<div class="tl-intel-note">Zaman çizelgesi mevcut kaynak akışının kronolojik görünümüdür; tek başına doğruluk kanıtı değildir.</div>';
    return h+'</div>';
  }
  function framingHTML(e){
    const hs = e.headlines||[];
    if(hs.length<2) return '';
    const ev = e.evidence||{};
    let h='<div class="framing"><div class="framing-head">Kim nasıl başlık attı?<span>'+hs.length+' kaynak · yalnız o kaynağın kullandığı kelimeler işaretli</span></div>';
    hs.forEach(x=>{
      const shown = x.translated || x.title;
      let t = esc(shown);
      (x.distinct||[]).forEach(w=>{
        const ew=esc(w).replace(/[.*+?^${}()|[\]\\]/g,'\\$&');
        t = t.replace(new RegExp('(^|[^\\p{L}\\p{N}])('+ew+')(?=$|[^\\p{L}\\p{N}])','u'),'$1<mark>$2</mark>');
      });
      h+='<div class="framing-row" data-url="'+encodeURIComponent(x.link||'')+'" data-act="open">'
        +'<div class="framing-src">'+faviconHTML(x.link)+esc(x.source||hostOf(x.link))+(x.via?'<span class="via" title="Bu kaynak ajans metnini kullanıyor">'+esc(x.via)+' metni</span>':'')+'<span style="margin-left:auto;font-weight:500;color:var(--txt-faint)">'+clockTime(x.ts)+'</span></div>'
        +'<div class="framing-title">'+t+(Number(x.sens||0)>=40?' <span class="sig sens" title="Bu başlıkta abartılı ya da duygu yüklü dil var">⚠ Abartılı dil</span>':'')+'</div>'
        +(x.translated?'<div class="framing-tr">Özgün: '+esc(x.title)+'</div>':'')
        +'</div>';
    });
    if(e.calmTitle) h+='<div class="framing-note calm-note">🌿 <span>En sakin başlık</span>: <span data-noi18n>'+esc(e.calmTitle)+(e.calmSource?' — '+esc(e.calmSource):'')+'</span></div>';
    if(ev.syndicated) h+='<div class="framing-note">'+Number(ev.syndicated)+' kaynak aynı ajans metnini'+((ev.agencies||[]).length?' ('+esc(ev.agencies.join(', '))+')':'')+' kullanıyor; bağımsız kaynak sayısında tek kaynak sayıldı.</div>';
    return h+'</div>';
  }
  function eventBodyHTML(e){
    const dur = eventDuration(e.firstTs, e.lastTs);
    let h = '<div class="event-meta">'
       + (e.lifecycle?'<span class="lc">'+esc(lifecycleLabel(e.lifecycle))+'</span>':'')
       + '<span class="pill">'+e.independentSources+' bağımsız / '+e.sourceCount+' kaynak</span>'
       + (dur?'<span class="pill">'+dur+'</span>':'')
       + '<span class="pill" title="Önem puanı">önem '+e.score+'</span>'
       + '<span class="pill" title="Kaynak mutabakatı">güven '+e.confidence+'</span>'
       + '</div>';
    if(e.lastVisit && Number(e.newSince||0)>0) h += '<div class="since-note" data-noi18n>🕑 '+esc(tf('Son baktığından beri ({t}) {n} yeni başlık — zaman çizelgesinde işaretli',{t:trList(relTime(e.lastVisit)),n:Number(e.newSince)}))+'</div>';
    else if(e.lastVisit) h += '<div class="since-note quiet">🕑 <span>Son baktığından beri yeni başlık yok.</span></div>';
    h += '<div style="display:flex;gap:8px;flex-wrap:wrap;margin:10px 0">'
       + '<button id="followEventBtn" type="button" class="btn primary'+(e.followed?' on':'')+'" data-follow-event="'+esc(e.eventId||'')+'" data-follow-link="'+encodeURIComponent(e._link||'')+'">'+(e.followed?'✓ Olayı takip ediyorsun':'☆ Olayı takip et')+'</button>'
       + '<button class="btn" data-follow-topic="'+esc(e.category||'')+'">☆ Konuyu takip et</button></div>';
    if(e.relText){
      const dot = e.relLevel==='yüksek' ? '🟢' : (e.relLevel==='orta' ? '🟡' : '🔴');
      h += '<div class="rel rel-'+(e.relLevel||'')+'">'+dot+' '+esc(e.relText)
         + (e.firstTs ? ' · İlk görülme: '+clockTime(e.firstTs) : '') + '</div>';
    }
    if(e.sources && e.sources.length) h += '<div class="event-sources"><span>Kaynaklar:</span> '+e.sources.map(esc).join(" · ")+'</div>';
    if(e.evidence){
      h += '<div class="evidence"><div class="evidence-head"><span>KANIT / KAYNAK YAPISI</span><span>'+Number(e.evidence.agreement||0)+'/100 mutabakat</span></div><div class="evidence-summary">'+esc(e.evidence.summary||'')+'</div><div class="evidence-src">';
      (e.evidence.sources||[]).forEach(s=>{ h += '<span class="evidence-chip '+(s.primary?'primary':'')+'">'+(s.primary?'✓ ':'')+esc(s.name||s.domain||'Kaynak')+'</span>'; });
      h += '</div></div>';
    }
    h += coverageHTML(e.coverage);
    h += ownershipHTML(e);
    h += factChecksHTML(e);
    h += '<div class="ctx-box" id="ctxBox"><button class="btn" data-ctx-link="'+encodeURIComponent(e._link||'')+'" onclick="loadContextCard(this)">📚 Bu konuya yeni misin? Bağlamı göster</button></div>';
    if(e.reason) h += '<div class="event-summary"><b>Neden önemli?</b><br>'+esc(e.reason)+'</div>';
    if(e.summary) h += '<div class="event-summary">'+esc(e.summary)+'</div>';
    h += framingHTML(e);
    h += timelineIntelHTML(e);
    h += '<div class="tl-head">Zaman çizelgesi</div><div class="timeline">';
    (e.timeline||[]).forEach(t=>{
      h += '<div class="tl-row'+(t.new?' tl-new':'')+'" data-url="'+encodeURIComponent(t.link||"")+'" data-act="open">'
         + '<div class="tl-time">'+clockTime(t.ts)+'</div>'
         + '<div class="tl-line"><span class="tl-dot"></span></div>'
         + '<div class="tl-body"><div class="tl-title">'+(t.new?'<span class="tl-newtag">yeni</span> ':'')+esc(t.title)+'</div><div class="tl-src">'+esc(t.source)+'</div></div>'
         + '</div>';
    });
    h += '</div>';
    return h;
  }
  function rightVisible(){ const rp=document.getElementById("rightpanel"); return rp && rp.offsetParent !== null; }
  function closeEventPanel(){
    const ep=document.getElementById("eventPanel"); if(!ep) return;
    ep.innerHTML='<div class="rp-empty">Birden çok kaynaklı bir habere tıklayın — kaynaklar, başlık karşılaştırması ve zaman çizelgesi burada açılır.</div>';
    const t=document.getElementById('rpTabOlay'); if(t) t.classList.add('hide');
    if(RIGHT_TAB==='olay') setRightTab(RIGHT_PREV||'ozet');
  }
  // Kısa bilgi balonu (ekranın altında, 2,6 sn).
  function nnToast(msg){
    let t=document.getElementById('nnToast');
    if(!t){ t=document.createElement('div'); t.id='nnToast'; t.className='nn-toast'; t.setAttribute('role','status'); document.body.appendChild(t); }
    t.textContent=msg; t.classList.add('show');
    clearTimeout(nnToast._h); nnToast._h=setTimeout(()=>t.classList.remove('show'),2600);
  }
  // ---- RC93: uzaktan güncelleme ----
  let UPD=null;
  window.nnOnUpdate=function(s){
    try{ UPD=typeof s==='string'?JSON.parse(s):s; }catch(e){ return; }
    const b=document.getElementById('updBtn'); if(b) b.classList.toggle('hide',!UPD.app);
    if(UPD.changed && !window._updToastShown){ window._updToastShown=true; showUpdToast(); }
  };
  function showUpdToast(){
    let t=document.getElementById('updToast');
    if(!t){ t=document.createElement('div'); t.id='updToast'; t.className='upd-toast'; document.body.appendChild(t); }
    t.innerHTML='<span>Kanal ve çeviri listeleri güncellendi.</span><button class="mini" onclick="window.nnReloadPage()">Şimdi uygula</button><button class="mini" onclick="this.parentNode.remove()">Sonra</button>';
  }
  function openAppUpdate(){
    if(!UPD||!UPD.app) return;
    const a=UPD.app, mb=(a.size/1048576).toFixed(1);
    const m=document.createElement('div'); m.className='modal-overlay open'; m.id='updModal';
    m.onclick=e=>{ if(e.target===m) m.remove(); };
    m.innerHTML='<div class="modal live-add"><div class="modal-head"><h2>⬆ Yeni sürüm: '+esc(a.version)+'</h2><button class="btn" onclick="document.getElementById(\'updModal\').remove()">Kapat</button></div><div class="live-ask-body">'
      +'<p>Kurulu sürüm: <b>'+esc(UPD.appCurrent||'')+'</b> · indirme: '+mb+' MB</p>'
      +(a.notes?'<p class="upd-notes">'+esc(a.notes)+'</p>':'')
      +'<p class="la-note">Dosya GitHub sürüm sayfasından indirilir; imzalı güncelleme listesindeki özetle birebir tutmazsa kurulmaz. Ayarların, kanalların ve favorilerin korunur.</p>'
      +'<div id="updStatus" class="la-status"></div></div>'
      +'<div class="modal-foot"><button class="btn" onclick="try{window.nnOpen(\'https://github.com/Eirahoutmoss/nexnews/releases\')}catch(e){}">Sürüm sayfası</button><button class="btn primary" id="updGo" onclick="runAppUpdate()">Güncelle ve yeniden başlat</button></div></div>';
    document.body.appendChild(m);
  }
  async function runAppUpdate(){
    const st=document.getElementById('updStatus'), b=document.getElementById('updGo');
    b.disabled=true; st.textContent='İndiriliyor ve doğrulanıyor…';
    try{ await nnAsync('nnAppUpdate'); st.textContent='Kuruldu, yeniden başlatılıyor…'; }
    catch(e){ st.textContent=String(e&&e.message||e)+' — Sürüm sayfasından elle indirebilirsin.'; b.disabled=false; }
  }
  async function checkUpdatesNow(btn){
    if(btn){ btn.disabled=true; }
    try{ const s=JSON.parse(await nnAsync('nnCheckUpdates')); window.nnOnUpdate(s); paintUpdInfo(s);
      nnToast(s.err?('Güncelleme denetlenemedi: '+s.err):(s.app?('Yeni sürüm var: '+s.app.version):'Her şey güncel')); }
    catch(e){ nnToast('Güncelleme denetlenemedi'); }
    if(btn) btn.disabled=false;
  }
  function paintUpdInfo(s){
    const el=document.getElementById('updInfo'); if(!el||!s) return;
    const t=s.checked?new Date(s.checked*1000).toLocaleString(uiLocale()):'—';
    el.textContent='Liste sürümü: '+(s.seq||'yerleşik')+' · son denetim: '+t+(s.err?(' · '+s.err):'');
  }
  async function toggleFollowEvent(id, link){
    if(!id && !link) return;
    if(!window.nnToggleFollowEvent){ nnToast('Takip özelliği hazır değil — uygulamayı yeniden başlatın.'); return; }
    let state="off";
    try{ state=await window.nnToggleFollowEvent(JSON.stringify({id:id||'',link:link||''})); }
    catch(e){ nnToast('Takip kaydedilemedi: '+(e&&e.message||e)); return; }
    document.querySelectorAll('[data-follow-event]').forEach(b=>{
      if(b.getAttribute('data-follow-event')!==id && decodeURIComponent(b.getAttribute('data-follow-link')||'')!==link) return;
      b.textContent=(state==="on"?"✓ Olayı takip ediyorsun":"☆ Olayı takip et"); b.classList.toggle('on',state==="on");
    });
    nnToast(state==="on" ? 'Olay takibe alındı — yeni gelişmelerde bildirim alacaksın. Sol menü › Takip ettiklerim' : 'Olay takipten çıkarıldı');
    refreshFollowCount();
    const active=document.activeElement; if(active) active.blur();
  }
  async function toggleFollowTopic(topic){
    if(!topic) return;
    let st='off'; try{ st=await window.nnToggleFollowTopic(topic); }catch(e){}
    document.querySelectorAll('[data-follow-topic]').forEach(b=>{ if(b.getAttribute('data-follow-topic')===topic) b.textContent=(st==='on'?'✓ Konuyu takip ediyorsun':'☆ Konuyu takip et'); });
    nnToast(st==='on' ? '“'+topic+'” konusu takibe alındı' : '“'+topic+'” konu takibinden çıkarıldı');
  }
  async function refreshFollowCount(){
    let n=0; try{ n=Number(await window.nnFollowCount())||0; }catch(e){}
    const c=document.getElementById('navFollowCnt'); if(c){ c.textContent=n?String(n):''; c.classList.toggle('has',n>0); }
  }
  async function openFollowed(){
    let rows=[]; try{ rows=JSON.parse(await window.nnFollowed())||[]; }catch(e){}
    document.getElementById("overview").classList.add("hide");
    const p=document.getElementById("catpage"); p.classList.remove("hide");
    let h='<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button><h1>🔔 Takip ettiklerim</h1><span class="pill">'+rows.length+' olay</span></div>';
    if(!rows.length) h+='<div class="empty">Henüz takip ettiğin bir olay yok. Birden çok kaynaklı bir habere tıkla ve <b>☆ Olayı takip et</b> düğmesine bas; olay büyüdükçe burada güncellenir ve bildirim alırsın.</div>';
    else{
      h+='<div class="list">';
      rows.forEach(r=>{
        const badge = r.new>0 ? '<span class="follow-new">+'+r.new+' yeni</span>' : (r.live?'':'<span class="follow-old">akıştan düştü</span>');
        h+='<div class="follow-row">'+badge+itemHTML(r.item)+'<button class="mini follow-x" type="button" title="Takibi bırak" onclick="unfollowRow(\''+encodeURIComponent(r.item.link||'')+'\')">✕</button></div>';
      });
      h+='</div>';
    }
    p.innerHTML=h; translateDOM(); window.scrollTo(0,0);
    try{ await window.nnFollowedSeen(); }catch(e){}
  }
  async function unfollowRow(enc){ try{ await window.nnUnfollow(decodeURIComponent(enc)); }catch(e){} refreshFollowCount(); openFollowed(); }
  function navFollowed(el){ setNavActive(el); openFollowed(); }
  async function loadFollowCard(){
    const box=document.getElementById('followCardList'), up=document.getElementById('followCardUpd'); if(!box) return;
    let rows=[]; try{ rows=JSON.parse(await window.nnFollowed())||[]; }catch(e){ return; }
    const fresh=rows.filter(r=>r.new>0).length;
    if(up) up.textContent=rows.length?(rows.length+' olay'+(fresh?(' · '+fresh+' gelişme'):'')):'';
    if(!rows.length){ box.innerHTML='<div class="rp-empty">Bir olayı ☆ Olayı takip et ile izlemeye al; gelişmeler burada görünür.</div>'; return; }
    box.innerHTML=rows.slice(0,6).map(r=>'<div class="follow-mini" data-url="'+encodeURIComponent(r.item.link||'')+'" data-act="event"><span class="fm-t">'+esc(r.item.title||'')+'</span>'+(r.new>0?'<b class="follow-new-s">+'+r.new+'</b>':(r.live?'':'<i>akıştan düştü</i>'))+'</div>').join('')
      +(rows.length>6?'<div class="follow-more" onclick="openFollowed()">Tümünü gör ('+rows.length+') ›</div>':'');
  }

  async function openEvent(link){
    let e; try{ e = JSON.parse(await window.nnStory(link)); }catch(err){ return; }
    if(!e || !e.found){ nnToast('Bu olay artık güncel akışta değil.'); return; }
    e._link = link;
    if(rightVisible()){
      // Geniş ekran: olayı SAĞ panelde göster, merkezdeki akışı bozma
      const ep = document.getElementById("eventPanel");
      ep.innerHTML = '<div class="rp-close" onclick="closeEventPanel()">✕ Kapat</div>'
        + '<h2 style="margin:0 0 12px;font-size:18px">'+esc(e.title)+'</h2>'
        + eventBodyHTML(e);
      const t=document.getElementById('rpTabOlay'); if(t) t.classList.remove('hide');
      setRightTab('olay');
      const rp=document.getElementById('rightpanel'); if(rp) rp.scrollTop=0;
    }else{
      // Dar ekran: merkezde tam sayfa
      document.getElementById("overview").classList.add("hide");
      const p = document.getElementById("catpage");
      p.classList.remove("hide");
      p.innerHTML = '<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button>'
        + '<h1><span class="kdot k-'+(e.kind||'genel')+'"></span>'+esc(e.title)+'</h1></div>'
        + eventBodyHTML(e);
      window.scrollTo(0,0);
    }
  }


  // ---- RC95: olayı kimin anlattığı (sahiplik dağılımı) ----
  function ownershipHTML(e){
    const seen={}, rows=[];
    (e.timeline||[]).forEach(t=>{ if(!t||!t.source||seen[t.source]) return; seen[t.source]=1; rows.push({src:t.source,o:ownOf(t.link)}); });
    if(rows.length<2) return '';
    const cnt={}; let unknown=0;
    rows.forEach(r=>{ if(r.o) cnt[r.o[0]]=(cnt[r.o[0]]||0)+1; else unknown++; });
    if(!Object.keys(cnt).length) return '';
    const order=['kamu','devlet','ticari','bagimsiz','resmi'];
    let h='<div class="coverage ownership"><div class="evidence-head"><span>KİM ANLATIYOR?</span><span>'+rows.length+' kaynak</span></div><div class="cov-row">';
    h+=order.filter(t=>cnt[t]).map(t=>'<span class="cov-chip own-chip"><i class="own-dot own-'+t+'"></i>'+esc(ownTypeLabel(t))+' <small>'+cnt[t]+'</small></span>').join('');
    if(unknown) h+='<span class="cov-chip off">Bilinmiyor <small>'+unknown+'</small></span>';
    h+='</div><div class="own-list">'+rows.filter(r=>r.o).map(r=>'<span title="'+esc(ownTypeLabel(r.o[0])+' — '+r.o[1])+'"><i class="own-dot own-'+r.o[0]+'"></i>'+esc(r.src)+'</span>').join(' · ')+'</div>';
    h+='<div class="cov-note">Sahiplik bilgisi herkese açık kaynaklardan derlenmiştir ve bilgi amaçlıdır; zamanla değişmiş olabilir.</div></div>';
    return h;
  }

  // ---- RC97: doğrulama yazıları ve bağlam kartı ----
  const FC_VERDICT={yanlis:['✗','Yanlış'],yaniltici:['!','Yanıltıcı / eksik'],dogru:['✓','Doğru']};
  function factChecksHTML(e){
    const fs=e.factChecks||[];
    if(!fs.length) return '';
    let h='<div class="coverage factcheck"><div class="evidence-head"><span>🔎 DOĞRULAMA YAZILARI</span><span>'+fs.length+'</span></div><div class="fc-list">';
    fs.forEach(f=>{
      const v=FC_VERDICT[f.verdict];
      h+='<div class="fc-row" data-url="'+encodeURIComponent(f.link||'')+'" data-act="browser">'
        +(v?'<span class="fc-v fc-'+esc(f.verdict)+'">'+v[0]+' '+v[1]+'</span>':'<span class="fc-v">İnceleme</span>')
        +'<span class="fc-t" data-noi18n>'+esc(f.title)+'</span><span class="fc-s">'+esc(f.source)+(f.ts?' · '+relTime(f.ts):'')+'</span></div>';
    });
    h+='</div><div class="cov-note">Bu yazılar bağımsız doğrulama kuruluşlarına aittir ve olayla kelime benzerliğiyle eşleştirildi; NexNews kendisi doğru/yanlış hükmü vermez. Hükmü kuruluşun yazısından oku.</div></div>';
    return h;
  }
  async function loadContextCard(btn){
    const box=btn.closest('.ctx-box'); if(!box) return;
    const link=decodeURIComponent(btn.getAttribute('data-ctx-link')||'');
    btn.disabled=true; btn.textContent='Bağlam getiriliyor…';
    let c={};
    try{ c=JSON.parse(await nnAsync('nnContext',link))||{}; }catch(e){ btn.disabled=false; btn.textContent='📚 Bu konuya yeni misin? Bağlamı göster'; nnToast('Bağlam getirilemedi.'); return; }
    const ents=c.entities||[], rel=c.related||[];
    let h='<div class="coverage ctx-card"><div class="evidence-head"><span>📚 BAĞLAM</span><span>Vikipedi</span></div>';
    if(!ents.length && !rel.length) h+='<div class="cov-note">Bu olay için bağlam bulunamadı.</div>';
    ents.forEach(x=>{
      h+='<div class="ctx-ent">'+(x.thumb?'<img src="'+esc(x.thumb)+'" alt="" loading="lazy" referrerpolicy="no-referrer">':'')
        +'<div><b data-noi18n>'+esc(x.title)+'</b>'+(x.desc?' <small data-noi18n>'+esc(x.desc)+'</small>':'')
        +'<p data-noi18n>'+esc(x.extract)+'</p>'+(x.url?'<a class="mini" data-act="openurl" data-url="'+encodeURIComponent(x.url)+'">Vikipedi\'de oku ↗</a>':'')+'</div></div>';
    });
    if(rel.length){
      h+='<div class="tl-head">Bu konudaki önceki olaylar</div>';
      rel.forEach(r=>{ h+='<div class="tl-row" data-url="'+encodeURIComponent(r.link||'')+'" data-act="event"><div class="tl-time">'+(r.first?new Date(r.first*1000).toLocaleDateString(uiLocale(),{day:'numeric',month:'short'}):'')+'</div><div class="tl-line"><span class="tl-dot"></span></div><div class="tl-body"><div class="tl-title">'+esc(r.title)+'</div><div class="tl-src">'+esc(r.category||'')+'</div></div></div>'; });
    }
    h+='<div class="cov-note">Özetler Vikipedi\'nin herkese açık özetlerinden alınır (CC BY-SA); güncel olmayabilir.</div></div>';
    box.innerHTML=h;
  }
