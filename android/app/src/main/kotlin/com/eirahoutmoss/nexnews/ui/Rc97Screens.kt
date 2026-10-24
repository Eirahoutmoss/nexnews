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
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
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
import androidx.compose.ui.unit.dp
import com.eirahoutmoss.nexnews.AgendaItem
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.I18n
import com.eirahoutmoss.nexnews.MirrorBoard
import com.eirahoutmoss.nexnews.MirrorCount
import com.eirahoutmoss.nexnews.MirrorNote
import com.eirahoutmoss.nexnews.t
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

@Composable
private fun PageScaffold(title: String, nav: NexNav, content: @Composable (PaddingValues) -> Unit) {
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(title, fontWeight = FontWeight.Bold) },
                navigationIcon = { IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, t("Geri")) } },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)
            )
        },
        containerColor = MaterialTheme.colorScheme.background,
        content = content
    )
}

private val AGENDA_IC = mapOf("ekonomi" to "💹", "siyaset" to "🗳️", "spor" to "⚽", "bilim" to "🔭")

/** RC97 gündem takvimi: önümüzdeki 14 günün önceden bilinen olayları. */
@Composable
fun AgendaScreen(nav: NexNav, bottomPad: PaddingValues) {
    val ctx = LocalContext.current
    var list by remember { mutableStateOf<List<AgendaItem>?>(null) }
    LaunchedEffect(Unit) { list = Core.agenda() }
    val loc = remember(I18n.lang) { Locale.forLanguageTag(I18n.lang) }
    val dayFmt = remember(loc) { SimpleDateFormat("EEEE d MMMM", loc) }
    val timeFmt = remember(loc) { SimpleDateFormat("HH:mm", loc) }
    PageScaffold(t("🗓️ Gündem takvimi"), nav) { pad ->
        val l = list
        if (l == null) { Box(Modifier.padding(pad)) { Loading() }; return@PageScaffold }
        LazyColumn(Modifier.fillMaxSize().padding(top = pad.calculateTopPadding()), contentPadding = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 24.dp)) {
            item {
                Text(t("Önceden bilinen, gündemi belirleyecek olaylar. Saatler cihazının saat dilimine göre."),
                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp))
            }
            if (l.isEmpty()) item { Text(t("Önümüzdeki iki haftada kayıtlı bir olay yok."), modifier = Modifier.padding(16.dp)) }
            var lastDay = ""
            l.forEach { a ->
                val d = dayFmt.format(Date(a.ts * 1000))
                if (d != lastDay) {
                    lastDay = d
                    item(key = "d" + a.ts) { SectionHeader(d.replaceFirstChar { it.titlecase(loc) }) }
                }
                item(key = "a" + a.ts + a.title) {
                    Row(Modifier.fillMaxWidth().clickable(enabled = a.link.isNotBlank()) { openInBrowser(ctx, a.link) }.padding(horizontal = 16.dp, vertical = 10.dp)) {
                        Text(if (a.allDay) "—" else timeFmt.format(Date(a.ts * 1000)), fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.primary, modifier = Modifier.width(56.dp))
                        Column(Modifier.weight(1f)) {
                            Text((AGENDA_IC[a.kind] ?: "📌") + " " + t(a.title), fontWeight = FontWeight.SemiBold)
                            if (a.note.isNotBlank()) Text(t(a.note), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f), modifier = Modifier.padding(start = 72.dp))
                }
            }
            item {
                Text(t("Tarihler kurumların yayımladığı resmî takvimlerden derlenir; kurumlar tarihi değiştirebilir."),
                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp))
            }
        }
    }
}

private fun trList(csv: String) = csv.split(",").map { it.trim() }.filter { it.isNotEmpty() }.joinToString(", ") { t(it) }

private fun mirrorNoteText(n: MirrorNote): String = when (n.kind) {
    "az" -> t("Okuma aynası için henüz yeterli okuma yok; birkaç gün kullandıktan sonra burada kendi haber alışkanlığını göreceksin.")
    "tekKaynak" -> tfmt("Okuduklarının %{p} kadarı tek kaynaktan: {n}. Aynı olayı farklı kaynaklardan okumak resmin tamamını görmeye yardımcı olur.", "p" to "${n.pct}", "n" to n.name)
    "devlet" -> tfmt("Sahipliği bilinen okumalarının %{p} kadarı devlet medyası ya da devlet fonlu kaynaklardan.", "p" to "${n.pct}")
    "kategori" -> tfmt("Okumalarının %{p} kadarı tek bir başlıkta: {n}.", "p" to "${n.pct}", "n" to trList(n.name))
    "bolgeYok" -> tfmt("Son 30 günde hiç okumadığın bölgeler: {n}.", "n" to trList(n.name))
    "cesitli" -> t("Kaynak çeşitliliğin yüksek: farklı kaynaklardan okuyorsun.")
    else -> ""
}

