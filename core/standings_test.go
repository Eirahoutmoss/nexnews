package core

import (
	"testing"
	"time"
)

const wikiFixture = `<div><p>intro</p>
<table class="wikitable"><tr><th>Round</th><th>1</th></tr><tr><td>A</td><td>1</td></tr></table>
<table class="wikitable" style="text-align:center;"><tbody>
<tr><th scope="col"><abbr title="Position">Pos</abbr></th><th scope="col">Team<div class="navbar plainlinks"><ul><li>v</li><li>t</li><li>e</li></ul></div></th>
<th scope="col"><abbr title="Played">Pld</abbr></th><th scope="col"><abbr title="Won">W</abbr></th><th scope="col"><abbr title="Lost">L</abbr></th>
<th scope="col"><abbr title="Points for">PF</abbr></th><th scope="col"><abbr title="Points against">PA</abbr></th><th scope="col"><abbr title="Point difference">PD</abbr></th>
<th scope="col"><abbr title="Points">Pts</abbr></th><th scope="col">Qualification or relegation</th></tr>
<tr><th scope="row">1</th><td style="text-align:left"><a href="/wiki/F">Fenerbahçe Beko</a> <b>(C)</b></td><td>30</td><td>25</td><td>5</td><td>2644</td><td>2393</td><td>+251</td><td>55</td><td rowspan="3">Advance to <a>playoffs</a></td></tr>
<tr><th scope="row">2</th><td><a>Beşiktaş Gain</a><sup class="reference">[a]</sup></td><td>30</td><td>25</td><td>5</td><td>2602</td><td>2304</td><td>+298</td><td>55</td></tr>
<tr><th scope="row">3</th><td>Bahçeşehir Koleji</td><td>30</td><td>21</td><td>9</td><td>2495</td><td>2349</td><td>+146</td><td>51</td></tr>
<tr><th scope="row">4</th><td>Mersin <span class="sortkey" style="display:none">x</span>Spor</td><td>30</td><td>4</td><td>26</td><td>2200</td><td>2600</td><td>&#8722;400</td><td>34</td><td>Relegation to TBL</td></tr>
</tbody></table>
<p>Updated to match(es) played on 6 September 2026. Source: TBF</p></div>`

func TestParseWikiStandings(t *testing.T) {
	rows, asOf, err := parseWikiStandings(wikiFixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("rows=%d %+v", len(rows), rows)
	}
	if rows[0].Team != "Fenerbahçe Beko" || rows[0].Points != 55 || rows[0].Diff != 251 || rows[0].Note != "Play-off" {
		t.Fatalf("row0 %+v", rows[0])
	}
	if rows[1].Team != "Beşiktaş Gain" || rows[1].Note != "Play-off" {
		t.Fatalf("row1 %+v", rows[1])
	}
	if rows[3].Diff != -400 || rows[3].Note != "Küme düşme" || rows[3].Team != "Mersin Spor" {
		t.Fatalf("row3 %+v", rows[3])
	}
	if asOf != "6 Eylül 2026" {
		t.Fatalf("asOf %q", asOf)
	}
}

func TestParseESPNStandings(t *testing.T) {
	body := []byte(`{"season":{"year":2026},"children":[{"standings":{"entries":[
	{"team":{"displayName":"Fenerbahce"},"stats":[{"name":"rank","value":2},{"name":"points","value":10},{"name":"gamesPlayed","value":5},{"name":"ties","value":1}]},
	{"team":{"displayName":"Galatasaray"},"note":{"description":"Champions League"},"stats":[{"name":"rank","value":1},{"name":"points","value":13},{"name":"gamesPlayed","value":5},{"name":"pointDifferential","value":7},{"name":"deductions"}]}]}}]}`)
	rows, season, err := parseESPNStandings(body)
	if err != nil {
		t.Fatal(err)
	}
	if season != "2026–27" || rows[0].Team != "Galatasaray" || rows[0].Note != "Şampiyonlar Ligi" || rows[1].Team != "Fenerbahçe" || rows[1].Drawn != 1 {
		t.Fatalf("%s %+v", season, rows)
	}
}

func TestSeasonStartYear(t *testing.T) {
	if seasonStartYear(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)) != 2026 || seasonStartYear(time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)) != 2026 {
		t.Fatal("season")
	}
	if seasonLabel(2026) != "2026–27" {
		t.Fatal(seasonLabel(2026))
	}
}

func TestMarketTag(t *testing.T) {
	cases := map[string]string{
		"Borsa güne yükselişle başladı":         "Borsa",
		"Merkez Bankası faiz kararını açıkladı": "Faiz",
		"Dolar/TL rekor kırdı":                  "Kur",
		"Gram altın 6.500 lirayı aştı":          "Altın",
		"Galatasaray deplasmanda kazandı":       "",
		"Konser sonrası sponsorlar açıklandı":   "",
		"Fed holds interest rates steady":       "Faiz",
	}
	for in, want := range cases {
		if got := marketTag(in); got != want {
			t.Errorf("%q -> %q, beklenen %q", in, got, want)
		}
	}
}
