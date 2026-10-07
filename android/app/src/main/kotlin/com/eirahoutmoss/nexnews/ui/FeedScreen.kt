@file:OptIn(ExperimentalMaterial3Api::class)

package com.eirahoutmoss.nexnews.ui

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
                    IconButton(onClick = { nav.search() }) { Icon(Icons.Filled.Search, "Ara") }
                    IconButton(onClick = { Core.refreshAsync(ctx) }) { Icon(Icons.Filled.Refresh, "Yenile") }
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
                    Loading(if (refreshing) "Başlıklar indiriliyor… İlk açılış biraz sürebilir." else "Hazırlanıyor…")
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
                        (if (o.updatedTs > 0) "Güncellendi: " + agoText(o.updatedTs) else "") +
                            (if (o.totalUnread > 0) " · ${o.totalUnread} okunmamış" else ""),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 2.dp)
                    )
                }
                if (o.breaking.isNotEmpty()) {
                    item { BreakingStrip(o.breaking) { nav.open(it) } }
                }
                if (o.forYou.isNotEmpty()) {
                    item { SectionHeader("Senin için") }
                    items(o.forYou, key = { "fy" + it.link }) { row(it) }
                }
                if (o.important.isNotEmpty()) {
                    item { SectionHeader("Şu anda önemli") }
                    items(o.important, key = { "im" + it.link }) { row(it) }
                }
                o.discover?.let { d ->
                    item { SectionHeader("Farklı bir pencere", sub = o.discoverReason) }
                    item(key = "disc" + d.link) { row(d) }
                }
                for (c in o.categories) {
                    if (c.items.isEmpty()) continue
                    item(key = "h" + c.name) {
                        SectionHeader(
                            c.name,
                            sub = if (c.unread > 0) "${c.unread} okunmamış · ${c.total} başlık" else "${c.total} başlık",
                            action = "Tümü",
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
        info = if (h.isEmpty()) "" else "${h.size} kaynaktan $bad tanesine ulaşılamadı." + if (err.isNotBlank()) "\nÖrnek hata: $err" else ""
    }
    androidx.compose.foundation.rememberScrollState().let { sc ->
        Column(
            Modifier.fillMaxSize().verticalScroll(sc).padding(32.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Spacer(Modifier.height(60.dp))
            Text("Başlık alınamadı", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
            Spacer(Modifier.height(8.dp))
            Text("İnternet bağlantını kontrol edip yeniden dene. Uygulama bir süre sonra kendisi de yeniden deneyecek.",
                textAlign = TextAlign.Center, color = MaterialTheme.colorScheme.onSurfaceVariant)
            if (info.isNotBlank()) {
                Spacer(Modifier.height(10.dp))
                Text(info, textAlign = TextAlign.Center, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Spacer(Modifier.height(16.dp))
            androidx.compose.material3.Button(onClick = onRetry) { Text("Yeniden dene") }
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
            Text("SON DAKİKA", color = nex.breaking, fontWeight = FontWeight.Bold, fontSize = 12.sp, letterSpacing = 0.8.sp)
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
                        it.source + (if (it.independentSources > 1) " · ${it.independentSources} kaynak" else "") + " · " + relTime(it.ts),
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
        Text("Önemli gelişmelerin hepsini gördün", fontWeight = FontWeight.Bold, textAlign = TextAlign.Center)
        Spacer(Modifier.height(4.dp))
        Text(
            if (unread > 0) "Kategorilerde $unread okunmamış başlık var; istersen göz at, istersen burada bırak." else "Şimdilik yeni bir şey yok.",
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
            if (item.count > 1) act("Olay ayrıntısı ve başlık karşılaştırması") { Core.markRead(item); nav.story(item.link) }
            act("Uygulamada oku") { Core.markRead(item); nav.reader(item.link) }
            act("Tarayıcıda aç") { Core.markRead(item); openInBrowser(ctx, item.link) }
            act(if (item.link in favs) "Favorilerden çıkar" else "Favorilere ekle") { Core.toggleFav(item) }
            act("Okundu olarak işaretle") { Core.markRead(item) }
            act("Paylaş") { shareText(ctx, item.title, item.link) }
        }
    }
}

