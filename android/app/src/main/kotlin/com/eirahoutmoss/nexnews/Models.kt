package com.eirahoutmoss.nexnews

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull

/** Go çekirdeğinin JSON çıktılarının Kotlin karşılıkları (bilinmeyen alanlar yok sayılır). */
val NexJson = Json {
    ignoreUnknownKeys = true
    isLenient = true
    coerceInputValues = true
    explicitNulls = false
    encodeDefaults = true
}

@Serializable
data class SourceEvidence(
    val name: String = "",
    val domain: String = "",
    val type: String = "",
    val independent: Boolean = true,
    val primary: Boolean = false,
    val via: String = "",
)

@Serializable
data class Evidence(
    val independentSources: Int = 0,
    val primarySources: Int = 0,
    val agreement: Int = 0,
    val sources: List<SourceEvidence> = emptyList(),
    val summary: String = "",
    val syndicated: Int = 0,
    val agencies: List<String> = emptyList(),
)

@Serializable
data class NewsItem(
    val title: String = "",
    val link: String = "",
    val source: String = "",
    val sources: List<String> = emptyList(),
    val sourceCount: Int = 0,
    val count: Int = 1,
    val category: String = "",
    val ts: Long = 0,
    val breaking: Boolean = false,
    val score: Int = 0,
    val kind: String = "",
    val summary: String = "",
    val eventId: String = "",
    val independentSources: Int = 0,
    val confidence: Int = 0,
    val lifecycle: String = "",
    val reason: String = "",
    val followed: Boolean = false,
    val image: String = "",
    val syndicated: Int = 0,
    val read: Boolean = false,
)

@Serializable
data class CategoryBlock(
    val name: String = "",
    val total: Int = 0,
    val unread: Int = 0,
    val items: List<NewsItem> = emptyList(),
)

@Serializable
data class Overview(
    val countryCode: String = "",
    val countryName: String = "",
    val categories: List<CategoryBlock> = emptyList(),
    val breaking: List<NewsItem> = emptyList(),
    val important: List<NewsItem> = emptyList(),
    val forYou: List<NewsItem> = emptyList(),
    val updatedTs: Long = 0,
    val loading: Boolean = false,
    val pulse: Int = 0,
    val pulseBand: String = "",
    val discover: NewsItem? = null,
    val discoverReason: String = "",
    val totalUnread: Int = 0,
)

@Serializable
data class CategoryPage(
    val name: String = "",
    val total: Int = 0,
    val items: List<NewsItem> = emptyList(),
)

@Serializable
data class TimelineItem(val title: String = "", val source: String = "", val link: String = "", val ts: Long = 0)

@Serializable
data class Headline(
    val source: String = "",
    val title: String = "",
    val translated: String = "",
    val link: String = "",
    val ts: Long = 0,
    val via: String = "",
    val distinct: List<String> = emptyList(),
)

@Serializable
data class StoryDetail(
    val found: Boolean = false,
    val title: String = "",
    val category: String = "",
    val kind: String = "",
    val score: Int = 0,
    val sources: List<String> = emptyList(),
    val sourceCount: Int = 0,
    val count: Int = 0,
    val firstTs: Long = 0,
    val lastTs: Long = 0,
    val summary: String = "",
    val timeline: List<TimelineItem> = emptyList(),
    val relLevel: String = "",
    val relText: String = "",
    val eventId: String = "",
    val independentSources: Int = 0,
    val primarySources: Int = 0,
    val confidence: Int = 0,
    val lifecycle: String = "",
    val reason: String = "",
    val evidence: Evidence = Evidence(),
    val followed: Boolean = false,
    val headlines: List<Headline> = emptyList(),
)

@Serializable
data class SearchResult(
    val query: String = "",
    val windowLabel: String = "",
    val total: Int = 0,
    val sourceCount: Int = 0,
    val items: List<NewsItem> = emptyList(),
)

@Serializable
data class Article(val title: String = "", val html: String = "", val url: String = "", val err: String = "")

@Serializable
data class SavedItem(
    val link: String = "",
    val title: String = "",
    val source: String = "",
    val cat: String = "",
    val ts: Long = 0,
    val added: Long = 0,
)

@Serializable
data class Alert(
    val title: String = "",
    val link: String = "",
    val source: String = "",
    val category: String = "",
    val breaking: Boolean = false,
    val score: Int = 0,
    val sources: Int = 0,
    val followed: Boolean = false,
)

