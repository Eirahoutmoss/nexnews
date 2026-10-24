/* ---- RC92: arayüz dili (NexI18n) ----
   Sayfa Türkçe yazılmıştır. Seçilen dil Türkçe değilse ekrana yazılan her
   arayüz metni sözlükten çevrilir. Haber başlıkları/içerikleri asla
   çevrilmez: hem bilinen haber kapsayıcıları atlanır hem de bir metin ancak
   tamamı sözlükteki parçalardan (artı sayı/simge) oluşuyorsa çevrilir. */
(function(){
  const DATA = __NEXI18N__;
  const SKIP_TAGS = {SCRIPT:1,STYLE:1,TEXTAREA:1,CODE:1,PRE:1,NOSCRIPT:1};
  const NEWS = '.item .title,.item-title,.tk-item,.tk-ttl,.wb-txt,.t-tx,.art,.tl-title,.tl-src,.event-summary,.evidence-summary,.predict-title,.intel2-title,.deep-event-title,.radio-name,.disc-name,.fm-t,.world-hotspot-label,.reader-body,.reader-title,.rd-body,.summary,.item .src,[data-noi18n],[contenteditable]';
  const ATTRS = ['title','placeholder','aria-label'];
  const RTL_LANGS = {ar:1, fa:1, he:1, ur:1};
  let lang='tr', dict=null, re=null, cache=new Map(), regionNames=null, observer=null;
  const orig = new WeakMap();      // text node -> Turkish original
  const wrote = new WeakMap();     // text node -> the value we last wrote
  const origAttr = new WeakMap();  // element -> {attr: original}
  const esc = s => s.replace(/[.*+?^${}()|[\]\\]/g,'\\$&');
  function countryName(code){
    try{ if(!regionNames) regionNames=new Intl.DisplayNames([lang],{type:'region'}); return regionNames.of(code)||null; }catch(e){ return null; }
  }
  function build(){
    cache=new Map(); regionNames=null;
    if(lang==='tr'){ dict=null; re=null; return; }
    const base=(DATA.langs[lang]||DATA.langs.en||{});
    dict=Object.assign({}, lang==='en'?{}:(DATA.langs.en||{}), base);
    // ülke adları: tarayıcının kendi bölge adları (her dilde doğru yazım)
    for(const [tr,code] of Object.entries(DATA.countries||{})){ if(!dict[tr]){ const n=countryName(code); if(n) dict[tr]=n; } }
    const keys=Object.keys(dict).filter(k=>k.length>=2).sort((a,b)=>b.length-a.length);
    re=new RegExp('(?<![\\p{L}\\p{N}])('+keys.map(esc).join('|')+')(?![\\p{L}\\p{N}])','gu');
  }
  function tr(s){
    if(!dict || !s) return s;
    const t=s.trim(); if(!t || !/\p{L}/u.test(t)) return s;
    if(cache.has(s)) return cache.get(s);
    let out=s;
    if(dict[t]!==undefined){ out=s.replace(t,dict[t]); }
    else{
      re.lastIndex=0;
      const rep=t.replace(re,m=>'\u0001'+dict[m]+'\u0002');
      // Kalan (çevrilmemiş) kısımda 3+ harfli bir kelime varsa dokunma: büyük olasılıkla haber metni.
      const rest=rep.replace(/\u0001[^\u0002]*\u0002/g,' ');
      if(rep!==t && !/\p{L}{3,}/u.test(rest)) out=s.replace(t,rep.replace(/[\u0001\u0002]/g,''));
    }
    if(cache.size>5000) cache.clear();
    cache.set(s,out); return out;
  }
  function skipEl(el){ return !el || SKIP_TAGS[el.tagName] || (el.closest && el.closest(NEWS)); }
  function doText(n){
    const p=n.parentElement; if(skipEl(p)) return;
    let o=orig.get(n);
    // Düğümdeki değer bizim yazdığımız değilse sayfa yeni (Türkçe) metin koymuştur.
    if(o===undefined || (n.nodeValue!==o && n.nodeValue!==wrote.get(n))){ o=n.nodeValue; orig.set(n,o); }
    const v=lang==='tr'?o:tr(o);
    wrote.set(n,v);
    if(n.nodeValue!==v) n.nodeValue=v;
  }
  function doAttrs(el){
    if(skipEl(el)) return;
    let m=origAttr.get(el);
    for(const a of ATTRS){
      if(!el.hasAttribute(a)) continue;
      const cur=el.getAttribute(a);
      if(!m) { m={w:{}}; origAttr.set(el,m); }
      if(m[a]===undefined || (cur!==m[a] && cur!==m.w[a])) m[a]=cur;
      const v=lang==='tr'?m[a]:tr(m[a]);
      m.w[a]=v;
      if(cur!==v) el.setAttribute(a,v);
    }
  }
  function walk(root){
    if(!root) return;
    if(root.nodeType===3){ doText(root); return; }
    if(root.nodeType!==1 || skipEl(root)) return;
    doAttrs(root);
    const tw=document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT|NodeFilter.SHOW_TEXT, {acceptNode:n=>{
      if(n.nodeType===1) return skipEl(n)?NodeFilter.FILTER_REJECT:NodeFilter.FILTER_ACCEPT;
      return NodeFilter.FILTER_ACCEPT;
    }});
    let n; while((n=tw.nextNode())){ if(n.nodeType===3) doText(n); else doAttrs(n); }
  }
  function start(){
    if(observer) return;
    observer=new MutationObserver(ms=>{
      if(lang==='tr' && !dictWasOn) return;
      for(const m of ms){
        if(m.type==='characterData') doText(m.target);
        else if(m.type==='attributes') doAttrs(m.target);
        else m.addedNodes.forEach(walk);
      }
    });
    observer.observe(document.documentElement,{subtree:true,childList:true,characterData:true,attributes:true,attributeFilter:ATTRS});
  }
  let dictWasOn=false;
  function setLang(l){
    l=String(l||'').toLowerCase().split(/[-_]/)[0]||'tr';
    if(l!=='tr' && !DATA.langs[l]) l='en'; // sözlüğü olmayan diller İngilizce görür
    if(l===lang && (dict||l==='tr')) return lang;
    lang=l; build(); if(l!=='tr') dictWasOn=true;
    document.documentElement.lang=l;
    // RC95: sağdan sola yazılan diller için sayfa yönü. Haber metinleri kendi
    // yönünde kalır (CSS'te unicode-bidi: plaintext).
    document.documentElement.dir = RTL_LANGS[l] ? 'rtl' : 'ltr';
    if(document.body) walk(document.body);
    start();
    return lang;
  }
  window.NexI18n={ setLang, t:tr, get lang(){ return lang; }, langs:()=>Object.keys(DATA.langs),
    locale:()=>({tr:'tr-TR',en:'en-GB',ar:'ar-QA',de:'de-DE',fr:'fr-FR',es:'es-ES',ru:'ru-RU'})[lang]||lang };
  // İlk tahmin: sistem/tarayıcı dili (ayar sonradan okunur).
  const first=(navigator.language||'tr');
  const go=()=>setLang(first);
  if(document.readyState==='loading') document.addEventListener('DOMContentLoaded',go); else go();
})();
