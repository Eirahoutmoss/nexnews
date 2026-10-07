package com.eirahoutmoss.nexnews.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

/** Masaüstüyle aynı renk dili: lacivert zemin, turkuaz vurgu, kırmızı yalnızca son dakika. */
data class NexExtra(
    val breaking: Color,
    val faint: Color,
    val signalBg: Color,
    val mark: Color,
    val accent2: Color,
)

val LocalNex = staticCompositionLocalOf {
    NexExtra(Color(0xFFE23B4E), Color(0xFF6D7F9E), Color(0x1A9FB0CC), Color(0x3836C9B6), Color(0xFF5B8CFF))
}

private val Dark = darkColorScheme(
    primary = Color(0xFF36C9B6),
    onPrimary = Color(0xFF00201C),
    secondary = Color(0xFF5B8CFF),
    background = Color(0xFF0E1320),
    onBackground = Color(0xFFE7ECF6),
    surface = Color(0xFF141B2B),
    onSurface = Color(0xFFE7ECF6),
    surfaceVariant = Color(0xFF1A2439),
    onSurfaceVariant = Color(0xFF9FB0CC),
    surfaceContainer = Color(0xFF141B2B),
    surfaceContainerHigh = Color(0xFF1A2439),
    outline = Color(0xFF2D3B57),
    outlineVariant = Color(0xFF243049),
    error = Color(0xFFFF6B6B),
)

private val Light = lightColorScheme(
    primary = Color(0xFF0C8A7B),
    onPrimary = Color.White,
    secondary = Color(0xFF2F66FF),
    background = Color(0xFFEEF1F7),
    onBackground = Color(0xFF15223B),
    surface = Color(0xFFFFFFFF),
    onSurface = Color(0xFF15223B),
    surfaceVariant = Color(0xFFEEF2F8),
    onSurfaceVariant = Color(0xFF43536F),
    surfaceContainer = Color(0xFFFFFFFF),
    surfaceContainerHigh = Color(0xFFF2F5FB),
    outline = Color(0xFFC8D2E2),
    outlineVariant = Color(0xFFDCE3EE),
)

private val NexType = Typography(
    titleLarge = TextStyle(fontWeight = FontWeight.Bold, fontSize = 22.sp),
    titleMedium = TextStyle(fontWeight = FontWeight.SemiBold, fontSize = 16.sp, lineHeight = 21.sp),
    bodyLarge = TextStyle(fontSize = 16.sp, lineHeight = 24.sp),
    bodyMedium = TextStyle(fontSize = 14.sp, lineHeight = 20.sp),
    labelSmall = TextStyle(fontSize = 11.sp, fontWeight = FontWeight.Medium),
)

@Composable
fun NexTheme(theme: String, content: @Composable () -> Unit) {
    val dark = when (theme) {
        "koyu" -> true
        "acik" -> false
        else -> isSystemInDarkTheme()
    }
    val extra = if (dark) NexExtra(Color(0xFFE23B4E), Color(0xFF6D7F9E), Color(0x1A9FB0CC), Color(0x3836C9B6), Color(0xFF5B8CFF))
    else NexExtra(Color(0xFFC42638), Color(0xFF6A7890), Color(0x1443536F), Color(0x2E0C8A7B), Color(0xFF2F66FF))
    CompositionLocalProvider(LocalNex provides extra) {
        MaterialTheme(colorScheme = if (dark) Dark else Light, typography = NexType, content = content)
    }
}
