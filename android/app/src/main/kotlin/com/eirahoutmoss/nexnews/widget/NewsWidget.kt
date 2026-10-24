package com.eirahoutmoss.nexnews.widget

import com.eirahoutmoss.nexnews.t

import android.content.Context
import android.content.Intent
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.action.actionStartActivity
import androidx.glance.appwidget.cornerRadius
import androidx.glance.appwidget.lazy.LazyColumn
import androidx.glance.appwidget.lazy.items
import androidx.glance.appwidget.provideContent
import androidx.glance.appwidget.updateAll
import androidx.glance.background
import androidx.glance.layout.Alignment
import androidx.glance.layout.Column
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.width
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import androidx.glance.unit.ColorProvider
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.MainActivity
import com.eirahoutmoss.nexnews.NewsItem
import com.eirahoutmoss.nexnews.Prefs
import com.eirahoutmoss.nexnews.notify.Notifier

/** Ana ekran widget'ı: son dakika + en önemli başlıklar. Zemin opaklığı ayarlardan değişir. */
class NewsWidget : GlanceAppWidget() {

    companion object {
        suspend fun refreshAll(ctx: Context) {
            NewsWidget().updateAll(ctx)
        }
    }

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        Core.ensureInit(context)
        Core.reloadLocal()
        val ov = Core.overview.value
        val items = buildList {
            ov?.breaking?.let { addAll(it.take(3)) }
            ov?.forYou?.let { addAll(it.take(3)) }
            ov?.important?.let { addAll(it) }
        }.distinctBy { it.link }.take(8)
        val alpha = Prefs.get(context).widgetOpacity.coerceIn(0, 100) / 100f
        val updated = ov?.updatedTs ?: 0L
        provideContent { WidgetBody(context, items, alpha, updated) }
    }
}

private fun openApp(ctx: Context, link: String?): Intent =
    Intent(ctx, MainActivity::class.java).apply {
        flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
        if (link != null) putExtra(Notifier.EXTRA_LINK, link)
    }

@Composable
private fun WidgetBody(ctx: Context, items: List<NewsItem>, alpha: Float, updated: Long) {
    val txt = ColorProvider(Color(0xFFE7ECF6))
    val faint = ColorProvider(Color(0xFF9FB0CC))
    val accent = ColorProvider(Color(0xFF36C9B6))
    val red = ColorProvider(Color(0xFFFF6B78))
    Column(
        modifier = GlanceModifier.fillMaxSize()
            .background(Color(0xFF121B2A).copy(alpha = alpha))
            .cornerRadius(18.dp)
            .padding(horizontal = 12.dp, vertical = 10.dp)
    ) {
        Row(
            modifier = GlanceModifier.fillMaxWidth().clickable(actionStartActivity(openApp(ctx, null))),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text("NexNews", style = TextStyle(color = accent, fontWeight = FontWeight.Bold, fontSize = 14.sp))
            Spacer(GlanceModifier.width(8.dp))
            Text(agoText(updated), style = TextStyle(color = faint, fontSize = 11.sp))
        }
        Spacer(GlanceModifier.height(6.dp))
        if (items.isEmpty()) {
            Text(t("Başlıklar yükleniyor… Uygulamayı bir kez açın."), style = TextStyle(color = faint, fontSize = 12.sp))
        } else {
            LazyColumn {
                items(items) { item ->
                    Column(modifier = GlanceModifier.fillMaxWidth().padding(vertical = 4.dp)
                        .clickable(actionStartActivity(openApp(ctx, item.link)))) {
                        Text(
                            (if (item.breaking) "⚡ " else "") + item.title,
                            maxLines = 2,
                            style = TextStyle(color = if (item.breaking) red else txt, fontSize = 13.sp, fontWeight = FontWeight.Medium)
                        )
                        Text(
                            item.source + if (item.independentSources > 1) " · ${item.independentSources} kaynak" else "",
                            maxLines = 1,
                            style = TextStyle(color = faint, fontSize = 11.sp)
                        )
                    }
                }
            }
        }
    }
}

private fun agoText(ts: Long): String {
    if (ts <= 0) return ""
    val d = System.currentTimeMillis() / 1000 - ts
    return when {
        d < 60 -> t("az önce")
        d < 3600 -> "${d / 60} " + t("dk") + " " + t("önce")
        d < 86400 -> "${d / 3600} " + t("sa") + " " + t("önce")
        else -> "${d / 86400} " + t("gün") + " " + t("önce")
    }
}

class NewsWidgetReceiver : GlanceAppWidgetReceiver() {
    override val glanceAppWidget: GlanceAppWidget = NewsWidget()
}
