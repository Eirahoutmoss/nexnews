@file:OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)

package com.eirahoutmoss.nexnews.ui

import android.content.Intent
import android.os.Build
import android.provider.Settings
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.background
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.ExpandLess
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Slider
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.eirahoutmoss.nexnews.BuildConfig
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.NexJson
import com.eirahoutmoss.nexnews.Prefs
import com.eirahoutmoss.nexnews.SourceHealth
import com.eirahoutmoss.nexnews.widget.NewsWidget
import com.eirahoutmoss.nexnews.work.Scheduler
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonPrimitive

private fun JsonElement?.str(def: String = "") = (this as? JsonPrimitive)?.content ?: def
private fun JsonElement?.bool(def: Boolean) = (this as? JsonPrimitive)?.booleanOrNull ?: def
private fun JsonElement?.int(def: Int) = (this as? JsonPrimitive)?.intOrNull ?: def
private fun JsonElement?.strList(): List<String> = runCatching { this!!.jsonArray.map { it.jsonPrimitive.content } }.getOrDefault(emptyList())
private fun strArray(l: List<String>) = JsonArray(l.map { JsonPrimitive(it) })
private fun csv(s: String) = s.split(',').map { it.trim() }.filter { it.isNotEmpty() }

@Composable
private fun Group(title: String) {
    Text(title.uppercase(java.util.Locale("tr")), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary,
        fontWeight = FontWeight.Bold, modifier = Modifier.padding(start = 16.dp, top = 22.dp, bottom = 4.dp))
}

@Composable
private fun SwitchRow(title: String, sub: String = "", checked: Boolean, onChange: (Boolean) -> Unit) {
    Row(Modifier.fillMaxWidth().clickable { onChange(!checked) }.padding(horizontal = 16.dp, vertical = 10.dp), verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) {
            Text(title)
            if (sub.isNotBlank()) Text(sub, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Switch(checked = checked, onCheckedChange = onChange)
    }
}

@Composable
private fun ChoiceRow(title: String, options: List<Pair<String, String>>, selected: String, onSelect: (String) -> Unit) {
    Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp)) {
        Text(title)
        Spacer(Modifier.height(6.dp))
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            options.forEach { (v, label) -> FilterChip(selected = selected == v, onClick = { onSelect(v) }, label = { Text(label) }) }
        }
    }
}

