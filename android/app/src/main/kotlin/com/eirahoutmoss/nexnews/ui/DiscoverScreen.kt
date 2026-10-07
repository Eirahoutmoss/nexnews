@file:OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)

package com.eirahoutmoss.nexnews.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Radio
import androidx.compose.material.icons.filled.Tv
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.ScrollableTabRow
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.FinanceBoard
import com.eirahoutmoss.nexnews.RadioBoard
import com.eirahoutmoss.nexnews.SportsBoard
import com.eirahoutmoss.nexnews.TvChannel
import com.eirahoutmoss.nexnews.WeatherBoard
import com.eirahoutmoss.nexnews.media.PlayerHub
import kotlinx.coroutines.launch
import java.util.Locale

@Composable
fun DiscoverScreen(nav: NexNav, bottomPad: PaddingValues) {
    var tab by rememberSaveable { mutableIntStateOf(0) }
    val tabs = listOf("Radyo", "Canlı TV", "Hava", "Piyasa", "Spor")
    Scaffold(
        topBar = { TopAppBar(title = { Text("Keşfet", fontWeight = FontWeight.Bold) }, colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)) },
        containerColor = MaterialTheme.colorScheme.background
    ) { pad ->
        Column(Modifier.fillMaxSize().padding(top = pad.calculateTopPadding())) {
            ScrollableTabRow(selectedTabIndex = tab, edgePadding = 12.dp, containerColor = MaterialTheme.colorScheme.background) {
                tabs.forEachIndexed { i, t -> Tab(tab == i, onClick = { tab = i }, text = { Text(t) }) }
            }
            val bp = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 16.dp)
            when (tab) {
                0 -> RadioTab(bp)
                1 -> TvTab(nav, bp)
                2 -> WeatherTab(bp)
                3 -> FinanceTab(bp)
                else -> SportsTab(bp)
            }
        }
    }
}

@Composable
private fun RadioTab(bp: PaddingValues) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    var board by remember { mutableStateOf<RadioBoard?>(null) }
    var filter by rememberSaveable { mutableStateOf("") }
    LaunchedEffect(Unit) { board = runCatching { Core.radio() }.getOrElse { RadioBoard(error = "Radyo dizinine ulaşılamadı.") } }
    val b = board
    if (b == null) { Loading("İstasyonlar yükleniyor…"); return }
    if (b.stations.isEmpty()) { Empty(b.error.ifBlank { "İstasyon bulunamadı." }); return }
    val groups = b.stations.map { it.continent.ifBlank { "Dünya" } }.distinct()
    val shown = b.stations.filter { filter.isBlank() || it.continent.ifBlank { "Dünya" } == filter }
    LazyColumn(contentPadding = bp) {
        item {
            Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal = 12.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(selected = filter.isBlank(), onClick = { filter = "" }, label = { Text("Tümü") })
                groups.forEach { g -> FilterChip(selected = filter == g, onClick = { filter = g }, label = { Text(g) }) }
            }
        }
        items(shown, key = { it.uuid.ifBlank { it.stream } }) { st ->
            Row(
                Modifier.fillMaxWidth().clickable { scope.launch { PlayerHub.playRadio(ctx, st) } }.padding(horizontal = 16.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Box(Modifier.size(42.dp).clip(RoundedCornerShape(10.dp)).background(MaterialTheme.colorScheme.surfaceVariant), contentAlignment = Alignment.Center) {
                    Icon(Icons.Filled.Radio, null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
                    if (st.favicon.isNotBlank()) AsyncImage(st.favicon, null, contentScale = ContentScale.Crop, modifier = Modifier.size(42.dp))
                }
                Spacer(Modifier.width(12.dp))
                Column(Modifier.weight(1f)) {
                    Text(st.name, fontWeight = FontWeight.SemiBold, maxLines = 1)
                    Text(listOf(st.country, st.language, st.tags.split(',').take(2).joinToString(", ")).filter { it.isNotBlank() }.joinToString(" · "),
                        style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1)
                }
                Icon(Icons.Filled.PlayArrow, "Çal", tint = MaterialTheme.colorScheme.primary)
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f), modifier = Modifier.padding(start = 70.dp))
        }
        item { Text("İstasyon dizini: Radio Browser. Yayınlar istasyonların herkese açık akışlarından çalınır.", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp)) }
    }
}

