package com.eirahoutmoss.nexnews.ui

import android.content.Context
import android.speech.tts.TextToSpeech
import android.speech.tts.UtteranceProgressListener
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.eirahoutmoss.nexnews.I18n
import java.util.Locale

/**
 * RC97 sesli brifing: metni telefonun kendi konuşma motoru okur (ses dosyası
 * indirilmez). Tek örnek; ekran değişse de okuma sürer, "Durdur" ile biter.
 */
object Speaker {
    private var tts: TextToSpeech? = null
    private var ready = false
    private var pending: String? = null
    var speaking by mutableStateOf(false)
        private set

    fun speak(ctx: Context, text: String) {
        if (text.isBlank()) return
        val t = tts
        if (t != null && ready) { run(t, text); return }
        pending = text
        if (t == null) {
            var created: TextToSpeech? = null
            created = TextToSpeech(ctx.applicationContext) { status ->
                val c = created ?: return@TextToSpeech
                if (status == TextToSpeech.SUCCESS) {
                    ready = true
                    c.setOnUtteranceProgressListener(object : UtteranceProgressListener() {
                        override fun onStart(utteranceId: String?) { speaking = true }
                        override fun onDone(utteranceId: String?) { if (utteranceId == LAST) speaking = false }
                        @Deprecated("Deprecated in Java")
                        override fun onError(utteranceId: String?) { speaking = false }
                    })
                    pending?.let { run(c, it) }
                    pending = null
                } else {
                    speaking = false
                }
            }
            tts = created
        }
    }

    private fun run(t: TextToSpeech, text: String) {
        val loc = Locale.forLanguageTag(I18n.lang)
        if (t.isLanguageAvailable(loc) >= TextToSpeech.LANG_AVAILABLE) t.language = loc
        // TTS tek seferde ~4000 karakter kabul eder; cümle sınırında parçala.
        val parts = text.split(Regex("(?<=[.!?])\\s+")).fold(mutableListOf<StringBuilder>()) { acc, s ->
            if (acc.isEmpty() || acc.last().length + s.length > 3500) acc.add(StringBuilder())
            acc.last().append(s).append(' ')
            acc
        }
        speaking = true
        parts.forEachIndexed { i, p ->
            t.speak(p.toString(), if (i == 0) TextToSpeech.QUEUE_FLUSH else TextToSpeech.QUEUE_ADD, null,
                if (i == parts.lastIndex) LAST else "nexbrief$i")
        }
    }

    fun stop() {
        tts?.stop()
        speaking = false
    }

    private const val LAST = "nexbrief-last"
}
