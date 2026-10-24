@file:OptIn(ExperimentalMaterial3Api::class)

package com.eirahoutmoss.nexnews.ui

import com.eirahoutmoss.nexnews.t

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.filled.StarBorder
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Checkbox
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.LiveCandidate
import com.eirahoutmoss.nexnews.LiveHealth
import com.eirahoutmoss.nexnews.LiveState
import com.eirahoutmoss.nexnews.NexJson
import com.eirahoutmoss.nexnews.TvChannel
import com.eirahoutmoss.nexnews.UserList
import com.eirahoutmoss.nexnews.UserLists
import com.eirahoutmoss.nexnews.media.PlayerHub
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withPermit
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

private val LIVE_REGIONS = listOf("Türkiye", "Avrupa", "Orta Doğu", "Asya", "Afrika", "Kuzey Amerika", "Güney Amerika", "Okyanusya", "Rusya", "Belgesel & Spor")
private val LIVE_KINDS = listOf("all" to "Tümü", "haber" to "Haber", "ekonomi" to "Ekonomi", "spor" to "Spor", "belgesel" to "Belgesel",
    "kultur" to "Kültür", "genel" to "Genel", "muzik" to "Müzik", "cocuk" to "Çocuk / Eğitim")
private const val G_FAV = "⭐ Favoriler"
private const val G_MINE = "📌 Kendi kanallarım"
private const val G_ALL = "🌐 Tüm dünya"
private const val G_HIDDEN = "🚫 Gizlenenler"

/**
 * Canlı TV (masaüstüyle aynı mantık): bölgeler, kategoriler, arama, favoriler,
 * kanalı listeden kaldırma, kendi kanalını ekleme ve kullanıcı onayıyla kanal
 * denetimi (izlenemeyenler gizlenir).
 */