@Composable
private fun NavRow(title: String, sub: String = "", onClick: () -> Unit) {
    Row(Modifier.fillMaxWidth().clickable(onClick = onClick).padding(horizontal = 16.dp, vertical = 14.dp), verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f)) {
            Text(title)
            if (sub.isNotBlank()) Text(sub, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
private fun TextSetting(title: String, value: String, hint: String, onSave: (String) -> Unit) {
    var open by remember { mutableStateOf(false) }
    var v by remember(value) { mutableStateOf(value) }
    NavRow(title, value.ifBlank { hint }) { open = true }
    if (open) AlertDialog(
        onDismissRequest = { open = false },
        title = { Text(title) },
        text = { OutlinedTextField(v, { v = it }, placeholder = { Text(hint) }) },
        confirmButton = { TextButton(onClick = { onSave(v); open = false }) { Text("Kaydet") } },
        dismissButton = { TextButton(onClick = { open = false }) { Text("Vazgeç") } }
    )
}

@Composable
fun SettingsScreen(nav: NexNav, bottomPad: PaddingValues) {
    val ctx = LocalContext.current
    val prefs = Prefs.get(ctx)
    val pv by prefs.version.collectAsState()
    val scope = rememberCoroutineScope()
    var cfgV by remember { mutableIntStateOf(0) }
    val cfg = remember(cfgV) { Core.config() }
    fun save(edit: (MutableMap<String, JsonElement>) -> Unit) = scope.launch { Core.saveConfig(edit); cfgV++ }
    @Suppress("UNUSED_VARIABLE") val unused = pv

    var net by remember { mutableStateOf("") }
    var usage by remember { mutableStateOf<com.eirahoutmoss.nexnews.UsageStats?>(null) }
    var usageV by remember { mutableIntStateOf(0) }
    LaunchedEffect(usageV) { usage = runCatching { Core.usage() }.getOrNull() }
    LaunchedEffect(Unit) {
        val r = NetMeter.sample()
        net = "Bu oturumda NexNews ${fmtBytes(r.appSession)} indirdi · son 5 dk ${fmtBytes(r.app5m)}"
    }

    Scaffold(
        topBar = { TopAppBar(title = { Text("Ayarlar", fontWeight = FontWeight.Bold) }, colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)) },
        containerColor = MaterialTheme.colorScheme.background
    ) { pad ->
        LazyColumn(Modifier.fillMaxSize().padding(top = pad.calculateTopPadding()), contentPadding = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 32.dp)) {
            item {
                Group("Görünüm")
                ChoiceRow("Tema", listOf("sistem" to "Sistem", "koyu" to "Koyu", "acik" to "Açık"), prefs.theme) { prefs.theme = it }
                ChoiceRow("Liste", listOf("siki" to "Sıkı", "rahat" to "Rahat (görselli)"), prefs.density) { prefs.density = it }
                SwitchRow("Görselleri yalnızca Wi-Fi'de yükle", "Mobil veride küçük resimler indirilmez", prefs.imagesWifiOnly) { prefs.imagesWifiOnly = it }
            }
            item {
                Group("Yenileme ve veri")
                ChoiceRow("Arka planda yenile", listOf("30" to "30 dk", "60" to "1 saat", "120" to "2 saat", "240" to "4 saat"), prefs.refreshMinutes.toString()) {
                    prefs.refreshMinutes = it.toInt(); Scheduler.apply(ctx)
                }
                SwitchRow("Yalnızca Wi-Fi'de yenile", "Arka plan yenilemesi mobil veride çalışmaz", prefs.wifiOnly) { prefs.wifiOnly = it; Scheduler.apply(ctx) }
                Text(net, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 16.dp, vertical = 6.dp))
            }
            item {
                Group("Veri akışı")
                val u = usage
                val mode = u?.mode ?: "tam"
                ChoiceRow("Mod", listOf("tam" to "⚡ Tam veri", "sinirli" to "🐢 Sınırlı veri"), mode) { m ->
                    scope.launch { Core.setDataMode(m, u?.limitKBps ?: 256, u?.quotaMB ?: 0); usageV++ }
                }
                Text("Sınırlı modda NexNews'in indirme hızı sınırlanır, canlı TV en çok 480p oynar, otomatik yenileme seyrekleşir.",
                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 16.dp))
                ChoiceRow("Sınırlı modda en yüksek hız", listOf("128" to "128 KB/s", "256" to "256 KB/s", "512" to "512 KB/s", "1024" to "1 MB/s"), (u?.limitKBps ?: 256).toString()) { v ->
                    scope.launch { Core.setDataMode(mode, v.toInt(), u?.quotaMB ?: 0); usageV++ }
                }
                TextSetting("Aylık kota (MB)", if ((u?.quotaMB ?: 0) > 0) "${u?.quotaMB} MB" else "", "0 = yok · dolunca sınırlı moda geçer") { v ->
                    scope.launch { Core.setDataMode(mode, u?.limitKBps ?: 256, v.filter { it.isDigit() }.toIntOrNull() ?: 0); usageV++ }
                }
                if (u != null) {
                    Column(Modifier.padding(horizontal = 16.dp, vertical = 6.dp)) {
                        fun line(label: String, p: com.eirahoutmoss.nexnews.UsagePeriod) = "$label: ⇣ ${fmtBytes(p.down)}  ⇡ ${fmtBytes(p.up)}" + if (p.secs > 0) "  · ${p.secs / 3600} sa ${(p.secs % 3600) / 60} dk açık" else ""
                        listOf("Bu oturum" to u.session, "Bugün" to u.today, "Bu hafta" to u.week, "Bu ay" to u.month, "Son 30 gün" to u.last30, "Toplam" to u.all)
                            .forEach { (k, v) -> Text(line(k, v), style = MaterialTheme.typography.bodySmall) }
                        if (u.quotaMB > 0) Text("Bu ay kotanın %${(u.quotaUsed * 100).toInt()} kısmı kullanıldı" + if (u.autoLimited) " · kota dolduğu için sınırlı moda geçildi" else "",
                            style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
                        TextButton(onClick = { scope.launch { Core.resetUsage(); usageV++ } }) { Text("Sayaçları sıfırla") }
                    }
                }
            }
            item {
                Group("Haberler")
                TextSetting("İlgi alanların", cfg["interests"].strList().joinToString(", "), "yapay zeka, deprem, uzay") { v -> save { it["interests"] = strArray(csv(v)) } }
                TextSetting("Gizlenecek konular", cfg["hideWords"].strList().joinToString(", "), "magazin, transfer") { v -> save { it["hideWords"] = strArray(csv(v)) } }
                SwitchRow("Başlıkları cihaz diline çevir", "Farklı dillerdeki aynı haberler de birleşir", cfg["translate"].bool(true)) { b -> save { it["translate"] = JsonPrimitive(b) } }
                SwitchRow("Haberleri uygulama içinde aç", "Kapalıysa tek kaynaklı haberler tarayıcıda açılır", cfg["openInApp"].bool(true)) { b -> save { it["openInApp"] = JsonPrimitive(b) } }
                ChoiceRow("Kategori başına başlık", listOf("3" to "3", "5" to "5", "8" to "8", "12" to "12"), cfg["maxPerCategory"].int(5).toString()) { v -> save { it["maxPerCategory"] = JsonPrimitive(v.toInt()) } }
                NavRow("Kaynaklar", "Kategoriler, feed'ler ve kaynak sağlığı") { nav.sources() }
            }
            item {
                Group("Bildirimler")
                SwitchRow("Önemli gelişmeleri bildir", "Son dakika ve önem eşiğini geçen olaylar", cfg["notify"].bool(true)) { b -> save { it["notify"] = JsonPrimitive(b) } }
                var thr by remember(cfgV) { mutableFloatStateOf(cfg["notifyMinScore"].int(75).toFloat()) }
                Column(Modifier.padding(horizontal = 16.dp)) {
                    Text("Önem eşiği: ${thr.toInt()}", style = MaterialTheme.typography.bodyMedium)
                    Slider(value = thr, onValueChange = { thr = it }, valueRange = 40f..100f, onValueChangeFinished = { save { it["notifyMinScore"] = JsonPrimitive(thr.toInt()) } })
                }
                SwitchRow("Sessiz saatler", "${cfg["quietFrom"].str("23:00")} – ${cfg["quietTo"].str("07:00")} arası bildirim yok", cfg["quietHours"].bool(false)) { b -> save { it["quietHours"] = JsonPrimitive(b) } }
                ChoiceRow("Sessizlik başlangıcı", listOf("21:00", "22:00", "23:00", "00:00").map { it to it }, cfg["quietFrom"].str("23:00")) { v -> save { it["quietFrom"] = JsonPrimitive(v) } }
                ChoiceRow("Sessizlik bitişi", listOf("06:00", "07:00", "08:00", "09:00").map { it to it }, cfg["quietTo"].str("07:00")) { v -> save { it["quietTo"] = JsonPrimitive(v) } }
                SwitchRow("Sabah ve akşam özeti", "${prefs.digestMorning}:00 ve ${prefs.digestEvening}:00 civarı", prefs.digestOn) { prefs.digestOn = it; Scheduler.apply(ctx) }
                if (Build.VERSION.SDK_INT >= 26) NavRow("Bildirim kanalları", "Son dakika / önemli / özet ayrı ayrı") {
                    ctx.startActivity(Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).putExtra(Settings.EXTRA_APP_PACKAGE, ctx.packageName).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
                }
            }
            item {
                Group("Widget")
                var wo by remember { mutableFloatStateOf(prefs.widgetOpacity.toFloat()) }
                Column(Modifier.padding(horizontal = 16.dp)) {
                    Text("Zemin opaklığı: %${wo.toInt()}" + if (wo.toInt() == 0) " (saydam)" else "", style = MaterialTheme.typography.bodyMedium)
                    Slider(value = wo, onValueChange = { wo = it }, valueRange = 0f..100f, onValueChangeFinished = {
                        prefs.widgetOpacity = wo.toInt(); scope.launch { runCatching { NewsWidget.refreshAll(ctx) } }
                    })
                    Text("Ana ekrana eklemek için: ana ekranda boş bir yere basılı tut → Widget'lar → NexNews.", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
            item {
                Group("Hakkında")
                Text("NexNews ${BuildConfig.VERSION_NAME} · çekirdek ${Core.coreVersion()}", modifier = Modifier.padding(horizontal = 16.dp, vertical = 6.dp))
                Text("Dünyanın nabzı, tek akışta. Haberleri, olay takibini ve dünyanın canlı yayınlarını tek pencerede, kullanıcıyı yormadan bir araya getiren, dil ve kaynak bağımsız haber radarı.",
                    style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp))
                Text("Geliştirici: Hasan Güler", style = MaterialTheme.typography.labelSmall, modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp))
                NavRow("YouTube", "@bakis_acisi") { openInBrowser(ctx, "https://www.youtube.com/@bakis_acisi") }
                NavRow("Instagram", "@elem_ya") { openInBrowser(ctx, "https://www.instagram.com/elem_ya") }
                NavRow("GitHub", "github.com/Eirahoutmoss") { openInBrowser(ctx, "https://github.com/Eirahoutmoss") }
                Text("Haber başlıkları, içerikler ve yayınlar ilgili kaynaklara aittir; NexNews yalnızca derler, bağlantı verir ve yayıncıların herkese açık akışlarını oynatır.",
                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp))
            }
        }
    }
}

