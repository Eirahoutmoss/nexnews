@file:OptIn(ExperimentalMaterial3Api::class)

package com.eirahoutmoss.nexnews.ui

import androidx.annotation.OptIn as AndroidOptIn
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Radio
import androidx.compose.material.icons.filled.Tv
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.media3.common.util.UnstableApi
import androidx.media3.ui.PlayerView
import com.eirahoutmoss.nexnews.media.PlayerHub

/** Tam ekran video. Ekrandan çıkınca yayın kapanmaz; alttaki mini çubukta sürer. */
@AndroidOptIn(UnstableApi::class)
@Composable
fun PlayerScreen(nav: NexNav) {
    val now by PlayerHub.now.collectAsState()
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(now?.title ?: "Canlı yayın", maxLines = 1) },
                navigationIcon = { IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Geri") } },
                actions = { IconButton(onClick = { PlayerHub.stop(); nav.back() }) { Icon(Icons.Filled.Close, "Yayını kapat") } },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = Color.Black, titleContentColor = Color.White, navigationIconContentColor = Color.White, actionIconContentColor = Color.White)
            )
        },
        containerColor = Color.Black
    ) { pad ->
        Column(Modifier.fillMaxSize().padding(pad), verticalArrangement = androidx.compose.foundation.layout.Arrangement.Center) {
            AndroidView(
                modifier = Modifier.fillMaxWidth().aspectRatio(16f / 9f),
                factory = { c -> PlayerView(c).apply { useController = true; setShowBuffering(PlayerView.SHOW_BUFFERING_ALWAYS) } },
                update = { it.player = PlayerHub.player() },
                onRelease = { it.player = null }
            )
            now?.let {
                Spacer(Modifier.height(12.dp))
                Text(it.subtitle, color = Color(0xFF9FB0CC), modifier = Modifier.padding(horizontal = 16.dp))
                if (it.error.isNotBlank()) Text(it.error, color = Color(0xFFFF6B78), modifier = Modifier.padding(16.dp))
            }
        }
    }
}

/** Uygulamanın her yerinde altta görünen oynatıcı çubuğu. */
@Composable
fun MiniPlayer(onOpenVideo: () -> Unit) {
    val now by PlayerHub.now.collectAsState()
    val n = now ?: return
    Row(
        Modifier.fillMaxWidth().background(MaterialTheme.colorScheme.surfaceVariant)
            .clickable(enabled = n.kind == "tv") { onOpenVideo() }
            .padding(horizontal = 12.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Box(Modifier.size(36.dp).clip(RoundedCornerShape(8.dp)).background(MaterialTheme.colorScheme.surface), contentAlignment = Alignment.Center) {
            Icon(if (n.kind == "tv") Icons.Filled.Tv else Icons.Filled.Radio, null, tint = MaterialTheme.colorScheme.primary)
        }
        Spacer(Modifier.width(10.dp))
        Column(Modifier.weight(1f)) {
            Text(n.title, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text(
                when {
                    n.error.isNotBlank() -> n.error
                    n.buffering -> "Bağlanıyor…"
                    else -> n.subtitle.ifBlank { if (n.kind == "tv") "Canlı TV" else "Canlı radyo" }
                },
                style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1
            )
        }
        if (n.buffering) CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp)
        IconButton(onClick = { PlayerHub.toggle() }) { Icon(if (n.playing) Icons.Filled.Pause else Icons.Filled.PlayArrow, if (n.playing) "Duraklat" else "Çal") }
        IconButton(onClick = { PlayerHub.stop() }) { Icon(Icons.Filled.Close, "Kapat") }
    }
}
