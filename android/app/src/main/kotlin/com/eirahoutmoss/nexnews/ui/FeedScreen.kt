@file:OptIn(ExperimentalMaterial3Api::class)

package com.eirahoutmoss.nexnews.ui

import com.eirahoutmoss.nexnews.t

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
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
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.NewsItem
import com.eirahoutmoss.nexnews.Prefs
import com.eirahoutmoss.nexnews.Overview
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive

@Composable
fun FeedScreen(nav: NexNav, bottomPad: PaddingValues) {
    val ctx = LocalContext.current
    val ov by Core.overview.collectAsState()
    val refreshing by Core.refreshing.collectAsState()
    val favs by Core.favLinks.collectAsState()
    val read by Core.readLinks.collectAsState()
    val prefsV by Prefs.get(ctx).version.collectAsState()
    val dense = remember(prefsV) { Prefs.get(ctx).density == "siki" }
    val images = showImages()
    var sheetItem by remember { mutableStateOf<NewsItem?>(null) }
    val scope = androidx.compose.runtime.rememberCoroutineScope()
    // Sakin görünüm (masaüstüyle aynı): günün birkaç önemli olayı ve "bugünlük bu kadar".
    var calm by remember { mutableStateOf(homeViewCalm()) }
    var crisisOff by remember { mutableStateOf(Prefs.get(ctx).crisisDismissed) }
    androidx.compose.runtime.LaunchedEffect(ov) { if (ov != null) calm = homeViewCalm() }
    fun setCalm(v: Boolean) {
        calm = v
        scope.launch { runCatching { Core.saveConfig { it["homeView"] = JsonPrimitive(if (v) "sakin" else "tam") } } }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("NexNews", fontWeight = FontWeight.Bold)
                        NetMeterLine()
                    }
                },
                actions = {
                    IconButton(onClick = { nav.search() }) { Icon(Icons.Filled.Search, t("Ara")) }
                    IconButton(onClick = { Core.refreshAsync(ctx) }) { Icon(Icons.Filled.Refresh, t("Yenile")) }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { pad ->
        PullToRefreshBox(
            isRefreshing = refreshing,
            onRefresh = { Core.refreshAsync(ctx) },
            modifier = Modifier.fillMaxSize().padding(top = pad.calculateTopPadding())
        ) {
            val o = ov
            val empty = o == null || o.categories.all { it.items.isEmpty() }
            if (empty && (refreshing || o == null || o.updatedTs == 0L)) {
                Column(Modifier.fillMaxSize()) {
                    Loading(if (refreshing) t("Başlıklar indiriliyor… İlk açılış biraz sürebilir.") else t("Hazırlanıyor…"))
                }
                return@PullToRefreshBox
            }
            if (empty) {
                NoNewsState { Core.refreshAsync(ctx) }
                return@PullToRefreshBox
            }
            o!!
            val row: @Composable (NewsItem) -> Unit = { item ->
                NewsRow(
                    item = item, read = item.read || item.link in read, fav = item.link in favs, dense = dense,
                    onOpen = { nav.open(item) }, onFav = { Core.toggleFav(item) }, onRead = { Core.markRead(item) },
                    onLong = { sheetItem = item }, images = images
                )
            }
            LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 24.dp)) {
                item {
                    Text(
                        (if (o.updatedTs > 0) t("Güncellendi: ") + agoText(o.updatedTs) else "") +
                            (if (o.totalUnread > 0) " · ${o.totalUnread} " + t("okunmamış") else ""),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 2.dp)
                    )
                }
                o.crisis?.let { c -> if (c.key != crisisOff) item(key = "crisis") { CrisisCard(c, nav, onDismiss = { crisisOff = c.key; Prefs.get(ctx).crisisDismissed = c.key }) } }
                if (o.today.isNotEmpty()) item(key = "today") { TodayStrip(o.today) { nav.agenda() } }
                item { ViewToggle(calm) { setCalm(it) } }
                item { ToolChips(nav, Core.configBool("easyRead", false)) }
                if (calm) {
                    if (o.breaking.isNotEmpty()) item { BreakingStrip(o.breaking.take(1)) { nav.open(it) } }
                    val wd = java.util.Calendar.getInstance().get(java.util.Calendar.DAY_OF_WEEK)
                    if (wd == java.util.Calendar.SUNDAY || wd == java.util.Calendar.MONDAY) item { WeeklyCta { nav.weekly() } }
                    val picks = calmPicks(o)
                    item {
                        SectionHeader(t("🌿 Bugün bilmen gerekenler"), sub = t("Önem, bağımsız kaynak sayısı ve güncelliğe göre seçilen") + " ${picks.size} " + t("olay"))
                    }
                    items(picks, key = { "calm" + it.link }) { row(it) }
                    o.discover?.let { d ->
                        item { SectionHeader(t("Farklı bir pencere"), sub = o.discoverReason) }
                        item(key = "cdisc" + d.link) { row(d) }
                    }
                    item { CalmClosing(o.categories.filter { it.total > 0 }.map { it.name to it.unread }, onCat = { nav.category(it) }, onFull = { setCalm(false) }) }
                    return@LazyColumn
                }
                if (o.breaking.isNotEmpty()) {
                    item { BreakingStrip(o.breaking) { nav.open(it) } }
                }
                if (o.forYou.isNotEmpty()) {
                    item { SectionHeader(t("Senin için")) }
                    items(o.forYou, key = { "fy" + it.link }) { row(it) }
                }
                if (o.important.isNotEmpty()) {
                    item { SectionHeader(t("Şu anda önemli")) }
                    items(o.important, key = { "im" + it.link }) { row(it) }
                }
                o.discover?.let { d ->
                    item { SectionHeader(t("Farklı bir pencere"), sub = o.discoverReason) }
                    item(key = "disc" + d.link) { row(d) }
                }
                for (c in o.categories) {
                    if (c.items.isEmpty()) continue
                    item(key = "h" + c.name) {
                        SectionHeader(
                            t(c.name),
                            sub = if (c.unread > 0) "${c.unread} " + t("okunmamış") + " · ${c.total} " + t("başlık") else "${c.total} " + t("başlık"),
                            action = t("Tümü"),
                            onAction = { nav.category(c.name) }
                        )
                    }
                    items(c.items, key = { "c" + c.name + it.link }) { row(it) }
                }
                item { ClosingCard(o.totalUnread) }
            }
        }
    }
    sheetItem?.let { ItemActionsSheet(it, nav) { sheetItem = null } }
}

