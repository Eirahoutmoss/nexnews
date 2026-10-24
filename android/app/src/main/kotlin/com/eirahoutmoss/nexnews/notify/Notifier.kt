package com.eirahoutmoss.nexnews.notify

import com.eirahoutmoss.nexnews.t

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import com.eirahoutmoss.nexnews.Alert
import com.eirahoutmoss.nexnews.Briefing
import com.eirahoutmoss.nexnews.MainActivity
import com.eirahoutmoss.nexnews.R

/** Bildirim kanalları: kullanıcı her birini sistem ayarlarından ayrı ayrı açıp kapatabilir. */
object Notifier {
    const val CH_BREAKING = "son_dakika"
    const val CH_IMPORTANT = "onemli"
    const val CH_DIGEST = "ozet"
    const val EXTRA_LINK = "nexnews.link"

    fun createChannels(ctx: Context) {
        if (Build.VERSION.SDK_INT < 26) return
        val nm = ctx.getSystemService(NotificationManager::class.java) ?: return
        nm.createNotificationChannel(NotificationChannel(CH_BREAKING, t("Son dakika"), NotificationManager.IMPORTANCE_HIGH).apply {
            description = t("Birden çok kaynağın doğruladığı son dakika gelişmeleri")
        })
        nm.createNotificationChannel(NotificationChannel(CH_IMPORTANT, t("Önemli gelişmeler"), NotificationManager.IMPORTANCE_DEFAULT).apply {
            description = t("Önem eşiğini geçen yeni olaylar")
        })
        nm.createNotificationChannel(NotificationChannel(CH_DIGEST, t("Günlük özet"), NotificationManager.IMPORTANCE_LOW).apply {
            description = t("Sabah ve akşam brifingi")
        })
    }

    private fun canPost(ctx: Context): Boolean =
        Build.VERSION.SDK_INT < 33 ||
            ContextCompat.checkSelfPermission(ctx, Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED

    private fun openIntent(ctx: Context, link: String?, req: Int): PendingIntent {
        val i = Intent(ctx, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            if (link != null) putExtra(EXTRA_LINK, link)
        }
        return PendingIntent.getActivity(ctx, req, i, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
    }

    @Suppress("MissingPermission")
    fun postAlerts(ctx: Context, alerts: List<Alert>) {
        if (alerts.isEmpty() || !canPost(ctx)) return
        val nm = NotificationManagerCompat.from(ctx)
        for (a in alerts) {
            val ch = if (a.breaking) CH_BREAKING else CH_IMPORTANT
            val id = a.link.hashCode()
            val sub = buildString {
                append(a.source)
                if (a.sources > 1) append(" · ${a.sources} " + t("bağımsız kaynak"))
                if (a.category.isNotBlank()) append(" · ${a.category}")
            }
            val n = NotificationCompat.Builder(ctx, ch)
                .setSmallIcon(R.drawable.ic_stat_news)
                .setContentTitle(if (a.followed) t("Takip ettiğin olay") else if (a.breaking) t("Son dakika") else t("Önemli gelişme"))
                .setContentText(a.title)
                .setSubText(sub)
                .setStyle(NotificationCompat.BigTextStyle().bigText(a.title + "\n" + sub))
                .setContentIntent(openIntent(ctx, a.link, id))
                .setAutoCancel(true)
                .setCategory(NotificationCompat.CATEGORY_RECOMMENDATION)
                .setPriority(if (a.breaking) NotificationCompat.PRIORITY_HIGH else NotificationCompat.PRIORITY_DEFAULT)
                .build()
            nm.notify(id, n)
        }
    }

    @Suppress("MissingPermission")
    fun postDigest(ctx: Context, b: Briefing) {
        if (b.items.isEmpty() || !canPost(ctx)) return
        val lines = b.items.take(5).joinToString("\n") { "${it.rank}. ${it.title}" }
        val n = NotificationCompat.Builder(ctx, CH_DIGEST)
            .setSmallIcon(R.drawable.ic_stat_news)
            .setContentTitle(b.windowLabel.ifBlank { t("Günün özeti") })
            .setContentText(b.headline)
            .setStyle(NotificationCompat.BigTextStyle().bigText(lines))
            .setContentIntent(openIntent(ctx, null, 7001))
            .setAutoCancel(true)
            .build()
        NotificationManagerCompat.from(ctx).notify(7001, n)
    }
}
