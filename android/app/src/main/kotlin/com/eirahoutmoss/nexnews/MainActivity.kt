package com.eirahoutmoss.nexnews

import android.Manifest
import android.content.Intent
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.filled.Bookmarks
import androidx.compose.material.icons.filled.Explore
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.lifecycleScope
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.eirahoutmoss.nexnews.notify.Notifier
import com.eirahoutmoss.nexnews.ui.CategoriesScreen
import com.eirahoutmoss.nexnews.ui.CategoryScreen
import com.eirahoutmoss.nexnews.ui.DiscoverScreen
import com.eirahoutmoss.nexnews.ui.FeedScreen
import com.eirahoutmoss.nexnews.ui.MiniPlayer
import com.eirahoutmoss.nexnews.ui.NexNav
import com.eirahoutmoss.nexnews.ui.NexTheme
import com.eirahoutmoss.nexnews.ui.PlayerScreen
import com.eirahoutmoss.nexnews.ui.ReaderScreen
import com.eirahoutmoss.nexnews.ui.SavedScreen
import com.eirahoutmoss.nexnews.ui.SearchScreen
import com.eirahoutmoss.nexnews.ui.SettingsScreen
import com.eirahoutmoss.nexnews.ui.SourcesScreen
import com.eirahoutmoss.nexnews.ui.StoryScreen
import com.eirahoutmoss.nexnews.ui.WeeklyScreen
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.unit.LayoutDirection
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {
    /** Bildirimden ya da widget'tan gelen haber bağlantısı. */
    private val pendingLink = MutableStateFlow<String?>(null)

    private val notifPermission = registerForActivityResult(ActivityResultContracts.RequestPermission()) { }

    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        pendingLink.value = intent?.getStringExtra(Notifier.EXTRA_LINK)

        val prefs = Prefs.get(this)
        if (Build.VERSION.SDK_INT >= 33 && !prefs.askedNotificationPermission) {
            prefs.askedNotificationPermission = true
            notifPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
        }

        lifecycleScope.launch {
            Core.ensureInit(this@MainActivity)
            // Açılışta veri 10 dakikadan eskiyse yenile. Hiç başlık gelmezse
            // (ör. ağ yeni bağlanıyorsa) artan aralıklarla birkaç kez yeniden dene.
            val age = System.currentTimeMillis() / 1000 - Core.lastFetch()
            if (age > 600 || Core.isEmpty()) Core.refresh(this@MainActivity)
            for (wait in listOf(15_000L, 30_000L, 60_000L)) {
                if (!Core.isEmpty()) break
                kotlinx.coroutines.delay(wait)
                if (Core.isEmpty()) Core.refresh(this@MainActivity)
            }
        }

        setContent {
            val pv by prefs.version.collectAsState()
            val theme = remember(pv) { prefs.theme }
            // Dil değişince (I18n.version) yeniden çizilir; Arapça sağdan sola.
            val lv = I18n.version
            val dir = remember(lv) { if (I18n.rtl) LayoutDirection.Rtl else LayoutDirection.Ltr }
            // RC97 kolay okuma: ayar açıksa tüm yazılar büyür.
            val ov by Core.overview.collectAsState()
            val easy = remember(ov) { Core.configBool("easyRead", false) }
            val dens = androidx.compose.ui.platform.LocalDensity.current
            val scaled = remember(dens, easy) { androidx.compose.ui.unit.Density(dens.density, dens.fontScale * (if (easy) 1.25f else 1f)) }
            CompositionLocalProvider(LocalLayoutDirection provides dir, androidx.compose.ui.platform.LocalDensity provides scaled) {
                NexTheme(theme) { AppRoot(pendingLink) }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        intent.getStringExtra(Notifier.EXTRA_LINK)?.let { pendingLink.value = it }
    }
}

private data class Tab(val route: String, val label: String, val icon: ImageVector)

private val tabs = listOf(
    Tab("feed", "Akış", Icons.Filled.Home),
    Tab("categories", "Kategoriler", Icons.AutoMirrored.Filled.List),
    Tab("discover", "Keşfet", Icons.Filled.Explore),
    Tab("saved", "Kaydedilen", Icons.Filled.Bookmarks),
    Tab("settings", "Ayarlar", Icons.Filled.Settings),
)

@Composable
private fun AppRoot(pendingLink: MutableStateFlow<String?>) {
    val ctx = LocalContext.current
    val navController = rememberNavController()
    val nav = remember(navController) { NexNav(navController, ctx) }
    val entry by navController.currentBackStackEntryAsState()
    val route = entry?.destination?.route ?: "feed"
    val link by pendingLink.collectAsState()

    LaunchedEffect(link) {
        val l = link ?: return@LaunchedEffect
        Core.ensureInit(ctx)
        val s = runCatching { Core.story(l) }.getOrNull()
        if (s != null && s.found && s.sourceCount > 1) nav.story(l) else nav.article(l)
        pendingLink.value = null
    }

    val showBottom = tabs.any { it.route == route }
    Scaffold(
        bottomBar = {
            Column {
                if (route != "player") MiniPlayer(onOpenVideo = { nav.player() })
                if (showBottom) NavigationBar(containerColor = MaterialTheme.colorScheme.surface) {
                    tabs.forEach { tb ->
                        NavigationBarItem(
                            selected = route == tb.route,
                            onClick = {
                                navController.navigate(tb.route) {
                                    popUpTo(navController.graph.findStartDestination().id) { saveState = true }
                                    launchSingleTop = true
                                    restoreState = true
                                }
                            },
                            icon = { Icon(tb.icon, t(tb.label)) },
                            label = { Text(t(tb.label), maxLines = 1) }
                        )
                    }
                }
            }
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { inner ->
        // Alt boşluk her ekranın kendi listesine verilir; üst boşluğu ekranlar kendisi yönetir.
        val bottom = PaddingValues(bottom = inner.calculateBottomPadding())
        NavHost(navController, startDestination = "feed") {
            composable("feed") { FeedScreen(nav, bottom) }
            composable("categories") { CategoriesScreen(nav, bottom) }
            composable("discover") { DiscoverScreen(nav, bottom) }
            composable("saved") { SavedScreen(nav, bottom) }
            composable("settings") { SettingsScreen(nav, bottom) }
            composable("search") { SearchScreen(nav, bottom) }
            composable("sources") { SourcesScreen(nav, bottom) }
            composable("player") { PlayerScreen(nav) }
            composable("weekly") { WeeklyScreen(nav, bottom) }
            composable("agenda") { com.eirahoutmoss.nexnews.ui.AgendaScreen(nav, bottom) }
            composable("mirror") { com.eirahoutmoss.nexnews.ui.MirrorScreen(nav, bottom) }
            composable("category/{name}", arguments = listOf(navArgument("name") { type = NavType.StringType })) {
                CategoryScreen(it.arguments?.getString("name") ?: "", nav, bottom)
            }
            composable("story/{link}", arguments = listOf(navArgument("link") { type = NavType.StringType })) {
                StoryScreen(it.arguments?.getString("link") ?: "", nav, bottom)
            }
            composable("reader/{link}", arguments = listOf(navArgument("link") { type = NavType.StringType })) {
                ReaderScreen(it.arguments?.getString("link") ?: "", nav, bottom)
            }
        }
    }
}

