package com.eirahoutmoss.nexnews.ui

import android.net.TrafficStats
import android.os.Process
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import kotlinx.coroutines.delay
import java.util.Locale

/**
 * Masaüstündeki ağ göstergesinin telefondaki karşılığı. Anlık hız cihazın
 * tamamıdır; "5 dk" NexNews'in (haber, görsel, radyo/TV dahil) son beş
 * dakikada indirdiği veridir. Android'in uygulama bazlı sayaçlarından okunur.
 */
object NetMeter {
    private data class Sample(val t: Long, val uidRx: Long, val totalRx: Long)
    private val samples = ArrayDeque<Sample>()

    data class Reading(val rateBps: Long, val app5m: Long, val appSession: Long)

    private var first: Sample? = null

    fun sample(): Reading {
        val uid = Process.myUid()
        val s = Sample(System.currentTimeMillis(), TrafficStats.getUidRxBytes(uid).coerceAtLeast(0), TrafficStats.getTotalRxBytes().coerceAtLeast(0))
        if (first == null) first = s
        samples.addLast(s)
        while (samples.size > 2 && s.t - samples.first().t > 5 * 60_000) samples.removeFirst()
        val prev = samples.getOrNull(samples.size - 2)
        val rate = if (prev != null && s.t > prev.t) ((s.totalRx - prev.totalRx) * 1000 / (s.t - prev.t)).coerceAtLeast(0) else 0
        val app5 = (s.uidRx - samples.first().uidRx).coerceAtLeast(0)
        val sess = (s.uidRx - (first?.uidRx ?: s.uidRx)).coerceAtLeast(0)
        return Reading(rate, app5, sess)
    }
}

fun fmtBytes(n: Long): String {
    var v = n.toDouble(); var i = 0
    val u = listOf("B", "KB", "MB", "GB")
    while (v >= 1024 && i < u.lastIndex) { v /= 1024; i++ }
    return if (i == 0) "${v.toLong()} B" else String.format(Locale("tr"), if (v < 10) "%.1f %s" else "%.0f %s", v, u[i])
}

fun fmtRate(bytesPerSec: Long): String {
    val bits = bytesPerSec * 8.0
    return when {
        bits < 1_000 -> "${bits.toLong()} bps"
        bits < 1_000_000 -> String.format(Locale("tr"), if (bits < 10_000) "%.1f Kbps" else "%.0f Kbps", bits / 1e3)
        bits < 1_000_000_000 -> String.format(Locale("tr"), if (bits < 10_000_000) "%.1f Mbps" else "%.0f Mbps", bits / 1e6)
        else -> String.format(Locale("tr"), "%.2f Gbps", bits / 1e9)
    }
}

@Composable
fun NetMeterLine() {
    var r by remember { mutableStateOf<NetMeter.Reading?>(null) }
    LaunchedEffect(Unit) {
        while (true) { r = NetMeter.sample(); delay(2000) }
    }
    val x = r ?: return
    Text(
        "⇣ ${fmtRate(x.rateBps)} · 5 dk: ${fmtBytes(x.app5m)}",
        style = MaterialTheme.typography.labelSmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant
    )
}
