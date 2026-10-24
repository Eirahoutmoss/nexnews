@file:OptIn(ExperimentalMaterial3Api::class, ExperimentalFoundationApi::class)

package com.eirahoutmoss.nexnews.ui

import com.eirahoutmoss.nexnews.t

import android.content.Context
import android.content.Intent
import android.net.ConnectivityManager
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Done
import androidx.compose.material.icons.filled.Star
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SwipeToDismissBox
import androidx.compose.material3.SwipeToDismissBoxValue
import androidx.compose.material3.Text
import androidx.compose.material3.rememberSwipeToDismissBoxState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.eirahoutmoss.nexnews.NewsItem
import com.eirahoutmoss.nexnews.Prefs

fun relTime(ts: Long): String {
    if (ts <= 0) return ""
    val d = (System.currentTimeMillis() / 1000 - ts).coerceAtLeast(0)
    return when {
        d < 60 -> t("az önce")
        d < 3600 -> "${d / 60} dk"
        d < 86400 -> "${d / 3600} sa"
        d < 604800 -> "${d / 86400} " + t("gün")
        else -> java.text.SimpleDateFormat("d MMM", java.util.Locale("tr")).format(java.util.Date(ts * 1000))
    }
}

fun agoText(ts: Long): String = relTime(ts).let { if (it.isEmpty() || it == t("az önce")) it else "$it " + t("önce") }

fun clock(ts: Long): String =
    if (ts <= 0) "" else java.text.SimpleDateFormat("HH:mm", java.util.Locale("tr")).format(java.util.Date(ts * 1000))

fun isMetered(ctx: Context): Boolean {
    val cm = ctx.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager ?: return false
    return cm.isActiveNetworkMetered
}

/** Bağlantıyı sistemin tarayıcısında (Custom Tabs) açar; yalnızca http/https. */
fun openInBrowser(ctx: Context, url: String) {
    val u = runCatching { Uri.parse(url) }.getOrNull() ?: return
    if (u.scheme != "http" && u.scheme != "https") return
    runCatching { CustomTabsIntent.Builder().setShowTitle(true).build().launchUrl(ctx, u) }
        .onFailure { runCatching { ctx.startActivity(Intent(Intent.ACTION_VIEW, u).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)) } }
}