@Composable
fun LiveTvTab(nav: NexNav, bp: PaddingValues) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    var base by remember { mutableStateOf<List<TvChannel>?>(null) }
    var state by remember { mutableStateOf(LiveState()) }
    val health = remember { mutableStateMapOf<String, LiveHealth>() }
    var group by rememberSaveable { mutableStateOf("Türkiye") }
    var kind by rememberSaveable { mutableStateOf("all") }
    var query by rememberSaveable { mutableStateOf("") }
    var onlyOk by rememberSaveable { mutableStateOf(true) }
    var askCheck by remember { mutableStateOf(false) }
    var addOpen by remember { mutableStateOf(false) }
    var checkJob by remember { mutableStateOf<Job?>(null) }
    var progress by remember { mutableStateOf("") }
    var autoStart by remember { mutableStateOf(false) }
    var lists by remember { mutableStateOf(UserLists()) }
    var listsOpen by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        base = runCatching { Core.liveTV() }.getOrDefault(emptyList())
        state = Core.liveState()
        lists = Core.liveLists()
        health.putAll(Core.liveHealth())
        if (state.favs.isNotEmpty()) group = G_FAV
        val now = System.currentTimeMillis() / 1000
        val stale = base.orEmpty().count { c -> health[c.id]?.let { h -> now - h.ts > (if (h.ok) 7 else 3) * 86400 } ?: true }
        if (stale > 0 && state.check == "ask") askCheck = true
        else if (stale > 0 && state.check == "auto" && !isMetered(ctx)) autoStart = true
    }
    val l0 = base ?: run { Loading(t("Kanallar yükleniyor…")); return }
    val custom = state.custom.map { it.copy(custom = true, continent = "Kendi kanallarım") }
    val fromLists = remember(lists) { listChannels(lists) }
    val all = l0 + custom + fromLists
    // Hazır liste kapalıysa yalnızca kullanıcının kendi kanalları ve listeleri görünür.
    val visible = all.filter { it.id !in state.hidden && !(lists.builtinOff && !it.custom && it.listId.isEmpty()) }

    fun persist(s: LiveState) { state = s; scope.launch { Core.saveLiveState(s) } }

    fun runCheck(list: List<TvChannel>) {
        if (checkJob?.isActive == true) { checkJob?.cancel(); progress = ""; return }
        checkJob = scope.launch {
            var done = 0; var ok = 0
            val sem = Semaphore(3)
            progress = t("Denetleniyor…") + " 0/${list.size}"
            list.map { c ->
                async {
                    sem.withPermit {
                        val r = Core.probeLive(c)
                        health[c.id] = r
                        done++; if (r.ok) ok++
                        progress = t("Deneniyor") + "… $done/${list.size} ($ok " + t("çalışıyor") + ")"
                    }
                }
            }.awaitAll()
            progress = t("Denetim bitti") + ": $ok/$done " + t("kanal çalışıyor")
        }
    }

    LaunchedEffect(autoStart) { if (autoStart) { autoStart = false; runCheck(visible) } }

    val groups = buildList {
        add(G_FAV to state.favs.mapNotNull { id -> visible.find { it.id == id } })
        add(G_MINE to visible.filter { it.custom })
        lists.lists.forEach { l -> add(listGroupName(l) to visible.filter { it.listId == l.id }) }
        LIVE_REGIONS.forEach { r -> visible.filter { it.continent == r }.takeIf { it.isNotEmpty() }?.let { add(r to it) } }
        add(G_ALL to visible)
        if (state.hidden.isNotEmpty()) add(G_HIDDEN to state.hidden.mapNotNull { id -> all.find { it.id == id } })
    }
    val current = groups.firstOrNull { it.first == group }?.second ?: groups.first { it.first == G_ALL }.second
    fun hiddenByFilter(c: TvChannel): Boolean {
        if (kind != "all" && c.kindOrGuess != kind) return true
        if (onlyOk && group != G_HIDDEN && health[c.id]?.ok == false) return true
        if (query.isNotBlank() && !(c.name + " " + c.country + " " + c.meta).lowercase().contains(query.trim().lowercase())) return true
        return false
    }
    val shown = current.filterNot(::hiddenByFilter)
    val badCount = current.count { health[it.id]?.ok == false }

    LazyColumn(contentPadding = bp) {
        item {
            Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal = 12.dp, vertical = 6.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                groups.forEach { (g, items) -> FilterChip(selected = group == g, onClick = { group = g }, label = { Text(t(g) + " ${items.size}") }) }
            }
            Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal = 12.dp), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                LIVE_KINDS.forEach { (k, label) ->
                    val n = if (k == "all") current.size else current.count { it.kindOrGuess == k }
                    FilterChip(selected = kind == k, onClick = { kind = k }, label = { Text(t(label) + " $n") }, enabled = n > 0 || k == "all")
                }
            }
            OutlinedTextField(query, { query = it }, singleLine = true, placeholder = { Text(t("Kanal ya da ülke ara…")) },
                modifier = Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 6.dp))
            Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp), verticalAlignment = Alignment.CenterVertically) {
                Switch(checked = onlyOk, onCheckedChange = { onlyOk = it })
                Spacer(Modifier.width(8.dp))
                Text(t("Yalnızca izlenebilenler"), style = MaterialTheme.typography.bodySmall, modifier = Modifier.weight(1f))
                TextButton(onClick = { runCheck(current) }) { Text(if (checkJob?.isActive == true) t("Durdur") else t("Denetle")) }
                IconButton(onClick = { listsOpen = true }) { Text("📋") }
                IconButton(onClick = { addOpen = true }) { Icon(Icons.Filled.Add, t("Kanal ekle")) }
            }
            val info = progress.ifBlank { if (onlyOk && badCount > 0) "$badCount " + t("kanal bu bağlantıdan izlenemediği için gizlendi") else "" }
            if (info.isNotBlank()) Text(info, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary, modifier = Modifier.padding(horizontal = 16.dp, vertical = 2.dp))
        }
        if (shown.isEmpty()) item {
            Text(if (group == G_FAV) t("Henüz favori kanal yok — bir kanalın ☆ işaretine dokun.") else if (group == G_MINE) t("Kendi kanalını + ile ekleyebilirsin.") else t("Bu filtreye uyan kanal yok."),
                style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp))
        }
        items(shown, key = { it.id }) { ch ->
            val h = health[ch.id]
            val fav = ch.id in state.favs
            var menu by remember { mutableStateOf(false) }
            Row(
                Modifier.fillMaxWidth().clickable {
                    persist(state.copy(last = ch.id))
                    scope.launch { PlayerHub.playTv(ctx, ch) }
                    nav.player()
                }.padding(start = 16.dp, end = 4.dp, top = 8.dp, bottom = 8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                val dot = when { h == null -> MaterialTheme.colorScheme.outline; h.ok -> Color(0xFF22C55E); else -> Color(0xFFEF4444) }
                Box(Modifier.size(9.dp).clip(CircleShape).background(dot))
                Spacer(Modifier.width(12.dp))
                Column(Modifier.weight(1f)) {
                    Text(ch.name, fontWeight = FontWeight.SemiBold, maxLines = 1)
                    val sub = listOf(if (group == G_ALL || group == G_FAV) ch.country else "", ch.meta, if (ch.geo.isNotBlank()) "🌐 ${ch.geo}" else "")
                        .filter { it.isNotBlank() }.distinct().joinToString(" · ")
                    Text(sub, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1)
                    if (h != null && !h.ok && h.reason.isNotBlank()) Text(h.reason, style = MaterialTheme.typography.labelSmall, color = Color(0xFFEF4444), maxLines = 2)
                }
                IconButton(onClick = { persist(state.copy(favs = if (fav) state.favs - ch.id else state.favs + ch.id)) }) {
                    Icon(if (fav) Icons.Filled.Star else Icons.Filled.StarBorder, t("Favori"), tint = if (fav) Color(0xFFF5B301) else MaterialTheme.colorScheme.onSurfaceVariant)
                }
                Box {
                    IconButton(onClick = { menu = true }) { Icon(Icons.Filled.MoreVert, t("Seçenekler")) }
                    DropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                        when {
                            group == G_HIDDEN -> DropdownMenuItem(text = { Text(t("Listeye geri al")) }, onClick = { menu = false; persist(state.copy(hidden = state.hidden - ch.id)) })
                            ch.custom -> DropdownMenuItem(text = { Text(t("Kanalı sil")) }, onClick = {
                                menu = false; persist(state.copy(custom = state.custom.filterNot { it.id == ch.id }, favs = state.favs - ch.id))
                            })
                            else -> DropdownMenuItem(text = { Text(t("Listeden kaldır")) }, onClick = {
                                menu = false; persist(state.copy(hidden = state.hidden + ch.id, favs = state.favs - ch.id))
                            })
                        }
                        DropdownMenuItem(text = { Text(t("Bu kanalı denetle")) }, onClick = { menu = false; scope.launch { health[ch.id] = Core.probeLive(ch) } })
                    }
                }
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f), modifier = Modifier.padding(start = 37.dp))
        }
        item {
            Text(t("Yalnızca yayıncıların kendi ya da lisanslı ücretsiz platformların yayınları oynatılır. ☆ favori · ⋮ listeden kaldır · + kendi kanalını ekle. Ekrandan çıkınca ses arka planda sürer."),
                style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp))
        }
    }

    if (askCheck) {
        var keep by remember { mutableStateOf(false) }
        val metered = isMetered(ctx)
        AlertDialog(
            onDismissRequest = { askCheck = false },
            title = { Text(t("Kanal listesini doğrulayalım mı?")) },
            text = {
                Column {
                    Text(t("NexNews kanalları senin bağlantından kısaca deneyip izlenemeyenleri (kapanmış ya da yalnızca kendi ülkesinde açık yayınları) gizleyebilir. Tüm liste için birkaç dakika ve birkaç MB veri gerekir; bu sırada uygulamayı kullanabilirsin."))
                    if (metered) Text(t("Şu an mobil verideysin."), color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.labelSmall, modifier = Modifier.padding(top = 6.dp))
                    Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 8.dp)) {
                        Checkbox(checked = keep, onCheckedChange = { keep = it }); Text(t("Seçimimi hatırla"), style = MaterialTheme.typography.bodySmall)
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    askCheck = false
                    if (keep) persist(state.copy(check = "auto"))
                    runCheck(visible)
                }) { Text(t("Tümünü doğrula")) }
            },
            dismissButton = {
                Row {
                    TextButton(onClick = { askCheck = false; if (keep) persist(state.copy(check = "never")) }) { Text(t("Doğrulama")) }
                    TextButton(onClick = { askCheck = false; runCheck(current) }) { Text(t("Bu bölge")) }
                }
            }
        )
    }

    if (listsOpen) ListsDialog(
        lists = lists,
        onDismiss = { listsOpen = false },
        onChanged = { newLists, select -> lists = newLists; if (select != null) { group = select; listsOpen = false } }
    )

    if (addOpen) AddChannelDialog(
        onDismiss = { addOpen = false },
        onAdded = { ch ->
            persist(state.copy(custom = state.custom + ch))
            health[ch.id] = LiveHealth(ok = true, ts = System.currentTimeMillis() / 1000)
            group = G_MINE; addOpen = false
        }
    )
}

