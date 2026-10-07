package com.eirahoutmoss.nexnews

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

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
    val urls: List<String> = emptyList(),
    val official: String = "",
    val officialOnly: Boolean = false,
) {
    val streams: List<String> get() = (if (urls.isNotEmpty()) urls else listOf(url)).filter { it.isNotBlank() }
}

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
data class FinanceBoard(val ok: Boolean = false, val updated: Long = 0, val quotes: List<Quote> = emptyList(), val error: String = "")

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
)

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
