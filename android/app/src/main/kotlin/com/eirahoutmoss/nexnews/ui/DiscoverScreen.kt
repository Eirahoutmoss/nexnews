@file:OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)

package com.eirahoutmoss.nexnews.ui

import com.eirahoutmoss.nexnews.t

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
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.FinanceBoard
import com.eirahoutmoss.nexnews.RadioBoard
import com.eirahoutmoss.nexnews.SportsBoard
import com.eirahoutmoss.nexnews.StandingTable
import com.eirahoutmoss.nexnews.StandingsBoard
import com.eirahoutmoss.nexnews.TvChannel
import com.eirahoutmoss.nexnews.WeatherBoard
import com.eirahoutmoss.nexnews.media.PlayerHub
import kotlinx.coroutines.launch
import java.util.Locale

@Composable
fun DiscoverScreen(nav: NexNav, bottomPad: PaddingValues) {
    var tab by rememberSaveable { mutableIntStateOf(0) }
    val tabs = listOf(t("Canlı TV"), t("Uyarılar"), t("Radyo"), t("Hava"), t("Borsa"), t("Spor"))
    Scaffold(
        topBar = { TopAppBar(title = { Text(t("Keşfet"), fontWeight = FontWeight.Bold) }, colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)) },
        containerColor = MaterialTheme.colorScheme.background
    ) { pad ->
        Column(Modifier.fillMaxSize().padding(top = pad.calculateTopPadding())) {
            ScrollableTabRow(selectedTabIndex = tab, edgePadding = 12.dp, containerColor = MaterialTheme.colorScheme.background) {
                tabs.forEachIndexed { i, label -> Tab(tab == i, onClick = { tab = i }, text = { Text(label) }) }
            }
            val bp = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 16.dp)
            when (tab) {
                0 -> LiveTvTab(nav, bp)
                1 -> SmartAlertsTab(nav, bp)
                2 -> RadioTab(bp)
                3 -> WeatherTab(bp)
                4 -> FinanceTab(nav, bp)
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
    var filter by rememberSaveable { mutableStateOf(t("Yerel")) }
    var query by rememberSaveable { mutableStateOf("") }
    LaunchedEffect(Unit) { board = runCatching { Core.radio() }.getOrElse { RadioBoard(error = t("Radyo dizinine ulaşılamadı.")) } }
    val b = board
    if (b == null) { Loading(t("İstasyonlar yükleniyor…")); return }
    if (b.stations.isEmpty()) { Empty(b.error.ifBlank { t("İstasyon bulunamadı.") }); return }
    val groups = b.stations.map { it.continent.ifBlank { t("Dünya") } }.distinct()
    val q = query.trim().lowercase()
    val shown = if (q.isNotEmpty())
        b.stations.filter { "${it.name} ${it.country} ${it.tags} ${it.language}".lowercase().contains(q) }.take(400)
    else b.stations.filter { val f = if (filter in groups) filter else ""; f.isBlank() || it.continent.ifBlank { t("Dünya") } == f }
    LazyColumn(contentPadding = bp) {
        item {
            OutlinedTextField(value = query, onValueChange = { query = it }, singleLine = true,
                placeholder = { Text(t("İstasyon, ülke ya da tür ara (ör. jazz, haber)")) },
                modifier = Modifier.fillMaxWidth().padding(start = 12.dp, end = 12.dp, top = 8.dp))
        }
        item {
            Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal = 12.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(selected = filter.isBlank(), onClick = { filter = "" }, label = { Text(t("Tümü")) })
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
                Icon(Icons.Filled.PlayArrow, t("Çal"), tint = MaterialTheme.colorScheme.primary)
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f), modifier = Modifier.padding(start = 70.dp))
        }
        item { Text(t("İstasyon dizini: Radio Browser. Yayınlar istasyonların herkese açık akışlarından çalınır."), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp)) }
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
                OutlinedTextField(place, { place = it }, Modifier.weight(1f), singleLine = true, placeholder = { Text(t("Şehir (boş = otomatik)")) })
                TextButton(onClick = {
                    scope.launch { Core.saveConfig { it["weatherPlace"] = kotlinx.serialization.json.JsonPrimitive(place.trim()) } }
                    load()
                }) { Text(t("Göster")) }
            }
        }
        val x = w
        item {
            when {
                busy -> Loading(t("Hava durumu alınıyor…"))
                x == null || !x.ok -> Empty(x?.err?.ifBlank { null } ?: t("Hava durumu alınamadı."))
                else -> Card {
                    Text(x.city, style = MaterialTheme.typography.titleMedium)
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(x.now.icon, fontSize = 40.sp)
                        Spacer(Modifier.width(12.dp))
                        Text(String.format(Locale("tr"), "%.0f°", x.now.temp), fontSize = 44.sp, fontWeight = FontWeight.Bold)
                        Spacer(Modifier.width(12.dp))
                        Column {
                            Text(x.now.desc)
                            Text(t("Rüzgâr") + " ${x.now.wind.toInt()} km/h ${x.now.dirName}", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                    listOf(x.rainText, x.windText, x.thunderText).filter { it.isNotBlank() }.forEach {
                        Spacer(Modifier.height(6.dp)); Text(it, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    if (x.models > 0) { Spacer(Modifier.height(6.dp)); Text("${x.models} " + t("model birleşimi") + " · " + t("güven") + ": ${x.confidence}", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                }
            }
        }
        val steps = x?.steps.orEmpty()
        if (!busy && steps.isNotEmpty()) {
            item {
                Card {
                    Text(t("Önümüzdeki saatler"), fontWeight = FontWeight.SemiBold)
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
    "^XU100" to "BIST 100", "^XU030" to "BIST 30", "THYAO.IS" to t("Türk Hava Yolları"), "ASELS.IS" to "Aselsan",
    "GARAN.IS" to "Garanti BBVA", "BIMAS.IS" to t("BİM"), "USDTRY=X" to "Dolar / TL", "EURTRY=X" to "Euro / TL", "GC=F" to t("Altın (ons)"),
)

@Composable
private fun FinanceTab(nav: NexNav, bp: PaddingValues) {
    var f by remember { mutableStateOf<FinanceBoard?>(null) }
    LaunchedEffect(Unit) { f = runCatching { Core.finance() }.getOrNull() }
    val b = f ?: run { Loading(t("Piyasa verileri alınıyor…")); return }
    if (b.quotes.isEmpty()) { Empty(b.error.ifBlank { t("Piyasa verisi alınamadı.") }); return }
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
                        Text(t("veri alınamadı"), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
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
        if (b.news.isNotEmpty()) {
            item { Text(t("PİYASAYI İLGİLENDİREN GÜNDEM"), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary, modifier = Modifier.padding(start = 16.dp, top = 20.dp, bottom = 4.dp)) }
            items(b.news, key = { "n-" + it.link }) { n ->
                Column(Modifier.fillMaxWidth().clickable { nav.story(n.link) }.padding(horizontal = 16.dp, vertical = 9.dp)) {
                    Text(n.title, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Medium, maxLines = 3)
                    Text(n.tag + " · " + n.source, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f))
            }
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
        item { Text(t("Veriler gecikmeli olabilir; yatırım tavsiyesi değildir. Takip listesi Ayarlar'dan değişir."), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp)) }
    }
}

@Composable
private fun SportsTab(bp: PaddingValues) {
    var view by rememberSaveable { mutableStateOf("futbol") }
    var st by remember { mutableStateOf<StandingsBoard?>(null) }
    var games by remember { mutableStateOf<SportsBoard?>(null) }
    LaunchedEffect(Unit) { st = runCatching { Core.standings() }.getOrNull() ?: StandingsBoard() }
    LaunchedEffect(view) { if (view == "maclar" && games == null) games = runCatching { Core.sports() }.getOrNull() ?: SportsBoard() }
    Column(Modifier.fillMaxSize()) {
        Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal = 12.dp, vertical = 6.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            FilterChip(selected = view == "futbol", onClick = { view = "futbol" }, label = { Text(t("⚽ Süper Lig")) })
            FilterChip(selected = view == "basketbol", onClick = { view = "basketbol" }, label = { Text(t("🏀 Basketbol Süper Ligi")) })
            FilterChip(selected = view == "maclar", onClick = { view = "maclar" }, label = { Text(t("Maçlar")) })
        }
        when (view) {
            "maclar" -> GamesList(games, bp)
            else -> {
                val b = st ?: run { Loading(t("Puan durumu yükleniyor…")); return@Column }
                val tbl = b.tables.firstOrNull { it.key == view }
                if (tbl == null || tbl.rows.isEmpty()) Empty(t("Puan durumu şu an alınamadı") + (tbl?.error?.takeIf { it.isNotBlank() }?.let { " ($it)" } ?: "") + ".")
                else StandingsTable(tbl, b.favorites, bp)
            }
        }
    }
}

private fun noteColor(note: String): Color? = when (note) {
    "Şampiyonlar Ligi" -> Color(0xFF2E9E5B)
    "Avrupa Ligi" -> Color(0xFF3D7BD9)
    "Konferans Ligi" -> Color(0xFF22A6A6)
    "Play-off" -> Color(0xFFE8A33D)
    "Küme düşme" -> Color(0xFFD64545)
    else -> null
}

@Composable
private fun StandingsTable(tbl: StandingTable, favorites: List<String>, bp: PaddingValues) {
    val uri = LocalUriHandler.current
    val dim = MaterialTheme.colorScheme.onSurfaceVariant
    @Composable
    fun Num(v: String, bold: Boolean = false, color: Color = dim) =
        Text(v, Modifier.width(30.dp), textAlign = androidx.compose.ui.text.style.TextAlign.End,
            fontWeight = if (bold) FontWeight.Bold else FontWeight.Normal, color = color, style = MaterialTheme.typography.bodySmall)
    LazyColumn(contentPadding = bp) {
        item {
            Text("${tbl.title} · ${tbl.season}", style = MaterialTheme.typography.labelMedium, color = dim, modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp))
            Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 6.dp), verticalAlignment = Alignment.CenterVertically) {
                Text("#", Modifier.width(26.dp), color = dim, style = MaterialTheme.typography.labelSmall)
                Text(t("Takım"), Modifier.weight(1f), color = dim, style = MaterialTheme.typography.labelSmall)
                Num("O"); Num("G"); if (tbl.hasDraws) Num("B"); Num("M"); Num("AV"); Num("P", bold = true)
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
        }
        items(tbl.rows, key = { tbl.key + it.pos + it.team }) { r ->
            val fav = favorites.any { f -> f.isNotBlank() && r.team.contains(f, ignoreCase = true) }
            Row(
                Modifier.fillMaxWidth()
                    .background(if (fav) MaterialTheme.colorScheme.primary.copy(alpha = 0.10f) else Color.Transparent)
                    .padding(horizontal = 12.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(Modifier.width(3.dp).height(18.dp).background(noteColor(r.note) ?: Color.Transparent))
                Text("${r.pos}", Modifier.width(23.dp).padding(start = 5.dp), color = dim, style = MaterialTheme.typography.bodySmall)
                Text(r.team, Modifier.weight(1f), maxLines = 1, fontWeight = if (fav) FontWeight.Bold else FontWeight.Normal, style = MaterialTheme.typography.bodyMedium)
                Num("${r.played}"); Num("${r.won}"); if (tbl.hasDraws) Num("${r.drawn}"); Num("${r.lost}")
                Num((if (r.diff > 0) "+" else "") + r.diff)
                Num("${r.points}", bold = true, color = MaterialTheme.colorScheme.onSurface)
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f))
        }
        item {
            val notes = tbl.rows.map { it.note }.filter { noteColor(it) != null }.distinct()
            if (notes.isNotEmpty()) Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal = 16.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                notes.forEach { n ->
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(Modifier.size(8.dp).background(noteColor(n)!!)); Spacer(Modifier.width(4.dp))
                        Text(n, style = MaterialTheme.typography.labelSmall, color = dim)
                    }
                }
            }
            Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
                Text(t("Kaynak") + ": ${tbl.source}" + (if (tbl.asOf.isNotBlank()) " · ${tbl.asOf} " + t("maçlarına kadar") else ""), Modifier.weight(1f), style = MaterialTheme.typography.labelSmall, color = dim)
                if (tbl.sourceUrl.isNotBlank()) TextButton(onClick = { runCatching { uri.openUri(tbl.sourceUrl) } }) { Text(t("Ayrıntılar ↗")) }
            }
        }
    }
}

@Composable
private fun GamesList(s: SportsBoard?, bp: PaddingValues) {
    val b = s ?: run { Loading(t("Maçlar yükleniyor…")); return }
    if (b.games.isEmpty()) { Empty(b.error.ifBlank { t("Bugün için maç bulunamadı.") }); return }
    LazyColumn(contentPadding = bp) {
        items(b.games, key = { it.id.ifBlank { it.home + it.away } }) { g ->
            val fav = g.fav
            Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp)) {
                val when_ = when (g.status) {
                    "in" -> "● CANLI " + g.statusDetail
                    "post" -> g.statusDetail.ifBlank { t("Bitti") }
                    else -> if (g.start > 0) java.text.SimpleDateFormat("EEE d MMM HH:mm", java.util.Locale("tr")).format(java.util.Date(g.start * 1000)) else ""
                }
                Text((if (fav) "⭐ " else "") + g.leagueName + " · " + when_, style = MaterialTheme.typography.labelSmall,
                    color = if (g.status == "in") LocalNex.current.breaking else MaterialTheme.colorScheme.onSurfaceVariant)
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(g.home, Modifier.weight(1f), fontWeight = if (fav) FontWeight.Bold else FontWeight.Normal)
                    Text(if (g.status == "prematch") "–" else "${g.homeScore} – ${g.awayScore}", fontWeight = FontWeight.Bold)
                    Text(g.away, Modifier.weight(1f).padding(start = 12.dp), textAlign = androidx.compose.ui.text.style.TextAlign.End, fontWeight = if (fav) FontWeight.Bold else FontWeight.Normal)
                }
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f))
        }
    }
}
