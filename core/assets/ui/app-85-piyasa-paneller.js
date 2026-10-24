  // ---------- markets ----------
  let marketTab='borsa';
  function sparkline(q){ const a=Array.isArray(q&&q.history)?q.history.map(Number).filter(Number.isFinite):[]; if(a.length<2)return ''; let mn=Math.min(...a),mx=Math.max(...a),range=mx-mn||1; const pts=a.map((v,i)=>((i/(a.length-1))*74).toFixed(1)+','+(20-((v-mn)/range)*18).toFixed(1)).join(' '); const up=a[a.length-1]>=a[0]; return '<svg class="market-spark" viewBox="0 0 74 22" preserveAspectRatio="none" aria-hidden="true"><polyline fill="none" stroke="'+(up?'#36c98f':'#ff6b7a')+'" stroke-width="1.8" points="'+pts+'"/></svg>'; }
  function marketMoney(q){
    if(!q || q.error) return '<span class="market-flat">veri yok</span>';
    const p=Number(q.price||0), ch=Number(q.change||0), pc=Number(q.changePct||0), d5=Number(q.change5dPct||0);
    const cls=ch>0?'market-pos':(ch<0?'market-neg':'market-flat'), sign=ch>0?'+':'';
    const cur=q.currency||'';
    const price=p>=1000?p.toLocaleString(uiLocale(),{maximumFractionDigits:2}):p.toLocaleString(uiLocale(),{minimumFractionDigits:2,maximumFractionDigits:4});
    const extra=q.dayHigh?(' <span class="market-flat">G '+Number(q.dayHigh).toLocaleString(uiLocale(),{maximumFractionDigits:2})+' / D '+Number(q.dayLow||0).toLocaleString(uiLocale(),{maximumFractionDigits:2})+'</span>'):'';
    return '<div class="market-price">'+sparkline(q)+'<b>'+price+' '+esc(cur)+'</b><span class="'+cls+'">'+sign+ch.toLocaleString(uiLocale(),{maximumFractionDigits:2})+' ('+sign+pc.toFixed(2)+'%) · 5g '+(d5>=0?'+':'')+d5.toFixed(2)+'%</span>'+extra+'</div>';
  }
  function financeLabel(q){ const s=(q.symbol||'').toUpperCase(); const names={'^XU100':'BIST 100','^XU030':'BIST 30','THYAO.IS':'Türk Hava Yolları','ASELS.IS':'Aselsan','GARAN.IS':'Garanti BBVA','BIMAS.IS':'BİM','USDTRY=X':'Dolar / TL','EURTRY=X':'Euro / TL','GC=F':'Altın (Ons)'}; return names[s]||q.name||s; }
  function financeGroup(q){ const s=(q.symbol||'').toUpperCase(); if(s.includes('=X')) return 'fx'; if(s==='GC=F'||s.includes('=F')) return 'emtia'; return 'borsa'; }
  function toggleMarketPanel(){ const p=document.getElementById('marketPanel'); if(!p)return; p.classList.toggle('collapsed'); p.classList.toggle('market-open',!p.classList.contains('collapsed')); }
  function setMarketTab(tab){ marketTab=tab; document.querySelectorAll('.market-tab').forEach(b=>b.classList.toggle('active',b.dataset.mtab===tab)); if(window._financeBoard) renderFinance(window._financeBoard); }
  function detailSpark(q){ const a=Array.isArray(q&&q.history)?q.history.map(Number).filter(Number.isFinite):[]; if(a.length<2)return '<div class="market-detail-note">5 günlük grafik için yeterli veri yok.</div>'; let mn=Math.min(...a),mx=Math.max(...a),range=mx-mn||1; const pts=a.map((v,i)=>((i/(a.length-1))*100).toFixed(2)+','+(96-((v-mn)/range)*84).toFixed(2)).join(' '); const up=a[a.length-1]>=a[0]; return '<svg viewBox="0 0 100 100" preserveAspectRatio="none" aria-label="5 günlük trend"><polyline fill="none" stroke="'+(up?'#36c98f':'#ff6b7a')+'" stroke-width="2" points="'+pts+'"/></svg>'; }
  function closeMarketDetail(){ const el=document.getElementById('marketDetail'); if(el) el.remove(); }
  function closeGoldDetail(){ const el=document.getElementById('goldDetail'); if(el) el.remove(); }
  function openMarketDetail(symbol){
    const b=window._financeBoard||{}, q=(b.quotes||[]).find(x=>String(x.symbol||'').toUpperCase()===String(symbol||'').toUpperCase());
    if(!q){return;}
    let el=document.getElementById('marketDetail');
    if(!el){el=document.createElement('div');el.id='marketDetail';el.className='market-detail';const panel=document.getElementById('marketPanel');panel.parentNode.insertBefore(el,panel.nextSibling);}
    const p=Number(q.price||0), d=Number(q.change||0), pc=Number(q.changePct||0), d5=Number(q.change5dPct||0); const cls=pc>0?'market-pos':pc<0?'market-neg':'market-flat';
    const price=p.toLocaleString(uiLocale(),{minimumFractionDigits:2,maximumFractionDigits:4});
    el.innerHTML='<div class="market-detail-head"><strong class="market-detail-title">📊 '+esc(financeLabel(q))+'</strong><button class="market-detail-close" onclick="closeMarketDetail()">Kapat</button></div>'
      +'<div class="market-detail-grid"><div class="market-detail-stat"><span>Fiyat</span><b>'+price+' '+esc(q.currency||'')+'</b></div><div class="market-detail-stat"><span>Günlük</span><b class="'+cls+'">'+(d>=0?'+':'')+d.toLocaleString(uiLocale(),{maximumFractionDigits:2})+' ('+(pc>=0?'+':'')+pc.toFixed(2)+'%)</b></div><div class="market-detail-stat"><span>5 gün</span><b class="'+(d5>0?'market-pos':d5<0?'market-neg':'market-flat')+'">'+(d5>=0?'+':'')+d5.toFixed(2)+'%</b></div><div class="market-detail-stat"><span>Gün aralığı</span><b>'+(q.dayLow?q.dayLow.toLocaleString(uiLocale(),{maximumFractionDigits:2}):'—')+' / '+(q.dayHigh?q.dayHigh.toLocaleString(uiLocale(),{maximumFractionDigits:2}):'—')+'</b></div></div>'
      +'<div class="market-detail-chart">'+detailSpark(q)+'</div>'
      +'<div class="market-detail-note">5 günlük trend, NexNews tarafında alınan günlük kapanış verilerinden oluşturulur. Veri kaynağı piyasa kotasyonudur; işlem önerisi değildir.</div>';
    el.scrollIntoView({behavior:'smooth',block:'nearest'});
  }
  function openGoldDetail(){
    const b=window._financeBoard||{}, gold=Array.isArray(b.gold)?b.gold:[];
    if(!gold.length){ alert('Altın referans verisi için Ons Altın ve USD/TRY verisi henüz alınamadı.'); return; }
    let el=document.getElementById('goldDetail');
    if(!el){ el=document.createElement('div'); el.id='goldDetail'; el.className='market-detail'; const panel=document.getElementById('marketPanel'); panel.parentNode.insertBefore(el,panel.nextSibling); }
    el.innerHTML='<div class="market-detail-head"><strong class="market-detail-title">🪙 Altın Detayları</strong><button class="market-detail-close" onclick="closeGoldDetail()">Kapat</button></div>'
      +'<div class="gold-detail">'+gold.map(g=>'<div class="gold-detail-row"><b>'+esc(g.name)+'</b><span>'+Number(g.valueTry||0).toLocaleString(uiLocale(),{minimumFractionDigits:2,maximumFractionDigits:2})+' TL · '+(Number(g.purity||0)*24).toFixed(1)+' ayar · 5g '+(Number(g.change5dPct||0)>=0?'+':'')+Number(g.change5dPct||0).toFixed(2)+'%</span></div>').join('')+'</div>'
      +'<div class="market-detail-note">'+esc(gold[0].note||'Teorik referans değeridir.')+' Kaynak hesabı: '+esc(gold[0].basis||'Ons Altın + USD/TRY')+'. Gerçek kuyumcu fiyatı değildir.</div>';
    el.scrollIntoView({behavior:'smooth',block:'nearest'});
  }
  function renderFinance(board){
    window._financeBoard=board; const list=document.getElementById('marketList'), up=document.getElementById('marketUpdated'), sum=document.getElementById('marketSummary'), movers=document.getElementById('marketMovers'); if(!list) return;
    if(up) up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok';
    const qs=(board&&board.quotes)||[];
    if(sum){ const bist=qs.filter(q=>financeGroup(q)==='borsa'&&!q.error), fx=qs.filter(q=>financeGroup(q)==='fx'&&!q.error), em=qs.filter(q=>financeGroup(q)==='emtia'&&!q.error); const xu100=bist.find(q=>(q.symbol||'').toUpperCase()==='^XU100'); const xu030=bist.find(q=>(q.symbol||'').toUpperCase()==='^XU030'); const avg=a=>a.length?(a.reduce((x,q)=>x+Number(q.changePct||0),0)/a.length):0; sum.innerHTML=[['BIST 100',xu100?.changePct??avg(bist)],['BIST 30',xu030?.changePct??avg(bist)],['Döviz',avg(fx)],['Emtia',avg(em)]].map(x=>'<div class="market-stat"><span>'+x[0]+'</span><b class="'+(x[1]>0?'market-pos':x[1]<0?'market-neg':'market-flat')+'">'+(x[1]>=0?'+':'')+Number(x[1]).toFixed(2)+'%</b></div>').join(''); }
    if(movers){ const g=(board&&board.gainers)||[], l=(board&&board.losers)||[]; const parse=x=>{const a=String(x).split('|');return {n:a[0]||'',p:Number(a[1]||0)}}; const top=g.length?parse(g[0]):null, low=l.length?parse(l[0]):null; const mv=(lbl,x)=>'<div class="market-mover"><span>'+lbl+'</span><b class="'+(x.p>0?'market-pos':x.p<0?'market-neg':'market-flat')+'">'+esc(x.n)+' '+(x.p>0?'+':'')+x.p.toFixed(2)+'%</b></div>'; movers.innerHTML=(top?mv(top.p>=0?'En çok yükselen':'En az düşen',top):'')+(low?mv(low.p<0?'En çok düşen':'En az yükselen',low):''); }
    const filtered=qs.filter(q=>financeGroup(q)===marketTab);
    if(!filtered.length){ list.innerHTML='<div class="market-empty">Bu grupta veri yok.</div>'; renderMarketNews(board); return; }
    list.innerHTML=filtered.slice(0,7).map(q=>'<div class="market-row" data-market-symbol="'+esc(q.symbol||'')+'" style="cursor:pointer"><div class="market-name"><b>'+esc(financeLabel(q))+(q.symbol==='^XU100'||q.symbol==='^XU030'?'<span class="market-badge">endeks</span>':'')+'</b><span>'+esc(q.symbol||'')+' · ayrıntılar</span></div>'+marketMoney(q)+'</div>').join('');
    list.querySelectorAll('[data-market-symbol]').forEach(el=>el.addEventListener('click',()=>openMarketDetail(el.dataset.marketSymbol)));
    renderMarketNews(board);
  }
  function renderMarketNews(board){
    const box=document.getElementById('marketNews'); if(!box)return; const n=(board&&board.news)||[];
    if(!n.length){ box.innerHTML='<div class="market-empty">Son 36 saatte piyasayla ilgili başlık bulunamadı.</div>'; return; }
    box.innerHTML=n.map(x=>'<div class="mnews-item" data-link="'+esc(x.link||'')+'"><b><span class="mnews-tag">'+esc(x.tag||'')+'</span>'+esc(x.title||'')+'</b><span>'+esc(x.source||'')+(x.published?' · '+agoText(x.published):'')+'</span></div>').join('');
    box.querySelectorAll('.mnews-item').forEach(el=>el.addEventListener('click',()=>openMarketNews(el.dataset.link)));
  }
  async function openMarketNews(link){ let e=null; try{ e=JSON.parse(await window.nnStory(link)); }catch(err){} if(e&&e.found) openEvent(link); else openItem(link); }
  window.nnOnFinance=function(board){ renderFinance(board); }; async function loadFinance(){ try{ window.nnFinance(); }catch(e){} }

  // ---------- smart alerts ----------
  const smartAlertLabels={critical:'Kritik',warning:'Uyarı',info:'Bilgi'};
  function toggleSmartAlertPanel(){
    const p=document.getElementById('smartAlertPanel'); if(!p)return;
    p.classList.toggle('collapsed'); p.classList.toggle('smart-alert-open',!p.classList.contains('collapsed'));
  }
  function renderSmartAlerts(board){
    const up=document.getElementById('smartAlertUpdated'), stats=document.getElementById('smartAlertStats'), list=document.getElementById('smartAlertList');
    if(!list)return; window._smartAlerts=board||{};
    if(up)up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok';
    if(stats)stats.innerHTML='<div class="smart-alert-stat"><b>'+Number(board&&board.critical||0)+'</b><span>kritik</span></div><div class="smart-alert-stat"><b>'+Number(board&&board.warning||0)+'</b><span>uyarı</span></div><div class="smart-alert-stat"><b>'+Number(board&&board.info||0)+'</b><span>bilgi</span></div>';
    const alerts=(board&&board.alerts)||[];
    list.innerHTML=alerts.map(a=>'<div class="smart-alert-row '+esc(a.severity||'info')+'" data-smart-alert-id="'+esc(a.id||'')+'"><div class="smart-alert-title"><i class="smart-alert-dot"></i><span>'+esc(a.title||smartAlertLabels[a.severity]||'Uyarı')+'</span></div><div class="smart-alert-message">'+esc(a.message||'')+' <b style="opacity:.9">'+esc(a.eventTitle||'')+'</b></div><div class="smart-alert-meta"><span class="smart-alert-pill">Önem '+Number(a.score||0)+'</span><span class="smart-alert-pill">Güven '+Number(a.confidence||0)+'</span><span class="smart-alert-pill">'+Number(a.sources||0)+' bağımsız</span><span class="smart-alert-pill">Risk '+Number(a.risk||0)+'</span></div></div>').join('')||'<div class="rp-empty">Şu anda dikkat gerektiren bir uyarı yok. Bir olay son dakika olduğunda, hızla tırmandığında ya da en az 3 bağımsız kaynakça doğrulandığında burada görünür; tıklayınca olay dosyası açılır.</div>';
  }
  window.nnOnSmartAlerts=function(board){renderSmartAlerts(board)};
  async function loadSmartAlerts(){ const up=document.getElementById('smartAlertUpdated'); if(up && !window._smartAlerts) up.textContent='toplanıyor…'; try{const b=await window.nnSmartAlerts();renderSmartAlerts(JSON.parse(b));}catch(e){ if(up) up.textContent='alınamadı'; }}

  // ---------- daily briefing ----------
  const briefingLife={detected:'Tespit',confirmed:'Doğrulandı',developing:'Gelişiyor',stabilizing:'Dengeleniyor',archived:'Arşiv'};
  function personalBar(name,score){ return '<div class="personal-meter"><span>'+esc(name)+'</span><i><em style="width:'+Math.max(0,Math.min(100,score))+'%"></em></i><b>'+score+'</b></div>'; }
  function renderPersonalIntelligence(board){
    const score=document.getElementById('personalScore'); if(score) score.textContent=(board.personalScore||0)+'%';
    const up=document.getElementById('personalUpdated'); if(up) up.textContent=board.updated?fmtTime(board.updated):'hazırlanıyor…';
    const ex=document.getElementById('personalExplicit'); if(ex) ex.innerHTML=(board.explicit||[]).map(x=>'<span class="personal-chip">'+esc(x)+'</span>').join('') || '<span class="personal-empty">Ayarlar → İlgi alanları</span>';
    const tp=document.getElementById('personalTopics'); if(tp) tp.innerHTML=(board.topics||[]).map(x=>personalBar(x.name,x.score)).join('') || '<div class="personal-empty">Eşleşen konu yok.</div>';
    const ca=document.getElementById('personalCategories'); if(ca) ca.innerHTML=(board.categories||[]).map(x=>personalBar(x.name,x.score)).join('') || '<div class="personal-empty">Henüz sinyal yok.</div>';
    const en=document.getElementById('personalEntities'); if(en) en.innerHTML=(board.entities||[]).map(x=>personalBar(x.name,x.score)).join('') || '<div class="personal-empty">Henüz sinyal yok.</div>';
    const rs=document.getElementById('personalReasons'); if(rs) rs.innerHTML=(board.reasons||[]).map(x=>'<div class="personal-reason">• '+esc(x.text)+'</div>').join('');
    const st=document.getElementById('personalStories'); if(st) st.innerHTML=(board.topStories||[]).map(x=>'<div class="personal-story" data-event="'+esc(x.eventId||'')+'" data-link="'+esc(x.link||'')+'"><b>'+esc(x.title)+'</b><small>'+esc(x.category||'')+' · '+(x.score||0)+' önem</small></div>').join('') || '<div class="personal-empty">İlgi alanlarınla eşleşen güçlü bir olay henüz yok.</div>';
    if(st) st.querySelectorAll('.personal-story').forEach(el=>el.addEventListener('click',()=>{ const id=el.dataset.event; if(id && typeof openEvent==='function') openEvent(id); else if(el.dataset.link) window.open(el.dataset.link,'_blank'); }));
  }
  window.nnOnPersonalIntelligence=function(board){renderPersonalIntelligence(board)};
  async function loadPersonalIntelligence(){try{const b=await window.nnPersonalIntelligence();renderPersonalIntelligence(JSON.parse(b));}catch(e){}}

  function renderBriefing(board){
    const up=document.getElementById('briefingUpdated'), win=document.getElementById('briefingWindow'), head=document.getElementById('briefingHeadline'), sum=document.getElementById('briefingSummary'), list=document.getElementById('briefingItems');
    if(!list)return; window._briefing=board||{};
    if(up)up.textContent=board&&board.generatedAt?('güncellendi '+agoText(board.generatedAt)):'veri yok';
    if(win)win.textContent=esc(board?.windowLabel||'Günlük brifing');
    if(head)head.textContent=board?.headline||'Bugünün önemli gelişmeleri';
    if(sum)sum.textContent=board?.summary||'';
    const items=board?.items||[];
    list.innerHTML=items.map(it=>'<div class="briefing-item" data-briefing-id="'+esc(it.eventId||'')+'"><div class="briefing-title"><span class="briefing-rank">'+Number(it.rank||0)+'.</span><span>'+esc(it.title||'')+'</span></div><div class="briefing-meta"><span class="briefing-pill">'+esc(it.category||'Genel')+'</span><span class="briefing-pill">Önem '+Number(it.score||0)+'</span><span class="briefing-pill">Güven '+Number(it.confidence||0)+'</span><span class="briefing-pill">'+Number(it.sources||0)+' bağımsız</span><span class="briefing-pill">'+esc(briefingLife[it.lifecycle]||it.lifecycle||'')+'</span></div><div class="briefing-reason">'+esc(it.reason||'')+'</div></div>').join('')||'<div class="rp-empty">Henüz günlük brifing oluşturacak kadar önemli sinyal yok.</div>';
  }
  window.nnOnBriefing=function(board){renderBriefing(board)};
  async function loadBriefing(){try{const b=await window.nnBriefing();renderBriefing(JSON.parse(b));}catch(e){}}

  // ---------- official publications ----------
  function renderOfficialPublications(board){
    const up=document.getElementById('officialUpdated'), country=document.getElementById('officialCountry'), list=document.getElementById('officialList'), portals=document.getElementById('officialPortals');
    if(!list)return; window._officialBoard=board||{};
    if(up)up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok';
    if(country)country.innerHTML='<b>'+esc((board?.flag||'🏛️')+' '+(board?.country||'Resmî yayınlar'))+'</b><small>birincil resmî kaynak</small>';
    const items=board?.publications||[];
    list.innerHTML=items.map(it=>'<a class="official-item" href="'+esc(it.link||'#')+'" target="_blank" rel="noopener"><b><span class="official-type">'+esc(it.type||'Resmî')+'</span>'+esc(it.title||'Yayın')+'</b><small>'+(it.date?fmtTime(it.date):'Güncel resmî yayın')+'</small></a>').join('') || '<div class="official-item"><b>Güncel yayın başlığı alınamadı.</b><small>Resmî portal bağlantıları aşağıda.</small></div>';
    const ps=board?.portals||[]; portals.innerHTML=ps.map(it=>'<a class="official-portal" href="'+esc(it.link||'#')+'" target="_blank" rel="noopener">↗ '+esc(it.name||'Resmî portal')+'</a>').join('');
  }
  window.nnOnOfficialPublications=function(board){try{renderOfficialPublications(board);}catch(e){}};
  function loadOfficialPublications(){try{window.nnOfficialPublications();}catch(e){}}


