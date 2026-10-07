package core

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// Smart alerts are explainable state-change signals derived from the same
// event/prediction data shown elsewhere in the UI. They do not claim to
// predict facts; they identify conditions that deserve immediate attention.
type smartAlert struct {
	ID         string `json:"id"`
	EventID    string `json:"eventId"`
	EventTitle string `json:"eventTitle"`
	Kind       string `json:"kind"`
	Severity   string `json:"severity"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	Score      int    `json:"score"`
	Confidence int    `json:"confidence"`
	Sources    int    `json:"sources"`
	Risk       int    `json:"risk"`
	Link       string `json:"link"`
	TS         int64  `json:"ts"`
}

type smartAlertBoard struct {
	Updated  int64        `json:"updated"`
	Total    int          `json:"total"`
	Critical int          `json:"critical"`
	Warning  int          `json:"warning"`
	Info     int          `json:"info"`
	Alerts   []smartAlert `json:"alerts"`
}

func addSmartAlert(out *[]smartAlert, seen map[string]bool, st *Story, kind, severity, title, message string, risk int) {
	if st == nil || st.EventID == "" || seen[st.EventID+"|"+kind] {
		return
	}
	seen[st.EventID+"|"+kind] = true
	*out = append(*out, smartAlert{
		ID: st.EventID + "-" + kind, EventID: st.EventID, EventTitle: st.Title, Kind: kind,
		Severity: severity, Title: title, Message: message,
		Score: st.Score, Confidence: st.Confidence, Sources: st.IndependentSources,
		Risk: risk, Link: st.Link, TS: st.Published.Unix(),
	})
}

func (s *Store) smartAlertsJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	board := smartAlertBoard{Updated: time.Now().Unix()}
	seen := map[string]bool{}
	var alerts []smartAlert
	candidates := append([]*Story{}, s.important...)
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })

	for _, st := range candidates {
		if st == nil || st.EventID == "" || s.isHidden(st) {
			continue
		}
		p := predictiveForStory(st)
		if st.Breaking {
			addSmartAlert(&alerts, seen, st, "breaking", "critical", "SON DAKİKA", "Yeni bir son dakika sinyali oluştu.", alertRisk(p.Risk, 85))
		}
		if p.Direction == "escalating" && p.Risk >= 60 {
			addSmartAlert(&alerts, seen, st, "escalating", "critical", "OLAY TIRMANIYOR", "Haber yoğunluğu ve olay sinyalleri yükseliyor.", p.Risk)
		}
		if st.IndependentSources >= 3 && st.Confidence >= 60 {
			addSmartAlert(&alerts, seen, st, "corroborated", "warning", "DOĞRULAMA GÜÇLENDİ", "Olay birden fazla bağımsız kaynak tarafından destekleniyor.", p.Risk)
		}
		if p.Risk >= 80 && !strings.EqualFold(p.Direction, "declining") {
			addSmartAlert(&alerts, seen, st, "risk", "warning", "YÜKSEK TAKİP RİSKİ", "Önem, güven, kaynak ve hız sinyalleri birlikte yüksek.", p.Risk)
		}
		if p.Direction == "emerging" && st.Score >= 60 {
			addSmartAlert(&alerts, seen, st, "emerging", "info", "YENİ SİNYAL YÜKSELİYOR", "Yeni olay sinyali hız kazanıyor ve yakından izlenmeye değer.", p.Risk)
		}
	}

	rank := map[string]int{"critical": 3, "warning": 2, "info": 1}
	sort.SliceStable(alerts, func(i, j int) bool {
		if rank[alerts[i].Severity] != rank[alerts[j].Severity] {
			return rank[alerts[i].Severity] > rank[alerts[j].Severity]
		}
		if alerts[i].Risk != alerts[j].Risk {
			return alerts[i].Risk > alerts[j].Risk
		}
		return alerts[i].TS > alerts[j].TS
	})
	if len(alerts) > 8 {
		alerts = alerts[:8]
	}
	for _, a := range alerts {
		switch a.Severity {
		case "critical":
			board.Critical++
		case "warning":
			board.Warning++
		default:
			board.Info++
		}
	}
	board.Total = len(alerts)
	board.Alerts = alerts
	return json.Marshal(board)
}

func alertRisk(a, b int) int {
	if a > b {
		return a
	}
	return b
}