fun shareText(ctx: Context, title: String, link: String) {
    val i = Intent(Intent.ACTION_SEND).apply {
        type = "text/plain"
        putExtra(Intent.EXTRA_SUBJECT, title)
        putExtra(Intent.EXTRA_TEXT, "$title\n$link")
    }
    ctx.startActivity(Intent.createChooser(i, t("Paylaş")).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
}

@Composable
fun showImages(): Boolean {
    val ctx = LocalContext.current
    return !(Prefs.get(ctx).imagesWifiOnly && isMetered(ctx))
}

@Composable
fun SectionHeader(title: String, modifier: Modifier = Modifier, sub: String = "", action: String? = null, onAction: (() -> Unit)? = null) {
    Row(
        modifier.fillMaxWidth().padding(start = 16.dp, end = 8.dp, top = 18.dp, bottom = 6.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Column(Modifier.weight(1f)) {
            Text(title.uppercase(java.util.Locale("tr")), style = MaterialTheme.typography.labelSmall.copy(fontSize = 12.sp, letterSpacing = 0.8.sp, fontWeight = FontWeight.Bold), color = MaterialTheme.colorScheme.onSurface)
            if (sub.isNotBlank()) Text(sub, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        if (action != null && onAction != null) {
            Text(
                action,
                color = MaterialTheme.colorScheme.primary,
                style = MaterialTheme.typography.labelSmall.copy(fontSize = 13.sp, fontWeight = FontWeight.SemiBold),
                modifier = Modifier.clip(RoundedCornerShape(8.dp)).combinedClickable(onClick = onAction).padding(8.dp)
            )
        }
    }
}

@Composable
fun Pill(text: String, modifier: Modifier = Modifier, color: Color = MaterialTheme.colorScheme.onSurfaceVariant, filled: Color? = null) {
    Box(
        modifier
            .clip(RoundedCornerShape(50))
            .then(if (filled != null) Modifier.background(filled) else Modifier.border(1.dp, MaterialTheme.colorScheme.outline, RoundedCornerShape(50)))
            .padding(horizontal = 8.dp, vertical = 2.dp)
    ) {
        Text(text, color = if (filled != null) Color.White else color, style = MaterialTheme.typography.labelSmall, maxLines = 1)
    }
}

/** Tek sinyal: son dakika ya da bağımsız kaynak sayısı (masaüstüyle aynı kural). */
@Composable
fun SignalPill(item: NewsItem) {
    val nex = LocalNex.current
    when {
        item.breaking -> Pill(t("Son dakika"), filled = nex.breaking)
        item.independentSources > 1 -> Pill("${item.independentSources} " + t("bağımsız kaynak"))
    }
}

/**
 * Haber satırı. Sağa kaydır: favori; sola kaydır: okundu. Uzun bas: işlemler.
 * dense=true: tek satır, görselsiz.
 */
@Composable
fun NewsRow(
    item: NewsItem,
    read: Boolean,
    fav: Boolean,
    dense: Boolean,
    onOpen: () -> Unit,
    onFav: () -> Unit,
    onRead: () -> Unit,
    onLong: () -> Unit,
    images: Boolean = true,
) {
    val state = rememberSwipeToDismissBoxState(confirmValueChange = { v ->
        when (v) {
            SwipeToDismissBoxValue.StartToEnd -> onFav()
            SwipeToDismissBoxValue.EndToStart -> onRead()
            else -> {}
        }
        false // satır listede kalır
    })
    SwipeToDismissBox(
        state = state,
        backgroundContent = {
            val toFav = state.dismissDirection == SwipeToDismissBoxValue.StartToEnd
            Row(
                Modifier.fillMaxSize().background(if (toFav) Color(0x33FFD45E) else MaterialTheme.colorScheme.primary.copy(alpha = 0.18f)).padding(horizontal = 20.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = if (toFav) Arrangement.Start else Arrangement.End
            ) {
                Icon(if (toFav) Icons.Filled.Star else Icons.Filled.Done, null, tint = if (toFav) Color(0xFFFFC94A) else MaterialTheme.colorScheme.primary)
                Spacer(Modifier.width(8.dp))
                Text(if (toFav) (if (fav) t("Favoriden çıkar") else "Favorilere ekle") else "Okundu", style = MaterialTheme.typography.labelSmall)
            }
        }
    ) {
        NewsRowBody(item, read, fav, dense, onOpen, onLong, images)
    }
}

@Composable
private fun NewsRowBody(item: NewsItem, read: Boolean, fav: Boolean, dense: Boolean, onOpen: () -> Unit, onLong: () -> Unit, images: Boolean) {
    val nex = LocalNex.current
    val titleColor = if (read) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.onSurface
    Column(
        Modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.background)
            .combinedClickable(onClick = onOpen, onLongClick = onLong)
    ) {
        Row(Modifier.padding(horizontal = 16.dp, vertical = if (dense) 9.dp else 12.dp), verticalAlignment = Alignment.Top) {
            Box(
                Modifier.padding(top = if (dense) 7.dp else 8.dp, end = 10.dp).size(7.dp).clip(CircleShape)
                    .background(if (item.breaking) nex.breaking else MaterialTheme.colorScheme.outline.copy(alpha = if (read) 0.4f else 1f))
            )
            Column(Modifier.weight(1f)) {
                Text(
                    item.title,
                    style = if (dense) MaterialTheme.typography.bodyMedium else MaterialTheme.typography.titleMedium,
                    fontWeight = if (read) FontWeight.Normal else FontWeight.SemiBold,
                    color = titleColor,
                    maxLines = if (dense) 2 else 4,
                    overflow = TextOverflow.Ellipsis
                )
                Spacer(Modifier.height(4.dp))
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    if (fav) Icon(Icons.Filled.Star, null, tint = Color(0xFFFFC94A), modifier = Modifier.size(13.dp))
                    Text(
                        item.source + if (item.sources.size > 1) " +${item.sources.size - 1}" else "",
                        color = MaterialTheme.colorScheme.primary,
                        style = MaterialTheme.typography.labelSmall.copy(fontSize = 12.sp),
                        maxLines = 1, overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f, fill = false)
                    )
                    Text("· " + relTime(item.ts), color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.labelSmall.copy(fontSize = 12.sp), maxLines = 1)
                    if (!dense) SignalPill(item) else if (item.breaking) SignalPill(item)
                }
            }
            if (!dense && images && item.image.isNotBlank()) {
                Spacer(Modifier.width(12.dp))
                AsyncImage(
                    model = item.image,
                    contentDescription = null,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.size(width = 92.dp, height = 64.dp).clip(RoundedCornerShape(10.dp)).background(MaterialTheme.colorScheme.surfaceVariant)
                )
            }
        }
        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f), modifier = Modifier.padding(start = 33.dp))
    }
}

@Composable
fun Loading(text: String = t("Yükleniyor…")) {
    Box(Modifier.fillMaxSize().padding(32.dp), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            CircularProgressIndicator(strokeWidth = 3.dp)
            Spacer(Modifier.height(14.dp))
            Text(text, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
fun Empty(text: String) {
    Box(Modifier.fillMaxWidth().padding(40.dp), contentAlignment = Alignment.Center) {
        Text(text, color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.bodyMedium)
    }
}

/** Ekran ilk görünür olduğunda bir kez çalışan yükleyici. */
@Composable
fun OnFirstShow(key: Any?, block: suspend () -> Unit) {
    LaunchedEffect(key) { block() }
}