@Serializable
private data class Src(val name: String = "", val url: String = "")

@Serializable
private data class Cat(val name: String = "", val sources: List<Src> = emptyList())

/** Kaynak yönetimi: kategoriler, feed'ler, sağlık durumu, siteden feed bulma. */
@Composable
fun SourcesScreen(nav: NexNav, bottomPad: PaddingValues) {
    val scope = rememberCoroutineScope()
    var cats by remember { mutableStateOf<List<Cat>>(emptyList()) }
    var health by remember { mutableStateOf<Map<String, SourceHealth>>(emptyMap()) }
    var open by remember { mutableStateOf<String?>(null) }
    var addTo by remember { mutableStateOf<String?>(null) }
    var info by remember { mutableStateOf("") }
    fun norm(u: String) = u.trim().lowercase().removePrefix("https://").removePrefix("http://").removePrefix("www.").trimEnd('/')
    fun reload() {
        cats = runCatching { NexJson.decodeFromJsonElement<List<Cat>>(Core.config()["categories"] ?: JsonArray(emptyList())) }.getOrDefault(emptyList())
    }
    fun persist(newCats: List<Cat>) = scope.launch {
        Core.saveConfig { it["categories"] = NexJson.encodeToJsonElement(newCats) }
        reload()
    }
    LaunchedEffect(Unit) {
        reload()
        health = runCatching { Core.sourceHealth() }.getOrDefault(emptyList()).associateBy { norm(it.url) }
    }
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Kaynaklar", fontWeight = FontWeight.Bold) },
                navigationIcon = { IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Geri") } },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { pad ->
        LazyColumn(Modifier.fillMaxSize().padding(top = pad.calculateTopPadding()), contentPadding = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 32.dp)) {
            item {
                val total = cats.sumOf { it.sources.size }
                Text("$total kaynak · telefonda mobil veriyi korumak için her kategoride birkaç kaynakla başlanır.",
                    style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp))
                Row(Modifier.padding(horizontal = 8.dp)) {
                    TextButton(onClick = { scope.launch { val n = Core.restoreAllSources(); info = "$n kaynak eklendi"; reload() } }) { Text("Tüm hazır kaynakları ekle") }
                }
                if (info.isNotBlank()) Text(info, color = MaterialTheme.colorScheme.primary, modifier = Modifier.padding(horizontal = 16.dp))
                Row(Modifier.padding(horizontal = 16.dp, vertical = 6.dp), horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                    listOf(Color(0xFF38C172) to "çalışıyor", Color(0xFFE0A83A) to "yavaş/eski", Color(0xFFE2566A) to "hata", MaterialTheme.colorScheme.outline to "denenmedi").forEach { (c, l) ->
                        Row(verticalAlignment = Alignment.CenterVertically) { Box(Modifier.size(8.dp).clip(CircleShape).background(c)); Spacer(Modifier.width(5.dp)); Text(l, style = MaterialTheme.typography.labelSmall) }
                    }
                }
            }
            cats.forEach { cat ->
                item(key = "cat" + cat.name) {
                    val isOpen = open == cat.name
                    val bad = cat.sources.count { s -> health[norm(s.url)]?.let { it.status != "healthy" } == true }
                    Row(Modifier.fillMaxWidth().clickable { open = if (isOpen) null else cat.name }.padding(horizontal = 16.dp, vertical = 14.dp), verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text(cat.name, fontWeight = FontWeight.SemiBold)
                            Text("${cat.sources.size} kaynak" + if (bad > 0) " · $bad sorunlu" else "", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                        Icon(if (isOpen) Icons.Filled.ExpandLess else Icons.Filled.ExpandMore, null)
                    }
                    if (isOpen) {
                        cat.sources.forEach { src ->
                            val h = health[norm(src.url)]
                            val dot = when (h?.status) { "healthy" -> Color(0xFF38C172); "stale" -> Color(0xFFE0A83A); null -> MaterialTheme.colorScheme.outline; else -> Color(0xFFE2566A) }
                            Row(Modifier.fillMaxWidth().padding(start = 24.dp, end = 4.dp, top = 4.dp, bottom = 4.dp), verticalAlignment = Alignment.CenterVertically) {
                                Box(Modifier.size(8.dp).clip(CircleShape).background(dot))
                                Spacer(Modifier.width(10.dp))
                                Column(Modifier.weight(1f)) {
                                    Text(src.name.ifBlank { src.url }, maxLines = 1)
                                    Text(src.url, maxLines = 1, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                }
                                IconButton(onClick = {
                                    persist(cats.map { c -> if (c.name == cat.name) c.copy(sources = c.sources.filter { it.url != src.url }) else c })
                                }) { Icon(Icons.Filled.Delete, "Kaldır", tint = MaterialTheme.colorScheme.onSurfaceVariant) }
                            }
                        }
                        TextButton(onClick = { addTo = cat.name }, modifier = Modifier.padding(start = 16.dp)) {
                            Icon(Icons.Filled.Add, null); Spacer(Modifier.width(6.dp)); Text("Kaynak ekle")
                        }
                    }
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                }
            }
        }
    }
    addTo?.let { catName ->
        AddSourceDialog(onDismiss = { addTo = null }) { name, url ->
            persist(cats.map { c -> if (c.name == catName && c.sources.none { it.url == url }) c.copy(sources = c.sources + Src(name, url)) else c })
            addTo = null
        }
    }
}

