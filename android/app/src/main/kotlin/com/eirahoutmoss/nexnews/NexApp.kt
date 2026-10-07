package com.eirahoutmoss.nexnews

import android.app.Application
import com.eirahoutmoss.nexnews.notify.Notifier
import com.eirahoutmoss.nexnews.work.Scheduler
import kotlinx.coroutines.launch

class NexApp : Application() {
    override fun onCreate() {
        super.onCreate()
        Notifier.createChannels(this)
        Scheduler.apply(this)
        // Çekirdek arka planda hazırlanır; arayüz hazır olana kadar yükleniyor gösterir.
        Core.scope.launch { runCatching { Core.ensureInit(this@NexApp) } }
    }
}
