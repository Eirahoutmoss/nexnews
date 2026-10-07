@file:OptIn(ExperimentalMaterial3Api::class)

package com.eirahoutmoss.nexnews.ui

import android.annotation.SuppressLint
import android.content.Context
import android.speech.tts.TextToSpeech
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.VolumeUp
import androidx.compose.material.icons.filled.OpenInBrowser
import androidx.compose.material.icons.filled.Share
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.text.HtmlCompat
import com.eirahoutmoss.nexnews.Article
import com.eirahoutmoss.nexnews.Core
import java.util.Locale

private fun cssColor(c: androidx.compose.ui.graphics.Color): String {
    val v = c.toArgb()
    return String.format("#%06X", 0xFFFFFF and v)
}

@SuppressLint("SetJavaScriptEnabled")
@Composable
fun ReaderScreen(link: String, nav: NexNav, bottomPad: androidx.compose.foundation.layout.PaddingValues) {
    val ctx = LocalContext.current
    var art by remember { mutableStateOf<Article?>(null) }
    var speaking by remember { mutableStateOf(false) }
    val tts = remember { mutableStateOf<TextToSpeech?>(null) }
    LaunchedEffect(link) { art = runCatching { Core.article(link) }.getOrElse { Article(err = "Sayfaya ulaşılamadı.") } }
    DisposableEffect(Unit) { onDispose { tts.value?.stop(); tts.value?.shutdown() } }

    val bg = cssColor(MaterialTheme.colorScheme.background)
    val fg = cssColor(MaterialTheme.colorScheme.onBackground)
    val acc = cssColor(MaterialTheme.colorScheme.primary)
    val faint = cssColor(MaterialTheme.colorScheme.onSurfaceVariant)
    val images = showImages()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(art?.title?.ifBlank { "Okuyucu" } ?: "Okuyucu", maxLines = 1, overflow = TextOverflow.Ellipsis) },
                navigationIcon = { IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Geri") } },
                actions = {
                    val a = art
                    if (a != null && a.html.isNotBlank()) {
                        IconButton(onClick = {
                            if (speaking) { tts.value?.stop(); speaking = false }
                            else speak(ctx, tts, a) { speaking = it }
                        }) { Icon(if (speaking) Icons.Filled.Stop else Icons.AutoMirrored.Filled.VolumeUp, if (speaking) "Durdur" else "Sesli oku") }
                    }
                    IconButton(onClick = { shareText(ctx, art?.title ?: "", link) }) { Icon(Icons.Filled.Share, "Paylaş") }
                    IconButton(onClick = { openInBrowser(ctx, link) }) { Icon(Icons.Filled.OpenInBrowser, "Tarayıcıda aç") }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background)
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { pad ->
        val a = art
        if (a == null) { Box(Modifier.padding(pad)) { Loading("Makale getiriliyor…") }; return@Scaffold }
        val body = if (a.html.isNotBlank()) a.html else
            "<p class='err'>${a.err.ifBlank { "Bu sayfa uygulama içinde okunamıyor." }}</p><p><a href='$link'>Tarayıcıda aç →</a></p>"
        val html = """
            <html><head><meta name="viewport" content="width=device-width, initial-scale=1">
            <style>
              body{background:$bg;color:$fg;font-family:sans-serif;font-size:18px;line-height:1.7;padding:4px 18px 60px;margin:0;word-wrap:break-word}
              h1{font-size:24px;line-height:1.3}
              a{color:$acc}
              img{max-width:100%;height:auto;border-radius:10px}
              blockquote{border-left:3px solid $acc;margin:0;padding-left:14px;color:$faint}
              .err{color:$faint}
            </style></head><body><h1>${HtmlCompat.fromHtml(a.title, 0)}</h1>$body</body></html>
        """.trimIndent()
        AndroidView(
            modifier = Modifier.fillMaxSize().padding(top = pad.calculateTopPadding(), bottom = bottomPad.calculateBottomPadding()),
            factory = { c ->
                WebView(c).apply {
                    settings.javaScriptEnabled = false
                    settings.blockNetworkImage = !images
                    setBackgroundColor(MaterialTheme_bg(bg))
                    webViewClient = object : WebViewClient() {
                        override fun shouldOverrideUrlLoading(view: WebView, request: WebResourceRequest): Boolean {
                            openInBrowser(c, request.url.toString()); return true
                        }
                    }
                }
            },
            update = { it.loadDataWithBaseURL(link, html, "text/html", "utf-8", null) }
        )
    }
}

@Suppress("FunctionName")
private fun MaterialTheme_bg(hex: String): Int = android.graphics.Color.parseColor(hex)

private fun speak(ctx: Context, holder: androidx.compose.runtime.MutableState<TextToSpeech?>, a: Article, onState: (Boolean) -> Unit) {
    val text = HtmlCompat.fromHtml(a.html, 0).toString().replace(Regex("\\s+"), " ").trim()
    if (text.isBlank()) return
    val existing = holder.value
    val run: (TextToSpeech) -> Unit = { t ->
        t.language = Locale.getDefault()
        // TTS tek seferde ~4000 karakter kabul eder; parçalara böl.
        text.chunked(3500).forEachIndexed { i, part ->
            t.speak(part, if (i == 0) TextToSpeech.QUEUE_FLUSH else TextToSpeech.QUEUE_ADD, null, "nex$i")
        }
        onState(true)
    }
    if (existing != null) { run(existing); return }
    var created: TextToSpeech? = null
    created = TextToSpeech(ctx.applicationContext) { status ->
        if (status == TextToSpeech.SUCCESS) created?.let { run(it) }
    }
    holder.value = created
}
