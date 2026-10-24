  // ---------- widget (akış) modu ----------
  let WIDGET_ITEMS = [];
  let widgetIdx = 0;
  let widgetTimer = null;
  let marketTimer = null;
  const WIDGET_HOLD = 3000; // her başlık 3 sn görünür, sonra kayar

  function enterWidget(){
    applyWidgetLookCSS();
    document.body.classList.add("widget");
    const nm=document.getElementById('netMeter'); if(nm) nm.classList.add('hide');
    try{ window.nnSetWidget(true); }catch(e){}
    renderWidget();
  }
  function exitWidget(){
    document.body.classList.remove("widget");
    if(widgetTimer){ clearInterval(widgetTimer); widgetTimer=null; }
    try{ window.nnSetWidget(false); }catch(e){}
    hideWMenu();
    setTimeout(forceRepaint, 160); // tam ekrana dönüşte hayalet kalıntıları temizle
    setTimeout(()=>{ pollNet(); refreshVisibleTab(); }, 400);
  }

  function widgetAllowed(catName){
    const sel = (CONFIG && CONFIG.widgetCats) || [];
    if(!sel.length) return true;           // hiçbiri seçili değilse tümü
    return sel.indexOf(catName) >= 0;
  }
  function renderWidget(){
    const data = LAST_OV || {categories:[],breaking:[],important:[]};
    const impOnly = CONFIG && CONFIG.widgetImpOnly;
    // son dakika satırı (seçili kategorilere göre)
    const wb = document.getElementById("wBreak");
    const brk = (data.breaking||[]).filter(b=>widgetAllowed(b.category));
    if(brk.length && !(CONFIG && CONFIG.widgetSingle)){
      const txt = brk.slice(0,6).map(b=>b.title).join("   •   ");
      wb.innerHTML = '<span class="live"></span><span class="wb-txt">'+esc(txt)+'</span>';
      wb.classList.remove("hide");
    }else{
      wb.innerHTML = ""; wb.classList.add("hide");
    }
    // akış listesi
    const seen = {}; const items = [];
    const push = it => { if(it && it.link && !seen[it.link]){ seen[it.link]=1; items.push(it); } };
    brk.forEach(push);
    (data.important||[]).forEach(it=>{ if(widgetAllowed(it.category)) push(it); });
    if(!impOnly){
      (data.categories||[]).forEach(c=>{ if(widgetAllowed(c.name)) (c.items||[]).forEach(push); });
    }
    WIDGET_ITEMS = items.slice(0,60);
    widgetIdx = 0;
    if(widgetTimer){ clearInterval(widgetTimer); widgetTimer=null; }
    showWidgetItem();
    const hold = Math.max(1, Math.min(30, (CONFIG && CONFIG.widgetSlideSec) || 3)) * 1000;
    if(WIDGET_ITEMS.length > 1){ widgetTimer = setInterval(()=>{ if(!WIDGET_PAUSED) nextWidgetItem(); }, hold); }
  }
  // Fare widget'ın üzerindeyken akış durur; okumaya fırsat verir.
  let WIDGET_PAUSED = false;
  (function(){
    const w=document.getElementById('widget'); if(!w) return;
    w.addEventListener('mouseenter',()=>{ WIDGET_PAUSED=true; document.body.classList.add('w-paused'); });
    w.addEventListener('mouseleave',()=>{ WIDGET_PAUSED=false; document.body.classList.remove('w-paused'); });
  })();
  function showWidgetItem(){
    const el = document.getElementById("wTicker");
    if(!el) return;
    if(!WIDGET_ITEMS.length){ el.textContent = "Başlık yok"; el.removeAttribute("data-url"); return; }
    const it = WIDGET_ITEMS[widgetIdx % WIDGET_ITEMS.length];
    el.className = "tk-item" + (it.breaking ? " brk" : "");
    el.setAttribute("data-url", encodeURIComponent(it.link||""));
    const srcLbl = esc(it.source||"") + (it.sourceCount>1 ? ' +'+(it.sourceCount-1) : '');
    el.innerHTML = (it.breaking?'⚡ ':'')
      + '<span class="tk-ttl">'+esc(it.title)+'</span>'
      + '<span class="tk-src">'+srcLbl+'</span>';
    const wc=document.getElementById('wCount'); if(wc) wc.textContent=(widgetIdx % WIDGET_ITEMS.length + 1)+'/'+WIDGET_ITEMS.length;
    translateDOM();
  }
  function nextWidgetItem(){
    const el = document.getElementById("wTicker");
    if(!el || !WIDGET_ITEMS.length) return;
    el.classList.add("out");              // eski başlık sola kayarak çıkar
    setTimeout(()=>{
      widgetIdx = (widgetIdx + 1) % WIDGET_ITEMS.length;
      showWidgetItem();
      el.classList.remove("out"); el.classList.add("in"); // yeni başlık sağdan gelsin
      void el.offsetHeight;                // reflow
      el.classList.remove("in");           // 0'a kayarak yerleşir
    }, 420);
  }

  // --- masaüstü widget: taşıma + sağ-tık menüsü ---
  let widgetTopmost = false;
  function setupWidgetWindow(){
    const w = document.getElementById("widget");
    if(!w) return;
    w.addEventListener("mousedown", e=>{
      if(e.button !== 0) return;
      if(e.target.closest("button, input, .tk-item, .wmenu, .w-resize")) return;
      try{ window.nnDragMove(); }catch(_){}
    });
    const grip = w.querySelector(".w-resize");
    if(grip){
      let active=false, sx=0, sy=0, sw=0, sh=0, raf=0, lastW=0, lastH=0;
      grip.addEventListener("pointerdown", e=>{
        if(e.button!==0) return;
        e.preventDefault(); e.stopPropagation();
        active=true; sx=e.clientX; sy=e.clientY; sw=window.innerWidth; sh=window.innerHeight;
        lastW=sw; lastH=sh;
        try{ grip.setPointerCapture(e.pointerId); }catch(_){}
      });
      grip.addEventListener("pointermove", e=>{
        if(!active) return;
        lastW=Math.max(300,Math.min(1000,sw+(e.clientX-sx)));
        lastH=Math.max(60,Math.min(500,sh+(e.clientY-sy)));
        if(raf) return;
        raf=requestAnimationFrame(()=>{
          raf=0;
          try{ window.nnResizeWidget(Math.round(lastW),Math.round(lastH)); }catch(_){}
        });
      });
      const stop=()=>{ active=false; if(raf){cancelAnimationFrame(raf);raf=0;} try{ window.nnSaveWidgetSize(); }catch(_){} };
      grip.addEventListener("pointerup", stop); grip.addEventListener("pointercancel", stop);
    }
    w.addEventListener("contextmenu", e=>{ e.preventDefault(); showWMenu(); });
  }
  // Widget hızlı ayar paneli: widget penceresi küçük olduğu için açılır menü
  // yerine pencerenin içine sığan bir panel kullanılır.
  function showWMenu(){
    const m = document.getElementById("wmenu");
    const mode=(CONFIG&&CONFIG.widgetMode)||'cam';
    const op=Number(CONFIG&&CONFIG.widgetOpacity!=null?CONFIG.widgetOpacity:78);
    const tint=Number(CONFIG&&CONFIG.widgetTint!=null?CONFIG.widgetTint:45);
    m.className='wmenu wpanel';
    m.innerHTML =
      '<div class="wp-row"><div class="seg" id="wpMode">'
      + [['opak','Opak'],['cam','Cam'],['bulanik','Bulanık'],['saydam','Saydam']].map(x=>'<button type="button" data-v="'+x[0]+'" class="'+(x[0]===mode?'active':'')+'">'+x[1]+'</button>').join('')
      + '</div><span class="spacer"></span><button class="mini" onclick="hideWMenu()" title="Kapat">✕</button></div>'
      + '<div class="wp-row"><span class="wp-lbl">Yüzey</span><input type="range" min="0" max="100" value="'+op+'" oninput="CONFIG.widgetOpacity=+this.value;pushWidgetLook()" title="Yüzey opaklığı">'
      + ((mode==='cam'||mode==='saydam')?'<span class="wp-lbl">Ton</span><input type="range" min="0" max="100" value="'+tint+'" oninput="CONFIG.widgetTint=+this.value;pushWidgetLook()" title="'+(mode==='saydam'?'Gölge tonu':'Cam tonu')+'">':'')
      + '</div>'
      + '<div class="wp-row"><button class="mini" onclick="toggleWTopmost()">'+(widgetTopmost?'✓ ':'')+'Üstte kal</button>'
      + '<button class="mini" onclick="CONFIG.widgetSingle=!CONFIG.widgetSingle;pushWidgetLook();renderWidget();showWMenu()">'+(CONFIG&&CONFIG.widgetSingle?'✓ ':'')+'Tek satır</button>'
      + '<button class="mini" onclick="hideWMenu();exitWidget()">Tam ekran</button>'
      + '<button class="mini" onclick="hideWMenu();nnMin()">Küçült</button></div>';
    bindSegOnce(m.querySelector('#wpMode'), v=>setWidgetMode(v));
    m.classList.remove("hide");
  }
  function bindSegOnce(el,cb){ if(!el) return; el.addEventListener('click',e=>{ const b=e.target.closest('button[data-v]'); if(b){ cb(b.dataset.v); } }); }
  function hideWMenu(){ const m=document.getElementById("wmenu"); if(m){ m.classList.add("hide"); m.className='wmenu hide'; } }
  function toggleWTopmost(){ widgetTopmost=!widgetTopmost; try{window.nnSetTopmost(widgetTopmost);}catch(e){} if(CONFIG) CONFIG.widgetTopmost=widgetTopmost; showWMenu(); }
  function nnMin(){ try{window.nnMinimize();}catch(e){} }