@Composable
private fun AddChannelDialog(onDismiss: () -> Unit, onAdded: (TvChannel) -> Unit) {
    val scope = rememberCoroutineScope()
    var name by remember { mutableStateOf("") }
    var url by remember { mutableStateOf("") }
    var ref by remember { mutableStateOf("") }
    var kind by remember { mutableStateOf("genel") }
    var status by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text(t("Kanal ekle")) },
        text = {
            Column {
                OutlinedTextField(name, { name = it }, label = { Text(t("Kanal adı")) }, singleLine = true)
                OutlinedTextField(url, { url = it }, label = { Text(t("Yayın adresi (.m3u8)")) }, singleLine = true)
                OutlinedTextField(ref, { ref = it }, label = { Text(t("Referer (gerekirse)")) }, singleLine = true)
                Row(Modifier.horizontalScroll(rememberScrollState()).padding(top = 6.dp), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    LIVE_KINDS.drop(1).forEach { (k, l) -> FilterChip(selected = kind == k, onClick = { kind = k }, label = { Text(t(l)) }) }
                }
                Text(t("Yalnızca yayıncısının herkese açık sunduğu yayınları ekle. Adres önce denenir; çalışmazsa kaydedilmez."),
                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 6.dp))
                if (status.isNotBlank()) Text(status, color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(top = 6.dp))
            }
        },
        confirmButton = {
            TextButton(enabled = !busy, onClick = {
                val u = url.trim()
                if (name.isBlank() || !(u.startsWith("http://") || u.startsWith("https://"))) { status = t("Ad ve http(s) ile başlayan bir adres gerekli."); return@TextButton }
                busy = true; status = t("Deneniyor…")
                scope.launch {
                    val cand = if (ref.isBlank()) JsonPrimitive(u) else buildJsonObject { put("u", u); put("r", ref.trim()) }
                    val label = LIVE_KINDS.firstOrNull { it.first == kind }?.second?.let { t(it) } ?: t("Genel")
                    val ch = TvChannel(id = "my-" + System.currentTimeMillis().toString(36), name = name.trim().take(60), meta = t("Kendi kanalım") + " · $label",
                        country = t("Kendi kanalım"), continent = "Kendi kanallarım", kind = kind, urls = listOf(cand), custom = true)
                    val r = Core.probeLive(ch)
                    busy = false
                    if (r.ok) onAdded(ch) else status = t("Adres çalışmadı: ") + r.reason.ifBlank { t("yayın açılamadı") }
                }
            }) { Text(if (busy) t("Deneniyor…") else t("Dene ve ekle")) }
        },
        dismissButton = { TextButton(enabled = !busy, onClick = onDismiss) { Text(t("Vazgeç")) } }
    )
}