@Serializable
data class BriefingItem(val rank: Int = 0, val title: String = "", val category: String = "", val link: String = "")

@Serializable
data class Briefing(
    val generatedAt: Long = 0,
    val windowLabel: String = "",
    val headline: String = "",
    val summary: String = "",
    val items: List<BriefingItem> = emptyList(),
)

@Serializable
data class RadioStation(
    val uuid: String = "",
    val name: String = "",
    val country: String = "",
    val countryCode: String = "",
    val language: String = "",
    val tags: String = "",
    val continent: String = "",
    val homepage: String = "",
    val stream: String = "",
    val favicon: String = "",
)

@Serializable
data class RadioBoard(
    val localCountry: String = "",
    val stations: List<RadioStation> = emptyList(),
    val error: String = "",
)

@Serializable
data class TvChannel(
    val id: String = "",
    val name: String = "",
    val meta: String = "",
    val continent: String = "",
    val country: String = "",
    val url: String = "",
    /** Düz adres ya da {u, r, ua} nesnesi (Referer/User-Agent isteyen yayınlar). */
    val urls: List<JsonElement> = emptyList(),
    val official: String = "",
    val officialOnly: Boolean = false,
    val kind: String = "",
    val geo: String = "",
    val note: String = "",
    val tvg: String = "",
    val rep: String = "",
    val custom: Boolean = false,
) {
    val candidates: List<LiveCandidate>
        get() = (if (urls.isNotEmpty()) urls.mapNotNull { e ->
            when (e) {
                is JsonPrimitive -> e.contentOrNull?.let { LiveCandidate(u = it) }
                is JsonObject -> runCatching { NexJson.decodeFromJsonElement(LiveCandidate.serializer(), e) }.getOrNull()
                else -> null
            }
        } else listOf(LiveCandidate(u = url))).filter { it.u.isNotBlank() }
    val streams: List<String> get() = candidates.map { it.u }

    /** Kategori: listedeki alan, yoksa açıklamadan tahmin. */
    val kindOrGuess: String
        get() {
            if (kind.isNotBlank()) return if (kind == "egitim") "cocuk" else kind
            val m = meta.lowercase()
            return when {
                Regex("spor|sport|yarış").containsMatchIn(m) -> "spor"
                Regex("ekonomi|finans|business").containsMatchIn(m) -> "ekonomi"
                Regex("belgesel|doğa|tarih|bilim|uzay|tarım").containsMatchIn(m) -> "belgesel"
                Regex("kültür|sanat").containsMatchIn(m) -> "kultur"
                Regex("haber|news|parlamento|meclis|uluslararası").containsMatchIn(m) -> "haber"
                Regex("müzik|music").containsMatchIn(m) -> "muzik"
                Regex("çocuk|eğitim").containsMatchIn(m) -> "cocuk"
                else -> "genel"
            }
        }
}

@Serializable
data class LiveCandidate(val u: String = "", val r: String = "", val ua: String = "")

@Serializable
data class LiveHealth(val ok: Boolean = false, val url: String = "", val reason: String = "", val code: Int = 0, val ms: Long = 0, val ts: Long = 0, val repair: LiveCandidate? = null)

@Serializable
data class LiveState(
    val favs: List<String> = emptyList(),
    val last: String = "",
    val hidden: List<String> = emptyList(),
    val custom: List<TvChannel> = emptyList(),
    val check: String = "ask",
)

@Serializable
data class FollowedEvent(val item: NewsItem = NewsItem(), val live: Boolean = false, val new: Int = 0, val added: Long = 0, val lastSeen: Long = 0)

@Serializable
data class SmartAlert(
    val id: String = "", val eventId: String = "", val eventTitle: String = "", val kind: String = "", val severity: String = "",
    val title: String = "", val message: String = "", val score: Int = 0, val confidence: Int = 0, val sources: Int = 0, val risk: Int = 0,
    val link: String = "", val ts: Long = 0,
)

@Serializable
data class SmartAlertBoard(val updated: Long = 0, val total: Int = 0, val critical: Int = 0, val warning: Int = 0, val info: Int = 0, val alerts: List<SmartAlert> = emptyList())

