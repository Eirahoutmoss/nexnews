  // ---------- overview ----------
  function renderPulseChip(data){
    const el=document.getElementById("pulseChip"); if(!el) return;
    const p=Math.max(0,Math.min(100,Number(data.pulse||0)));
    el.innerHTML='Nabız <b>'+p+'</b> '+esc(String(data.pulseBand||'').toLowerCase());
    el.title='Haber yoğunluğu, önem, güven ve güncelliğin birleşik göstergesi (0–100)';
    el.classList.toggle('hide', !data.updatedTs);
  }
  function pulseHTML(data){
    const p=Math.max(0,Math.min(100,Number(data.pulse||0)));
    const band=esc(data.pulseBand||"sakin");
    return '<section class="pulse-card"><div class="pulse-ring" style="--p:'+p+'">'+p+'</div><div><div class="pulse-title">NEX PULSE <span class="pulse-band">'+band+'</span></div><div class="pulse-note">Haber yoğunluğu, önem, güven ve güncellik sinyallerinin birleşik görünümü.</div></div></section>';
  }
  function breakingBandHTML(list){
    if(!list || !list.length) return "";
    let h = '<div class="breaking-band"><div class="bb-label"><span class="live"></span>SON DAKİKA</div><div class="bb-scroll">';
    for(const it of list){
      h += '<div class="bb-item" data-url="'+encodeURIComponent(it.link||"")+'" data-act="open">'
         + '<span class="t">'+esc(it.title)+'</span>'
         + '<span class="s">'+esc(it.source)+(it.category?' · '+esc(it.category):'')+'</span></div>';
    }
    h += '</div></div>';
    return h;
  }

  async function loadOverview(){
    let data;
    try{ data = JSON.parse(await window.nnOverview()); }catch(e){ return; }
    LAST_OV = data;
    indexSummaries(data.breaking); indexSummaries(data.important);
    (data.categories||[]).forEach(c=>indexSummaries(c.items));
    if(document.body.classList.contains("widget")){ renderWidget(); }
    const el = document.getElementById("overview");
    const up = document.getElementById("updated");
    up.textContent = data.updatedTs ? ("Güncellendi: "+agoText(data.updatedTs))
                   : (data.loading ? "Başlıklar yükleniyor…" : "");

    if(!data.categories || !data.categories.length){
      el.innerHTML = '<div class="empty">Kaynak yok. Sağ üstten “Kaynaklar” ile ekleyin.</div>';
      return;
    }
    const anyItems = data.categories.some(c=>c.items && c.items.length);
    if(!anyItems){
      el.innerHTML = '<div class="loading-note">'+(data.updatedTs?'Başlık bulunamadı.':'Başlıklar yükleniyor…')+'</div>';
      if(!data.updatedTs) return;
    }

    renderPulseChip(data);
    updateNavCounts(data);
    if(homeView()==='sakin'){
      el.innerHTML = calmHTML(data);
      if(KBD_IDX>=0) KBD_IDX=-1;
      translateDOM();
      return;
    }
    let html = viewToggleHTML() + breakingBandHTML(data.breaking) + forYouHTML(data.forYou) + importantHTML(data.important) + discoverHTML(data) + '<div class="grid">';
    const personalized = !!String(data.countryCode||'').trim();
    for(const c of data.categories){
      if(personalized && Number(c.total||0)===0) continue;
      html += '<section class="cat"><div class="cat-head"><h2>'+esc(c.name)+'</h2>'
            + '<span class="pill" title="'+Number(c.unread||0)+' okunmamış / '+c.total+' başlık">'+(Number(c.unread||0)?Number(c.unread)+' yeni':c.total)+'</span></div><div class="cat-body">';
      if(!c.items || !c.items.length){
        html += '<div class="empty" style="padding:22px 10px">—</div>';
      }else{
        for(const it of c.items) html += itemHTML(it);
      }
      html += '</div>';
      if(c.total > (c.items?c.items.length:0)){
        html += '<div class="cat-foot"><a class="showall" data-cat="'+encodeURIComponent(c.name)+'">Tümünü Göster ('+c.total+') →</a></div>';
      }
      html += '</section>';
    }
    html += '</div>' + closingHTML(data);
    el.innerHTML = html;
    if(KBD_IDX>=0) KBD_IDX=-1;
    translateDOM();
  }

  function sourcesText(it){
    const s = it.sources||[];
    if(s.length<=1) return esc(it.source||(s[0]||""));
    return esc(s[0]) + ' <span class="src-more">+'+(s.length-1)+'</span>';
  }
  function hostOf(u){ try{ return new URL(u).hostname.replace(/^www\./,''); }catch(e){ return ""; } }
  function faviconHTML(link){
    const h=hostOf(link); if(!h) return "";
    return '<img class="fav-ic" src="https://'+esc(h)+'/favicon.ico" alt="" loading="lazy" referrerpolicy="no-referrer" onerror="this.remove()">';
  }
  function scoreTier(n){ return n>=75?"hi":(n>=55?"mid":"lo"); }
  function lifecycleLabel(x){ return ({detected:'Algılandı',confirmed:'Teyitli',developing:'Gelişiyor',stabilizing:'Durulan',archived:'Arşiv'})[x] || x; }
  // Listede tek bir sinyal: son dakika > bağımsız kaynak sayısı. Puan, güven ve
  // olay evresi ipucunda (title) durur; ayrıntı olay panelindedir.
  function signalHTML(it){
    const tip = ['Önem '+(it.score||0)+'/100','güven '+(it.confidence||0)+'/100',
      it.lifecycle?('evre: '+lifecycleLabel(it.lifecycle).toLowerCase()):'', it.reason||''].filter(Boolean).join(' · ');
    if(it.breaking) return '<span class="sig brk" title="'+esc(tip)+'">Son dakika</span>';
    const n = Number(it.independentSources||0);
    if(n>1){
      const w = Number(it.syndicated||0);
      return '<span class="sig'+(w?' wire':'')+'" title="'+esc(tip+(w?' · '+w+' kaynak aynı ajans metnini kullanıyor':''))+'">'+n+' bağımsız kaynak</span>';
    }
    return '';
  }
  function itemHTML(it, opts){
    opts = opts||{};
    const u = encodeURIComponent(it.link||"");
    const kind = it.kind||"genel";
    const titleAct = (it.count>1) ? 'event' : 'open';
    const read = it.read || READ.has(it.link);
    const thumb = (opts.thumbs && it.image && !document.body.classList.contains('data-lite')) ? '<img class="thumb" src="'+esc(it.image)+'" alt="" loading="lazy" referrerpolicy="no-referrer" onerror="this.parentNode.classList.remove(\'has-thumb\');this.remove()">' : '';
    return '<div class="item'+(it.breaking?' brk':'')+(read?' read':'')+(thumb?' has-thumb':'')+'" data-url="'+u+'"'+(it.category?' data-cat-name="'+esc(it.category)+'"':'')+'>'
         + thumb
         + '<span class="title" data-act="'+titleAct+'"><span class="kdot k-'+kind+'"></span><span class="t-tx">'+esc(it.title)
         + (it.count>1?'<span class="evt-tag" title="Aynı olayı bildiren '+Number(it.sourceCount||it.count)+' kaynak — ayrıntı için tıklayın">'+Number(it.sourceCount||it.count)+' kaynak</span>':'')+'</span></span>'
         + '<div class="meta"><span class="src" title="'+esc(ownTitle(it.link))+'">'+faviconHTML(it.link)+ownDot(it.link)+'<span class="src-tx">'+sourcesText(it)+'</span></span>'
         + '<span class="time" title="'+esc(it.ts?new Date(it.ts*1000).toLocaleString(uiLocale()):'')+'">'+relTime(it.ts)+'</span>'
         + signalHTML(it)
         + '</div><div class="actions">'
         + starHTML(it.link)
         + '<button class="mini" data-act="read" title="Uygulamada oku (R)">Oku</button>'
         + '<button class="mini" data-act="browser" title="Tarayıcıda aç (B)">Tarayıcı</button>'
         + '</div></div>';
  }
  const SEC_IC = {
    star:'<svg class="sec-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m12 3 2.7 5.6 6.1.9-4.4 4.3 1 6.1L12 17l-5.4 2.9 1-6.1-4.4-4.3 6.1-.9z"/></svg>',
    user:'<svg class="sec-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="8" r="4"/><path d="M4 21c0-4 4-6 8-6s8 2 8 6"/></svg>',
    compass:'<svg class="sec-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="9"/><path d="m15.5 8.5-2 5-5 2 2-5z"/></svg>',
    check:'<svg class="cc-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="9"/><path d="m8 12 3 3 5-6"/></svg>'
  };
  function sectionHTML(cls, head, list, sub){
    if(!list || !list.length) return "";
    let h = '<section class="important '+cls+'"><div class="imp-head">'+head+(sub?'<span class="imp-sub">'+sub+'</span>':'')+'</div><div class="imp-list">';
    for(const it of list) h += itemHTML(it,{thumbs:true});
    h += '</div></section>';
    return h;
  }
  function importantHTML(list){ return sectionHTML("", SEC_IC.star+' Şu anda önemli', list); }
  function forYouHTML(list){ return sectionHTML("foryou", SEC_IC.user+' Senin için', list, (CONFIG&&CONFIG.interests&&CONFIG.interests.length)?esc(CONFIG.interests.slice(0,4).join(', ')):''); }
  function discoverHTML(data){
    if(!data.discover) return "";
    return '<section class="important discover-win"><div class="imp-head">'+SEC_IC.compass+' Farklı bir pencere</div>'
      + (data.discoverReason?'<div class="discover-reason">'+esc(data.discoverReason)+'</div>':'')
      + '<div class="imp-list">'+itemHTML(data.discover,{thumbs:true})+'</div></section>';
  }
  // ---- RC93: kör nokta göstergesi ----
  function coverageHTML(cv){
    if(!cv) return '';
    const cov=cv.covered||[], sil=cv.silent||[];
    let h='<div class="coverage"><div class="evidence-head"><span>KİM HABER YAPTI, KİM SUSTU?</span><span>'+cov.length+'/'+(cov.length+sil.length)+' bölge</span></div><div class="cov-row">';
    h+=cov.map(c=>'<a class="cov-chip on" data-url="'+encodeURIComponent(c.link||'')+'" data-act="open" title="'+esc(c.title||'')+'">✓ '+esc(c.region)+' <small>'+Number(c.sources||0)+'</small></a>').join('');
    h+=sil.map(r=>'<span class="cov-chip off" title="Bu bölgenin kaynakları son 48 saatte başka haberler yayımladı ama bu olaya değinmedi">○ '+esc(r)+'</span>').join('');
    h+='</div>';
    if(sil.length) h+='<div class="cov-note">Sessiz kalan bölgeler bir kör nokta olabilir: olayın oradaki yankısını merak ediyorsan o bölgenin kaynaklarına göz at.</div>';
    return h+'</div>';
  }
  // ---- RC93: sakin görünüm ----
  // Sonsuz akış yerine günün birkaç önemli olayı ve açık bir "bugünlük bu
  // kadar". Haber yorgunluğu (news fatigue) ve kaygı döngüsü, NexNews'in
  // "yormadan bilgilendirme" amacıyla çelişir; tam akış tek tıkla açılır.
  const CALM_MAX=7;
  function homeView(){ return (CONFIG&&CONFIG.homeView)==='tam'?'tam':'sakin'; }
  function setHomeView(v){
    CONFIG=CONFIG||{}; CONFIG.homeView=v;
    try{ window.nnSetHomeView(v); }catch(e){}
    loadOverview(); window.scrollTo(0,0);
  }
  function viewToggleHTML(){
    const v=homeView();
    return '<div class="view-toggle" role="tablist"><button class="'+(v==='sakin'?'active':'')+'" onclick="setHomeView(\'sakin\')" title="Günün önemli olayları, sonu belli kısa bir özet">🌿 Sakin</button>'
      + '<button class="'+(v==='tam'?'active':'')+'" onclick="setHomeView(\'tam\')" title="Tüm kategoriler ve başlıklar">Tam akış</button></div>';
  }
  function calmPicks(data){
    const seen=new Set(), out=[];
    const push=it=>{ if(!it||!it.link||seen.has(it.link)) return; seen.add(it.link); out.push(it); };
    (data.important||[]).forEach(push);
    if(out.length<CALM_MAX){
      const rest=[]; (data.categories||[]).forEach(c=>(c.items||[]).forEach(it=>rest.push(it)));
      rest.sort((a,b)=>(Number(b.score||0)-Number(a.score||0))||((b.ts||0)-(a.ts||0)));
      rest.forEach(push);
    }
    return out.slice(0,CALM_MAX);
  }
  function calmHTML(data){
    const picks=calmPicks(data);
    const brk=(data.breaking||[]).slice(0,1);
    let h=viewToggleHTML()+breakingBandHTML(brk);
    // Pazar ve pazartesi: haftanın özeti hazır kartı
    const wd=new Date().getDay();
    if(wd===0||wd===1) h+='<a class="weekly-cta" onclick="openWeekly()">📰 <b>Haftanın özeti hazır</b><span>Son 7 günün en önemli olayları ve takip ettiklerin, tek sayfada →</span></a>';
    h+='<section class="important calm-list"><div class="imp-head">🌿 Bugün bilmen gerekenler</div>'
      +'<div class="calm-sub">Önem, bağımsız kaynak sayısı ve güncelliğe göre seçilen '+picks.length+' olay</div>'
      +'<div class="imp-list">'+(picks.length?picks.map(it=>itemHTML(it)).join(''):'<div class="empty">Başlıklar yükleniyor…</div>')+'</div></section>';
    if(data.discover) h+=discoverHTML(data);
    const cats=(data.categories||[]).filter(c=>Number(c.total||0)>0);
    h+='<section class="closing-card calm-close">'+SEC_IC.check+'<h3>Bugünlük bu kadar</h3>'
      +'<p>Dünyada önemli bir şey olursa burada ve bildirimlerde görürsün. Merak ettiğin bir konu varsa:</p>'
      +'<div class="calm-cats">'+cats.map(c=>'<a class="calm-cat" data-cat="'+encodeURIComponent(c.name)+'">'+esc(c.name)+(Number(c.unread||0)?' <span>'+Number(c.unread)+'</span>':'')+'</a>').join('')+'</div>'
      +'<button class="btn" onclick="setHomeView(\'tam\')">Tüm akışı göster</button></section>';
    return h;
  }
  function closingHTML(data){
    const unread = Number(data.totalUnread||0);
    const next = (CONFIG && CONFIG.refreshMinutes) ? (' Bir sonraki otomatik yenileme en geç '+CONFIG.refreshMinutes+' dk içinde.') : '';
    return '<section class="closing-card">'+SEC_IC.check+'<h3>Önemli gelişmelerin hepsini gördün</h3>'
      + '<p>Kategorilerde '+unread+' okunmamış başlık var; istersen göz atabilir ya da burada bırakabilirsin.'+next+'</p></section>';
  }

  // ---------- category detail ----------
  async function openCategory(name){
    let data;
    try{ data = JSON.parse(await window.nnCategory(name)); }catch(e){ return; }
    document.getElementById("overview").classList.add("hide");
    const p = document.getElementById("catpage");
    p.classList.remove("hide");
    let html = '<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button>'
             + '<h1>'+esc(data.name)+'</h1><span class="pill">'+data.total+' başlık</span></div>';
    html += '<div class="list">';
    for(const it of (data.items||[])) html += itemHTML(it);
    html += '</div>';
    p.innerHTML = html;
    indexSummaries(data.items);
    translateDOM();
    window.scrollTo(0,0);
  }
  function backToOverview(){
    if(WXTIMER){ clearInterval(WXTIMER); WXTIMER=null; }
    stopRadar();
    if(RADAR && RADAR.map){ try{ RADAR.map.remove(); }catch(e){} RADAR=null; }
    document.getElementById("catpage").classList.add("hide");
    document.getElementById("overview").classList.remove("hide");
  }

  // ---------- favoriler + okuma geçmişi ----------
  function savedRowHTML(it){
    ITEMS[it.link] = {title:it.title, source:it.source, cat:it.cat||"", ts:it.ts||0};
    const u = encodeURIComponent(it.link);
    return '<div class="item" data-url="'+u+'"><span class="title" data-act="open"><span class="t-tx">'+esc(it.title)+'</span></span>'
      + '<div class="meta"><span class="src">'+esc(it.source||"")+'</span><span class="time">'+relTime(it.ts||it.added)+'</span>'
      + '</div>' + (it.reason?'<div class="reason">'+esc(it.reason)+'</div>':'') + '<div class="actions">'+starHTML(it.link)+'<button class="mini" data-act="browser">Tarayıcı</button></span></div></div>';
  }
  async function openList(which){
    let items; try{ items = JSON.parse(which==='favs' ? await window.nnFavs() : await window.nnHistory()); }catch(e){ items=[]; }
    document.getElementById("overview").classList.add("hide");
    const p = document.getElementById("catpage"); p.classList.remove("hide");
    const title = which==='favs' ? '★ Favoriler' : '🕘 Okuma Geçmişi';
    let h = '<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button><h1>'+title+'</h1>'
          + '<span class="pill">'+items.length+'</span>'
          + (which==='history' && items.length ? '<button class="btn" style="margin-left:auto" onclick="clearHist()">Geçmişi temizle</button>' : '')
          + '</div>';
    if(!items.length){ h += '<div class="empty">Boş.</div>'; }
    else { h += '<div class="list">'; for(const it of items) h += savedRowHTML(it); h += '</div>'; }
    p.innerHTML = h; translateDOM(); window.scrollTo(0,0);
  }
  async function clearHist(){ try{ await window.nnClearHistory(); }catch(e){} openList('history'); }


  // ---- RC95: haftanın özeti ----
  async function openWeekly(){
    let w={}; try{ w=JSON.parse(await window.nnWeekly()); }catch(e){}
    document.getElementById('overview').classList.add('hide');
    const p=document.getElementById('catpage'); p.classList.remove('hide');
    const fmt=ts=>new Date(ts*1000).toLocaleDateString(uiLocale(),{day:'numeric',month:'long'});
    const top=w.top||[], regs=w.regions||[], fol=w.follows||[];
    let h='<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button><h1>📰 Haftanın özeti</h1><span class="pill">'+(w.from?fmt(w.from)+' – '+fmt(w.to):'')+'</span></div>';
    if(Number(w.days||0)<7) h+='<div class="weekly-note">Bu özet NexNews\'in tuttuğu '+Number(w.days||1)+' günlük kayda dayanıyor; tam hafta için birkaç gün daha kullanman gerekiyor.</div>';
    h+='<div class="weekly-stats"><div><b>'+Number(w.total||0)+'</b><span>önemli olay</span></div><div><b>'+Number(w.breaking||0)+'</b><span>son dakika</span></div><div><b>'+regs.length+'</b><span>bölge</span></div><div><b>'+fol.length+'</b><span>takip ettiğin olay</span></div></div>';
    h+='<section class="important calm-list"><div class="imp-head">⭐ Haftanın en önemli olayları</div><div class="imp-list">'
      +(top.length?top.map((e,i)=>'<div class="item weekly-item" data-url="'+encodeURIComponent(e.link||'')+'"><div class="title" data-act="'+(e.id?'event':'open')+'"><span class="weekly-rank">'+(i+1)+'</span><span class="t-tx">'+esc(e.title)+'</span></div><div class="meta"><span>'+esc(e.category||'')+'</span><span class="time">'+fmt(e.first)+'</span>'+(e.sources>1?'<span class="sig">'+Number(e.sources)+' bağımsız kaynak</span>':'')+'</div></div>').join(''):'<div class="empty">Henüz kayıtlı önemli olay yok.</div>')
      +'</div></section>';
    if(regs.length){
      const mx=Math.max(...regs.map(r=>r.count));
      h+='<section class="important"><div class="imp-head">🌍 Gündemin bölgelere dağılımı</div><div class="weekly-regs">'+regs.map(r=>'<div class="wr-row"><span>'+esc(r.name)+'</span><i style="width:'+Math.max(6,Math.round(r.count/mx*100))+'%"></i><b>'+r.count+'</b></div>').join('')+'</div>'
        +'<div class="cov-note" style="padding:0 16px 12px">Az görünen bölgeler bir kör nokta olabilir: bu hafta oradan pek haber okumadın.</div></section>';
    }
    if(fol.length){
      h+='<section class="important"><div class="imp-head">🔔 Takip ettiğin olaylar</div><div class="imp-list">'+fol.map(f=>'<div class="item" data-url="'+encodeURIComponent(f.link||'')+'"><div class="title" data-act="event"><span class="t-tx">'+esc(f.title)+'</span></div><div class="meta"><span>'+Number(f.count||0)+' haber · '+Number(f.sources||0)+' bağımsız kaynak</span>'+(f.new?'<span class="sig">+'+Number(f.new)+' yeni</span>':'')+'</div></div>').join('')+'</div></section>';
    }
    h+='<section class="closing-card calm-close">'+SEC_IC.check+'<h3>Haftanın özeti bu kadar</h3><p>Gelecek haftanın özeti pazar günü hazır olur.</p></section>';
    p.innerHTML=h; window.scrollTo(0,0);
  }