/** Akıllı uyarılar: açıldığında toplanır, dokununca olay sayfası açılır. */
@Composable
fun SmartAlertsTab(nav: NexNav, bp: PaddingValues) {
    var board by remember { mutableStateOf<com.eirahoutmoss.nexnews.SmartAlertBoard?>(null) }
    LaunchedEffect(Unit) { board = runCatching { Core.smartAlerts() }.getOrDefault(com.eirahoutmoss.nexnews.SmartAlertBoard()) }
    val b = board ?: run { Loading(t("Uyarılar toplanıyor…")); return }
    LazyColumn(contentPadding = bp) {
        item {
            Row(Modifier.padding(16.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Pill("${b.critical} " + t("kritik"), color = Color(0xFFEF4444))
                Pill("${b.warning} " + t("uyarı"), color = Color(0xFFE0A83A))
                Pill("${b.info} " + t("bilgi"))
            }
        }
        if (b.alerts.isEmpty()) item {
            Text(t("Şu anda dikkat gerektiren bir uyarı yok. Bir olay son dakika olduğunda, hızla tırmandığında ya da en az 3 bağımsız kaynakça doğrulandığında burada görünür."),
                style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp))
        }
        items(b.alerts, key = { it.id }) { a ->
            val c = when (a.severity) { "critical" -> Color(0xFFEF4444); "warning" -> Color(0xFFE0A83A); else -> MaterialTheme.colorScheme.primary }
            Column(Modifier.fillMaxWidth().clickable { if (a.link.isNotBlank()) nav.story(a.link) }.padding(horizontal = 16.dp, vertical = 10.dp)) {
                Text(a.title, color = c, style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.Bold)
                Text(a.eventTitle, fontWeight = FontWeight.SemiBold)
                Text(a.message + " · " + t("önem") + " ${a.score} · " + t("güven") + " ${a.confidence} · ${a.sources} " + t("bağımsız kaynak"),
                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f))
        }
    }
}