@Composable
private fun TvTab(nav: NexNav, bp: PaddingValues) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    var list by remember { mutableStateOf<List<TvChannel>?>(null) }
    var group by rememberSaveable { mutableStateOf("Türkiye") }
    LaunchedEffect(Unit) { list = runCatching { Core.liveTV() }.getOrDefault(emptyList()) }
    val l = list ?: run { Loading(); return }
    val groups = l.map { it.continent }.distinct()
    LazyColumn(contentPadding = bp) {
        item {
            Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal = 12.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                groups.forEach { g -> FilterChip(selected = group == g, onClick = { group = g }, label = { Text(g) }) }
            }
        }
        items(l.filter { it.continent == group }, key = { it.id }) { ch ->
            val playable = ch.streams.isNotEmpty() && !ch.officialOnly
            Row(
                Modifier.fillMaxWidth().clickable {
                    if (playable) { scope.launch { PlayerHub.playTv(ctx, ch) }; nav.player() }
                    else if (ch.official.isNotBlank()) openInBrowser(ctx, ch.official)
                }.padding(horizontal = 16.dp, vertical = 12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(Icons.Filled.Tv, null, tint = if (playable) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
                Spacer(Modifier.width(14.dp))
                Column(Modifier.weight(1f)) {
                    Text(ch.name, fontWeight = FontWeight.SemiBold)
                    Text(ch.meta, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                Pill(if (playable) "Canlı" else "Resmî site")
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f), modifier = Modifier.padding(start = 54.dp))
        }
        item { Text("Yayın erişimi kanalların herkese açık akışlarına bağlıdır; zaman zaman değişebilir. Ekrandan çıkınca ses arka planda sürer.", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp)) }
    }
}

@Composable
private fun Card(content: @Composable () -> Unit) {
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp).clip(RoundedCornerShape(16.dp))
            .background(MaterialTheme.colorScheme.surface)
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(16.dp))
            .padding(16.dp)
    ) { content() }
}