@Serializable
data class UsagePeriod(val down: Long = 0, val up: Long = 0, val secs: Long = 0)

@Serializable
data class UsageDay(val day: String = "", val down: Long = 0, val secs: Long = 0)

@Serializable
data class UsageStats(
    val mode: String = "tam", val limitKBps: Int = 256, val quotaMB: Int = 0,
    val session: UsagePeriod = UsagePeriod(), val today: UsagePeriod = UsagePeriod(), val week: UsagePeriod = UsagePeriod(),
    val month: UsagePeriod = UsagePeriod(), val last30: UsagePeriod = UsagePeriod(), val all: UsagePeriod = UsagePeriod(),
    val since: Long = 0, val dailyAvg: Long = 0, val days: List<UsageDay> = emptyList(), val quotaUsed: Double = 0.0, val autoLimited: Boolean = false,
)

@Serializable
data class LiveHeaders(val referer: String = "", val origin: String = "", val userAgent: String = "")

@Serializable
data class WeatherStep(
    val label: String = "",
    val ts: Long = 0,
    val temp: Double = 0.0,
    val icon: String = "",
    val desc: String = "",
    val precip: Double = 0.0,
    val precipProb: Int = 0,
    val wind: Double = 0.0,
    val dirName: String = "",
)

@Serializable
data class WeatherBoard(
    val ok: Boolean = false,
    val err: String = "",
    val city: String = "",
    val models: Int = 0,
    val now: WeatherStep = WeatherStep(),
    val steps: List<WeatherStep> = emptyList(),
    val rainText: String = "",
    val windText: String = "",
    @SerialName("thunderText") val thunderText: String = "",
    val confidence: String = "",
)

@Serializable
data class Quote(
    val symbol: String = "",
    val name: String = "",
    val price: Double = 0.0,
    val changePct: Double = 0.0,
    val change5dPct: Double = 0.0,
    val currency: String = "",
    val history: List<Double> = emptyList(),
    val error: String = "",
)

@Serializable
data class FinanceBoard(val ok: Boolean = false, val updated: Long = 0, val quotes: List<Quote> = emptyList(), val gold: List<GoldQuote> = emptyList(), val news: List<MarketNews> = emptyList(), val error: String = "")

@Serializable
data class MarketNews(val title: String = "", val link: String = "", val source: String = "", val published: Long = 0, val tag: String = "")

@Serializable
data class GoldQuote(val name: String = "", val valueTry: Double = 0.0, val change5dPct: Double = 0.0, val note: String = "")

@Serializable
data class Game(
    val id: String = "",
    val leagueName: String = "",
    val home: String = "",
    val away: String = "",
    val homeScore: String = "",
    val awayScore: String = "",
    val status: String = "",
    val statusDetail: String = "",
    val start: Long = 0,
    val fav: Boolean = false,
)

@Serializable
data class StandingRow(
    val pos: Int = 0, val team: String = "", val played: Int = 0, val won: Int = 0, val drawn: Int = 0,
    val lost: Int = 0, val diff: Int = 0, val points: Int = 0, val note: String = "",
)

@Serializable
data class StandingTable(
    val key: String = "", val title: String = "", val season: String = "", val source: String = "",
    val sourceUrl: String = "", val asOf: String = "", val hasDraws: Boolean = false,
    val rows: List<StandingRow> = emptyList(), val error: String = "",
)

@Serializable
data class StandingsBoard(val ok: Boolean = false, val updated: Long = 0, val tables: List<StandingTable> = emptyList(), val favorites: List<String> = emptyList())

@Serializable
data class SportsBoard(val ok: Boolean = false, val games: List<Game> = emptyList(), val favorites: List<String> = emptyList(), val error: String = "")

@Serializable
data class SourceHealth(
    val url: String = "",
    val name: String = "",
    val status: String = "",
    val itemCount: Int = 0,
    val latencyMs: Long = 0,
    val lastError: String = "",
)

@Serializable
data class DiscoveredFeed(val url: String = "", val title: String = "")

@Serializable
data class DiscoverResult(val feeds: List<DiscoveredFeed> = emptyList(), val err: String = "")

@Serializable
data class NetStats(
    val appDownBps: Long = 0,
    val down5m: Long = 0,
    val up5m: Long = 0,
    val sessionDown: Long = 0,
    val sessionUp: Long = 0,
)
