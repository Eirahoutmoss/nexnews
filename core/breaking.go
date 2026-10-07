package core

// Breaking / emergency detection — language-independent by design. A headline is
// flagged "breaking" when either (a) it carries an explicit breaking marker in
// any of several languages, or (b) it mentions a hazard term AND is recent. This
// is a heuristic, not an oracle: it is tuned to favor real national/international
// emergencies (earthquake, tsunami, eruption, evacuation…) while keeping obvious
// everyday news out. Authoritative alert feeds (GDACS, USGS) are also carried as
// sources, and their titles naturally match these terms.

import (
	"strings"
	"time"
)

// Explicit "breaking news" markers across languages. Presence alone is a strong
// signal.
var breakingMarkers = []string{
	// Turkish
	"son dakika", "flaş", "flas haber", "acil",
	// English
	"breaking", "urgent", "just in", "developing",
	// Spanish / Portuguese
	"última hora", "ultima hora", "urgente", "de última",
	// Italian
	"ultima ora", "ultim'ora",
	// French
	"dernière minute", "derniere minute", "alerte info",
	// German
	"eilmeldung",
	// Japanese / Chinese / Korean / Arabic / Russian
	"速報", "緊急", "紧急", "속보", "عاجل", "срочно", "молния",
}

// Hazard / emergency terms across languages. A match counts only together with
// recency.
var hazardTerms = []string{
	// quakes
	"earthquake", "deprem", "terremoto", "séisme", "seisme", "sismo", "erdbeben",
	"地震", "지진", "землетрясение", "magnitude", "büyüklüğünde deprem",
	// tsunami (near-universal spelling)
	"tsunami", "津波",
	// volcano / eruption
	"volcano", "volcanic", "eruption", "yanardağ", "volkan", "püskür",
	"vulcano", "volcán", "volcan", "火山", "噴火",
	// flood
	"flood", "flash flood", "sel", "su baskını", "inundación", "inundacao",
	"alluvione", "inondation", "hochwasser", "洪水", "наводнение",
	// wildfire
	"wildfire", "yangın", "orman yangını", "incendio", "incêndio", "incendie",
	"waldbrand", "山火事",
	// storms
	"hurricane", "typhoon", "cyclone", "kasırga", "tayfun", "furacão", "台風",
	// evacuation / emergency
	"evacuat", "tahliye", "evacuación", "evacuacao", "evacuazione", "évacuation",
	"避難", "state of emergency", "olağanüstü hal", "emergency",
	// landslide / collapse
	"landslide", "heyelan", "göçük", "deslizamento", "frana", "glissement",
	"山崩れ", "土砂崩れ",
	// explosion / attack (high-signal, kept minimal)
	"explosion", "patlama", "explosão", "explosión", "esplosione", "爆発",
}

// classifyBreaking decides whether a headline is breaking/emergency.
func classifyBreaking(title string, published time.Time, hasDate bool) bool {
	low := strings.ToLower(title)

	strong := containsAny(low, breakingMarkers)
	hazard := containsAny(low, hazardTerms)

	recent := hasDate && time.Since(published) <= 6*time.Hour

	// Explicit marker: breaking if recent, or if the feed gave no date at all
	// (many "breaking" feeds omit timestamps).
	if strong && (recent || !hasDate) {
		return true
	}
	// Hazard term needs recency to avoid flagging retrospectives/analysis.
	if hazard && recent {
		return true
	}
	return false
}

func containsAny(s string, terms []string) bool {
	for _, t := range terms {
		if t != "" && strings.Contains(s, t) {
			return true
		}
	}
	return false
}
