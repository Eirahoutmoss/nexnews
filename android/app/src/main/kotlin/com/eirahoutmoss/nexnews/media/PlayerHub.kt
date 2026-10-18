package com.eirahoutmoss.nexnews.media

import android.content.ComponentName
import android.content.Context
import android.net.Uri
import androidx.media3.common.MediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.common.MimeTypes
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.session.MediaController
import androidx.media3.session.SessionToken
import com.eirahoutmoss.nexnews.RadioStation
import com.eirahoutmoss.nexnews.TvChannel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.guava.await

/** Arayüz ile PlaybackService arasındaki köprü (tek MediaController). */
object PlayerHub {
    data class NowPlaying(
        val kind: String,      // "radio" | "tv"
        val title: String,
        val subtitle: String,
        val artwork: String = "",
        val playing: Boolean = false,
        val buffering: Boolean = false,
        val error: String = "",
    )

    private var controller: MediaController? = null
    private val _now = MutableStateFlow<NowPlaying?>(null)
    val now: StateFlow<NowPlaying?> = _now.asStateFlow()

    /** TV için yedek akış adresleri: biri açılmazsa sıradakine geçilir. */
    private var tvFallbacks: List<String> = emptyList()
    private var tvChannel: TvChannel? = null

    suspend fun controller(ctx: Context): MediaController {
        controller?.let { return it }
        val token = SessionToken(ctx, ComponentName(ctx, PlaybackService::class.java))
        val c = MediaController.Builder(ctx.applicationContext, token).buildAsync().await()
        c.addListener(object : Player.Listener {
            override fun onIsPlayingChanged(isPlaying: Boolean) { update { it.copy(playing = isPlaying, error = if (isPlaying) "" else it.error) } }
            override fun onPlaybackStateChanged(state: Int) { update { it.copy(buffering = state == Player.STATE_BUFFERING) } }
            override fun onPlayerError(error: PlaybackException) {
                if (tvFallbacks.isNotEmpty()) {
                    val next = tvFallbacks.first(); tvFallbacks = tvFallbacks.drop(1)
                    tvChannel?.let { ch -> setItem(c, next, ch.id, ch.name, ch.meta, "", hls = true) }
                    c.prepare(); c.play()
                } else {
                    update { it.copy(playing = false, buffering = false, error = "Yayın şu anda açılamıyor") }
                }
            }
        })
        controller = c
        return c
    }

    private fun update(f: (NowPlaying) -> NowPlaying) { _now.value?.let { _now.value = f(it) } }

    private fun setItem(c: MediaController, url: String, id: String, title: String, sub: String, art: String, hls: Boolean) {
        val meta = MediaMetadata.Builder().setTitle(title).setArtist(sub).apply {
            if (art.isNotBlank()) setArtworkUri(Uri.parse(art))
        }.build()
        val item = MediaItem.Builder().setUri(url).setMediaId(id).setMediaMetadata(meta).apply {
            if (hls || url.contains(".m3u8", ignoreCase = true)) setMimeType(MimeTypes.APPLICATION_M3U8)
        }.build()
        c.setMediaItem(item)
    }

    suspend fun playRadio(ctx: Context, st: RadioStation) {
        val c = controller(ctx)
        tvFallbacks = emptyList(); tvChannel = null
        _now.value = NowPlaying("radio", st.name, listOf(st.country, st.language).filter { it.isNotBlank() }.joinToString(" · "), st.favicon, buffering = true)
        setItem(c, st.stream, "radio:" + st.uuid.ifBlank { st.stream }, st.name, st.country, st.favicon, hls = false)
        c.prepare(); c.play()
    }

    suspend fun playTv(ctx: Context, ch: TvChannel) {
        val c = controller(ctx)
        val streams = ch.streams
        if (streams.isEmpty()) {
            _now.value = NowPlaying("tv", ch.name, ch.meta, error = "Bu kanal için uygulama içi yayın yok")
            return
        }
        tvChannel = ch
        tvFallbacks = streams.drop(1)
        // Sınırlı veri modunda en çok 480p.
        val lim = com.eirahoutmoss.nexnews.Core.dataLimited()
        c.trackSelectionParameters = c.trackSelectionParameters.buildUpon()
            .setMaxVideoSize(if (lim) 854 else Int.MAX_VALUE, if (lim) 480 else Int.MAX_VALUE).build()
        _now.value = NowPlaying("tv", ch.name, ch.meta, buffering = true)
        setItem(c, streams.first(), "tv:" + ch.id, ch.name, ch.meta, "", hls = true)
        c.prepare(); c.play()
    }

    fun toggle() { controller?.let { if (it.isPlaying) it.pause() else it.play() } }

    fun stop() {
        controller?.let { it.stop(); it.clearMediaItems() }
        tvFallbacks = emptyList(); tvChannel = null
        _now.value = null
    }

    fun player(): Player? = controller
}