private fun listGroupName(l: UserList) = "📋 " + l.name

/** M3U grup adından kanal türünü tahmin eder (masaüstüyle aynı kural). */
private fun guessListKind(g: String): String {
    val x = g.lowercase()
    return when {
        Regex("news|haber|info|noticias|nachrichten|أخبار").containsMatchIn(x) -> "haber"
        Regex("sport|spor|deport").containsMatchIn(x) -> "spor"
        Regex("doc|belgesel").containsMatchIn(x) -> "belgesel"
        Regex("music|müzik|musique").containsMatchIn(x) -> "muzik"
        Regex("kid|child|çocuk|cocuk|anim|educ|eğitim").containsMatchIn(x) -> "cocuk"
        Regex("business|econom|finans|ekonomi").containsMatchIn(x) -> "ekonomi"
        Regex("cult|kültür|kultur|relig").containsMatchIn(x) -> "kultur"
        else -> "genel"
    }
}

private fun listChannels(u: UserLists): List<TvChannel> = u.lists.flatMap { l ->
    l.channels.map { ch ->
        val kind = guessListKind(ch.group)
        val where = ch.country.ifBlank { l.name }
        TvChannel(
            id = ch.id, name = ch.name, listId = l.id, continent = listGroupName(l), country = where, kind = kind,
            meta = where + " · " + ch.group.ifBlank { LIVE_KINDS.firstOrNull { it.first == kind }?.second?.let { t(it) } ?: t("Genel") },
            urls = ch.urls,
        )
    }
}