/** Hiç başlık alınamadıysa: bağlantı uyarısı, kaynak durumu ve yeniden dene. */
@Composable
private fun NoNewsState(onRetry: () -> Unit) {
    var info by remember { mutableStateOf("") }
    androidx.compose.runtime.LaunchedEffect(Unit) {
        val h = runCatching { Core.sourceHealth() }.getOrDefault(emptyList())
        val bad = h.count { it.status != "healthy" }
        val err = h.firstOrNull { it.lastError.isNotBlank() }?.lastError ?: ""
        info = if (h.isEmpty()) "" else "$bad/${h.size} " + t("kaynağa ulaşılamadı.") + if (err.isNotBlank()) "\n" + t("Örnek hata") + ": $err" else ""
    }
    androidx.compose.foundation.rememberScrollState().let { sc ->
        Column(
            Modifier.fillMaxSize().verticalScroll(sc).padding(32.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Spacer(Modifier.height(60.dp))
            Text(t("Başlık alınamadı"), style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
            Spacer(Modifier.height(8.dp))
            Text(t("İnternet bağlantını kontrol edip yeniden dene. Uygulama bir süre sonra kendisi de yeniden deneyecek."),
                textAlign = TextAlign.Center, color = MaterialTheme.colorScheme.onSurfaceVariant)
            if (info.isNotBlank()) {
                Spacer(Modifier.height(10.dp))
                Text(info, textAlign = TextAlign.Center, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Spacer(Modifier.height(16.dp))
            androidx.compose.material3.Button(onClick = onRetry) { Text(t("Yeniden dene")) }
        }
    }
}

@Composable
private fun BreakingStrip(list: List<NewsItem>, onOpen: (NewsItem) -> Unit) {
    val nex = LocalNex.current
    Column(Modifier.padding(top = 8.dp)) {
        Row(Modifier.padding(horizontal = 16.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.size(8.dp).clip(CircleShape).background(nex.breaking))
            Spacer(Modifier.width(8.dp))
            Text(t("SON DAKİKA"), color = nex.breaking, fontWeight = FontWeight.Bold, fontSize = 12.sp, letterSpacing = 0.8.sp)
        }
        LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            items(list, key = { it.link }) { it ->
                Column(
                    Modifier.width(270.dp).clip(RoundedCornerShape(14.dp))
                        .background(nex.breaking.copy(alpha = 0.12f))
                        .border(1.dp, nex.breaking.copy(alpha = 0.35f), RoundedCornerShape(14.dp))
                        .clickable { onOpen(it) }.padding(12.dp)
                ) {
                    Text(it.title, maxLines = 3, overflow = TextOverflow.Ellipsis, fontWeight = FontWeight.SemiBold, style = MaterialTheme.typography.bodyMedium)
                    Spacer(Modifier.height(6.dp))
                    Text(
                        it.source + (if (it.independentSources > 1) " · ${it.independentSources} " + t("kaynak") else "") + " · " + relTime(it.ts),
                        style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1
                    )
                }
            }
        }
    }
}

@Composable
private fun ClosingCard(unread: Int) {
    Column(
        Modifier.fillMaxWidth().padding(16.dp).clip(RoundedCornerShape(16.dp))
            .background(MaterialTheme.colorScheme.surface)
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(16.dp))
            .padding(22.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Icon(Icons.Filled.CheckCircle, null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(30.dp))
        Spacer(Modifier.height(8.dp))
        Text(t("Önemli gelişmelerin hepsini gördün"), fontWeight = FontWeight.Bold, textAlign = TextAlign.Center)
        Spacer(Modifier.height(4.dp))
        Text(
            if (unread > 0) t("Kategorilerde") + " $unread " + t("okunmamış başlık var; istersen göz atabilir ya da burada bırakabilirsin.") else t("Şimdilik yeni bir şey yok."),
            color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center, style = MaterialTheme.typography.bodyMedium
        )
    }
}

/** Uzun basınca açılan işlemler. */
@Composable
fun ItemActionsSheet(item: NewsItem, nav: NexNav, onDismiss: () -> Unit) {
    val ctx = LocalContext.current
    val favs by Core.favLinks.collectAsState()
    androidx.compose.material3.ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(Modifier.padding(horizontal = 20.dp).padding(bottom = 28.dp)) {
            Text(item.title, fontWeight = FontWeight.SemiBold, maxLines = 3, overflow = TextOverflow.Ellipsis)
            Spacer(Modifier.height(4.dp))
            Text(item.source + " · " + relTime(item.ts), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.height(12.dp))
            val act: @Composable (String, () -> Unit) -> Unit = { label, f ->
                Text(label, modifier = Modifier.fillMaxWidth().clip(RoundedCornerShape(10.dp)).clickable { f(); onDismiss() }.padding(vertical = 13.dp, horizontal = 6.dp))
            }
            if (item.count > 1) act(t("Olay ayrıntısı ve başlık karşılaştırması")) { Core.markRead(item); nav.story(item.link) }
            act(t("Uygulamada oku")) { Core.markRead(item); nav.reader(item.link) }
            act(t("Tarayıcıda aç")) { Core.markRead(item); openInBrowser(ctx, item.link) }
            act(if (item.link in favs) t("Favorilerden çıkar") else t("Favorilere ekle")) { Core.toggleFav(item) }
            act(t("Okundu olarak işaretle")) { Core.markRead(item) }
            act(t("Paylaş")) { shareText(ctx, item.title, item.link) }
        }
    }
}


private const val CALM_MAX = 7

private fun homeViewCalm(): Boolean =
    runCatching { (Core.config()["homeView"] as? JsonPrimitive)?.content != "tam" }.getOrDefault(true)

/** Önce "şu anda önemli", eksik kalırsa en yüksek puanlı başlıklar; en çok 7 olay. */
private fun calmPicks(o: Overview): List<NewsItem> {
    val seen = HashSet<String>()
    val out = ArrayList<NewsItem>()
    fun push(it: NewsItem) { if (it.link.isNotBlank() && seen.add(it.link)) out.add(it) }
    o.important.forEach(::push)
    if (out.size < CALM_MAX) {
        o.categories.flatMap { it.items }.sortedWith(compareByDescending<NewsItem> { it.score }.thenByDescending { it.ts }).forEach(::push)
    }
    return out.take(CALM_MAX)
}

@Composable
private fun ViewToggle(calm: Boolean, onChange: (Boolean) -> Unit) {
    Row(Modifier.padding(horizontal = 16.dp, vertical = 4.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        androidx.compose.material3.FilterChip(selected = calm, onClick = { onChange(true) }, label = { Text(t("🌿 Sakin")) })
        androidx.compose.material3.FilterChip(selected = !calm, onClick = { onChange(false) }, label = { Text(t("Tam akış")) })
    }
}

@Composable
private fun WeeklyCta(onOpen: () -> Unit) {
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp).clip(RoundedCornerShape(14.dp))
            .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.10f))
            .border(1.dp, MaterialTheme.colorScheme.primary.copy(alpha = 0.35f), RoundedCornerShape(14.dp))
            .clickable(onClick = onOpen).padding(14.dp)
    ) {
        Text("📰 " + t("Haftanın özeti hazır"), fontWeight = FontWeight.Bold)
        Text(t("Son 7 günün en önemli olayları ve takip ettiklerin, tek sayfada →"), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@OptIn(androidx.compose.foundation.layout.ExperimentalLayoutApi::class)
@Composable
private fun CalmClosing(cats: List<Pair<String, Int>>, onCat: (String) -> Unit, onFull: () -> Unit) {
    Column(
        Modifier.fillMaxWidth().padding(16.dp).clip(RoundedCornerShape(16.dp))
            .background(MaterialTheme.colorScheme.surface)
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(16.dp))
            .padding(22.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Icon(Icons.Filled.CheckCircle, null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(30.dp))
        Spacer(Modifier.height(8.dp))
        Text(t("Bugünlük bu kadar"), fontWeight = FontWeight.Bold, textAlign = TextAlign.Center)
        Spacer(Modifier.height(4.dp))
        Text(t("Dünyada önemli bir şey olursa burada ve bildirimlerde görürsün. Merak ettiğin bir konu varsa:"),
            color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center, style = MaterialTheme.typography.bodyMedium)
        Spacer(Modifier.height(10.dp))
        androidx.compose.foundation.layout.FlowRow(
            horizontalArrangement = Arrangement.spacedBy(6.dp, Alignment.CenterHorizontally),
            verticalArrangement = Arrangement.spacedBy(6.dp)
        ) {
            cats.forEach { (name, unread) ->
                androidx.compose.material3.AssistChip(onClick = { onCat(name) }, label = { Text(if (unread > 0) t(name) + " $unread" else t(name)) })
            }
        }
        Spacer(Modifier.height(10.dp))
        androidx.compose.material3.OutlinedButton(onClick = onFull) { Text(t("Tüm akışı göster")) }
    }
}

/** RC97: sesli brifing, kolay okuma ve sayfa kısayolları. */
@Composable
private fun ToolChips(nav: NexNav, easy: Boolean) {
    val ctx = LocalContext.current
    val scope = androidx.compose.runtime.rememberCoroutineScope()
    LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        item {
            androidx.compose.material3.FilterChip(selected = Speaker.speaking, onClick = {
                if (Speaker.speaking) Speaker.stop()
                else scope.launch {
                    val b = Core.audioBrief()
                    if (b.text.isBlank()) android.widget.Toast.makeText(ctx, t("Sesli brifing hazırlanamadı."), android.widget.Toast.LENGTH_SHORT).show()
                    else Speaker.speak(ctx, b.text)
                }
            }, label = { Text(if (Speaker.speaking) t("■ Durdur") else t("🔊 Dinle")) })
        }
        item {
            androidx.compose.material3.FilterChip(selected = easy, onClick = {
                scope.launch { Core.saveConfig { it["easyRead"] = JsonPrimitive(!easy) } }
            }, label = { Text(t("Aa Kolay okuma")) })
        }
        item { androidx.compose.material3.AssistChip(onClick = { nav.weekly() }, label = { Text(t("📰 Haftanın özeti")) }) }
        item { androidx.compose.material3.AssistChip(onClick = { nav.agenda() }, label = { Text(t("🗓️ Gündem takvimi")) }) }
        item { androidx.compose.material3.AssistChip(onClick = { nav.mirror() }, label = { Text(t("🪞 Okuma aynan")) }) }
    }
}

private val CRISIS_KIND = mapOf("deprem" to "Deprem", "tsunami" to "Tsunami", "sel" to "Sel", "yangın" to "Orman yangını",
    "fırtına" to "Fırtına", "heyelan" to "Heyelan", "afet" to "Afet")

/** RC97 kriz modu kartı: resmî / çok kaynaklı gelişmeler, acil numaralar, paylaşmadan önce doğrula. */
@Composable
private fun CrisisCard(c: com.eirahoutmoss.nexnews.CrisisInfo, nav: NexNav, onDismiss: () -> Unit) {
    val ctx = LocalContext.current
    val col = if (c.level == "red") androidx.compose.ui.graphics.Color(0xFFEF4444) else androidx.compose.ui.graphics.Color(0xFFF59E0B)
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp).clip(RoundedCornerShape(14.dp))
            .background(col.copy(alpha = 0.09f)).border(2.dp, col, RoundedCornerShape(14.dp)).padding(14.dp)
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(t("⚠ KRİZ MODU"), color = androidx.compose.ui.graphics.Color.White, fontWeight = FontWeight.Black, fontSize = 12.sp,
                modifier = Modifier.clip(RoundedCornerShape(6.dp)).background(col).padding(horizontal = 8.dp, vertical = 2.dp))
            Spacer(Modifier.width(8.dp))
            Text(t(CRISIS_KIND[c.kind] ?: "Afet") + if (c.mag > 0) " · M " + String.format(java.util.Locale.US, "%.1f", c.mag) else "", fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f))
            androidx.compose.material3.TextButton(onClick = onDismiss) { Text(t("Kapat")) }
        }
        Text(c.title, fontWeight = FontWeight.SemiBold, modifier = Modifier.padding(vertical = 6.dp).clickable { if (c.link.isNotBlank()) nav.story(c.link) })
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            c.numbers.forEach { n ->
                androidx.compose.material3.AssistChip(onClick = {
                    runCatching { ctx.startActivity(android.content.Intent(android.content.Intent.ACTION_DIAL, android.net.Uri.parse("tel:" + n.number))) }
                }, label = { Text(n.number + " " + t(n.label), fontWeight = FontWeight.Bold) })
            }
        }
        if (c.stories.isNotEmpty()) {
            Text(t("Resmî ya da en az 3 bağımsız kaynakla doğrulanmış gelişmeler"), style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
            c.stories.take(5).forEach { s ->
                Text("• " + s.title, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.fillMaxWidth().clickable { nav.open(s) }.padding(vertical = 5.dp))
            }
        }
        Text(t("Doğrulanmamış bilgiyi paylaşmadan önce resmî kaynaklardan kontrol et. Afetlerde eski fotoğraf ve videolar sık sık yeniden dolaşıma sokulur."),
            style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 8.dp))
    }
}

/** RC98: bugün bayram, anma ya da dünya günü varsa ince şerit. */
@OptIn(androidx.compose.foundation.layout.ExperimentalLayoutApi::class)
@Composable
private fun TodayStrip(list: List<com.eirahoutmoss.nexnews.AgendaItem>, onMore: () -> Unit) {
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp).clip(RoundedCornerShape(12.dp))
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(12.dp)).clickable(onClick = onMore).padding(10.dp)
    ) {
        list.forEach { a ->
            Text(agendaIcon(a) + " " + agendaTitle(a), fontWeight = if (a.kind == "bayram") FontWeight.Bold else FontWeight.Medium,
                color = if (a.kind == "bayram") MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
                style = MaterialTheme.typography.bodyMedium)
        }
    }
}
