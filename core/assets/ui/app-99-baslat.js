  // ---------- init ----------
  (async function init(){
    try{ document.getElementById("ver").textContent = await window.nnVersion(); }catch(e){}
    try{ CONFIG = JSON.parse(await window.nnGetConfig()); }catch(e){}
    try{ TR_SYS = await window.nnSysLang(); }catch(e){}
    if(!TR_SYS) TR_SYS = normalizeLocalLang((navigator&&navigator.language)||"");
    applyDocumentLanguage();
    applyGlass();
    renderNav();
    initRightCardToggles();
    initRightTabs();
    await loadFavs();
    await loadReadSet();
    await loadOverview();
    loadWeather();
    widgetTopmost = !!(CONFIG && CONFIG.widgetTopmost);
    setupWidgetWindow();
    pollNet();
    // Zamanlayıcılar: pencere gizliyken ya da widget modundayken sağ panel
    // yoklanmaz; her sekme yalnızca görünürken yenilenir.
    setInterval(()=>{ if(!document.hidden) pollNet(); }, 2000);
    loadDataUsage(); setInterval(()=>{ if(!document.hidden) loadDataUsage(); }, 30000);
    // Yalnızca açık kartlar yenilenir.
    const ifOpen=(id,fn)=>()=>{ if(cardOpen(id)) fn(); };
    everyVisible(ifOpen('marketPanel',loadFinance), 120000, 'borsa');
    everyVisible(ifOpen('sportPanel',loadSports), 90000, 'spor');
    everyVisible(ifOpen('standingsPanel',loadStandings), 30*60000, 'spor');
    everyVisible(()=>{ ['followPanel','briefingPanel','personalPanel','smartAlertPanel','worldPanel'].forEach(id=>{ if(cardOpen(id)) CARD_LOADERS[id](); }); }, 60000, 'ozet');
    everyVisible(ifOpen('officialPanel',loadOfficialPublications), 300000, 'ozet');
    everyVisible(()=>{ ['intel2Panel','predictPanel','entityPanel','eventGraphPanel'].forEach(id=>{ if(cardOpen(id)) CARD_LOADERS[id](); }); }, 60000, 'analiz');
    everyVisible(()=>{ ['sourceHealthPanel','contentIntelPanel'].forEach(id=>{ if(cardOpen(id)) CARD_LOADERS[id](); }); }, 120000, 'analiz');
    document.addEventListener("click", e=>{ if(!e.target.isConnected || e.target.closest("#wmenu,[data-wmenu-toggle]")) return; hideWMenu(); });

    // Single delegated click handler — URLs travel in data-* (encodeURIComponent)
    // so quotes in links can never break an inline handler.
    document.addEventListener("click", function(e){
      const followEventEl = e.target.closest("[data-follow-event]");
      if(followEventEl){ e.preventDefault(); toggleFollowEvent(followEventEl.getAttribute("data-follow-event")||"", decodeURIComponent(followEventEl.getAttribute("data-follow-link")||"")); return; }
      const followTopicEl = e.target.closest("[data-follow-topic]");
      if(followTopicEl){ toggleFollowTopic(followTopicEl.getAttribute("data-follow-topic")||""); return; }
      const catEl = e.target.closest("[data-cat]");
      if(catEl){ openCategory(decodeURIComponent(catEl.dataset.cat)); return; }
      const actEl = e.target.closest("[data-act]");
      if(!actEl) return;
      const act = actEl.dataset.act;
      if(act==="openurl"){ browserItem(decodeURIComponent(actEl.dataset.url||"")); return; }
      const holder = actEl.closest("[data-url]");
      if(!holder) return;
      const url = decodeURIComponent(holder.dataset.url||"");
      if(!url) return;
      if(act==="fav"){ toggleFav(url); return; }
      if(act==="open") openItem(url);
      else if(act==="event") openEvent(url);
      else if(act==="read") readItem(url);
      else if(act==="browser") browserItem(url);
    });

    document.addEventListener("keydown",e=>{ if(e.key==="Escape"){ closeReader(); closeSettings(); hideWMenu(); closeEventPanel(); } });
    const radioAudio=document.getElementById('radioAudio'); if(radioAudio){ radioAudio.addEventListener('error',()=>{ if(MEDIA.kind==='radio' && radioAudio.getAttribute('src')){ const ms=document.getElementById('mdSub'); if(ms) ms.textContent='Yayın oynatılamıyor — başka istasyon seçin'; } const sub=document.getElementById('radioSub'); if(sub)sub.textContent='⚠️ Seçilen yayın akışı şu anda oynatılamıyor. Başka bir istasyon seçin.'; }); }
  })();
