package com.eirahoutmoss.nexnews.work

import android.content.Context
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import com.eirahoutmoss.nexnews.Alert
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.NexJson
import com.eirahoutmoss.nexnews.Prefs
import com.eirahoutmoss.nexnews.go.nexcore.Nexcore
import com.eirahoutmoss.nexnews.notify.Notifier
import com.eirahoutmoss.nexnews.widget.NewsWidget
import java.util.Calendar
import java.util.concurrent.TimeUnit

/** Arka planda yenileme: kaynakları çeker, yeni önemli olaylar için bildirim gönderir, widget'ı günceller. */
class RefreshWorker(ctx: Context, params: WorkerParameters) : CoroutineWorker(ctx, params) {
    override suspend fun doWork(): Result {
        return try {
            Core.refresh(applicationContext)
            val alerts = runCatching { NexJson.decodeFromString<List<Alert>>(Nexcore.newAlerts()) }.getOrDefault(emptyList())
            Notifier.postAlerts(applicationContext, alerts)
            runCatching { NewsWidget.refreshAll(applicationContext) }
            Result.success()
        } catch (e: Throwable) {
            Result.retry()
        }
    }
}

/** Sabah/akşam brifingi. Yeni veri çekmez; son yenilemenin özetini bildirir. */
class DigestWorker(ctx: Context, params: WorkerParameters) : CoroutineWorker(ctx, params) {
    override suspend fun doWork(): Result {
        val prefs = Prefs.get(applicationContext)
        if (!prefs.digestOn) return Result.success()
        return try {
            Core.ensureInit(applicationContext)
            if (System.currentTimeMillis() / 1000 - Core.lastFetch() > 3 * 3600) {
                Core.refresh(applicationContext)
            }
            if (!Nexcore.inQuietHours()) Notifier.postDigest(applicationContext, Core.briefing())
            Result.success()
        } catch (e: Throwable) {
            Result.retry()
        }
    }
}

object Scheduler {
    private const val REFRESH = "nexnews-refresh"
    private const val DIGEST_AM = "nexnews-digest-am"
    private const val DIGEST_PM = "nexnews-digest-pm"

    /** Tercihlere göre periyodik işleri kurar ya da günceller. */
    fun apply(ctx: Context) {
        val p = Prefs.get(ctx)
        val wm = WorkManager.getInstance(ctx)
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(if (p.wifiOnly) NetworkType.UNMETERED else NetworkType.CONNECTED)
            .setRequiresBatteryNotLow(true)
            .build()
        val minutes = p.refreshMinutes.coerceAtLeast(15).toLong()
        val refresh = PeriodicWorkRequestBuilder<RefreshWorker>(minutes, TimeUnit.MINUTES)
            .setConstraints(constraints)
            .setInitialDelay(minutes, TimeUnit.MINUTES)
            .build()
        wm.enqueueUniquePeriodicWork(REFRESH, ExistingPeriodicWorkPolicy.UPDATE, refresh)

        if (p.digestOn) {
            digest(ctx, DIGEST_AM, p.digestMorning, constraints)
            digest(ctx, DIGEST_PM, p.digestEvening, constraints)
        } else {
            wm.cancelUniqueWork(DIGEST_AM)
            wm.cancelUniqueWork(DIGEST_PM)
        }
    }

    private fun digest(ctx: Context, name: String, hour: Int, constraints: Constraints) {
        val now = Calendar.getInstance()
        val next = (now.clone() as Calendar).apply {
            set(Calendar.HOUR_OF_DAY, hour.coerceIn(0, 23)); set(Calendar.MINUTE, 0); set(Calendar.SECOND, 0)
            if (before(now)) add(Calendar.DAY_OF_YEAR, 1)
        }
        val delay = next.timeInMillis - now.timeInMillis
        val req = PeriodicWorkRequestBuilder<DigestWorker>(24, TimeUnit.HOURS)
            .setInitialDelay(delay, TimeUnit.MILLISECONDS)
            .setConstraints(constraints)
            .build()
        WorkManager.getInstance(ctx).enqueueUniquePeriodicWork(name, ExistingPeriodicWorkPolicy.UPDATE, req)
    }
}
