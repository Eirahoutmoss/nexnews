@file:OptIn(ExperimentalMaterial3Api::class)

package com.eirahoutmoss.nexnews.ui

import com.eirahoutmoss.nexnews.t

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.DeleteSweep
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.unit.dp
import com.eirahoutmoss.nexnews.CategoryPage
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.NewsItem
import com.eirahoutmoss.nexnews.Prefs
import com.eirahoutmoss.nexnews.SavedItem
import com.eirahoutmoss.nexnews.SearchResult
import kotlinx.coroutines.launch

@Composable
private fun BackBar(title: String, nav: NexNav, actions: @Composable () -> Unit = {}) {
    TopAppBar(
        title = { Text(title, fontWeight = FontWeight.Bold) },
        navigationIcon = { IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Geri") } },
        actions = { actions() },
        colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)
    )
}

@Composable
private fun ItemList(items: List<NewsItem>, nav: NexNav, bottomPad: PaddingValues, top: PaddingValues) {
    val ctx = LocalContext.current
    val favs by Core.favLinks.collectAsState()
    val read by Core.readLinks.collectAsState()
    val prefsV by Prefs.get(ctx).version.collectAsState()
    val dense = remember(prefsV) { Prefs.get(ctx).density == "siki" }
    val images = showImages()
    var sheet by remember { mutableStateOf<NewsItem?>(null) }
    LazyColumn(
        Modifier.fillMaxSize().padding(top = top.calculateTopPadding()),
        contentPadding = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 24.dp)
    ) {
        items(items, key = { it.link }) { item ->
            NewsRow(item, item.read || item.link in read, item.link in favs, dense,
                onOpen = { nav.open(item) }, onFav = { Core.toggleFav(item) }, onRead = { Core.markRead(item) },
                onLong = { sheet = item }, images = images)
        }
    }
    sheet?.let { ItemActionsSheet(it, nav) { sheet = null } }
}

