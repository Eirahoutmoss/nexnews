package com.eirahoutmoss.nexnews

import android.content.Context
import com.eirahoutmoss.nexnews.go.nexcore.Nexcore
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonObject
import java.util.Locale

/**
 * Go çekirdeğine (gomobile: Nexcore) tek giriş noktası. Ağ ya da disk işi yapan
 * her çağrı IO iş parçacığında çalışır; arayüz StateFlow'ları dinler.
 */
object Core {
    private val initLock = Mutex()
    @Volatile private var ready = false
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val _overview = MutableStateFlow<Overview?>(null)
    val overview: StateFlow<Overview?> = _overview.asStateFlow()

    private val _refreshing = MutableStateFlow(false)
    val refreshing: StateFlow<Boolean> = _refreshing.asStateFlow()

    private val _favLinks = MutableStateFlow<Set<String>>(emptySet())
    val favLinks: StateFlow<Set<String>> = _favLinks.asStateFlow()

    private val _readLinks = MutableStateFlow<Set<String>>(emptySet())
    val readLinks: StateFlow<Set<String>> = _readLinks.asStateFlow()

    suspend fun ensureInit(context: Context) {
        if (ready) return
        initLock.withLock {
            if (ready) return
            withContext(Dispatchers.IO) {
                val dir = context.applicationContext.filesDir.absolutePath
                Nexcore.init(dir, Locale.getDefault().language)
                ready = true
                reloadLocal()
            }
        }
    }

    /** Ağa gitmeden önbellekteki veriyi yeniden okur. */
    fun reloadLocal() {
        if (!ready) return
        _overview.value = runCatching { NexJson.decodeFromString<Overview>(Nexcore.overview()) }.getOrNull()
        _favLinks.value = runCatching { NexJson.decodeFromString<List<String>>(Nexcore.favLinks()).toSet() }.getOrDefault(emptySet())
        _readLinks.value = runCatching { NexJson.decodeFromString<List<SavedItem>>(Nexcore.history()).map { it.link }.toSet() }.getOrDefault(emptySet())
    }

    /** Tüm kaynakları yeniler; çeviri açıksa en önemli başlıkları çevirir. */
    suspend fun refresh(context: Context, translate: Boolean = true) {
        ensureInit(context)
        if (_refreshing.value) return
        _refreshing.value = true
        try {
            withContext(Dispatchers.IO) {
                Nexcore.refresh()
                reloadLocal()
                if (translate && configBool("translate", true)) {
                    Nexcore.translate(250, 40)
                    reloadLocal()
                }
            }
        } finally {
            _refreshing.value = false
        }
    }

    fun refreshAsync(context: Context) {
        scope.launch { refresh(context) }
    }

    fun lastFetch(): Long = if (ready) Nexcore.lastFetch() else 0L

    /** Hiç başlık yok mu (ilk açılış ya da tüm kaynaklar başarısız)? */
    fun isEmpty(): Boolean = _overview.value?.categories?.all { it.items.isEmpty() } ?: true

    suspend fun category(name: String): CategoryPage = io { NexJson.decodeFromString<CategoryPage>(Nexcore.category(name)) }
    suspend fun story(link: String): StoryDetail = io { NexJson.decodeFromString<StoryDetail>(Nexcore.story(link)) }
    suspend fun search(q: String): SearchResult = io { NexJson.decodeFromString<SearchResult>(Nexcore.search(q)) }
    suspend fun article(link: String): Article = io { NexJson.decodeFromString<Article>(Nexcore.readArticle(link)) }
    suspend fun briefing(): Briefing = io { NexJson.decodeFromString<Briefing>(Nexcore.briefing()) }
    suspend fun favorites(): List<SavedItem> = io { NexJson.decodeFromString<List<SavedItem>>(Nexcore.favorites()) }
    suspend fun history(): List<SavedItem> = io { NexJson.decodeFromString<List<SavedItem>>(Nexcore.history()) }
    suspend fun radio(): RadioBoard = io { NexJson.decodeFromString<RadioBoard>(Nexcore.radio()) }
    suspend fun liveTV(): List<TvChannel> = io { NexJson.decodeFromString<List<TvChannel>>(Nexcore.liveTV()) }
    suspend fun weather(place: String): WeatherBoard = io { NexJson.decodeFromString<WeatherBoard>(Nexcore.weather(place)) }
    suspend fun finance(): FinanceBoard = io { NexJson.decodeFromString<FinanceBoard>(Nexcore.finance()) }
    suspend fun sports(): SportsBoard = io { NexJson.decodeFromString<SportsBoard>(Nexcore.sports()) }
    suspend fun sourceHealth(): List<SourceHealth> = io { NexJson.decodeFromString<List<SourceHealth>>(Nexcore.sourceHealth()) }
    suspend fun discover(site: String): DiscoverResult = io { NexJson.decodeFromString<DiscoverResult>(Nexcore.discover(site)) }
    suspend fun netStats(): NetStats = io { NexJson.decodeFromString<NetStats>(Nexcore.netStats()) }
    suspend fun clearHistory() = io { Nexcore.clearHistory(); reloadLocal() }
    suspend fun restoreAllSources(): Long = io { Nexcore.restoreAllSources() }
    fun liveHeaders(url: String): LiveHeaders =
        runCatching { NexJson.decodeFromString<LiveHeaders>(Nexcore.liveHeaders(url)) }.getOrDefault(LiveHeaders())
    fun addTraffic(bytes: Long) { if (ready && bytes > 0) Nexcore.addTraffic(bytes) }
    fun coreVersion(): String = runCatching { Nexcore.version() }.getOrDefault("?")

    private suspend fun <T> io(block: () -> T): T = withContext(Dispatchers.IO) { block() }

    // ---- favoriler / geçmiş / takip ------------------------------------------

    private fun saved(item: NewsItem) = NexJson.encodeToString(
        SavedItem(link = item.link, title = item.title, source = item.source, cat = item.category, ts = item.ts)
    )

    fun toggleFav(item: NewsItem) {
        scope.launch {
            val on = Nexcore.toggleFav(saved(item))
            _favLinks.value = if (on) _favLinks.value + item.link else _favLinks.value - item.link
        }
    }

    fun toggleFavSaved(item: SavedItem) {
        scope.launch {
            val on = Nexcore.toggleFav(NexJson.encodeToString(item))
            _favLinks.value = if (on) _favLinks.value + item.link else _favLinks.value - item.link
        }
    }

    fun markRead(item: NewsItem) {
        if (item.link in _readLinks.value) return
        _readLinks.value = _readLinks.value + item.link
        scope.launch { Nexcore.addHistory(saved(item)) }
    }

    suspend fun toggleFollow(eventId: String): Boolean = io { Nexcore.toggleFollowEvent(eventId) }

    // ---- yapılandırma (Go config.json, alan kaybetmeden düzenlenir) ----------

    fun config(): JsonObject =
        runCatching { NexJson.parseToJsonElement(Nexcore.config()).jsonObject }.getOrDefault(JsonObject(emptyMap()))

    fun configBool(key: String, def: Boolean): Boolean =
        runCatching { (config()[key] as? kotlinx.serialization.json.JsonPrimitive)?.content?.toBooleanStrictOrNull() }.getOrNull() ?: def

    suspend fun saveConfig(edit: (MutableMap<String, kotlinx.serialization.json.JsonElement>) -> Unit) = io {
        val m = config().toMutableMap()
        edit(m)
        Nexcore.saveConfig(JsonObject(m).toString())
        reloadLocal()
    }
}
