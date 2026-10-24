  // ---------- sol gezinme (3 panel) ----------
  const NAV_IC = {
    home:'<path d="M3 11 12 4l9 7"/><path d="M5 10v10h14V10"/>',
    star:'<path d="m12 3 2.7 5.6 6.1.9-4.4 4.3 1 6.1L12 17l-5.4 2.9 1-6.1-4.4-4.3 6.1-.9z"/>',
    clock:'<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    search:'<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
    tv:'<rect x="3" y="6" width="18" height="12" rx="2"/><path d="m9 2 3 4 3-4"/>',
    radio:'<rect x="3" y="8" width="18" height="12" rx="2"/><circle cx="8" cy="14" r="2.5"/><path d="M14 12h4M14 16h4M6 8l10-5"/>',
    rain:'<path d="M7 15a4 4 0 1 1 1-7.9A5 5 0 0 1 18 8a3.5 3.5 0 0 1 0 7z"/><path d="m8 18-1 3M12 18l-1 3M16 18l-1 3"/>',
    info:'<circle cx="12" cy="12" r="9"/><path d="M12 11v6M12 7.5v.5"/>',
    sliders:'<path d="M4 6h10M18 6h2M4 12h4M12 12h8M4 18h12M20 18h0"/><circle cx="16" cy="6" r="2"/><circle cx="10" cy="12" r="2"/><circle cx="18" cy="18" r="2"/>',
    hash:'<path d="M5 9h14M5 15h14M10 4 8 20M16 4l-2 16"/>',
    alert:'<path d="M12 3 2 20h20z"/><path d="M12 10v4M12 17v.5"/>',
    bell:'<path d="M6 8a6 6 0 1 1 12 0c0 7 3 8 3 8H3s3-1 3-8"/><path d="M10 20a2 2 0 0 0 4 0"/>'
  };
  function navIc(k){ return '<svg class="n-ic" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">'+(NAV_IC[k]||NAV_IC.hash)+'</svg>'; }
  function renderNav(){
    const el = document.getElementById("leftnav");
    if(!el) return;
    const active = (document.querySelector('.navi.active')||{}).dataset||{};
    let h = '<div class="nav-grp">';
    h += '<div class="navi active" data-nav="home" onclick="navHome(this)">'+navIc('home')+'<span class="n-tx">Ana sayfa</span><span class="n-cnt" id="navTotal"></span></div>';
    (CONFIG && CONFIG.categories || []).forEach(c=>{
      const nm=String(c.name||''); const clean=nm.replace(/^[⚠-⛿☀-➿️\s]+/,'');
      const ic = /uyar|alert/i.test(nm) ? 'alert' : 'hash';
      h += '<div class="navi" data-nav="cat" data-cat-nav="'+esc(nm)+'" onclick="navCat(this,\''+encodeURIComponent(nm)+'\')">'+navIc(ic)+'<span class="n-tx">'+esc(clean||nm)+'</span><span class="n-cnt"></span></div>';
    });
    h += '</div><div class="nav-grp">';
    h += '<div class="navi" onclick="navFollowed(this)">'+navIc('bell')+'<span class="n-tx">Takip ettiklerim</span><span class="n-cnt" id="navFollowCnt"></span></div>';
    h += '<div class="navi" onclick="navWeekly(this)">'+navIc('clock')+'<span class="n-tx">Haftanın özeti</span></div>';
    h += '<div class="navi" onclick="navList(this,\'favs\')">'+navIc('star')+'<span class="n-tx">Favoriler</span></div>';
    h += '<div class="navi" onclick="navList(this,\'history\')">'+navIc('clock')+'<span class="n-tx">Okuma geçmişi</span></div>';
    h += '<div class="navi" onclick="navSearch()">'+navIc('search')+'<span class="n-tx">Ara</span></div>';
    h += '<div class="navi" onclick="navLiveTV(this)">'+navIc('tv')+'<span class="n-tx">Canlı TV</span></div>';
    h += '<div class="navi nav-sub" onclick="navRadio(this)">'+navIc('radio')+'<span class="n-tx">Dünya radyoları</span></div>';
    h += '<div class="navi" onclick="navRadar(this)">'+navIc('rain')+'<span class="n-tx">Yağış radarı</span></div>';
    h += '<div class="navi" onclick="openSettings(\'kaynak\')">'+navIc('sliders')+'<span class="n-tx">Kaynakları yönet</span></div>';
    h += '<div class="navi" onclick="navAbout(this)">'+navIc('info')+'<span class="n-tx">Hakkında</span></div>';
    h += '</div>';
    el.innerHTML = h;
    if(LAST_OV) updateNavCounts(LAST_OV);
    refreshFollowCount();
    fitLeftNav();
  }
  // RC83: sol menü kaydırma çubuğu olmadan sığsın. Taşarsa adım adım sıkılaştır:
  // sıkı → daha sıkı → kategoriler iki sütun.
  function fitLeftNav(){
    const col=document.getElementById('leftcol'); if(!col || getComputedStyle(col).display==='none' || document.body.classList.contains('widget')) return; // fixed öğede offsetParent hep null
    const steps=['nv-tight','nv-tighter','nv-two'];
    steps.forEach(c=>col.classList.remove(c));
    for(const c of steps){
      if(col.scrollHeight<=col.clientHeight+1) break;
      col.classList.add(c);
    }
  }
  window.addEventListener('resize',()=>{ clearTimeout(fitLeftNav._t); fitLeftNav._t=setTimeout(fitLeftNav,120); });
  const NAV_COUNTS = {};
  function paintNavCount(name){
    document.querySelectorAll('.navi[data-cat-nav]').forEach(n=>{
      if(n.dataset.catNav!==name) return;
      const c=n.querySelector('.n-cnt'); const v=NAV_COUNTS[name]||0;
      if(c){ c.textContent=v?String(v):''; c.classList.toggle('has',v>0); c.title=v?(v+' okunmamış'):''; }
    });
    const t=document.getElementById('navTotal');
    if(t){ const tot=Object.values(NAV_COUNTS).reduce((a,b)=>a+b,0); t.textContent=tot?String(tot):''; t.classList.toggle('has',tot>0); t.title=tot?(tot+' okunmamış başlık'):''; }
  }
  function updateNavCounts(data){
    (data.categories||[]).forEach(c=>{ NAV_COUNTS[c.name]=Number(c.unread||0); paintNavCount(c.name); });
  }
  function bumpNavCount(name,d){ if(name in NAV_COUNTS){ NAV_COUNTS[name]=Math.max(0,NAV_COUNTS[name]+d); paintNavCount(name); } }
  function setNavActive(el){ document.querySelectorAll('.navi').forEach(n=>n.classList.remove('active')); if(el) el.classList.add('active'); }
  function navHome(el){ setNavActive(el); backToOverview(); }
  function navCat(el,enc){ setNavActive(el); openCategory(decodeURIComponent(enc)); }
  function navList(el,which){ setNavActive(el); openList(which); }
  function navSearch(){ const s=document.getElementById("searchBox"); if(s){ s.focus(); } }
  function navAbout(el){ setNavActive(el); openAbout(); }
  function navRadar(el){ setNavActive(el); openRadar(); }
  function navLiveTV(el){ setNavActive(el); openLiveTV(); }
  function navRadio(el){ setNavActive(el); openWorldRadio(); }
  function navWeekly(el){ setNavActive(el); openWeekly(); }