/** RC97 okuma aynası: son 30 günün okuma alışkanlığı (yalnızca bu cihazdaki geçmişten). */
@Composable
fun MirrorScreen(nav: NexNav, bottomPad: PaddingValues) {
    var m by remember { mutableStateOf<MirrorBoard?>(null) }
    LaunchedEffect(Unit) { m = Core.mirror() }
    PageScaffold(t("🪞 Okuma aynan"), nav) { pad ->
        val b = m
        if (b == null) { Box(Modifier.padding(pad)) { Loading() }; return@PageScaffold }
        LazyColumn(Modifier.fillMaxSize().padding(top = pad.calculateTopPadding()), contentPadding = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 24.dp)) {
            item {
                Text(t("Bu sayfa yalnızca bu cihazdaki okuma geçmişinden hesaplanır; hiçbir yere gönderilmez. Amaç yargılamak değil, kendi haber alışkanlığını görmek."),
                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp))
                Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    MirrorStat("${b.total}", t("okunan haber"), Modifier.weight(1f))
                    MirrorStat("${b.sourceCount}", t("farklı kaynak"), Modifier.weight(1f))
                    MirrorStat("${b.diversity}", t("çeşitlilik (0–100)"), Modifier.weight(1f))
                }
            }
            items(b.notes, key = { "n" + it.kind }) { n ->
                val txt = mirrorNoteText(n)
                if (txt.isNotBlank()) Text((if (n.kind == "cesitli") "🌈 " else "💡 ") + txt, style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp).clip(RoundedCornerShape(12.dp))
                        .background(MaterialTheme.colorScheme.surface).border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(12.dp)).padding(12.dp))
            }
            if (b.total > 0) {
                item { SectionHeader(t("📰 En çok okuduğun kaynaklar")) }
                items(b.sources, key = { "s" + it.name }) { MirrorBar(it.name, it) }
                item { SectionHeader(t("🌍 Bölgeler")) }
                items(b.regions, key = { "r" + it.name }) { MirrorBar(t(it.name), it) }
                if (b.silentRegions.isNotEmpty()) item {
                    Text(tfmt("Hiç okumadığın bölgeler: {n}", "n" to trList(b.silentRegions.joinToString(","))), style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp))
                }
                item { SectionHeader(t("🏛️ Kaynak sahipliği")) }
                items(b.owners, key = { "o" + it.name }) { MirrorBar(t(it.label.ifBlank { it.name }), it) }
                if (b.ownerUnknown > 0) item {
                    Text(tfmt("{n} okumanın kaynağının sahipliği katalogda yok.", "n" to "${b.ownerUnknown}"), style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp))
                }
                item { SectionHeader(t("🗂️ Başlıklar")) }
                items(b.cats.take(8), key = { "c" + it.name }) { MirrorBar(t(it.name), it) }
            }
        }
    }
}

@Composable
private fun MirrorStat(value: String, label: String, modifier: Modifier = Modifier) {
    Column(
        modifier.clip(RoundedCornerShape(12.dp)).background(MaterialTheme.colorScheme.surface)
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(12.dp)).padding(vertical = 10.dp, horizontal = 4.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(value, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium)
        Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center, maxLines = 2)
    }
}

@Composable
private fun MirrorBar(label: String, c: MirrorCount) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
        Text(label, style = MaterialTheme.typography.bodySmall, modifier = Modifier.width(120.dp), maxLines = 1)
        Box(Modifier.weight(1f).height(10.dp)) {
            Box(Modifier.fillMaxWidth(maxOf(0.04f, c.pct / 100f)).fillMaxHeight().clip(RoundedCornerShape(5.dp))
                .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.7f)))
        }
        Text(if (I18n.lang == "tr") "%${c.pct}" else "${c.pct}%", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold, modifier = Modifier.padding(start = 8.dp))
    }
}
