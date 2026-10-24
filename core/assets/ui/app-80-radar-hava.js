  // ---------- yağış radarı (Leaflet + RainViewer) ----------
  let RADAR = null; // {map, frames, layers, idx, timer}
  function loadLeaflet(){
    if(window.L) return Promise.resolve();
    return new Promise((res,rej)=>{
      const css=document.createElement("link"); css.rel="stylesheet";
      css.href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"; document.head.appendChild(css);
      const js=document.createElement("script"); js.src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js";
      js.onload=res; js.onerror=rej; document.head.appendChild(js);
    });
  }
  function stopRadar(){ if(RADAR && RADAR.timer){ clearInterval(RADAR.timer); RADAR.timer=null; } }
  async function openRadar(){
    stopRadar();
    document.getElementById("overview").classList.add("hide");
    const p=document.getElementById("catpage"); p.classList.remove("hide");
    p.innerHTML = '<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button><h1>🌧️ Yağış Radarı</h1>'
      + '<span class="pill" id="radarTime">yükleniyor…</span></div>'
      + '<div id="radarMap" class="radar-map"></div>'
      + '<div class="radar-ctl"><button class="btn" id="radarPlay" onclick="radarToggle()">⏸ Duraklat</button>'
      + '<span class="radar-note">Geçmiş + yakın tahmin kareleri · kaynak: RainViewer · harita: OpenStreetMap/Carto</span></div>';
    try{ await loadLeaflet(); }catch(e){ document.getElementById("radarMap").innerHTML='<div class="rp-empty">Harita bileşeni (Leaflet) yüklenemedi — internete ulaşılamadı olabilir.</div>'; return; }
    // nnGeo asenkron: sonucu nnOnGeo ile gelir
    waitGeo = true;
    try{ window.nnGeo(); }catch(e){}
    setTimeout(()=>{ if(waitGeo){ const rt=document.getElementById("radarTime"); if(rt) rt.textContent="konum alınamadı"; } }, 16000);
  }
  let waitGeo=false;
  window.nnOnGeo=function(g){
    if(!waitGeo) return;
    waitGeo=false;
    if(g && g.ok && document.getElementById("radarMap")){ buildRadar(g.lat,g.lon,g.city); }
    else { const rt=document.getElementById("radarTime"); if(rt) rt.textContent="konum alınamadı"; }
  };
  function buildRadar(lat,lon,city){
    const el=document.getElementById("radarMap"); if(!el || !window.L) return;
    el.style.background="#0b1020";
    const map=L.map(el,{zoomControl:true,attributionControl:false}).setView([lat,lon], 7);
    // Anahtarsız taban: ESRI koyu altlık — OSM/Carto'nun aksine uygulama
    // isteklerini politika gereği engellemez. Engellenirse şeffaf hata karosu
    // kullanılır, böylece çirkin "403" karoları yerine temiz koyu zemin kalır
    // ve üstteki yağış katmanı yine çalışır.
    const blankTile="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==";
    L.tileLayer("https://services.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Dark_Gray_Base/MapServer/tile/{z}/{y}/{x}",
      {maxZoom:12,crossOrigin:true,errorTileUrl:blankTile}).addTo(map);
    L.circleMarker([lat,lon],{radius:6,color:"#36c9b6",fillColor:"#36c9b6",fillOpacity:.9}).addTo(map).bindTooltip(city||"Konum");
    RADAR={map,frames:[],layers:[],idx:0,timer:null};
    fetch("https://api.rainviewer.com/public/weather-maps.json").then(r=>{ if(!r.ok) throw new Error("HTTP "+r.status); return r.json(); }).then(d=>{
      const host=d.host; const past=(d.radar&&d.radar.past)||[]; const now=(d.radar&&d.radar.nowcast)||[];
      const frames=past.concat(now);
      if(!frames.length){ document.getElementById("radarTime").textContent="radar verisi yok"; return; }
      RADAR.frames=frames; RADAR.nowStart=past.length;
      frames.forEach(f=>{
        const layer=L.tileLayer(host+f.path+"/256/{z}/{x}/{y}/2/1_1.png",{opacity:0,maxZoom:12});
        layer.addTo(map); RADAR.layers.push(layer);
      });
      RADAR.idx=Math.max(0,past.length-1);
      showRadarFrame(RADAR.idx);
      RADAR.timer=setInterval(()=>{ RADAR.idx=(RADAR.idx+1)%RADAR.frames.length; showRadarFrame(RADAR.idx); }, 650);
    }).catch(e=>{ document.getElementById("radarTime").textContent="radar verisi alınamadı ("+(e&&e.message||"ağ")+")"; });
  }
  function showRadarFrame(i){
    if(!RADAR) return;
    RADAR.layers.forEach((l,j)=>l.setOpacity(j===i?0.75:0));
    const f=RADAR.frames[i]; const t=new Date(f.time*1000);
    const future=i>=RADAR.nowStart;
    document.getElementById("radarTime").textContent=(future?"tahmin ":"")+t.toLocaleTimeString(uiLocale(),{hour:"2-digit",minute:"2-digit"});
  }
  function radarToggle(){
    const b=document.getElementById("radarPlay"); if(!RADAR) return;
    if(RADAR.timer){ clearInterval(RADAR.timer); RADAR.timer=null; b.textContent="▶ Oynat"; }
    else { RADAR.timer=setInterval(()=>{ RADAR.idx=(RADAR.idx+1)%RADAR.frames.length; showRadarFrame(RADAR.idx); },650); b.textContent="⏸ Duraklat"; }
  }
  async function openAbout(){
    let ver="0.1.0"; try{ ver = await window.nnVersion(); }catch(e){}
    document.getElementById("overview").classList.add("hide");
    const p=document.getElementById("catpage"); p.classList.remove("hide");
    p.innerHTML =
      '<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button><h1>Hakkında</h1></div>'
      + '<div class="about">'
      + '<div class="about-hero"><img class="about-logo" src="__NEXLOGO__" alt="">'
      +   '<div><div class="about-name">NexNews <span class="about-ver">v'+esc(ver)+'</span></div>'
      +   '<div class="about-slogan">Dünyanın nabzı, tek akışta</div></div></div>'
      + '<p class="about-p">NexNews, dünyanın dört bir yanındaki haberleri <b>tek pencerede</b>, kullanıcıyı yormadan bir araya getiren, dil ve kaynak bağımsız bir <b>haber radarı</b>dır. Aynı olayı farklı kaynaklardan tek karta toplar, ajans metinlerini ayırt eder, önem sırasına dizer, olayları zaman çizelgesinde izler ve dünyanın canlı yayınlarını aynı pencerede sunar.</p>'
      + '<div class="about-links">'
      +   '<a class="about-link yt" onclick="window.nnOpen(\'https://www.youtube.com/@bakis_acisi\')"><span>▶</span><div><b>YouTube</b><small>@bakis_acisi</small></div></a>'
      +   '<a class="about-link ig" onclick="window.nnOpen(\'https://www.instagram.com/elem_ya\')"><span>◎</span><div><b>Instagram</b><small>@elem_ya</small></div></a>'
      +   '<a class="about-link gh" onclick="window.nnOpen(\'https://github.com/eirahoutmoss\')"><span>⌥</span><div><b>GitHub</b><small>eirahoutmoss</small></div></a>'
      + '</div>'
      + '<div class="about-grid">'
      +   aboutRow('Geliştirici', 'Hasan Güler')
      +   aboutRow('Platformlar', 'Windows masaüstü (Go + WebView2, tek .exe) · Android (Kotlin + ortak Go çekirdeği)')
      +   aboutRow('Haber motoru', 'Çok kaynaklı olay gruplama, Türkçe ek farkındalığı, diller arası birleştirme, ajans (AA, DHA, Reuters, AFP…) ayrımı, "Kim nasıl başlık attı?"')
      +   aboutRow('Olay takibi', 'Takip edilen olaylar başlık değişse de izlenir; yeni gelişmede bildirim')
      +   aboutRow('Görünüm', 'Sakin görünüm (günün önemli olayları ve “bugünlük bu kadar”) ya da tam akış; tek tıkla geçiş')
      +   aboutRow('Kör nokta', 'Her olayda hangi bölgelerin haber yaptığı, hangilerinin sessiz kaldığı')
      +   aboutRow('Haftanın özeti', 'Haftanın özeti: son 7 günün en önemli olayları, bölgelere dağılım ve takip ettiğin olaylar; pazar günleri hatırlatma kartı')
      +   aboutRow('Sahiplik', 'Her kaynağın kamu yayıncısı, devlet medyası, ticari grup ya da bağımsız olduğu; olay panelinde “Kim anlatıyor?” dağılımı')
      +   aboutRow('Okuma aynası', 'Son 30 günde hangi kaynakları, bölgeleri ve sahiplik türlerini okuduğun; yalnızca bu bilgisayarda hesaplanır')
      +   aboutRow('Duygu yükü', 'Abartılı / duygu yüklü başlıklar işaretlenir; istersen olayın en sakin başlığı gösterilir')
      +   aboutRow('Doğrulama', 'Bağımsız doğrulama kuruluşlarının yazıları olaylarla eşleştirilir (Teyit, Doğruluk Payı, Full Fact, PolitiFact, CORRECTIV…)')
      +   aboutRow('Kriz modu', 'Ülkende büyük bir afette resmî ve çok kaynaklı gelişmeler, acil numaralar ve doğrulama hatırlatması')
      +   aboutRow('Bağlam ve takvim', 'Olaydaki adlar için Vikipedi özeti ve önceki olaylar; merkez bankası kararları ve veri açıklamalarıyla gündem takvimi')
      +   aboutRow('Sesli brifing', 'Günün özeti Windows\'un konuşma motoruyla dinlenir; kolay okuma görünümü')
      +   aboutRow('Canlı TV', LIVE_TV_CHANNELS.length+' kanal · 10 bölge · kategoriler, favoriler, kendi kanalını ve açık kaynak M3U listelerini ekleme (yalnızca istenince güncellenir)')
      +   aboutRow('Yerel haberler', 'Sistem ülkesine göre o ülkenin başlıca haber sitelerinden otomatik “📍” kategorisi (90+ ülke)')
      +   aboutRow('Arayüz dili', 'Sistem diline göre otomatik: Türkçe · English · العربية · Deutsch · Français · Español · Русский (diğer diller İngilizce)')
      +   aboutRow('Güncellemeler', 'Kanal, çeviri ve ülke listeleri ile yeni sürümler GitHub üzerinden, Ed25519 imzası doğrulanarak gelir')
      +   aboutRow('Veri akışı', 'Tam / sınırlı mod, hız sınırı, aylık kota; oturum-gün-hafta-ay kullanım defteri')
      +   aboutRow('Sağ panel', 'Özet · Borsa · Spor · Analiz · Olay sekmeleri; kartlar yalnızca açıkken veri toplar')
      +   aboutRow('Masaüstü widget', 'Opak · cam · bulanık · saydam zemin, ayarlanabilir opaklık ve ton')
      +   aboutRow('Hava verisi', 'Open-Meteo (çok model: ICON·GFS·ECMWF·GEM) · RainViewer radar')
      +   aboutRow('Afet / deprem', 'GDACS · USGS')
      +   aboutRow('Çeviri', 'Yerel LibreTranslate → Google · Lingva · SimplyTranslate (anahtarsız, yedekli; çeviriler diskte saklanır)')
      +   aboutRow('Kanal dizini', 'Yayıncı CDN\'leri + iptv-org açık veritabanı (lisanslı kaynaklarla süzülmüş)')
      + '</div>'
      + '<div class="about-upd"><button class="btn" onclick="checkUpdatesNow(this)">⟳ Güncellemeleri denetle</button><span id="updInfo" class="access-note"></span></div>'
      + '<p class="about-note">Haber başlıkları, içerikler ve yayınlar ilgili kaynaklara aittir; NexNews yalnızca derler, bağlantı verir ve yayıncıların herkese açık akışlarını oynatır. Kişisel/eğitim amaçlı geliştirilmiştir.</p>'
      + '</div>';
    window.scrollTo(0,0);
    try{ paintUpdInfo(JSON.parse(await window.nnUpdateStatus())); }catch(e){}
  }
  function aboutRow(k,v){ return '<div class="about-k">'+esc(k)+'</div><div class="about-v">'+esc(v)+'</div>'; }

  // ---------- hava durumu (çok-modelli nowcast) ----------
  let WX = null;
  let WXTIMER = null;
  function shorten(s){ s=s||""; return s.length>58 ? s.slice(0,56)+"…" : s; }
  window.nnOnWeather = function(w){
    WX = w;
    const el = document.getElementById("wxCard");
    if(!el) return;
    el.classList.remove("hide");
    if(!w || !w.ok){
      el.innerHTML = '<div class="wx-city" style="margin:2px 0 0">Hava durumu</div>'
        + '<div class="wx-desc">'+esc((w&&w.err)||"Alınamadı.")+'</div>';
      return;
    }
    const n = w.now||{};
    // RC83: tek satırlık hava çubuğu; ayrıntılar tıklayınca Hava Panosu'nda.
    el.innerHTML =
      '<span class="wxb-ic">'+(n.icon||"🌡️")+'</span>'
      + '<span class="wxb-t">'+Math.round(n.temp)+'°</span>'
      + '<span class="wxb-tx"><b>'+esc(w.city||"")+'</b><small>'+esc(n.desc||"")+' · '+Math.round(n.wind)+' km/sa</small></span>'
      + '<span class="wxb-go">›</span>';
    el.title = (w.city||'')+' — '+(n.desc||'')+'\nHissedilen '+Math.round(w.feels)+'° · Nem %'+(w.humidity||0)+' · '+(n.dirName||'')+' '+Math.round(n.wind)+' km/sa\n🌧️ '+(w.rainText||'')+'\n\nTıkla: Hava panosu';
    fitLeftNav();
    // pano açıksa canlı yeniden çiz
    const cat = document.getElementById("catpage");
    if(document.getElementById("wxBoard") && cat && !cat.classList.contains("hide")){ openWeatherBoard(); }
  };
  function loadWeather(){
    const el = document.getElementById("wxCard");
    if(CONFIG && CONFIG.weather !== false){ try{ window.nnWeather(); }catch(e){} }
    else if(el){ el.classList.add("hide"); }
  }
  async function setWeatherLocation(){
    const input=document.getElementById("wxLocationInput");
    const status=document.getElementById("wxLocationStatus");
    const place=(input&&input.value||"").trim();
    if(!place){
      if(status) status.textContent="Bir şehir, ilçe veya posta kodu yazın.";
      if(input) input.focus();
      return;
    }
    if(status) status.textContent="Konum aranıyor…";
    CONFIG=CONFIG||{};
    CONFIG.weatherPlace=place;
    try{
      await window.nnSaveConfig(JSON.stringify(CONFIG));
      if(status) status.textContent="Konum kaydedildi: "+place+" · hava verisi yenileniyor…";
      loadWeather();
    }catch(e){
      if(status) status.textContent="Konum kaydedilemedi.";
    }
  }
  async function useAutoWeatherLocation(){
    const status=document.getElementById("wxLocationStatus");
    const input=document.getElementById("wxLocationInput");
    if(status) status.textContent="Otomatik konum hazırlanıyor…";
    CONFIG=CONFIG||{};
    CONFIG.weatherPlace="";
    if(input) input.value="";
    try{
      await window.nnSaveConfig(JSON.stringify(CONFIG));
      if(status) status.textContent="Otomatik konum etkin. Hava verisi yenileniyor…";
      loadWeather();
    }catch(e){
      if(status) status.textContent="Otomatik konuma geçilemedi.";
    }
  }

  function openWeatherBoard(){
    if(!WX || !WX.ok) return;
    const w = WX;
    document.getElementById("overview").classList.add("hide");
    const p = document.getElementById("catpage");
    p.classList.remove("hide");
    let h = '<span id="wxBoard" hidden></span>'
          + '<div class="page-head"><button class="btn" onclick="backToOverview()">← Geri</button>'
          + '<h1>Hava Panosu — '+esc(w.city||"")+'</h1>'
          + '<span class="pill">'+w.models+' model ort.</span>'
          + '<span class="pill">güven: '+esc(w.confidence)+'</span>'
          + '<span class="pill">güncelleme: '+(w.updated?new Date(w.updated*1000).toLocaleTimeString(uiLocale(),{hour:"2-digit",minute:"2-digit"}):"—")+'</span></div>'
          + '<div class="wx-location">'
          + '<span class="wx-location-label">📍 Konum</span>'
          + '<input id="wxLocationInput" class="wx-location-input" type="text" value="'+esc(CONFIG.weatherPlace||"")+'" placeholder="Şehir, ilçe veya posta kodu yazın…" autocomplete="off" spellcheck="false" onmousedown="event.stopPropagation()" onclick="event.stopPropagation()" onfocus="event.stopPropagation()" onkeydown="event.stopPropagation();if(event.key===\'Enter\'){event.preventDefault();setWeatherLocation()}" />'
          + '<button class="btn wx-location-btn" onclick="setWeatherLocation()">Konumu Uygula</button>'
          + '<button class="mini wx-location-btn" onclick="useAutoWeatherLocation()">Otomatik Konum</button>'
          + '<span id="wxLocationStatus" class="wx-location-status">Şehir adını yazıp Konumu Uygula butonuna basın. Konum Open-Meteo ile çözümlenir.</span>'
          + '</div>';
    h += '<div class="wx-strip">';
    (w.steps||[]).forEach(s=>{
      const head = ((s.dirFrom||0)+180)%360;
      h += '<div class="wx-col">'
         + '<div class="wx-col-lbl">'+esc(s.label)+'</div>'
         + '<div class="wx-col-ic">'+(s.icon||"")+'</div>'
         + '<div class="wx-col-t">'+Math.round(s.temp)+'°</div>'
         + '<div class="wx-col-p">'+(s.precip>=0.1 ? s.precip.toFixed(1)+' mm' : '—')+' <span class="wx-prob">%'+s.precipProb+'</span></div>'
         + '<div class="wx-col-w"><span class="wx-arrow" style="transform:rotate('+head+'deg)">↑</span> '+Math.round(s.wind)+'</div>'
         + '<div class="wx-col-dir">'+esc(s.dirName)+' → '+esc(s.toName)+'</div>'
         + '</div>';
    });
    h += '</div>';
    // zengin detay ızgarası
    const ws = (l,v)=>'<div class="wx-stat"><div class="ws-l">'+l+'</div><div class="ws-v">'+v+'</div></div>';
    const r0 = n => Math.round(n||0);
    h += '<div class="wx-stats">'
       + ws('Hissedilen', r0(w.feels)+'°')
       + ws('Nem', (w.humidity||0)+'%')
       + ws('Basınç', (w.pressure||0)+' hPa')
       + ws('Görüş', (w.visibility>0 ? r0(w.visibility) : '—')+' km')
       + ws('Bulut', (w.cloud||0)+'%')
       + ws('UV', r0(w.uv)+' / maks '+r0(w.uvMax))
       + ws('Bugün ↑ / ↓', r0(w.dayHi)+'° / '+r0(w.dayLo)+'°')
       + ws('Yağış (bugün)', (w.precipSum||0).toFixed(1)+' mm')
       + ws('🌅 Gün doğumu', w.sunrise||'—')
       + ws('🌇 Gün batımı', w.sunset||'—')
       + '</div>';
    // 24 saatlik görünüm
    if(w.hourly && w.hourly.length){
      h += '<div class="wx-section"><span>24 saatlik görünüm</span><small>saatlik sıcaklık · yağış · rüzgâr</small></div><div class="wx-hourly">';
      w.hourly.forEach((x,i)=>{
        const rain = Number(x.prob||0);
        h += '<div class="wx-hour '+(i===0?'now':'')+'">'
          + '<div class="wx-hour-time">'+esc(x.label||'')+'</div>'
          + '<div class="wx-hour-ic">'+(x.icon||'')+'</div>'
          + '<div class="wx-hour-temp">'+r0(x.temp)+'°</div>'
          + '<div class="wx-hour-rain">💧%'+rain+'</div>'
          + '<div class="wx-hour-wind">💨 '+r0(x.wind)+' km/sa</div>'
          + '</div>';
      });
      h += '</div>';
    }
    // Hava kalitesi
    if(Number(w.aqi)>0){
      h += '<div class="wx-section"><span>Hava kalitesi</span><small>CAMS · Avrupa Hava Kalitesi İndeksi</small></div>'
        + '<div class="wx-aqi">'
        + '<div class="wx-aqi-main"><div class="wx-aqi-score">'+r0(w.aqi)+'</div><div><b>'+esc(w.aqiLevel||'—')+'</b><div class="wx-aqi-label">Avrupa AQI</div></div></div>'
        + ws('PM2.5', Number(w.pm25||0).toFixed(1)+' µg/m³')
        + ws('PM10', Number(w.pm10||0).toFixed(1)+' µg/m³')
        + ws('Ozon', Number(w.ozone||0).toFixed(1)+' µg/m³')
        + ws('NO₂', Number(w.no2||0).toFixed(1)+' µg/m³')
        + '</div>';
    }
    // Kullanıcıya doğrudan anlamlı kısa özet
    const aqiText = Number(w.aqi)>0 ? (' Hava kalitesi '+esc(w.aqiLevel||'belirsiz')+' seviyesinde.') : '';
    h += '<div class="wx-summary">'
      + '<div class="wx-summary-card"><h3>🌤️ Bugünün özeti</h3><p>'+esc((w.city||'')+' için '+(w.now?.desc||'mevcut hava')+'. Gün içi sıcaklık '+r0(w.dayLo)+'° ile '+r0(w.dayHi)+'° arasında. '+(w.rainText||'')+'.')+'</p></div>'
      + '<div class="wx-summary-card"><h3>🧭 Rüzgâr</h3><p>'+esc(w.windText||'Rüzgâr bilgisi mevcut değil.')+'</p></div>'
      + '<div class="wx-summary-card"><h3>☀️ Güneş / UV</h3><p>Gün doğumu <b>'+esc(w.sunrise||'—')+'</b>, gün batımı <b>'+esc(w.sunset||'—')+'</b>. UV '+r0(w.uv)+' / gün maksimumu '+r0(w.uvMax)+'.'+aqiText+'</p></div>'
      + '</div>'
      + '<div class="wx-linkbar"><button class="wx-link" onclick="openRadar()">🌧️ Yağış radarını aç</button></div>';
    if(w.days && w.days.length){
      h += '<div class="tl-head" style="margin-top:2px">5 günlük tahmin</div><div class="wx-days">';
      w.days.forEach(d=>{
        h += '<div class="wx-day">'
           + '<div class="wd-lbl">'+esc(d.label)+'</div>'
           + '<div class="wd-ic">'+(d.icon||'')+'</div>'
           + '<div class="wd-t"><b>'+r0(d.hi)+'°</b> <span class="wd-lo">'+r0(d.lo)+'°</span></div>'
           + '<div class="wd-p">💧'+(d.prob||0)+'% · 💨'+r0(d.wind)+'</div>'
           + '</div>';
      });
      h += '</div>';
    }
    const tl = w.thunderLevel||"yok";
    h += '<div class="wx-insights">'
       + '<div class="wx-ins"><span class="wx-ins-ic">🌧️</span><span>'+esc(w.rainText)+'</span></div>'
       + '<div class="wx-ins"><span class="wx-ins-ic">💨</span><span>'+esc(w.windText)+'</span></div>'
       + '<div class="wx-ins thunder-'+tl+'"><span class="wx-ins-ic">⛈️</span><span>'+esc(w.thunderText)+'</span></div>'
       + '</div>';
    h += '<div class="wx-note">Yakın vade (+10/+30 dk) 15-dakikalık çözünürlükten; +1/+2/+4 saat dört modelin (ICON·GFS·ECMWF·GEM) ortalamasıdır — güven etiketi modellerin uyuşmasından gelir. Gök gürültüsü penceresi CAPE + fırtına kodundan türetilmiş bir <b>tahmindir</b>, canlı yıldırım verisi değildir.</div>';
    p.innerHTML = h;
    window.scrollTo(0,0);
    if(WXTIMER) clearInterval(WXTIMER);
    WXTIMER = setInterval(()=>{ try{ window.nnWeather(); }catch(e){} }, 60000); // canlı tazeleme
  }
  function forceRepaint(){
    const r = document.documentElement; const prev = r.style.display;
    r.style.display = "none"; void r.offsetHeight; r.style.display = prev || "";
  }

