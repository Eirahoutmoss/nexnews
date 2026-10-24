  // ---------- content intelligence ----------
  function renderContentIntelligence(board){
    const st=document.getElementById('contentIntelStats'), cls=document.getElementById('contentIntelClasses'), up=document.getElementById('contentIntelUpdated');
    if(!st)return; board=board||{};
    if(up)up.textContent=board.updated?('güncellendi '+agoText(board.updated)):'izleniyor…';
    st.innerHTML=[[''+Number(board.rawItems||0),'ham içerik'],[''+Number(board.eventCandidates||0),'olay adayı'],[''+Number(board.suppressed||0),'bastırılan'],['%'+Number(board.suppressionPct||0),'gürültü oranı']].map(x=>'<div class="content-intel-stat"><b>'+esc(x[0])+'</b><span>'+esc(x[1])+'</span></div>').join('');
    const items=board.topClasses||[];
    cls.innerHTML=items.length?items.map(x=>'<span class="content-intel-pill">'+esc(x.class)+': <b>'+Number(x.count||0)+'</b></span>').join(''):'<span class="content-intel-empty">Henüz sınıflandırılacak içerik yok.</span>'; const kinds=board.kinds||{}; cls.innerHTML += '<div class="content-intel-kindline">Olay '+Number(kinds.event||0)+' · Güncelleme '+Number(kinds.update||0)+' · Analiz '+Number(kinds.analysis||0)+' · Bülten '+Number(kinds.bulletin||0)+' · Canlı veri '+Number(kinds.liveData||0)+'</div>';
  }
  async function loadContentIntelligence(){try{const b=await window.nnContentIntelligence();renderContentIntelligence(JSON.parse(b));}catch(e){}}

  // ---------- source health ----------
  function renderSourceHealth(board){
    const st=document.getElementById('sourceHealthStats'), bar=document.getElementById('sourceHealthBar'), up=document.getElementById('sourceHealthUpdated'), fail=document.getElementById('sourceHealthFailures');
    if(!st)return; board=board||{};
    if(up)up.textContent=board.updated?('güncellendi '+agoText(board.updated)):'izleniyor…';
    st.innerHTML=[[''+Number(board.configured||0),'tanımlı'],[''+Number(board.healthy||0),'aktif'],[''+Number(board.stale||0),'yavaş/eski'],[''+Number(board.errors||0),'hatalı']].map(x=>'<div class="source-health-stat"><b>'+esc(x[0])+'</b><span>'+esc(x[1])+'</span></div>').join('');
    if(bar)bar.style.setProperty('--w',Math.max(0,Math.min(100,Number(board.healthyPct||0)))+'%');
    const failures=board.failures||[];
    if(fail)fail.innerHTML=(failures.length?failures.map(x=>'<div class="source-health-fail"><b>'+esc(x.name||x.url||'Kaynak')+'</b><span>×'+Number(x.consecutiveFail||1)+'</span></div>').join(''):'<div class="source-health-foot">Şu anda kayda değer bir kaynak hatası yok.</div>')+'<div class="source-health-foot">Kontrol kapsamı: '+Number(board.checked||0)+' / '+Number(board.configured||0)+' ('+Number(board.coveragePct||0)+'%) · '+Number(board.uniqueDomains||0)+' benzersiz domain</div>'+(Number(board.unchecked||0)?'<div class="source-health-foot">Kontrol edilmemiş: '+Number(board.unchecked)+'</div>':'');
  }
  async function loadSourceHealth(){try{const b=await window.nnSourceHealthSummary();renderSourceHealth(JSON.parse(b));}catch(e){}}

  // ---------- world pulse ----------
  let worldFilter='all';
  const WORLD_POINTS={
    'Türkiye':{x:405,y:144,dx:8,dy:-7},'Avrupa':{x:345,y:108,dx:7,dy:-8},'Asya':{x:500,y:126,dx:8,dy:-8},
    'Afrika':{x:380,y:215,dx:8,dy:17},'Amerika':{x:190,y:155,dx:8,dy:-8},'Okyanusya':{x:585,y:250,dx:8,dy:17},'Küresel':{x:360,y:62,dx:8,dy:-8}
  };
  function worldClass(e){ const life=String(e.lifecycle||'').toLowerCase(); const cat=String(e.category||'').toLocaleLowerCase('tr-TR'); if(worldFilter==='yüksek')return Number(e.score||0)>=75; if(worldFilter==='gelişen')return life.includes('develop')||life.includes('geliş')||life.includes('confirmed')||life.includes('doğr'); if(worldFilter==='ekonomi')return /ekonomi|finans|borsa|piyasa|enflasyon|döviz|merkez bank|faiz|economic|finance|market/.test(cat); return true; }
  function worldMapSVG(regions){
    const active=regions.filter(r=>Number(r.events||0)>0); const max=Math.max(1,...active.map(r=>Number(r.events||0)));
    const land=[
      'M70 90 L105 58 L155 54 L188 72 L202 103 L178 126 L140 120 L112 143 L82 130 Z',
      'M118 150 L152 142 L176 162 L166 198 L143 228 L119 218 L108 185 Z',
      'M267 88 L300 61 L345 63 L372 86 L358 111 L323 111 L306 132 L276 119 Z',
      'M353 124 L390 111 L427 125 L458 114 L500 125 L545 112 L592 131 L618 157 L596 179 L558 171 L532 191 L492 176 L460 186 L425 169 L391 180 L365 158 Z',
      'M440 184 L470 176 L500 194 L516 224 L492 244 L462 230 L446 208 Z',
      'M556 229 L586 219 L620 231 L636 254 L618 270 L581 264 L559 246 Z'
    ];
    let svg='<defs><radialGradient id="wmGlow"><stop offset="0" stop-color="currentColor" stop-opacity=".35"/><stop offset="1" stop-color="currentColor" stop-opacity="0"/></radialGradient></defs>';
    for(let x=40;x<=680;x+=80) svg+='<path class="world-grid" d="M'+x+' 35 V305"/>';
    for(let y=55;y<=295;y+=60) svg+='<path class="world-grid" d="M25 '+y+' H695"/>';
    svg+=land.map(d=>'<path class="world-land" d="'+d+'"/>').join('');
    active.forEach(r=>{const pt=WORLD_POINTS[r.name]||WORLD_POINTS['Küresel']; const w=Math.max(8,Math.min(26,8+Number(r.events||0)/max*18)); svg+='<g class="world-hotspot '+esc(r.level||'low')+'" data-region="'+esc(r.name)+'" transform="translate('+pt.x+' '+pt.y+')"><circle r="'+w+'" fill="url(#wmGlow)"/><circle class="world-hotspot-ring" r="'+Math.max(6,w*.55)+'"/><circle class="world-hotspot-core" r="4"/><text class="world-hotspot-label" x="'+pt.dx+'" y="'+pt.dy+'">'+esc(r.name)+' · '+r.events+'</text></g>';});
    return svg;
  }
  function renderWorldPulse(board){
    const up=document.getElementById('worldUpdated'), sum=document.getElementById('worldSummary'), regs=document.getElementById('worldRegions'), evs=document.getElementById('worldEvents'), map=document.getElementById('worldMap');
    if(!regs)return; window._worldPulse=board;
    if(up) up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok';
    const total=Number(board&&board.total||0), regions=(board&&board.regions)||[], events=(board&&board.events)||[];
    if(sum) sum.innerHTML='<b>'+total+'</b><span>önemli olay sinyali · bölgesel yoğunluk</span>';
    if(map) map.innerHTML=worldMapSVG(regions);
    const max=Math.max(1,...regions.map(r=>Number(r.score||0)));
    regs.innerHTML=regions.filter(r=>r.events>0).map(r=>{const w=Math.max(8,Math.round(Number(r.score||0)/max*100));return '<div class="world-region '+esc(r.level||'low')+'" data-region="'+esc(r.name)+'"><div class="world-rh"><span>'+esc(r.emoji||'🌐')+' '+esc(r.name)+'</span><b>'+r.events+'</b></div><div class="world-bar"><i style="--w:'+w+'%"></i></div></div>';}).join('')||'<div class="world-event-meta" style="padding:8px 0">Henüz yeterli olay sinyali yok.</div>';
    const filtered=events.filter(worldClass);
    evs.innerHTML=filtered.slice(0,5).map(e=>'<div class="world-event" data-eventid="'+esc(e.eventId||'')+'"><div class="world-event-title">'+esc(e.emoji||'🌐')+' '+esc(e.title||'')+'</div><div class="world-event-meta">'+esc(e.region||e.category||'Küresel')+' · '+e.sources+' bağımsız kaynak · güven '+e.confidence+' · '+relTime(e.ts)+' önce</div></div>').join('')||'<div class="world-event-meta" style="padding:8px 0">Bu filtrede olay yok.</div>';
    document.querySelectorAll('.world-filter').forEach(b=>b.classList.toggle('active',b.dataset.wfilter===worldFilter));
    const lab=document.getElementById('worldFilterLabel'); if(lab) lab.textContent=worldFilter==='all'?'Tümü':worldFilter[0].toLocaleUpperCase('tr-TR')+worldFilter.slice(1);
  }
  function setWorldFilter(f){worldFilter=f;renderWorldPulse(window._worldPulse||{});}
  window.nnOnWorldPulse=function(board){renderWorldPulse(board)};
  async function loadWorldPulse(){try{const b=await window.nnWorldPulse(); renderWorldPulse(JSON.parse(b));}catch(e){}}

  // ---------- event intelligence 2.0 ----------
  function intelLifecycleLabel(x){const m={detected:'Tespit',confirmed:'Doğrulandı',developing:'Gelişiyor',stabilizing:'Dengeleniyor',archived:'Arşiv'};return m[x]||x||'Bilinmiyor';}
  function intelLifecycleClass(x){return String(x||'').toLowerCase().replace(/[^a-z]/g,'');}
  function renderEventIntelligence(board){
    const up=document.getElementById('intel2Updated'), stats=document.getElementById('intel2Stats'), list=document.getElementById('intel2Events');
    if(!list)return; window._intel2=board||{};
    if(up)up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok';
    if(stats)stats.innerHTML='<div class="intel2-stat"><b>'+Number(board?.total||0)+'</b><span>önemli olay</span></div><div class="intel2-stat"><b>'+Number(board?.active||0)+'</b><span>aktif</span></div><div class="intel2-stat"><b>'+Number(board?.high||0)+'</b><span>yüksek önem</span></div><div class="intel2-stat"><b>'+Number(board?.developing||0)+'</b><span>gelişen</span></div>';
    const events=(board&&board.events)||[];
    list.innerHTML=events.slice(0,5).map(e=>{const w=Math.max(3,Math.min(100,Number(e.confidence||0)));return '<div class="intel2-row" data-intel-id="'+esc(e.eventId||'')+'"><div class="intel2-title">'+esc(e.title||'')+'</div><div class="intel2-meta"><span class="intel2-pill">'+esc(intelLifecycleLabel(e.lifecycle))+'</span><span class="intel2-pill">Önem '+Number(e.score||0)+'</span><span class="intel2-pill">Güven '+Number(e.confidence||0)+'</span><span class="intel2-pill">'+Number(e.independentSources||0)+' bağımsız</span>'+(Number(e.primarySources||0)>0?'<span class="intel2-pill">'+Number(e.primarySources)+' resmî</span>':'')+'</div><div class="intel2-meter" title="Güven '+w+'%"><i style="--w:'+w+'%"></i></div><div class="intel2-reason">'+esc(e.reason||e.evidenceSummary||'')+'</div></div>';}).join('')||'<div class="rp-empty">Henüz yeterli olay istihbaratı yok.</div>';
  }
  window.nnOnEventIntelligence=function(board){renderEventIntelligence(board)};
  async function loadEventIntelligence(){try{const b=await window.nnEventIntelligence();renderEventIntelligence(JSON.parse(b));}catch(e){}}
  // ---------- predictive intelligence ----------
  const predictLabels={escalating:'Tırmanıyor',emerging:'Yükseliyor',stable:'Stabil',declining:'Zayıflıyor'};
  function predictClass(x){return String(x||'').toLowerCase().replace(/[^a-z]/g,'');}
  function renderPredictive(board){
    const up=document.getElementById('predictUpdated'), stats=document.getElementById('predictStats'), list=document.getElementById('predictEvents');
    if(!list)return; window._predictive=board||{};
    if(up)up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok';
    if(stats)stats.innerHTML='<div class="predict-stat"><b>'+Number(board?.total||0)+'</b><span>izlenen olay</span></div><div class="predict-stat"><b>'+Number(board?.escalating||0)+'</b><span>tırmanıyor</span></div><div class="predict-stat"><b>'+Number(board?.emerging||0)+'</b><span>yükseliyor</span></div><div class="predict-stat"><b>'+Number(board?.declining||0)+'</b><span>zayıflıyor</span></div>';
    const events=(board&&board.events)||[];
    list.innerHTML=events.slice(0,6).map(e=>{const w=Math.max(3,Math.min(100,Number(e.risk||0))); return '<div class="predict-row" data-predict-id="'+esc(e.eventId||'')+'"><div class="predict-title">'+esc(e.title||'')+'</div><div class="predict-meta"><span class="predict-pill">'+esc(predictLabels[e.direction]||e.direction||'Stabil')+'</span><span class="predict-pill">Takip '+Number(e.watchScore||0)+'</span><span class="predict-pill">Risk '+Number(e.risk||0)+'</span><span class="predict-pill">'+esc(e.window||'')+'</span></div><div class="predict-meter" title="Risk '+w+'%"><i style="--w:'+w+'%"></i></div><div class="predict-reason">'+esc(e.reason||'')+'</div></div>'; }).join('')||'<div class="rp-empty">Henüz yeterli öngörü sinyali oluşmadı.</div>';
  }
  window.nnOnPredictive=function(board){renderPredictive(board)};
  async function loadPredictive(){try{const b=await window.nnPredictive();renderPredictive(JSON.parse(b));}catch(e){}}

  // ---------- entity intelligence ----------
  const entityTypeLabels={person:'Kişi',organization:'Kurum',place:'Yer',company:'Şirket'};
  function renderEntityIntelligence(board){
    const up=document.getElementById('entityUpdated'), stats=document.getElementById('entityStats'), cloud=document.getElementById('entityCloud'), rel=document.getElementById('entityRelations');
    if(!cloud)return; window._entityIntel=board||{};
    if(up)up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok';
    if(stats)stats.innerHTML='<div class="entity-stat"><b>'+Number(board?.total||0)+'</b><span>varlık</span></div><div class="entity-stat"><b>'+Number(board?.people||0)+'</b><span>kişi</span></div><div class="entity-stat"><b>'+Number(board?.organizations||0)+'</b><span>kurum</span></div><div class="entity-stat"><b>'+Number(board?.companies||0)+'</b><span>şirket</span></div>';
    const es=(board&&board.entities)||[];
    cloud.innerHTML=es.map(e=>'<button class="entity-chip '+esc(e.type||'')+'" data-entity-id="'+esc(e.id||'')+'"><span>'+esc(e.name||'')+'</span> <b>'+Number(e.events||0)+'</b></button>').join('')||'<div class="entity-empty">Henüz yeterli varlık sinyali oluşmadı.</div>';
    const rs=(board&&board.relations)||[];
    rel.innerHTML=rs.slice(0,7).map(r=>{const a=es.find(x=>x.id===r.source),b=es.find(x=>x.id===r.target);return '<div class="entity-rel"><span>'+esc(a?.name||r.source)+' ↔ '+esc(b?.name||r.target)+'</span><span>'+Number(r.weight||0)+' ortak olay</span></div>';}).join('')||'';
    if(rs.length&&rel.innerHTML)rel.innerHTML='<div class="entity-rel-title">Olaylarda birlikte geçen varlıklar</div>'+rel.innerHTML;
  }
  window.nnOnEntityIntelligence=function(board){renderEntityIntelligence(board)};
  async function loadEntityIntelligence(){try{const b=await window.nnEntityIntelligence();renderEntityIntelligence(JSON.parse(b));}catch(e){}}

  async function loadEventGraph(){try{const b=await window.nnEventGraph();renderEventGraph(JSON.parse(b));}catch(e){}}
  function graphNodeLabel(n){const x=String(n?.label||'');if(n?.type==='event')return x.length>42?x.slice(0,39)+'…':x;return x.length>18?x.slice(0,15)+'…':x;}
  function renderEventGraph(board){
    const svg=document.getElementById('eventGraph'), up=document.getElementById('egUpdated'), sum=document.getElementById('egSummary');
    if(!svg)return; window._eventGraph=board||{};
    const nodes=(board&&board.nodes)||[], edges=(board&&board.edges)||[];
    if(up)up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok';
    if(sum)sum.textContent=nodes.length+' düğüm · '+edges.length+' bağlantı · olaylar kaynak ve bağlamlarıyla ilişkilendirildi';
    if(!nodes.length){svg.innerHTML='<text x="360" y="150" text-anchor="middle" class="eg-empty">Henüz yeterli olay grafiği oluşmadı.</text>';return;}
    const ev=nodes.filter(n=>n.type==='event'), src=nodes.filter(n=>n.type==='source'), reg=nodes.filter(n=>n.type==='region'), cat=nodes.filter(n=>n.type==='category');
    const pos={}, cx=360, cy=150;
    function ring(list,r,offset){list.forEach((n,i)=>{const a=offset+(Math.PI*2*i/Math.max(1,list.length));pos[n.id]={x:cx+Math.cos(a)*r,y:cy+Math.sin(a)*r};});}
    ring(ev,82,-Math.PI/2); ring(src,132,0); ring(reg,112,Math.PI/4); ring(cat,142,Math.PI/2);
    let out='';
    edges.forEach(e=>{const a=pos[e.source],b=pos[e.target];if(!a||!b)return;out+='<line class="eg-edge '+esc(e.type||'')+'" x1="'+a.x.toFixed(1)+'" y1="'+a.y.toFixed(1)+'" x2="'+b.x.toFixed(1)+'" y2="'+b.y.toFixed(1)+'" stroke-width="'+(e.type==='related'?Math.max(1,Number(e.weight||.3)*3):1)+'"/>';});
    nodes.forEach(n=>{const p=pos[n.id]||{x:cx,y:cy};const r=n.type==='event'?Math.max(8,Math.min(15,8+Number(n.score||0)/18)):n.type==='source'?6:n.type==='region'?9:7;const title=esc(n.label||'');out+='<g class="eg-node '+esc(n.type||'')+'" data-eg-id="'+esc(n.id||'')+'" data-eg-type="'+esc(n.type||'')+'" transform="translate('+p.x.toFixed(1)+' '+p.y.toFixed(1)+')"><circle r="'+r.toFixed(1)+'"/><text x="0" y="'+(r+11).toFixed(1)+'" text-anchor="middle">'+graphNodeLabel(n).replace(/&/g,'&amp;').replace(/</g,'&lt;')+'</text><title>'+title+'</title></g>';});
    svg.innerHTML=out;
  }

  // ---------- sports ----------
  let sportFilter='all';
  function countdown(ts){ const d=(Number(ts||0)*1000-Date.now())/60000; if(!(d>0)||d>48*60) return ''; const h=Math.floor(d/60), m=Math.floor(d%60); return (h?h+' sa ':'')+m+' dk sonra'; }
  function sportStatus(g){ if(g.status==='in') return '● CANLI · '+(g.statusDetail||''); if(g.status==='post') return (g.statusDetail||'Bitti')+' · '+new Date((g.start||0)*1000).toLocaleDateString(uiLocale(),{day:'2-digit',month:'short'}); if(g.statusDetail) return g.statusDetail; const t=new Date((g.start||0)*1000); return t.toLocaleDateString(uiLocale(),{weekday:'short'})+' '+t.toLocaleDateString(uiLocale(),{day:'numeric',month:'short'})+' · '+t.toLocaleTimeString(uiLocale(),{hour:'2-digit',minute:'2-digit'}); }
  function setSportFilter(f){ sportFilter=f; document.querySelectorAll('.sport-filter[data-sfilter]').forEach(b=>b.classList.toggle('active',b.dataset.sfilter===f)); if(window._sportsBoard) renderSports(window._sportsBoard); }
  function sportFavorite(g, board){ if(g.fav) return true; const fold=x=>String(x||'').toLocaleLowerCase('tr-TR').normalize('NFD').replace(/[\u0300-\u036f]/g,'').replace(/ı/g,'i'); const a=fold((g.home||'')+' '+(g.away||'')); return ((board&&board.favorites)||[]).some(x=>x&&fold(x).length>=3&&a.includes(fold(x))); }
  function renderSports(board){ const list=document.getElementById('sportList'), up=document.getElementById('sportUpdated'), counts=document.getElementById('sportCounts'); if(!list)return; window._sportsBoard=board; if(up)up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok'; const games=[...((board&&board.games)||[])]; const live=games.filter(g=>g.status==='in').length, post=games.filter(g=>g.status==='post').length, upcoming=games.length-live-post; if(counts)counts.innerHTML='<span>Toplam <b>'+games.length+'</b></span><span>🔴 <b>'+live+'</b> canlı</span><span>Yaklaşan <b>'+upcoming+'</b></span><span>Sonuç <b>'+post+'</b></span>'; const filtered=games.filter(g=>sportFilter==='all'||(sportFilter==='live'&&g.status==='in')||(sportFilter==='upcoming'&&g.status==='prematch')||(sportFilter==='post'&&g.status==='post')); if(!filtered.length){list.innerHTML='<div class="sport-empty">'+(games.length?'Bu filtrede karşılaşma yok.':'Önümüzdeki günlerde takip ettiğin liglerde maç görünmüyor.')+'</div>';return;} list.innerHTML=filtered.slice(0,8).map(g=>{const fav=sportFavorite(g,board); const cd=g.status==='prematch'?countdown(g.start):''; const sc=(g.status==='prematch')?'–':(esc(g.homeScore||'0')+' - '+esc(g.awayScore||'0')); return '<div class="sport-row"><div class="sport-league">'+esc(g.leagueName||g.league||'Spor')+(fav?'<span class="sport-fav">⭐ FAVORİ</span>':'')+'</div><div class="sport-teams"><span class="sport-team">'+esc(g.home||'')+'</span><span class="sport-score">'+sc+'</span><span class="sport-team">'+esc(g.away||'')+'</span></div><div class="sport-status">'+esc(sportStatus(g))+(cd?'<span class="sport-countdown">· '+cd+'</span>':'')+'</div></div>';}).join(''); }
  // ---------- puan durumu ----------
  let standingsLeague='futbol'; try{ standingsLeague=localStorage.getItem('nexnews.standings')||'futbol'; }catch(e){}
  function setStandingsLeague(k){ standingsLeague=k; try{localStorage.setItem('nexnews.standings',k);}catch(e){} if(window._standings) renderStandings(window._standings); }
  const ST_Q={'Şampiyonlar Ligi':'sl','Avrupa Ligi':'al','Konferans Ligi':'kl','Play-off':'po','Küme düşme':'kd'};
  function stFav(team,fav){ const t=String(team||'').toLocaleLowerCase('tr-TR'); return (fav||[]).some(x=>x&&t.includes(String(x).toLocaleLowerCase('tr-TR'))); }
  function renderStandings(board){
    window._standings=board; const body=document.getElementById('standingsBody'), up=document.getElementById('standingsUpdated'); if(!body)return;
    document.querySelectorAll('#standingsTabs .sport-filter').forEach(b=>b.classList.toggle('active',b.dataset.league===standingsLeague));
    if(up) up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok';
    const t=((board&&board.tables)||[]).find(x=>x.key===standingsLeague);
    if(!t||!(t.rows||[]).length){ body.innerHTML='<div class="sport-empty">Puan durumu şu an alınamadı'+(t&&t.error?' ('+esc(t.error)+')':'')+'. Biraz sonra yeniden denenecek.</div>'; return; }
    const fav=board.favorites||[], d=t.hasDraws, used={};
    const head='<tr><th style="text-align:center">#</th><th class="st-team">Takım</th><th title="Oynanan">O</th><th title="Galibiyet">G</th>'+(d?'<th title="Beraberlik">B</th>':'')+'<th title="Mağlubiyet">M</th><th title="Averaj">AV</th><th title="Puan">P</th></tr>';
    const rows=t.rows.map(r=>{ const q=ST_Q[r.note]||''; if(q)used[r.note]=q; const cls=(q?'st-q-'+q:'')+(stFav(r.team,fav)?' st-fav':''); return '<tr class="'+cls+'"><td class="st-pos">'+r.pos+'</td><td class="st-team" title="'+esc(r.team)+'">'+esc(r.team)+'</td><td>'+r.played+'</td><td>'+r.won+'</td>'+(d?'<td>'+r.drawn+'</td>':'')+'<td>'+r.lost+'</td><td>'+(r.diff>0?'+':'')+r.diff+'</td><td class="st-pts">'+r.points+'</td></tr>'; }).join('');
    const colors={sl:'#2e9e5b',al:'#3d7bd9',kl:'#22a6a6',po:'var(--accent)',kd:'#d64545'};
    const legend=Object.keys(used).map(n=>'<span><i style="background:'+colors[used[n]]+'"></i>'+esc(n)+'</span>').join('');
    const asOf=t.asOf?(' · '+esc(t.asOf)+' maçlarına kadar'):'';
    body.innerHTML='<div class="st-meta"><span>'+esc(t.title)+' · '+esc(t.season||'')+'</span><span>'+t.rows.length+' takım</span></div><table class="st-table">'+head+rows+'</table>'+(legend?'<div class="st-legend">'+legend+'</div>':'')
      +'<div class="st-foot"><span>Kaynak: '+esc(t.source||'')+asOf+'</span><a onclick="window.nnOpen(\''+esc(t.sourceUrl||'')+'\')">Ayrıntılar ↗</a></div>';
  }
  window.nnOnStandings=function(board){ renderStandings(board); }; function loadStandings(){ try{ window.nnStandings(); }catch(e){} }

  window.nnOnSports=function(board){renderSports(board)}; async function loadSports(){try{window.nnSports()}catch(e){}}

  document.addEventListener("click", function(e){
    const ir=e.target.closest('.intel2-row'); if(ir){const id=ir.dataset.intelId||''; const e=(window._intel2&&window._intel2.events||[]).find(x=>x.eventId===id); if(e&&e.link)openEvent(e.link); return;}
    const pr=e.target.closest('.predict-row'); if(pr){const id=pr.dataset.predictId||''; const e=(window._predictive&&window._predictive.events||[]).find(x=>x.eventId===id); if(e&&e.link)openEvent(e.link); return;}
    const sa=e.target.closest('.smart-alert-row'); if(sa){const id=sa.dataset.smartAlertId||''; const a=(window._smartAlerts&&window._smartAlerts.alerts||[]).find(x=>x.id===id); if(a&&a.link)openEvent(a.link); return;}
    const br=e.target.closest('.briefing-item'); if(br){const id=br.dataset.briefingId||''; const it=(window._briefing&&window._briefing.items||[]).find(x=>x.eventId===id); if(it&&it.link)openEvent(it.link); else {const match=(LAST_OV&&[...(LAST_OV.important||[]),...(LAST_OV.breaking||[])].find(x=>x.eventId===id)); if(match&&match.link)openEvent(match.link);} return;}
    const gn=e.target.closest('.eg-node'); if(gn){const id=gn.dataset.egId||''; const n=(window._eventGraph&&window._eventGraph.nodes||[]).find(x=>x.id===id); if(n&&n.type==='event'){const match=(LAST_OV&&[...(LAST_OV.important||[]),...(LAST_OV.breaking||[])].find(x=>x.eventId===id)); if(match&&match.link)openEvent(match.link);} return;}
    const rr=e.target.closest('.radio-item'); if(rr){const idx=Number(rr.dataset.radioIndex||-1); const st=(window._radioBoard&&window._radioBoard.stations||[])[idx]; if(st&&st.stream){ playRadio(st); const sub=document.getElementById('radioSub'); if(sub)sub.textContent='▶ '+(st.name||'Radyo')+' · '+(st.country||''); } return;}
    const f=e.target.closest('.world-filter'); if(f){setWorldFilter(f.dataset.wfilter||'all');return;}
    const region=e.target.closest('.world-hotspot,.world-region');
    if(region){const name=region.dataset.region||''; const events=(window._worldPulse&&window._worldPulse.events||[]).filter(x=>x.region===name); if(events.length){const match=(LAST_OV&&[...(LAST_OV.important||[]),...(LAST_OV.breaking||[])].find(x=>x.eventId===events[0].eventId)); if(match&&match.link)openEvent(match.link); else {const first=document.querySelector('.world-event[data-eventid="'+CSS.escape(events[0].eventId||'')+'"]'); if(first)first.scrollIntoView({behavior:'smooth',block:'nearest'});}} return;}
    const row=e.target.closest('.world-event'); if(!row)return; const id=row.dataset.eventid||''; if(id){ const match=(LAST_OV&&[...(LAST_OV.important||[]),...(LAST_OV.breaking||[])].find(x=>x.eventId===id)); if(match&&match.link) openEvent(match.link); }
  });