@Composable
private fun WeatherTab(bp: PaddingValues) {
    val scope = rememberCoroutineScope()
    var place by rememberSaveable { mutableStateOf(Core.config()["weatherPlace"]?.toString()?.trim('"') ?: "") }
    var w by remember { mutableStateOf<WeatherBoard?>(null) }
    var busy by remember { mutableStateOf(true) }
    val load: () -> Unit = { scope.launch { busy = true; w = runCatching { Core.weather(place) }.getOrNull(); busy = false } }
    LaunchedEffect(Unit) { load() }
    LazyColumn(contentPadding = bp) {
        item {
            Row(Modifier.padding(horizontal = 16.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                OutlinedTextField(place, { place = it }, Modifier.weight(1f), singleLine = true, placeholder = { Text("Şehir (boş = otomatik)") })
                TextButton(onClick = {
                    scope.launch { Core.saveConfig { it["weatherPlace"] = kotlinx.serialization.json.JsonPrimitive(place.trim()) } }
                    load()
                }) { Text("Göster") }
            }
        }
        val x = w
        item {
            when {
                busy -> Loading("Hava durumu alınıyor…")
                x == null || !x.ok -> Empty(x?.err?.ifBlank { null } ?: "Hava durumu alınamadı.")
                else -> Card {
                    Text(x.city, style = MaterialTheme.typography.titleMedium)
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(x.now.icon, fontSize = 40.sp)
                        Spacer(Modifier.width(12.dp))
                        Text(String.format(Locale("tr"), "%.0f°", x.now.temp), fontSize = 44.sp, fontWeight = FontWeight.Bold)
                        Spacer(Modifier.width(12.dp))
                        Column {
                            Text(x.now.desc)
                            Text("Rüzgâr ${x.now.wind.toInt()} km/sa ${x.now.dirName}", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                    listOf(x.rainText, x.windText, x.thunderText).filter { it.isNotBlank() }.forEach {
                        Spacer(Modifier.height(6.dp)); Text(it, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    if (x.models > 0) { Spacer(Modifier.height(6.dp)); Text("${x.models} model birleşimi · güven: ${x.confidence}", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                }
            }
        }
        val steps = x?.steps.orEmpty()
        if (!busy && steps.isNotEmpty()) {
            item {
                Card {
                    Text("Önümüzdeki saatler", fontWeight = FontWeight.SemiBold)
                    Spacer(Modifier.height(8.dp))
                    Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                        steps.take(24).forEach { s ->
                            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                                Text(s.label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                Text(s.icon, fontSize = 22.sp)
                                Text(String.format(Locale("tr"), "%.0f°", s.temp), fontWeight = FontWeight.SemiBold)
                                if (s.precipProb > 0) Text("%${s.precipProb}", style = MaterialTheme.typography.labelSmall, color = Color(0xFF5B8CFF))
                            }
                        }
                    }
                }
            }
        }
    }
}

private val marketNames = mapOf(
    "^XU100" to "BIST 100", "^XU030" to "BIST 30", "THYAO.IS" to "Türk Hava Yolları", "ASELS.IS" to "Aselsan",
    "GARAN.IS" to "Garanti BBVA", "BIMAS.IS" to "BİM", "USDTRY=X" to "Dolar / TL", "EURTRY=X" to "Euro / TL", "GC=F" to "Altın (ons)",
)

@Composable
private fun FinanceTab(bp: PaddingValues) {
    var f by remember { mutableStateOf<FinanceBoard?>(null) }
    LaunchedEffect(Unit) { f = runCatching { Core.finance() }.getOrNull() }
    val b = f ?: run { Loading("Piyasa verileri alınıyor…"); return }
    if (b.quotes.isEmpty()) { Empty(b.error.ifBlank { "Piyasa verisi alınamadı." }); return }
    LazyColumn(contentPadding = bp) {
        items(b.quotes, key = { it.symbol }) { q ->
            Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 12.dp), verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text(marketNames[q.symbol.uppercase()] ?: q.name.ifBlank { q.symbol }, fontWeight = FontWeight.SemiBold, maxLines = 1)
                    Text(q.symbol, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                Column(horizontalAlignment = Alignment.End) {
                    if (q.error.isNotBlank() || q.price == 0.0) {
                        Text("—", fontWeight = FontWeight.SemiBold)
                        Text("veri alınamadı", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    } else {
                        Text(String.format(Locale("tr"), "%,.2f %s", q.price, q.currency), fontWeight = FontWeight.SemiBold)
                        val up = q.changePct >= 0
                        Text(String.format(Locale("tr"), "%s%%%.2f", if (up) "▲ " else "▼ ", kotlin.math.abs(q.changePct)),
                            color = if (q.changePct == 0.0) MaterialTheme.colorScheme.onSurfaceVariant else if (up) Color(0xFF36C98F) else Color(0xFFFF6B7A),
                            style = MaterialTheme.typography.labelSmall)
                    }
                }
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f))
        }
        if (b.gold.isNotEmpty()) {
            item { Text("ALTIN (teorik)", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary, modifier = Modifier.padding(start = 16.dp, top = 20.dp, bottom = 4.dp)) }
            items(b.gold, key = { "g-" + it.name }) { g ->
                Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
                    Text(g.name, Modifier.weight(1f))
                    Text(String.format(Locale("tr"), "%,.0f TRY", g.valueTry), fontWeight = FontWeight.SemiBold)
                }
            }
            item { Text(b.gold.first().note, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 16.dp)) }
        }
        item { Text("Veriler gecikmeli olabilir; yatırım tavsiyesi değildir. Takip listesi Ayarlar'dan değişir.", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp)) }
    }
}

@Composable
private fun SportsTab(bp: PaddingValues) {
    var s by remember { mutableStateOf<SportsBoard?>(null) }
    LaunchedEffect(Unit) { s = runCatching { Core.sports() }.getOrNull() }
    val b = s ?: run { Loading("Maçlar yükleniyor…"); return }
    if (b.games.isEmpty()) { Empty(b.error.ifBlank { "Bugün için maç bulunamadı." }); return }
    LazyColumn(contentPadding = bp) {
        items(b.games, key = { it.id.ifBlank { it.home + it.away } }) { g ->
            val fav = b.favorites.any { f -> g.home.contains(f, true) || g.away.contains(f, true) }
            Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp)) {
                Text(g.leagueName + " · " + g.statusDetail.ifBlank { clock(g.start) }, style = MaterialTheme.typography.labelSmall,
                    color = if (g.status == "in") LocalNex.current.breaking else MaterialTheme.colorScheme.onSurfaceVariant)
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(g.home, Modifier.weight(1f), fontWeight = if (fav) FontWeight.Bold else FontWeight.Normal)
                    Text("${g.homeScore} – ${g.awayScore}", fontWeight = FontWeight.Bold)
                    Text(g.away, Modifier.weight(1f).padding(start = 12.dp), fontWeight = if (fav) FontWeight.Bold else FontWeight.Normal)
                }
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f))
        }
    }
}