/** Kullanıcının M3U listeleri: ekle (adres ya da dosya), güncelle, yeniden adlandır, sil; hazır listeyi gizle. */
@Composable
private fun ListsDialog(lists: UserLists, onDismiss: () -> Unit, onChanged: (UserLists, String?) -> Unit) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    var name by remember { mutableStateOf("") }
    var url by remember { mutableStateOf("") }
    var fileText by remember { mutableStateOf("") }
    var fileName by remember { mutableStateOf("") }
    var status by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var renaming by remember { mutableStateOf<String?>(null) }
    var renameText by remember { mutableStateOf("") }
    var armedDelete by remember { mutableStateOf<String?>(null) }
    val fmt = remember { java.text.SimpleDateFormat("dd.MM.yyyy HH:mm", java.util.Locale.getDefault()) }

    val pick = androidx.activity.compose.rememberLauncherForActivityResult(androidx.activity.result.contract.ActivityResultContracts.GetContent()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            val r = kotlinx.coroutines.withContext(kotlinx.coroutines.Dispatchers.IO) {
                runCatching {
                    ctx.contentResolver.openInputStream(uri)?.use { ins ->
                        val b = ins.readBytes()
                        if (b.size > 8 * 1024 * 1024) null else String(b, Charsets.UTF_8)
                    }
                }.getOrNull()
            }
            if (r == null) { status = t("Dosya okunamadı ya da çok büyük (en çok 8 MB).") ; return@launch }
            fileText = r
            val n = uri.lastPathSegment?.substringAfterLast('/') ?: ""
            fileName = n
            if (name.isBlank()) name = n.replace(Regex("\\.(m3u8?|txt)$", RegexOption.IGNORE_CASE), "")
        }
    }

    suspend fun reload(): UserLists = Core.liveLists()

    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text(t("📋 Kanal listelerim")) },
        text = {
            LazyColumn {
                item {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Switch(checked = !lists.builtinOff, onCheckedChange = { on ->
                            scope.launch {
                                runCatching { Core.liveBuiltin(on) }
                                onChanged(lists.copy(builtinOff = !on), null)
                            }
                        })
                        Spacer(Modifier.width(8.dp))
                        Text(t("NexNews hazır kanal listesini göster"), style = MaterialTheme.typography.bodySmall)
                    }
                    Spacer(Modifier.size(8.dp))
                    if (lists.lists.isEmpty()) Text(t("Henüz liste eklemedin."), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                items(lists.lists, key = { it.id }) { l ->
                    Column(Modifier.fillMaxWidth().padding(vertical = 6.dp)) {
                        if (renaming == l.id) {
                            OutlinedTextField(renameText, { renameText = it.take(40) }, singleLine = true, modifier = Modifier.fillMaxWidth())
                            Row {
                                TextButton(onClick = {
                                    val v = renameText.trim(); renaming = null
                                    if (v.isNotEmpty() && v != l.name) scope.launch {
                                        runCatching { Core.liveListRename(l.id, v) }
                                        onChanged(reload(), null); status = t("Liste adı değişti")
                                    }
                                }) { Text(t("Kaydet")) }
                                TextButton(onClick = { renaming = null }) { Text(t("Vazgeç")) }
                            }
                        } else {
                            Text(l.name, fontWeight = FontWeight.SemiBold)
                        }
                        Text("${l.channels.size} " + t("kanal · son güncelleme") + ": " + (if (l.updated > 0) fmt.format(java.util.Date(l.updated * 1000)) else "—"),
                            style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        Text(l.url.ifBlank { t("Dosyadan / yapıştırarak eklendi") }, style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1)
                        Row(horizontalArrangement = Arrangement.spacedBy(2.dp)) {
                            TextButton(onClick = { onChanged(lists, listGroupName(l)) }) { Text(t("Aç")) }
                            if (l.url.isNotBlank()) TextButton(enabled = !busy, onClick = {
                                busy = true; status = t("Liste indiriliyor…")
                                scope.launch {
                                    runCatching { Core.liveListRefresh(l.id) }
                                        .onSuccess { nl -> onChanged(reload(), null); status = "“${nl.name}” " + t("güncellendi") + " · ${nl.channels.size} " + t("kanal") }
                                        .onFailure { e -> status = e.message ?: t("Liste güncellenemedi.") }
                                    busy = false
                                }
                            }) { Text(t("⟳ Güncelle")) }
                            TextButton(onClick = { renaming = l.id; renameText = l.name }) { Text("✎") }
                            TextButton(onClick = {
                                if (armedDelete != l.id) { armedDelete = l.id; status = "“${l.name}” " + t("silinsin mi? Onaylamak için yeniden bas."); return@TextButton }
                                armedDelete = null
                                scope.launch {
                                    runCatching { Core.liveListDelete(l.id) }
                                    onChanged(reload(), null); status = t("Liste silindi")
                                }
                            }) { Text(if (armedDelete == l.id) t("Emin misin?") else t("🗑 Sil"), color = MaterialTheme.colorScheme.error) }
                        }
                    }
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f))
                }
                item {
                    Spacer(Modifier.size(10.dp))
                    Text(t("Yeni liste ekle"), fontWeight = FontWeight.Bold)
                    OutlinedTextField(name, { name = it.take(40) }, label = { Text(t("Liste adı")) }, singleLine = true, modifier = Modifier.fillMaxWidth())
                    OutlinedTextField(url, { url = it }, label = { Text(t("Liste adresi (M3U / M3U8)")) }, placeholder = { Text("https://…/liste.m3u") },
                        singleLine = true, modifier = Modifier.fillMaxWidth())
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        TextButton(onClick = { pick.launch("*/*") }) { Text(t("📂 Dosya seç")) }
                        if (fileName.isNotBlank()) Text(fileName, style = MaterialTheme.typography.labelSmall, maxLines = 1)
                    }
                    Text(t("Listeyi sen seçersin ve yalnızca sen “Güncelle” dediğinde yenilenir; NexNews arka planda liste indirmez. Listedeki yayınların herkese açık ve yasal olmasından liste sahibi ve sen sorumlusunuz. Çalışmayan kanalları 🩺 denetimiyle ayıklayabilir, ✕ ile kaldırabilirsin."),
                        style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    if (status.isNotBlank()) Text(status, color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(top = 6.dp))
                }
            }
        },
        confirmButton = {
            TextButton(enabled = !busy, onClick = {
                val u = url.trim()
                if (u.isEmpty() && fileText.isEmpty()) { status = t("Bir liste adresi yaz ya da dosya seç."); return@TextButton }
                if (u.isNotEmpty() && !(u.startsWith("http://") || u.startsWith("https://"))) { status = t("Liste adresi http(s) ile başlamalı."); return@TextButton }
                busy = true; status = if (u.isNotEmpty()) t("Liste indiriliyor…") else t("Liste okunuyor…")
                scope.launch {
                    runCatching { Core.liveListAdd(name.trim(), u, if (u.isNotEmpty()) "" else fileText) }
                        .onSuccess { l ->
                            fileText = ""; fileName = ""; name = ""; url = ""
                            android.widget.Toast.makeText(ctx, "“${l.name}” " + t("listesi eklendi") + " · ${l.channels.size} " + t("kanal"), android.widget.Toast.LENGTH_SHORT).show()
                            onChanged(reload(), listGroupName(l))
                        }
                        .onFailure { e -> status = e.message ?: t("Liste eklenemedi.") }
                    busy = false
                }
            }) { Text(t("Listeyi ekle")) }
        },
        dismissButton = { TextButton(enabled = !busy, onClick = onDismiss) { Text(t("Kapat")) } }
    )
}

@Suppress("unused")
private fun candJson(c: LiveCandidate) = NexJson.encodeToString(LiveCandidate.serializer(), c)
