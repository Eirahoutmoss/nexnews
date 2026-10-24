package com.eirahoutmoss.nexnews

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.setValue
import com.eirahoutmoss.nexnews.go.nexcore.Nexcore
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * Arayüz dili (RC96). Uygulama Türkçe yazılmıştır; seçili dil Türkçe değilse
 * Go çekirdeğinin sözlüğünden (masaüstüyle aynı i18n.json) çevrilir. Haber
 * başlıkları burada çevrilmez. Sözlüğü olmayan diller İngilizce görür.
 */
object I18n {
    @Volatile var lang: String = "tr"
        private set
    @Volatile private var dict: Map<String, String> = emptyMap()

    /** Değişince Compose ekranları yeniden çizilsin diye okunan sürüm sayacı. */
    var version by mutableIntStateOf(0)
        private set

    fun load() {
        val l = runCatching { Nexcore.uiLang() }.getOrDefault("tr").ifBlank { "tr" }
        val d = if (l == "tr") emptyMap() else runCatching {
            NexJson.parseToJsonElement(Nexcore.i18n()).jsonObject.mapValues { it.value.jsonPrimitive.content }
        }.getOrDefault(emptyMap())
        lang = l
        dict = d
        version++
    }

    /** Sağdan sola yazılan diller. */
    val rtl: Boolean get() = lang == "ar" || lang == "fa" || lang == "he" || lang == "ur"

    fun t(s: String): String {
        @Suppress("UNUSED_VARIABLE") val v = version // Compose aboneliği
        if (dict.isEmpty() || s.isEmpty()) return s
        val k = s.trim()
        val tr = dict[k] ?: return s
        return if (k.length == s.length) tr else s.replace(k, tr)
    }
}

/** Kısa yol: t("Türkçe metin") → seçili dildeki karşılığı. */
fun t(s: String): String = I18n.t(s)