/** Alt menüdeki "Kategoriler": okunmamış sayılarıyla liste. */
@Composable
fun CategoriesScreen(nav: NexNav, bottomPad: PaddingValues) {
    val ov by Core.overview.collectAsState()
    Scaffold(
        topBar = { TopAppBar(title = { Text(t("Kategoriler"), fontWeight = FontWeight.Bold) }, colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)) },
        containerColor = MaterialTheme.colorScheme.background
    ) { pad ->
        val cats = ov?.categories.orEmpty()
        if (cats.isEmpty()) { Box(Modifier.padding(pad)) { Empty(t("Henüz başlık yok.")) }; return@Scaffold }
        LazyColumn(Modifier.fillMaxSize().padding(top = pad.calculateTopPadding()), contentPadding = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 16.dp)) {
            items(cats, key = { it.name }) { c ->
                Row(
                    Modifier.fillMaxWidth().clickable { nav.category(c.name) }.padding(horizontal = 18.dp, vertical = 16.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column(Modifier.weight(1f)) {
                        Text(t(c.name), style = MaterialTheme.typography.titleMedium)
                        c.items.firstOrNull()?.let {
                            Text(it.title, maxLines = 1, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                    Spacer(Modifier.width(10.dp))
                    if (c.unread > 0) Pill("${c.unread} yeni", color = MaterialTheme.colorScheme.primary) else Pill("${c.total}")
                    Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
            }
        }
    }
}

@Composable
fun CategoryScreen(name: String, nav: NexNav, bottomPad: PaddingValues) {
    var page by remember { mutableStateOf<CategoryPage?>(null) }
    val ov by Core.overview.collectAsState()
    LaunchedEffect(name, ov?.updatedTs) { page = runCatching { Core.category(name) }.getOrNull() }
    Scaffold(topBar = { BackBar(name, nav) }, containerColor = MaterialTheme.colorScheme.background) { pad ->
        val p = page
        when {
            p == null -> Box(Modifier.padding(pad)) { Loading() }
            p.items.isEmpty() -> Box(Modifier.padding(pad)) { Empty(t("Bu kategoride başlık yok.")) }
            else -> ItemList(p.items, nav, bottomPad, pad)
        }
    }
}

@Composable
fun SearchScreen(nav: NexNav, bottomPad: PaddingValues) {
    var q by remember { mutableStateOf("") }
    var res by remember { mutableStateOf<SearchResult?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val focus = remember { FocusRequester() }
    LaunchedEffect(Unit) { runCatching { focus.requestFocus() } }
    val run = { scope.launch { busy = true; res = runCatching { Core.search(q) }.getOrNull(); busy = false } }
    Scaffold(topBar = { BackBar(t("Ara"), nav) }, containerColor = MaterialTheme.colorScheme.background) { pad ->
        Column(Modifier.fillMaxSize().padding(top = pad.calculateTopPadding())) {
            OutlinedTextField(
                value = q, onValueChange = { q = it },
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp).focusRequester(focus),
                placeholder = { Text(t("deprem türkiye · son 24 saat")) },
                singleLine = true,
                trailingIcon = { if (q.isNotEmpty()) IconButton(onClick = { q = ""; res = null }) { Icon(Icons.Filled.Close, "Temizle") } },
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                keyboardActions = KeyboardActions(onSearch = { if (q.isNotBlank()) run() })
            )
            val r = res
            when {
                busy -> Loading(t("Aranıyor…"))
                r == null -> Empty(t("Kelime yazın; \"son 24 saat\", \"bugün\", \"son 3 gün\" gibi ifadeler zaman aralığı olarak anlaşılır."))
                r.items.isEmpty() -> Empty(t("Sonuç yok."))
                else -> {
                    Text(
                        "${r.total} olay · ${r.sourceCount} kaynak" + if (r.windowLabel.isNotBlank()) " · ${r.windowLabel}" else "",
                        style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
                    )
                    ItemList(r.items, nav, bottomPad, PaddingValues(0.dp))
                }
            }
        }
    }
}

@Composable
fun SavedScreen(nav: NexNav, bottomPad: PaddingValues) {
    var tab by remember { mutableIntStateOf(0) }
    var list by remember { mutableStateOf<List<SavedItem>?>(null) }
    val favs by Core.favLinks.collectAsState()
    val scope = rememberCoroutineScope()
    var follows by remember { mutableStateOf<List<com.eirahoutmoss.nexnews.FollowedEvent>?>(null) }
    LaunchedEffect(tab, favs) {
        if (tab == 2) { follows = runCatching { Core.followed() }.getOrDefault(emptyList()); runCatching { Core.followedSeen() } }
        else list = runCatching { if (tab == 0) Core.favorites() else Core.history() }.getOrDefault(emptyList())
    }
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Kaydedilenler", fontWeight = FontWeight.Bold) },
                actions = {
                    if (tab == 1 && !list.isNullOrEmpty()) IconButton(onClick = {
                        scope.launch { Core.clearHistory(); list = emptyList() }
                    }) { Icon(Icons.Filled.DeleteSweep, t("Geçmişi temizle")) }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { pad ->
        Column(Modifier.fillMaxSize().padding(top = pad.calculateTopPadding())) {
            TabRow(selectedTabIndex = tab, containerColor = MaterialTheme.colorScheme.background) {
                Tab(tab == 0, onClick = { tab = 0 }, text = { Text(t("Favoriler")) })
                Tab(tab == 1, onClick = { tab = 1 }, text = { Text(t("Okuma geçmişi")) })
                Tab(tab == 2, onClick = { tab = 2 }, text = { Text("Takip") })
            }
            val l = list
            val f = follows
            when {
                tab == 2 && f == null -> Loading()
                tab == 2 && f.isNullOrEmpty() -> Empty(t("Takip ettiğin olay yok. Bir olay sayfasında 🔔 simgesine dokun; olay büyüdükçe burada güncellenir ve bildirim alırsın."))
                tab == 2 -> LazyColumn(contentPadding = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 16.dp)) {
                    items(f.orEmpty(), key = { it.item.link + it.added }) { r ->
                        Row(
                            Modifier.fillMaxWidth().clickable { if (r.item.link.isNotBlank()) nav.story(r.item.link) }.padding(horizontal = 16.dp, vertical = 12.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Column(Modifier.weight(1f)) {
                                Text(r.item.title, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Medium, maxLines = 3)
                                Text(listOf(if (r.new > 0) "+${r.new} yeni haber" else "", if (!r.live) t("akıştan düştü") else "", r.item.category, relTime(r.item.ts)).filter { it.isNotBlank() }.joinToString(" · "),
                                    style = MaterialTheme.typography.labelSmall, color = if (r.new > 0) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            IconButton(onClick = { scope.launch { Core.unfollow(r.item.link); follows = Core.followed() } }) { Icon(Icons.Filled.Close, t("Takibi bırak")) }
                        }
                        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                    }
                }
                l == null -> Loading()
                l.isEmpty() -> Empty(if (tab == 0) t("Favori yok. Bir haberi sağa kaydırarak ekleyebilirsin.") else t("Okuma geçmişi boş."))
                else -> LazyColumn(contentPadding = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 16.dp)) {
                    items(l, key = { it.link }) { s ->
                        Row(
                            Modifier.fillMaxWidth().clickable { nav.article(s.link) }.padding(horizontal = 16.dp, vertical = 12.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Column(Modifier.weight(1f)) {
                                Text(s.title, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Medium, maxLines = 3)
                                Text(listOf(s.source, s.cat, relTime(if (s.ts > 0) s.ts else s.added)).filter { it.isNotBlank() }.joinToString(" · "),
                                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            if (tab == 0) IconButton(onClick = { Core.toggleFavSaved(s) }) { Icon(Icons.Filled.Close, t("Çıkar")) }
                        }
                        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                    }
                }
            }
        }
    }
}
