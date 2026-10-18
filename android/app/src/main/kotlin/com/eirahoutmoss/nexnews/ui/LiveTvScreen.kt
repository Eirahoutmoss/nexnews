@file:OptIn(ExperimentalMaterial3Api::class)

package com.eirahoutmoss.nexnews.ui

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

    LaunchedEffect(Unit) {
        base = runCatching { Core.liveTV() }.getOrDefault(emptyList())
        state = Core.liveState()
        health.putAll(Core.liveHealth())
        if (state.favs.isNotEmpty()) group = G_FAV
        val now = System.currentTimeMillis() / 1000
        val stale = base.orEmpty().count { c -> health[c.id]?.let { h -> now - h.ts > (if (h.ok) 7 else 3) * 86400 } ?: true }
        if (stale > 0 && state.check == "ask") askCheck = true
        else if (stale > 0 && state.check == "auto" && !isMetered(ctx)) autoStart = true
    }
    val l0 = base ?: run { Loading("Kanallar yükleniyor…"); return }
    val custom = state.custom.map { it.copy(custom = true, continent = "Kendi kanallarım") }
    val all = l0 + custom
    val visible = all.filter { it.id !in state.hidden }

    fun persist(s: LiveState) { state = s; scope.launch { Core.saveLiveState(s) } }

    fun runCheck(list: List<TvChannel>) {
        if (checkJob?.isActive == true) { checkJob?.cancel(); progress = ""; return }
        checkJob = scope.launch {
            var done = 0; var ok = 0
            val sem = Semaphore(3)
            progress = "Denetleniyor… 0/${list.size}"
            list.map { c ->
                async {
                    sem.withPermit {
                        val r = Core.probeLive(c)
                        health[c.id] = r
                        done++; if (r.ok) ok++
                        progress = "Denetleniyor… $done/${list.size} ($ok çalışıyor)"
                    }
                }
            }.awaitAll()
            progress = "Denetim bitti: $ok/$done kanal çalışıyor"
        }
    }

    LaunchedEffect(autoStart) { if (autoStart) { autoStart = false; runCheck(visible) } }

    val groups = buildList {
        add(G_FAV to state.favs.mapNotNull { id -> visible.find { it.id == id } })
        add(G_MINE to visible.filter { it.custom })
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
                groups.forEach { (g, items) -> FilterChip(selected = group == g, onClick = { group = g }, label = { Text("$g ${items.size}") }) }
            }
            Row(Modifier.horizontalScroll(rememberScrollState()).padding(horizontal = 12.dp), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                LIVE_KINDS.forEach { (k, label) ->
                    val n = if (k == "all") current.size else current.count { it.kindOrGuess == k }
                    FilterChip(selected = kind == k, onClick = { kind = k }, label = { Text("$label $n") }, enabled = n > 0 || k == "all")
                }
            }
            OutlinedTextField(query, { query = it }, singleLine = true, placeholder = { Text("Kanal ya da ülke ara…") },
                modifier = Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 6.dp))
            Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp), verticalAlignment = Alignment.CenterVertically) {
                Switch(checked = onlyOk, onCheckedChange = { onlyOk = it })
                Spacer(Modifier.width(8.dp))
                Text("Yalnızca izlenebilenler", style = MaterialTheme.typography.bodySmall, modifier = Modifier.weight(1f))
                TextButton(onClick = { runCheck(current) }) { Text(if (checkJob?.isActive == true) "Durdur" else "Denetle") }
                IconButton(onClick = { addOpen = true }) { Icon(Icons.Filled.Add, "Kanal ekle") }
            }
            val info = progress.ifBlank { if (onlyOk && badCount > 0) "$badCount kanal bu bağlantıdan izlenemediği için gizlendi" else "" }
            if (info.isNotBlank()) Text(info, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary, modifier = Modifier.padding(horizontal = 16.dp, vertical = 2.dp))
        }
        if (shown.isEmpty()) item {
            Text(if (group == G_FAV) "Henüz favori kanal yok — bir kanalın ☆ işaretine dokun." else if (group == G_MINE) "Kendi kanalını + ile ekleyebilirsin." else "Bu filtreye uyan kanal yok.",
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
                    Icon(if (fav) Icons.Filled.Star else Icons.Filled.StarBorder, "Favori", tint = if (fav) Color(0xFFF5B301) else MaterialTheme.colorScheme.onSurfaceVariant)
                }
                Box {
                    IconButton(onClick = { menu = true }) { Icon(Icons.Filled.MoreVert, "Seçenekler") }
                    DropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                        when {
                            group == G_HIDDEN -> DropdownMenuItem(text = { Text("Listeye geri al") }, onClick = { menu = false; persist(state.copy(hidden = state.hidden - ch.id)) })
                            ch.custom -> DropdownMenuItem(text = { Text("Kanalı sil") }, onClick = {
                                menu = false; persist(state.copy(custom = state.custom.filterNot { it.id == ch.id }, favs = state.favs - ch.id))
                            })
                            else -> DropdownMenuItem(text = { Text("Listeden kaldır") }, onClick = {
                                menu = false; persist(state.copy(hidden = state.hidden + ch.id, favs = state.favs - ch.id))
                            })
                        }
                        DropdownMenuItem(text = { Text("Bu kanalı denetle") }, onClick = { menu = false; scope.launch { health[ch.id] = Core.probeLive(ch) } })
                    }
                }
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f), modifier = Modifier.padding(start = 37.dp))
        }
        item {
            Text("Yalnızca yayıncıların kendi ya da lisanslı ücretsiz platformların yayınları oynatılır. ☆ favori · ⋮ listeden kaldır · + kendi kanalını ekle. Ekrandan çıkınca ses arka planda sürer.",
                style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp))
        }
    }

    if (askCheck) {
        var keep by remember { mutableStateOf(false) }
        val metered = isMetered(ctx)
        AlertDialog(
            onDismissRequest = { askCheck = false },
            title = { Text("Kanal listesini doğrulayalım mı?") },
            text = {
                Column {
                    Text("NexNews kanalları senin bağlantından kısaca deneyip izlenemeyenleri (kapanmış ya da yalnızca kendi ülkesinde açık yayınları) gizleyebilir. Tüm liste için birkaç dakika ve birkaç MB veri gerekir; bu sırada uygulamayı kullanabilirsin.")
                    if (metered) Text("Şu an mobil verideysin.", color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.labelSmall, modifier = Modifier.padding(top = 6.dp))
                    Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 8.dp)) {
                        Checkbox(checked = keep, onCheckedChange = { keep = it }); Text("Seçimimi hatırla", style = MaterialTheme.typography.bodySmall)
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    askCheck = false
                    if (keep) persist(state.copy(check = "auto"))
                    runCheck(visible)
                }) { Text("Tümünü doğrula") }
            },
            dismissButton = {
                Row {
                    TextButton(onClick = { askCheck = false; if (keep) persist(state.copy(check = "never")) }) { Text("Doğrulama") }
                    TextButton(onClick = { askCheck = false; runCheck(current) }) { Text("Bu bölge") }
                }
            }
        )
    }

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
        title = { Text("Kanal ekle") },
        text = {
            Column {
                OutlinedTextField(name, { name = it }, label = { Text("Kanal adı") }, singleLine = true)
                OutlinedTextField(url, { url = it }, label = { Text("Yayın adresi (.m3u8)") }, singleLine = true)
                OutlinedTextField(ref, { ref = it }, label = { Text("Referer (gerekirse)") }, singleLine = true)
                Row(Modifier.horizontalScroll(rememberScrollState()).padding(top = 6.dp), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    LIVE_KINDS.drop(1).forEach { (k, l) -> FilterChip(selected = kind == k, onClick = { kind = k }, label = { Text(l) }) }
                }
                Text("Yalnızca yayıncısının herkese açık sunduğu yayınları ekle. Adres önce denenir; çalışmazsa kaydedilmez.",
                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 6.dp))
                if (status.isNotBlank()) Text(status, color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(top = 6.dp))
            }
        },
        confirmButton = {
            TextButton(enabled = !busy, onClick = {
                val u = url.trim()
                if (name.isBlank() || !(u.startsWith("http://") || u.startsWith("https://"))) { status = "Ad ve http(s) ile başlayan bir adres gerekli."; return@TextButton }
                busy = true; status = "Deneniyor…"
                scope.launch {
                    val cand = if (ref.isBlank()) JsonPrimitive(u) else buildJsonObject { put("u", u); put("r", ref.trim()) }
                    val label = LIVE_KINDS.firstOrNull { it.first == kind }?.second ?: "Genel"
                    val ch = TvChannel(id = "my-" + System.currentTimeMillis().toString(36), name = name.trim().take(60), meta = "Kendi kanalım · $label",
                        country = "Kendi kanalım", continent = "Kendi kanallarım", kind = kind, urls = listOf(cand), custom = true)
                    val r = Core.probeLive(ch)
                    busy = false
                    if (r.ok) onAdded(ch) else status = "Adres çalışmadı: " + r.reason.ifBlank { "yayın açılamadı" }
                }
            }) { Text(if (busy) "Deneniyor…" else "Dene ve ekle") }
        },
        dismissButton = { TextButton(enabled = !busy, onClick = onDismiss) { Text("Vazgeç") } }
    )
}

