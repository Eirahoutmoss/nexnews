package com.eirahoutmoss.nexnews

import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.RectF
import android.graphics.Typeface
import android.text.Layout
import android.text.SpannableStringBuilder
import android.text.StaticLayout
import android.text.TextPaint
import android.text.style.BackgroundColorSpan
import androidx.core.content.FileProvider
import java.io.File
import java.io.FileOutputStream

/**
 * "Kim nasıl başlık attı?" paylaşım kartı: aynı olayın farklı kaynaklardaki
 * başlıkları tek görselde. WhatsApp gibi uygulamalarda tek başlıklı, bağlamsız
 * iletilere karşı küçük bir denge unsuru.
 */
object ShareCard {
    private const val W = 1080
    private const val PAD = 64f

    fun share(ctx: Context, s: StoryDetail) {
        val file = render(ctx, s)
        val uri = FileProvider.getUriForFile(ctx, ctx.packageName + ".files", file)
        val first = s.headlines.firstOrNull()?.link ?: ""
        val i = Intent(Intent.ACTION_SEND).apply {
            type = "image/png"
            putExtra(Intent.EXTRA_STREAM, uri)
            putExtra(Intent.EXTRA_TEXT, s.title + (if (first.isNotBlank()) "\n$first" else "") + "\n— NexNews ile karşılaştırıldı")
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        }
        ctx.startActivity(Intent.createChooser(i, "Karşılaştırmayı paylaş").addFlags(Intent.FLAG_ACTIVITY_NEW_TASK))
    }

    private fun layout(text: CharSequence, paint: TextPaint, width: Int): StaticLayout =
        StaticLayout.Builder.obtain(text, 0, text.length, paint, width)
            .setAlignment(Layout.Alignment.ALIGN_NORMAL)
            .setLineSpacing(0f, 1.18f)
            .build()

    fun render(ctx: Context, s: StoryDetail): File {
        val inner = (W - 2 * PAD).toInt()
        val titleP = TextPaint(Paint.ANTI_ALIAS_FLAG).apply { color = Color.WHITE; textSize = 52f; typeface = Typeface.DEFAULT_BOLD }
        val labelP = TextPaint(Paint.ANTI_ALIAS_FLAG).apply { color = Color.parseColor("#36C9B6"); textSize = 30f; typeface = Typeface.DEFAULT_BOLD; letterSpacing = 0.08f }
        val srcP = TextPaint(Paint.ANTI_ALIAS_FLAG).apply { color = Color.parseColor("#36C9B6"); textSize = 32f; typeface = Typeface.DEFAULT_BOLD }
        val headP = TextPaint(Paint.ANTI_ALIAS_FLAG).apply { color = Color.parseColor("#E7ECF6"); textSize = 38f }
        val faintP = TextPaint(Paint.ANTI_ALIAS_FLAG).apply { color = Color.parseColor("#9FB0CC"); textSize = 28f }

        val heads = s.headlines.take(6)
        val title = layout(s.title, titleP, inner)
        val blocks = heads.map { h ->
            val text = SpannableStringBuilder(h.translated.ifBlank { h.title })
            for (w in h.distinct) {
                val idx = text.indexOf(w)
                if (idx >= 0) text.setSpan(BackgroundColorSpan(Color.parseColor("#5536C9B6")), idx, idx + w.length, 0)
            }
            val src = h.source + if (h.via.isNotBlank()) "  ·  ${h.via} metni" else ""
            Pair(layout(src, srcP, inner), layout(text, headP, inner))
        }
        val foot = layout(
            "${s.independentSources} bağımsız kaynak · ${s.sourceCount} kaynak" +
                (if (s.evidence.syndicated > 0) " · ${s.evidence.syndicated} kaynak ajans metni" else "") +
                "\nİşaretli kelimeler yalnız o kaynağın başlığında geçiyor.  —  NexNews",
            faintP, inner
        )
        var h = PAD + 40 + 20 + title.height + 40
        blocks.forEach { (a, b) -> h += a.height + 8 + b.height + 36 }
        h += foot.height + PAD
        val bmp = Bitmap.createBitmap(W, h.toInt(), Bitmap.Config.ARGB_8888)
        val c = Canvas(bmp)
        c.drawColor(Color.parseColor("#0E1320"))
        val card = Paint(Paint.ANTI_ALIAS_FLAG).apply { color = Color.parseColor("#141B2B") }
        var y = PAD
        c.save(); c.translate(PAD, y); layout("KİM NASIL BAŞLIK ATTI?", labelP, inner).draw(c); c.restore()
        y += 60
        c.save(); c.translate(PAD, y); title.draw(c); c.restore()
        y += title.height + 40
        for ((a, b) in blocks) {
            val bh = a.height + 8 + b.height
            c.drawRoundRect(RectF(PAD - 24, y - 18, W - PAD + 24, y + bh + 18), 22f, 22f, card)
            c.save(); c.translate(PAD, y); a.draw(c); c.restore()
            c.save(); c.translate(PAD, y + a.height + 8); b.draw(c); c.restore()
            y += bh + 36
        }
        c.save(); c.translate(PAD, y); foot.draw(c); c.restore()

        val dir = File(ctx.cacheDir, "share").apply { mkdirs() }
        val f = File(dir, "nexnews-karsilastirma.png")
        FileOutputStream(f).use { bmp.compress(Bitmap.CompressFormat.PNG, 100, it) }
        bmp.recycle()
        return f
    }
}
