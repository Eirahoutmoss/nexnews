  "use strict";
  const LIVE_RELAY_TOKEN = "__NEXRELAYTOKEN__";
  let CONFIG = null;
  let currentReadURL = null;
  let currentReadSummary = "";
  let LAST_OV = null;
  const SUMMARIES = {}; // link -> feed summary (reader fallback)
  const ITEMS = {};     // link -> {title,source,cat,ts}
  let FAV = new Set();  // favorilenmiş linkler
  function indexSummaries(items){ (items||[]).forEach(it=>{
    if(!it || !it.link) return;
    if(it.summary) SUMMARIES[it.link]=it.summary;
    ITEMS[it.link] = {title:it.title||"", source:it.source||"", cat:it.category||it.cat||"", ts:it.ts||0};
  }); }
  async function loadFavs(){ try{ FAV = new Set(JSON.parse(await window.nnFavLinks())); }catch(e){} }
  function savedPayload(link){ const m=ITEMS[link]||{}; return JSON.stringify({link:link,title:m.title||"",source:m.source||"",cat:m.cat||"",ts:m.ts||0}); }
  const READ = new Set();
  async function loadReadSet(){ try{ JSON.parse(await window.nnHistory()).forEach(h=>h&&h.link&&READ.add(h.link)); }catch(e){} }
  function recordHistory(link){
    try{ window.nnAddHistory(savedPayload(link)); }catch(e){}
    if(!READ.has(link)){
      READ.add(link);
      document.querySelectorAll('.item[data-url="'+encodeURIComponent(link)+'"]').forEach(n=>n.classList.add('read'));
      const cat=(ITEMS[link]||{}).cat; if(cat) bumpNavCount(cat,-1);
    }
  }
  async function toggleFav(link){
    try{ const r = await window.nnToggleFav(savedPayload(link)); if(r==="on") FAV.add(link); else FAV.delete(link); }catch(e){}
    document.querySelectorAll('.item[data-url="'+encodeURIComponent(link)+'"] .fav').forEach(b=>{
      const on=FAV.has(link); b.textContent=on?'★':'☆'; b.classList.toggle('on',on);
    });
  }
  function starHTML(link){ const on=FAV.has(link); return '<button class="mini fav'+(on?' on':'')+'" data-act="fav" title="Favori">'+(on?'★':'☆')+'</button>'; }

  // ---------- helpers ----------
  // Zaman damgası unix saniye, milisaniye ya da ISO metin olabilir.
  function toTs(v){
    if(v==null || v==="") return 0;
    if(typeof v==="number") return v>1e12 ? v/1000 : v;
    const n=Number(v); if(isFinite(n) && n>0) return n>1e12 ? n/1000 : n;
    const p=Date.parse(v); return isFinite(p) ? p/1000 : 0;
  }
  function agoText(v){ const r=relTime(v); return !r ? "" : (r==="az önce" ? r : r+" önce"); }
  function relTime(ts){
    ts = toTs(ts);
    if(!ts) return "";
    const d = Math.max(0, Date.now()/1000 - ts);
    if(d < 60) return "az önce";
    if(d < 3600) return Math.floor(d/60)+" dk";
    if(d < 86400) return Math.floor(d/3600)+" sa";
    if(d < 604800) return Math.floor(d/86400)+" gün";
    const dt = new Date(ts*1000);
    return dt.toLocaleDateString(uiLocale(),{day:"numeric",month:"short"});
  }
  // HTML + öznitelik güvenli kaçış (tırnaklar da kaçırılır: feed'den gelen
  // başlık/URL'ler value="…" ve title="…" içinde de kullanılıyor).
  // ---- RC95: sahiplik şeffaflığı ----
  const OWN = __NEXOWN__;
  function ownOf(link){
    let h=''; try{ h=new URL(link).hostname.toLowerCase().replace(/^www\./,''); }catch(e){ return null; }
    const d=(OWN&&OWN.d)||{};
    while(h){ if(d[h]) return d[h]; const i=h.indexOf('.'); if(i<0) break; h=h.slice(i+1); if(h.indexOf('.')<0) break; }
    return null;
  }
  function ownTypeLabel(t){ const l=(OWN.types||{})[t]||''; try{ return NexI18n.t(l); }catch(e){ return l; } }
  function ownTitle(link){ const o=ownOf(link); return o?(ownTypeLabel(o[0])+' — '+o[1]):''; }
  function ownDot(link){ const o=ownOf(link); return o?'<i class="own-dot own-'+o[0]+'" aria-hidden="true"></i>':''; }
  function esc(s){ return String(s==null?"":s).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;").replace(/'/g,"&#39;"); }

  // temayı uygula: premium (kozmik+buzlu) / koyu / açık / yüksek kontrast
  function applyGlass(){
    const theme = (CONFIG && CONFIG.theme) || "premium";
    const premium = theme === "premium";
    let op = (CONFIG && CONFIG.glassOpacity) ? CONFIG.glassOpacity : 72;
    op = Math.max(40, Math.min(100, op));
    const r = document.documentElement;
    r.setAttribute("data-theme", theme);
    r.style.setProperty("--ga", (op/100).toFixed(2));
    let fs = Math.max(0.8, Math.min(1.6, Number((CONFIG && CONFIG.fontScale) || 100)/100));
    if(CONFIG && CONFIG.easyRead) fs = Math.max(fs, 1.25); // RC97: kolay okuma
    const lh = Math.max(1.4, Math.min(2.4, Number((CONFIG && CONFIG.lineHeight) || 1.75)));
    const rw = Math.max(520, Math.min(980, Number((CONFIG && CONFIG.readerWidth) || 760)));
    r.style.setProperty("--font-scale", fs.toFixed(2));
    r.style.setProperty("--reader-lh", lh.toFixed(2));
    r.style.setProperty("--reader-measure", rw+"px");
    r.classList.toggle("premium", premium);
    r.classList.toggle("frost", premium);
    r.classList.toggle("dyslexia", !!(CONFIG && CONFIG.dyslexiaFont));
    r.classList.toggle("density-siki", !!(CONFIG && CONFIG.density==="siki"));
    r.classList.toggle("easy-read", !!(CONFIG && CONFIG.easyRead)); // RC97
    if(typeof applyWidgetLookCSS==="function") applyWidgetLookCSS();
  }

