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
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.I18n
import com.eirahoutmoss.nexnews.WeeklySummary
import com.eirahoutmoss.nexnews.t
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/** Haftanın özeti (masaüstüyle aynı): son 7 günün önemli olayları, bölgeler, takip edilenler. */
@Composable
fun WeeklyScreen(nav: NexNav, bottomPad: PaddingValues) {
    var w by remember { mutableStateOf<WeeklySummary?>(null) }
    LaunchedEffect(Unit) { w = runCatching { Core.weekly() }.getOrDefault(WeeklySummary()) }
    val fmt = remember(I18n.lang) { SimpleDateFormat("d MMMM", Locale.forLanguageTag(I18n.lang)) }
    fun day(ts: Long) = if (ts > 0) fmt.format(Date(ts * 1000)) else ""
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(t("📰 Haftanın özeti"), fontWeight = FontWeight.Bold) },
                navigationIcon = { IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, t("Geri")) } },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { pad ->
        val s = w
        if (s == null) { Box(Modifier.padding(pad)) { Loading() }; return@Scaffold }
        LazyColumn(
            Modifier.fillMaxSize().padding(top = pad.calculateTopPadding()),
            contentPadding = PaddingValues(bottom = bottomPad.calculateBottomPadding() + 32.dp)
        ) {
            item {
                if (s.from > 0) Text(day(s.from) + " – " + day(s.to), style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 16.dp))
                if (s.days < 7) Text(t("Bu özet NexNews'in tuttuğu") + " ${maxOf(1, s.days)} " + t("günlük kayda dayanıyor; tam hafta için birkaç gün daha kullanman gerekiyor."),
                    style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 6.dp))
                Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Stat("${s.total}", t("önemli olay"), Modifier.weight(1f))
                    Stat("${s.breaking}", t("son dakika"), Modifier.weight(1f))
                    Stat("${s.regions.size}", t("bölge"), Modifier.weight(1f))
                    Stat("${s.follows.size}", t("takip ettiğin olay"), Modifier.weight(1f))
                }
                SectionHeader(t("⭐ Haftanın en önemli olayları"))
            }
            if (s.top.isEmpty()) item {
                Text(t("Henüz kayıtlı önemli olay yok."), color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(16.dp))
            }
            itemsIndexed(s.top, key = { i, e -> "w$i" + e.link }) { i, e ->
                Row(Modifier.fillMaxWidth().clickable { if (e.link.isNotBlank()) nav.story(e.link) }.padding(horizontal = 16.dp, vertical = 9.dp)) {
                    Text("${i + 1}", fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.primary, modifier = Modifier.width(28.dp))
                    Column(Modifier.weight(1f)) {
                        Text(e.title, fontWeight = FontWeight.SemiBold, style = MaterialTheme.typography.bodyMedium)
                        Text(listOf(e.category, day(e.first), if (e.sources > 1) "${e.sources} " + t("bağımsız kaynak") else "")
                            .filter { it.isNotBlank() }.joinToString(" · "),
                            style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f), modifier = Modifier.padding(start = 44.dp))
            }
            if (s.regions.isNotEmpty()) {
                item { SectionHeader(t("🌍 Gündemin bölgelere dağılımı")) }
                val mx = s.regions.maxOf { it.count }.coerceAtLeast(1)
                items(s.regions, key = { "r" + it.name }) { r ->
                    Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text(t(r.name), style = MaterialTheme.typography.bodySmall, modifier = Modifier.width(110.dp), maxLines = 1)
                        Box(Modifier.weight(1f).height(10.dp)) {
                            Box(Modifier.fillMaxWidth(maxOf(0.06f, r.count.toFloat() / mx)).fillMaxHeight()
                                .clip(RoundedCornerShape(5.dp)).background(MaterialTheme.colorScheme.primary.copy(alpha = 0.7f)))
                        }
                        Text("${r.count}", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold, modifier = Modifier.padding(start = 8.dp))
                    }
                }
                item {
                    Text(t("Az görünen bölgeler bir kör nokta olabilir: bu hafta oradan pek haber okumadın."),
                        style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 6.dp))
                }
            }
            if (s.follows.isNotEmpty()) {
                item { SectionHeader(t("🔔 Takip ettiğin olaylar")) }
                items(s.follows, key = { "f" + it.link }) { f ->
                    Column(Modifier.fillMaxWidth().clickable { if (f.link.isNotBlank()) nav.story(f.link) }.padding(horizontal = 16.dp, vertical = 9.dp)) {
                        Text(f.title, fontWeight = FontWeight.SemiBold, style = MaterialTheme.typography.bodyMedium)
                        Text("${f.count} " + t("haber") + " · ${f.sources} " + t("bağımsız kaynak") + if (f.new > 0) " · +${f.new} " + t("yeni") else "",
                            style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
            item {
                Column(
                    Modifier.fillMaxWidth().padding(16.dp).clip(RoundedCornerShape(16.dp))
                        .background(MaterialTheme.colorScheme.surface)
                        .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(16.dp)).padding(22.dp),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Icon(Icons.Filled.CheckCircle, null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(30.dp))
                    Spacer(Modifier.height(8.dp))
                    Text(t("Haftanın özeti bu kadar"), fontWeight = FontWeight.Bold, textAlign = TextAlign.Center)
                    Text(t("Gelecek haftanın özeti pazar günü hazır olur."), color = MaterialTheme.colorScheme.onSurfaceVariant,
                        textAlign = TextAlign.Center, style = MaterialTheme.typography.bodyMedium)
                }
            }
        }
    }
}

@Composable
private fun Stat(value: String, label: String, modifier: Modifier = Modifier) {
    Column(
        modifier.clip(RoundedCornerShape(12.dp)).background(MaterialTheme.colorScheme.surface)
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(12.dp)).padding(vertical = 10.dp, horizontal = 4.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(value, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium)
        Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center, maxLines = 2)
    }
}
