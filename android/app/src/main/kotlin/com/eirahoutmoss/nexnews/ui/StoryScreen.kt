@file:OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)

package com.eirahoutmoss.nexnews.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.NotificationsActive
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.ExperimentalMaterial3Api
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
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.Headline
import com.eirahoutmoss.nexnews.ShareCard
import com.eirahoutmoss.nexnews.StoryDetail
import kotlinx.coroutines.launch

fun lifecycleLabel(x: String) = when (x) {
    "detected" -> "Algılandı"; "confirmed" -> "Teyitli"; "developing" -> "Gelişiyor"
    "stabilizing" -> "Durulan"; "archived" -> "Arşiv"; else -> x
}

@Composable
fun StoryScreen(link: String, nav: NexNav, bottomPad: PaddingValues) {
    val ctx = LocalContext.current
    val scope = rememberCoroutineScope()
    var s by remember { mutableStateOf<StoryDetail?>(null) }
    var followed by remember { mutableStateOf(false) }
    LaunchedEffect(link) {
        s = runCatching { Core.story(link) }.getOrNull()
        followed = s?.followed == true
    }
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Olay", fontWeight = FontWeight.Bold) },
                navigationIcon = { IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Geri") } },
                actions = {
                    val st = s
                    if (st != null && st.found) {
                        IconButton(onClick = {
                            scope.launch { followed = Core.toggleFollow(st.eventId) }
                        }) { Icon(if (followed) Icons.Filled.NotificationsActive else Icons.Filled.Notifications, if (followed) "Takibi bırak" else "Olayı takip et") }
                        IconButton(onClick = {
                            if (st.headlines.size >= 2) ShareCard.share(ctx, st) else shareText(ctx, st.title, link)
                        }) { Icon(Icons.Filled.Share, "Paylaş") }
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { pad ->
        val st = s
        if (st == null) { Box(Modifier.padding(pad)) { Loading() }; return@Scaffold }
        if (!st.found) { Box(Modifier.padding(pad)) { Empty("Bu olay artık akışta değil.") }; return@Scaffold }
        LazyColumn(
            Modifier.fillMaxSize().padding(top = pad.calculateTopPadding()),
            contentPadding = PaddingValues(start = 16.dp, end = 16.dp, bottom = bottomPad.calculateBottomPadding() + 32.dp)
        ) {
            item {
                Text(st.title, style = MaterialTheme.typography.titleLarge, lineHeight = 28.sp)
                Spacer(Modifier.height(10.dp))
                FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    if (st.lifecycle.isNotBlank()) Pill(lifecycleLabel(st.lifecycle))
                    Pill("${st.independentSources} bağımsız / ${st.sourceCount} kaynak")
                    Pill("önem ${st.score}")
                    Pill("güven ${st.confidence}")
                    if (st.category.isNotBlank()) Pill(st.category)
                }
                if (st.relText.isNotBlank()) {
                    Spacer(Modifier.height(12.dp))
                    Note(st.relText + if (st.firstTs > 0) " · İlk görülme: ${clock(st.firstTs)}" else "")
                }
                if (st.evidence.summary.isNotBlank()) {
                    Spacer(Modifier.height(8.dp))
                    Note(st.evidence.summary)
                }
                if (st.summary.isNotBlank()) {
                    Spacer(Modifier.height(12.dp))
                    Text(st.summary, style = MaterialTheme.typography.bodyLarge)
                }
            }
            if (st.headlines.size >= 2) {
                item {
                    Spacer(Modifier.height(18.dp))
                    Text("Kim nasıl başlık attı?", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium)
                    Text("Yalnız o kaynağın kullandığı kelimeler işaretli", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Spacer(Modifier.height(8.dp))
                }
                st.headlines.forEach { h -> item(key = "h" + h.link) { HeadlineCard(h) { nav.article(h.link) } } }
            }
            item {
                Spacer(Modifier.height(18.dp))
                Text("Zaman çizelgesi", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium)
                Spacer(Modifier.height(8.dp))
            }
            st.timeline.forEach { t ->
                item(key = "t" + t.link + t.ts) {
                    Row(Modifier.fillMaxWidth().clickable { nav.article(t.link) }.padding(vertical = 8.dp)) {
                        Text(clock(t.ts), modifier = Modifier.width(52.dp), color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.labelSmall.copy(fontSize = 12.sp))
                        Box(Modifier.padding(top = 5.dp, end = 10.dp).size(8.dp).clip(CircleShape).background(MaterialTheme.colorScheme.primary))
                        Column(Modifier.weight(1f)) {
                            Text(t.title, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Medium)
                            Text(t.source, color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun Note(text: String) {
    Text(
        text,
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp))
            .background(MaterialTheme.colorScheme.surface)
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(12.dp))
            .padding(12.dp)
    )
}

@Composable
private fun HeadlineCard(h: Headline, onOpen: () -> Unit) {
    val nex = LocalNex.current
    val shown = h.translated.ifBlank { h.title }
    val text: AnnotatedString = buildAnnotatedString {
        append(shown)
        for (w in h.distinct) {
            val idx = shown.indexOf(w)
            if (idx >= 0) addStyle(SpanStyle(background = nex.mark, fontWeight = FontWeight.SemiBold), idx, idx + w.length)
        }
    }
    Column(
        Modifier.fillMaxWidth().padding(vertical = 5.dp).clip(RoundedCornerShape(14.dp))
            .background(MaterialTheme.colorScheme.surface)
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(14.dp))
            .clickable(onClick = onOpen).padding(12.dp)
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(h.source, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.labelSmall.copy(fontSize = 12.sp))
            if (h.via.isNotBlank()) { Spacer(Modifier.width(6.dp)); Pill(h.via + " metni") }
            Spacer(Modifier.weight(1f))
            Text(clock(h.ts), color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.labelSmall)
        }
        Spacer(Modifier.height(4.dp))
        Text(text, style = MaterialTheme.typography.bodyMedium)
        if (h.translated.isNotBlank()) {
            Spacer(Modifier.height(2.dp))
            Text("Özgün: " + h.title, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

