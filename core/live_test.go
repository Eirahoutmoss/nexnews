package core

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// Canlı kaynak kontrolü: yalnızca NEXNEWS_LIVE=1 iken çalışır (CI'da ayrı adım).
func TestLiveStandings(t *testing.T) {
	if os.Getenv("NEXNEWS_LIVE") != "1" {
		t.Skip("canlı test kapalı")
	}
	dataDirOverride = t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	var b standingsBoard
	if err := json.Unmarshal(standingsJSON(ctx, nil), &b); err != nil {
		t.Fatal(err)
	}
	for _, tb := range b.Tables {
		t.Logf("%s %s · %s · %s asOf=%q err=%q satır=%d", tb.Key, tb.Title, tb.Season, tb.Source, tb.AsOf, tb.Error, len(tb.Rows))
		for _, r := range tb.Rows {
			t.Logf("  %2d %-26s O%-2d G%-2d B%-2d M%-2d %d:%d (%+d) P%d %s", r.Pos, r.Team, r.Played, r.Won, r.Drawn, r.Lost, r.For, r.Against, r.Diff, r.Points, r.Note)
		}
		if len(tb.Rows) == 0 {
			t.Errorf("%s tablosu boş", tb.Key)
		}
	}
}

func TestLiveFinance(t *testing.T) {
	if os.Getenv("NEXNEWS_LIVE") != "1" {
		t.Skip("canlı test kapalı")
	}
	c := defaultConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	var b financeBoard
	if err := json.Unmarshal(financeJSON(ctx, c), &b); err != nil {
		t.Fatal(err)
	}
	for _, q := range b.Quotes {
		t.Logf("%-10s %12.2f %s  gün %+.2f%%  5g %+.2f%% err=%q", q.Symbol, q.Price, q.Currency, q.ChangePct, q.Change5dPct, q.Error)
	}
}

func TestLiveSports(t *testing.T) {
	if os.Getenv("NEXNEWS_LIVE") != "1" {
		t.Skip("canlı test kapalı")
	}
	c := defaultConfig()
	for _, k := range []string{"futbol", "premier", "ucl", "nba"} {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		g, err := espnScoreboard(ctx, k, sportsEndpoints[k])
		cancel()
		t.Logf("%s: %d maç err=%v", k, len(g), err)
		for i, x := range g {
			if i >= 4 {
				break
			}
			t.Logf("   %s %s %s-%s %s [%s %s]", time.Unix(x.Start, 0).UTC().Format("01-02 15:04"), x.Home, x.HomeScore, x.AwayScore, x.Away, x.Status, x.StatusDetail)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	var b sportsBoard
	_ = json.Unmarshal(sportsJSON(ctx, c), &b)
	t.Logf("pano: %d maç, ilk 8:", len(b.Games))
	for i, x := range b.Games {
		if i >= 8 {
			break
		}
		t.Logf("   fav=%v %s %s vs %s [%s %s]", x.Fav, x.LeagueName, x.Home, x.Away, x.Status, x.StatusDetail)
	}
}