/** Akıllı uyarılar: açıldığında toplanır, dokununca olay sayfası açılır. */
@Composable
fun SmartAlertsTab(nav: NexNav, bp: PaddingValues) {
    var board by remember { mutableStateOf<com.eirahoutmoss.nexnews.SmartAlertBoard?>(null) }
    LaunchedEffect(Unit) { board = runCatching { Core.smartAlerts() }.getOrDefault(com.eirahoutmoss.nexnews.SmartAlertBoard()) }
    val b = board ?: run { Loading("Uyarılar toplanıyor…"); return }
    LazyColumn(contentPadding = bp) {
        item {
            Row(Modifier.padding(16.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Pill("${b.critical} kritik", color = Color(0xFFEF4444))
                Pill("${b.warning} uyarı", color = Color(0xFFE0A83A))
                Pill("${b.info} bilgi")
            }
        }
        if (b.alerts.isEmpty()) item {
            Text("Şu anda dikkat gerektiren bir uyarı yok. Bir olay son dakika olduğunda, hızla tırmandığında ya da en az 3 bağımsız kaynakça doğrulandığında burada görünür.",
                style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp))
        }
        items(b.alerts, key = { it.id }) { a ->
            val c = when (a.severity) { "critical" -> Color(0xFFEF4444); "warning" -> Color(0xFFE0A83A); else -> MaterialTheme.colorScheme.primary }
            Column(Modifier.fillMaxWidth().clickable { if (a.link.isNotBlank()) nav.story(a.link) }.padding(horizontal = 16.dp, vertical = 10.dp)) {
                Text(a.title, color = c, style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.Bold)
                Text(a.eventTitle, fontWeight = FontWeight.SemiBold)
                Text(a.message + " · önem ${a.score} · güven ${a.confidence} · ${a.sources} bağımsız kaynak",
                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f))
        }
    }
}

@Suppress("unused")
private fun candJson(c: LiveCandidate) = NexJson.encodeToString(LiveCandidate.serializer(), c)