@Composable
private fun AddSourceDialog(onDismiss: () -> Unit, onAdd: (String, String) -> Unit) {
    val scope = rememberCoroutineScope()
    var site by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var msg by remember { mutableStateOf("") }
    var found by remember { mutableStateOf<List<Pair<String, String>>>(emptyList()) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Kaynak ekle") },
        text = {
            Column {
                Text("Site adresi ya da RSS adresi yazın; NexNews feed'i kendisi bulur.", style = MaterialTheme.typography.bodyMedium)
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(site, { site = it }, singleLine = true, placeholder = { Text("ör. ntv.com.tr") })
                if (busy) Text("Aranıyor…", modifier = Modifier.padding(top = 8.dp))
                if (msg.isNotBlank()) Text(msg, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 8.dp))
                found.forEach { (title, url) ->
                    Row(Modifier.fillMaxWidth().clickable { onAdd(title, url) }.padding(vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Filled.Add, null, tint = MaterialTheme.colorScheme.primary)
                        Spacer(Modifier.width(8.dp))
                        Column { Text(title.ifBlank { url }, maxLines = 1); Text(url, maxLines = 1, style = MaterialTheme.typography.labelSmall) }
                    }
                }
            }
        },
        confirmButton = {
            TextButton(enabled = site.isNotBlank() && !busy, onClick = {
                scope.launch {
                    busy = true; msg = ""
                    val r = runCatching { Core.discover(site) }.getOrNull()
                    busy = false
                    found = r?.feeds?.map { it.title to it.url }.orEmpty()
                    if (found.isEmpty()) msg = r?.err?.ifBlank { null } ?: "Feed bulunamadı."
                }
            }) { Text("Bul") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Kapat") } }
    )
}
