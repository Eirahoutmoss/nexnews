  // ---------- world radio ----------
  // RC94: Go tarafı her istasyonun grubunu (Yerel, Avrupa, Orta Doğu…) belirler.
  const RADIO_CONTINENTS=[
    {name:'Yerel',icon:'📍'},{name:'Avrupa',icon:'🇪🇺'},{name:'Orta Doğu',icon:'🕌'},{name:'Asya',icon:'🌏'},{name:'Rusya',icon:'🇷🇺'},
    {name:'Afrika',icon:'🌍'},{name:'Kuzey Amerika',icon:'🌎'},{name:'Güney Amerika',icon:'🌎'},{name:'Okyanusya',icon:'🌊'}
  ];
  let RADIO_Q='', RADIO_TAB='Yerel', RADIO_GROUPS={};
  function radioContinent(st){ return String(st?.continent||'').trim()||'Asya'; }
  function radioGroupIcon(name){const g=RADIO_CONTINENTS.find(x=>x.name===name);return g?g.icon:'🌐';}
  function radioItemHTML(st){
    const i=window._radioBoard.stations.indexOf(st);
    const tags=String(st.tags||'').split(',').map(x=>x.trim()).filter(Boolean).slice(0,3).join(' · ');
    return '<button class="radio-item" data-radio-index="'+i+'"><span class="radio-play">▶</span><span class="radio-info"><span class="radio-name">'+esc(st.name||'Radyo')+'</span><span class="radio-meta">'+esc(tags||st.language||'')+'</span></span><span class="radio-live">CANLI</span></button>';
  }
  function renderRadioGroup(name,stations){
    if(!stations.length) return '<div class="radio-empty">Bu bölge için şu anda istasyon bulunamadı.</div>';
    if(name==='Yerel') return '<div class="radio-grid">'+stations.map(radioItemHTML).join('')+'</div>';
    const countries={}; stations.forEach(st=>{const c=String(st.country||'Bilinmeyen ülke').trim()||'Bilinmeyen ülke';(countries[c]||(countries[c]=[])).push(st);});
    return Object.keys(countries).sort((a,b)=>a.localeCompare(b,'tr')).map(country=>
      '<details class="radio-country-block"><summary class="radio-country-title">'+(countryFlagOf(countries[country][0].countryCode))+' <b>'+esc(country)+'</b><small>'+countries[country].length+' istasyon</small></summary><div class="radio-grid">'+countries[country].map(radioItemHTML).join('')+'</div></details>').join('');
  }
  function countryFlagOf(code){ code=String(code||'').toUpperCase(); if(!/^[A-Z]{2}$/.test(code)) return '🌐'; return String.fromCodePoint(...[...code].map(c=>0x1F1E6+c.charCodeAt(0)-65)); }
  function paintRadioPanel(){
    const host=document.getElementById('radioPanel'); if(!host) return;
    if(RADIO_Q){
      const q=RADIO_Q.toLocaleLowerCase('tr-TR');
      const hits=(window._radioBoard.stations||[]).filter(st=>(String(st.name)+' '+st.country+' '+st.tags+' '+st.language).toLocaleLowerCase('tr-TR').includes(q));
      host.innerHTML='<div class="radio-group-title"><span>🔎</span><b>“'+esc(RADIO_Q)+'”</b><small>'+hits.length+' istasyon</small></div>'+(hits.length?'<div class="radio-grid">'+hits.slice(0,400).map(radioItemHTML).join('')+'</div>':'<div class="radio-empty">Sonuç bulunamadı.</div>');
      return;
    }
    const list=RADIO_GROUPS[RADIO_TAB]||[];
    const title=RADIO_TAB==='Yerel'?(window._radioBoard.localFlag||'📍')+' Yerel':radioGroupIcon(RADIO_TAB)+' '+RADIO_TAB;
    host.innerHTML='<div class="radio-group-title"><b>'+esc(title)+'</b><small>'+list.length+' istasyon</small></div>'+renderRadioGroup(RADIO_TAB,list);
  }
  function selectRadioContinent(name){
    RADIO_TAB=name; RADIO_Q=''; const s=document.getElementById('radioSearch'); if(s) s.value='';
    document.querySelectorAll('.radio-continent-tab').forEach(b=>b.classList.toggle('active',b.dataset.radioContinent===name));
    paintRadioPanel();
  }
  function radioSearch(v){ RADIO_Q=String(v||'').trim(); clearTimeout(radioSearch._t); radioSearch._t=setTimeout(paintRadioPanel,180); }
  function renderRadio(board){
    const up=document.getElementById('radioUpdated'), sub=document.getElementById('radioSub'), list=document.getElementById('radioList');
    if(!list)return; window._radioBoard=board||{};
    if(up)up.textContent=board&&board.updated?('güncellendi '+agoText(board.updated)):'veri yok';
    if(sub)sub.textContent=(board?.localFlag||'🌍')+' Yerel: '+Number(board?.localCount||0)+' · 🌍 '+Number(board?.worldCount||0)+' dünya istasyonu · '+(board?.directoryStatus==='healthy'?'dizin aktif':(board?.directoryStatus==='degraded'?'dizin sınırlı':'dizin erişilemiyor'));
    const stations=board?.stations||[];
    RADIO_GROUPS={}; RADIO_CONTINENTS.forEach(g=>RADIO_GROUPS[g.name]=[]);
    stations.forEach(st=>{const c=radioContinent(st);(RADIO_GROUPS[c]||(RADIO_GROUPS[c]=[])).push(st);});
    if(!(RADIO_GROUPS[RADIO_TAB]||[]).length) RADIO_TAB=(RADIO_CONTINENTS.find(g=>(RADIO_GROUPS[g.name]||[]).length)||{name:'Yerel'}).name;
    list.innerHTML='<div class="radio-toolbar"><input id="radioSearch" class="live-search" type="search" placeholder="İstasyon, ülke ya da tür ara… (ör. jazz, haber, Katar)" oninput="radioSearch(this.value)" onkeydown="event.stopPropagation()"></div>'
      +'<div class="radio-continent-nav">'+RADIO_CONTINENTS.filter(g=>(RADIO_GROUPS[g.name]||[]).length).map(g=>'<button class="radio-continent-tab '+(g.name===RADIO_TAB?'active':'')+'" data-radio-continent="'+esc(g.name)+'" onclick="selectRadioContinent(\''+g.name+'\')">'+(g.name==='Yerel'?(board.localFlag||g.icon):g.icon)+' '+esc(g.name)+' <span>'+RADIO_GROUPS[g.name].length+'</span></button>').join('')+'</div>'
      +'<div id="radioPanel" class="radio-continent-panel active"></div>';
    paintRadioPanel();
  }
  window.nnOnRadio=function(board){try{renderRadio(board);}catch(e){}};
  function loadRadio(){try{window.nnRadio();}catch(e){}}

