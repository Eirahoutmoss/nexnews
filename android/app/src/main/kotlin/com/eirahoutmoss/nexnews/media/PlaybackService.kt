package com.eirahoutmoss.nexnews.media

import android.app.PendingIntent
import android.content.Intent
import android.net.Uri
import androidx.media3.common.AudioAttributes
import androidx.annotation.OptIn
import androidx.media3.common.C
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.datasource.ResolvingDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.MainActivity
import java.util.concurrent.ConcurrentHashMap

/**
 * Radyo ve canlı TV oynatıcısı. Uygulamanın başka ekranlarında gezinirken ya da
 * uygulama arka plandayken çalmaya devam eder; kilit ekranı ve bildirim
 * denetimleri Media3 tarafından sağlanır.
 */
@OptIn(UnstableApi::class)
class PlaybackService : MediaSessionService() {
    private var session: MediaSession? = null
    private val headerCache = ConcurrentHashMap<String, Map<String, String>>()

    /** Yayıncıların beklediği Referer/Origin başlıkları (çekirdekteki liveReferer kuralları). */
    private fun headersFor(uri: Uri): Map<String, String> {
        val key = (uri.host ?: "") + "|" + (uri.path?.split('/')?.getOrNull(1) ?: "")
        return headerCache.getOrPut(key) {
            val h = Core.liveHeaders(uri.toString())
            buildMap {
                if (h.referer.isNotBlank()) put("Referer", h.referer)
                if (h.origin.isNotBlank()) put("Origin", h.origin)
            }
        }
    }

    override fun onCreate() {
        super.onCreate()
        val ua = Core.liveHeaders("https://example.com/").userAgent.ifBlank { "NexNews" }
        val http = DefaultHttpDataSource.Factory()
            .setUserAgent(ua)
            .setAllowCrossProtocolRedirects(true)
            .setConnectTimeoutMs(15_000)
            .setReadTimeoutMs(20_000)
        val resolving = ResolvingDataSource.Factory(http) { spec ->
            val extra = headersFor(spec.uri)
            if (extra.isEmpty()) spec else spec.withAdditionalHeaders(extra)
        }
        val player = ExoPlayer.Builder(this)
            .setMediaSourceFactory(DefaultMediaSourceFactory(this).setDataSourceFactory(resolving))
            .setAudioAttributes(
                AudioAttributes.Builder().setUsage(C.USAGE_MEDIA).setContentType(C.AUDIO_CONTENT_TYPE_MUSIC).build(),
                true
            )
            .setHandleAudioBecomingNoisy(true)
            .setWakeMode(C.WAKE_MODE_NETWORK)
            .build()
        val open = PendingIntent.getActivity(
            this, 9001,
            Intent(this, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        )
        session = MediaSession.Builder(this, player).setSessionActivity(open).build()
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? = session

    override fun onTaskRemoved(rootIntent: Intent?) {
        val p = session?.player
        if (p == null || !p.playWhenReady || p.mediaItemCount == 0) stopSelf()
    }

    override fun onDestroy() {
        session?.run {
            player.release()
            release()
        }
        session = null
        super.onDestroy()
    }
}
