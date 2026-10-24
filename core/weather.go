package core

// Multi-model weather nowcast board. Location resolves from a user-typed city
// (Open-Meteo geocoding) or, failing that, from IP (several providers tried in
// turn). The forecast is pulled from Open-Meteo for FOUR models (ICON, GFS,
// ECMWF, GEM) and averaged, with model disagreement turned into a confidence
// label — this is the "birden fazla kaynaktan ortalama tahmin" the user asked
// for. Near-term (10–30 min) detail comes from the 15-minutely series.
//
// Honest scope: true minute-level lightning/cell tracking needs radar/lightning
// networks that keyless APIs don't fully expose; thunder here is an ESTIMATE
// from CAPE + storm weather codes, labelled as such.

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var wxModels = []string{"icon_seamless", "gfs_seamless", "ecmwf_ifs025", "gem_seamless"}

type wbHour struct {
	TS     int64   `json:"ts"`
	Label  string  `json:"label"`
	Temp   float64 `json:"temp"`
	Icon   string  `json:"icon"`
	Desc   string  `json:"desc"`
	Precip float64 `json:"precip"`
	Prob   int     `json:"prob"`
	Wind   float64 `json:"wind"`
}

type wbStep struct {
	Label      string  `json:"label"`
	TS         int64   `json:"ts"`
	Temp       float64 `json:"temp"`
	Code       int     `json:"code"`
	Icon       string  `json:"icon"`
	Desc       string  `json:"desc"`
	Precip     float64 `json:"precip"`     // mm
	PrecipProb int     `json:"precipProb"` // %
	Wind       float64 `json:"wind"`       // km/h
	Gust       float64 `json:"gust"`
	DirFrom    int     `json:"dirFrom"` // degrees wind blows FROM
	DirName    string  `json:"dirName"`
	ToName     string  `json:"toName"` // compass it's heading TO
}

type weatherBoard struct {
	OK         bool     `json:"ok"`
	Err        string   `json:"err,omitempty"`
	City       string   `json:"city"`
	Models     int      `json:"models"`
	Updated    int64    `json:"updated"`
	Now        wbStep   `json:"now"`
	Steps      []wbStep `json:"steps"`
	RainText   string   `json:"rainText"`
	WindText   string   `json:"windText"`
	ThunderTxt string   `json:"thunderText"`
	ThunderLvl string   `json:"thunderLevel"` // yok/düşük/orta/yüksek
	Confidence string   `json:"confidence"`   // yüksek/orta/düşük

	// Absolute event times so the "X dk içinde" text can be recomputed live
	// instead of freezing inside the cached board.
	Raining      bool  `json:"-"`
	RainStartTS  int64 `json:"-"`
	RainEndTS    int64 `json:"-"`
	RainProb     int   `json:"-"`
	ThunderNow   bool  `json:"-"`
	ThunderStart int64 `json:"-"`
	ThunderEnd   int64 `json:"-"`

	// extended "intense" detail (single best model)
	Feels      float64  `json:"feels"`
	Humidity   int      `json:"humidity"`
	Pressure   int      `json:"pressure"`
	Visibility float64  `json:"visibility"` // km
	Cloud      int      `json:"cloud"`      // %
	UV         float64  `json:"uv"`
	UVMax      float64  `json:"uvMax"`
	Sunrise    string   `json:"sunrise"`
	Sunset     string   `json:"sunset"`
	DayHi      float64  `json:"dayHi"`
	DayLo      float64  `json:"dayLo"`
	PrecipSum  float64  `json:"precipSum"`
	Days       []wbDay  `json:"days"` // 5 günlük tahmin
	Hourly     []wbHour `json:"hourly"`
	AQI        float64  `json:"aqi"`
	AQILevel   string   `json:"aqiLevel"`
	PM25       float64  `json:"pm25"`
	PM10       float64  `json:"pm10"`
	Ozone      float64  `json:"ozone"`
	NO2        float64  `json:"no2"`
}

type wbDay struct {
	Label  string  `json:"label"`
	TS     int64   `json:"ts"`
	Code   int     `json:"code"`
	Icon   string  `json:"icon"`
	Desc   string  `json:"desc"`
	Hi     float64 `json:"hi"`
	Lo     float64 `json:"lo"`
	Prob   int     `json:"prob"`
	Precip float64 `json:"precip"`
	Wind   float64 `json:"wind"`
}

var (
	wxMu     sync.Mutex
	wxCache  weatherBoard
	wxAt     time.Time
	wxClient = &http.Client{Timeout: 12 * time.Second}
)

type geoOut struct {
	OK   bool    `json:"ok"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	City string  `json:"city"`
}

var (
	geoMu sync.Mutex
	geoC  geoOut
	geoAt time.Time
)

// geoJSON resolves (and caches) the user's coordinates for the radar map.
func geoJSON(ctx context.Context, place string) []byte {
	geoMu.Lock()
	if geoC.OK && time.Since(geoAt) < 30*time.Minute {
		b, _ := json.Marshal(geoC)
		geoMu.Unlock()
		return b
	}
	geoMu.Unlock()
	lat, lon, city, ok := resolveLocation(ctx, place)
	out := geoOut{OK: ok, Lat: lat, Lon: lon, City: city}
	if ok {
		geoMu.Lock()
		geoC, geoAt = out, time.Now()
		geoMu.Unlock()
	}
	b, _ := json.Marshal(out)
	return b
}

func resetWeatherCache() {
	geoMu.Lock()
	geoC = geoOut{}
	geoMu.Unlock()
	wxMu.Lock()
	wxAt = time.Time{}
	wxCache = weatherBoard{}
	wxMu.Unlock()
}

func getJSON(ctx context.Context, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "NexNews/0.1")
	resp, err := wxClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errURL
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// weatherJSON returns the cached board, refreshing when stale (10 min).
func weatherJSON(ctx context.Context, place string) []byte {
	wxMu.Lock()
	if wxCache.OK && time.Since(wxAt) < 10*time.Minute {
		refreshInsights(&wxCache) // güncel saate göre "X dk içinde" metnini tazele
		b, _ := json.Marshal(wxCache)
		wxMu.Unlock()
		return b
	}
	wxMu.Unlock()

	out := buildBoard(ctx, place)

	wxMu.Lock()
	if out.OK {
		wxCache = out
		wxAt = time.Now()
	}
	wxMu.Unlock()
	b, _ := json.Marshal(out)
	return b
}

// --- location -------------------------------------------------------------

func resolveLocation(ctx context.Context, place string) (lat, lon float64, city string, ok bool) {
	if place != "" {
		var geo struct {
			Results []struct {
				Latitude, Longitude float64
				Name                string
			} `json:"results"`
		}
		u := "https://geocoding-api.open-meteo.com/v1/search?count=1&language=tr&name=" + url.QueryEscape(place)
		if err := getJSON(ctx, u, &geo); err == nil && len(geo.Results) > 0 {
			return geo.Results[0].Latitude, geo.Results[0].Longitude, geo.Results[0].Name, true
		}
	}
	// IP fallbacks (different providers, in case one is blocked).
	var a struct {
		City      string  `json:"city"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := getJSON(ctx, "https://ipapi.co/json/", &a); err == nil && (a.Latitude != 0 || a.Longitude != 0) {
		return a.Latitude, a.Longitude, a.City, true
	}
	var b struct {
		Success   bool    `json:"success"`
		City      string  `json:"city"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := getJSON(ctx, "https://ipwho.is/", &b); err == nil && (b.Latitude != 0 || b.Longitude != 0) {
		return b.Latitude, b.Longitude, b.City, true
	}
	return 0, 0, "", false
}

// --- board build ----------------------------------------------------------

func buildBoard(ctx context.Context, place string) weatherBoard {
	lat, lon, city, ok := resolveLocation(ctx, place)
	if !ok {
		return weatherBoard{OK: false, Err: "Konum belirlenemedi. Ayarlar'dan şehir girin."}
	}

	ll := "&latitude=" + ftoa(lat) + "&longitude=" + ftoa(lon)

	// 1) multi-model hourly (ensemble average).
	hvars := "temperature_2m,precipitation,precipitation_probability,weather_code," +
		"wind_speed_10m,wind_direction_10m,wind_gusts_10m,cape,cloud_cover"
	u1 := "https://api.open-meteo.com/v1/forecast?timezone=auto&forecast_days=2&wind_speed_unit=kmh" +
		ll + "&models=" + strings.Join(wxModels, ",") + "&hourly=" + hvars
	var r1 struct {
		Hourly map[string]json.RawMessage `json:"hourly"`
	}
	if err := getJSON(ctx, u1, &r1); err != nil || r1.Hourly == nil {
		return weatherBoard{OK: false, Err: "Hava verisi alınamadı.", City: city}
	}
	times := parseStrArr(r1.Hourly["time"])
	if len(times) == 0 {
		return weatherBoard{OK: false, Err: "Hava verisi boş.", City: city}
	}

	temp, _ := avgSeries(r1.Hourly, "temperature_2m")
	precip, _ := avgSeries(r1.Hourly, "precipitation")
	wind, windSpread := avgSeries(r1.Hourly, "wind_speed_10m")
	gust, _ := avgSeries(r1.Hourly, "wind_gusts_10m")
	cape, _ := avgSeries(r1.Hourly, "cape")
	dir := avgDirSeries(r1.Hourly, "wind_direction_10m")
	code := firstModelInts(r1.Hourly, "weather_code")
	prob := firstModelInts(r1.Hourly, "precipitation_probability")

	now := time.Now()
	hourTimes := parseLocalTimes(times)
	idxNow := nearestIdx(hourTimes, now)

	// 2) near-term 15-minutely (single best model) for +10/+30 min detail.
	var mPrecip []float64
	var mWind []float64
	var mDir []float64
	var mTimes []time.Time
	var mCode []int
	{
		u2 := "https://api.open-meteo.com/v1/forecast?timezone=auto&forecast_days=1&wind_speed_unit=kmh" +
			ll + "&minutely_15=precipitation,weather_code,wind_speed_10m,wind_direction_10m"
		var r2 struct {
			Minutely map[string]json.RawMessage `json:"minutely_15"`
		}
		if err := getJSON(ctx, u2, &r2); err == nil && r2.Minutely != nil {
			mt := parseStrArr(r2.Minutely["time"])
			mTimes = parseLocalTimes(mt)
			mPrecip = parseFloatArr(r2.Minutely["precipitation"])
			mWind = parseFloatArr(r2.Minutely["wind_speed_10m"])
			mDir = parseFloatArr(r2.Minutely["wind_direction_10m"])
			mCode = toInts(parseFloatArr(r2.Minutely["weather_code"]))
		}
	}

	mk := func(label string, t time.Time) wbStep {
		// near-term from minutely if available, else hourly.
		if len(mTimes) > 0 && t.Sub(now) <= 90*time.Minute {
			i := nearestIdx(mTimes, t)
			s := wbStep{Label: label, TS: t.Unix()}
			s.Precip = at(mPrecip, i)
			s.Wind = at(mWind, i)
			s.DirFrom = int(math.Round(at(mDir, i)))
			if i < len(mCode) {
				s.Code = mCode[i]
			}
			// temp/prob from hourly (minutely has none)
			hi := nearestIdx(hourTimes, t)
			s.Temp = at(temp, hi)
			s.PrecipProb = iat(prob, hi)
			s.Gust = at(gust, hi)
			s.Icon, s.Desc = weatherTextUI(s.Code)
			s.DirName = dirName(s.DirFrom)
			s.ToName = dirName((s.DirFrom + 180) % 360)
			return s
		}
		i := nearestIdx(hourTimes, t)
		s := wbStep{Label: label, TS: t.Unix()}
		s.Temp = at(temp, i)
		s.Precip = at(precip, i)
		s.PrecipProb = iat(prob, i)
		s.Wind = at(wind, i)
		s.Gust = at(gust, i)
		s.DirFrom = int(math.Round(at(dir, i)))
		s.Code = iat(code, i)
		s.Icon, s.Desc = weatherTextUI(s.Code)
		s.DirName = dirName(s.DirFrom)
		s.ToName = dirName((s.DirFrom + 180) % 360)
		return s
	}

	board := weatherBoard{
		OK: true, City: city, Models: len(wxModels), Updated: now.Unix(),
	}
	board.Now = mk("Şimdi", now)
	board.Steps = []wbStep{
		board.Now,
		mk("+10 dk", now.Add(10*time.Minute)),
		mk("+30 dk", now.Add(30*time.Minute)),
		mk("+1 sa", now.Add(1*time.Hour)),
		mk("+2 sa", now.Add(2*time.Hour)),
		mk("+4 sa", now.Add(4*time.Hour)),
	}

	board.Raining, board.RainStartTS, board.RainEndTS = computeRain(now, mTimes, mPrecip, hourTimes, precip)
	board.RainProb = iat(prob, idxNow)
	board.WindText = windInsight(board.Now, mk("+2 sa", now.Add(2*time.Hour)))
	board.ThunderLvl, board.ThunderStart, board.ThunderEnd, board.ThunderNow = computeThunder(now, hourTimes, code, cape, precip)
	board.Confidence = confidence(windSpread, wind, idxNow)
	refreshInsights(&board) // ilk metinleri üret (sonra her çağrıda tazelenir)
	board.Cloud = int(math.Round(at(avgOrFirst(r1.Hourly, "cloud_cover"), idxNow)))

	// Bugünün saatlik görünümü.
	start := idxNow
	if start < 0 {
		start = 0
	}
	end := start + 24
	if end > len(hourTimes) {
		end = len(hourTimes)
	}
	for i := start; i < end; i++ {
		h := wbHour{TS: hourTimes[i].Unix(), Label: hourTimes[i].Format("15:04"), Temp: at(temp, i), Precip: at(precip, i), Prob: iat(prob, i), Wind: at(wind, i)}
		h.Icon, h.Desc = weatherTextUI(iat(code, i))
		board.Hourly = append(board.Hourly, h)
	}

	// extended detail from a single best-model call (current + daily + visibility/UV).
	enrichBoard(ctx, ll, now, &board)
	return board
}

// avgOrFirst is a tiny helper for a variable we only need roughly (cloud_cover).
func avgOrFirst(h map[string]json.RawMessage, base string) []float64 {
	m, _ := avgSeries(h, base)
	return m
}

func enrichBoard(ctx context.Context, ll string, now time.Time, board *weatherBoard) {
	u3 := "https://api.open-meteo.com/v1/forecast?timezone=auto&forecast_days=5&wind_speed_unit=kmh" + ll +
		"&current=relative_humidity_2m,apparent_temperature,surface_pressure" +
		"&hourly=visibility,uv_index" +
		"&daily=temperature_2m_max,temperature_2m_min,sunrise,sunset,uv_index_max,precipitation_sum," +
		"weather_code,precipitation_probability_max,wind_speed_10m_max"
	var r struct {
		Current struct {
			Humidity float64 `json:"relative_humidity_2m"`
			Feels    float64 `json:"apparent_temperature"`
			Pressure float64 `json:"surface_pressure"`
		} `json:"current"`
		Hourly struct {
			Time       []string  `json:"time"`
			Visibility []float64 `json:"visibility"`
			UV         []float64 `json:"uv_index"`
		} `json:"hourly"`
		Daily struct {
			Time      []string  `json:"time"`
			Max       []float64 `json:"temperature_2m_max"`
			Min       []float64 `json:"temperature_2m_min"`
			Sunrise   []string  `json:"sunrise"`
			Sunset    []string  `json:"sunset"`
			UVMax     []float64 `json:"uv_index_max"`
			PrecipSum []float64 `json:"precipitation_sum"`
			Code      []float64 `json:"weather_code"`
			Prob      []float64 `json:"precipitation_probability_max"`
			WindMax   []float64 `json:"wind_speed_10m_max"`
		} `json:"daily"`
	}
	if err := getJSON(ctx, u3, &r); err != nil {
		return
	}
	board.Feels = r.Current.Feels
	board.Humidity = int(math.Round(r.Current.Humidity))
	board.Pressure = int(math.Round(r.Current.Pressure))
	if len(r.Hourly.Time) > 0 {
		ht := parseLocalTimes(r.Hourly.Time)
		i := nearestIdx(ht, now)
		board.Visibility = at(r.Hourly.Visibility, i) / 1000.0 // m → km
		board.UV = at(r.Hourly.UV, i)
	}
	if len(r.Daily.Max) > 0 {
		board.DayHi = r.Daily.Max[0]
	}
	if len(r.Daily.Min) > 0 {
		board.DayLo = r.Daily.Min[0]
	}
	if len(r.Daily.UVMax) > 0 {
		board.UVMax = r.Daily.UVMax[0]
	}
	if len(r.Daily.PrecipSum) > 0 {
		board.PrecipSum = r.Daily.PrecipSum[0]
	}
	if len(r.Daily.Sunrise) > 0 {
		board.Sunrise = hhmm(r.Daily.Sunrise[0])
	}
	if len(r.Daily.Sunset) > 0 {
		board.Sunset = hhmm(r.Daily.Sunset[0])
	}

	// 5 günlük tahmin
	n := len(r.Daily.Time)
	for i := 0; i < n && i < 5; i++ {
		d := wbDay{}
		if t, err := time.ParseInLocation("2006-01-02", r.Daily.Time[i], time.Local); err == nil {
			d.TS = t.Unix()
			d.Label = dayLabel(t, i)
		}
		d.Hi = atv(r.Daily.Max, i)
		d.Lo = atv(r.Daily.Min, i)
		d.Precip = atv(r.Daily.PrecipSum, i)
		d.Prob = int(atv(r.Daily.Prob, i))
		d.Wind = atv(r.Daily.WindMax, i)
		d.Code = int(atv(r.Daily.Code, i))
		d.Icon, d.Desc = weatherTextUI(d.Code)
		board.Days = append(board.Days, d)
	}

	// CAMS tabanlı hava kalitesi. Ana hava verisi çalışmaya devam ederken
	// bu istek başarısız olursa yalnızca AQI bölümü boş kalır.
	uaq := "https://air-quality-api.open-meteo.com/v1/air-quality?timezone=auto&forecast_days=1" + ll + "&current=european_aqi,pm2_5,pm10,ozone,nitrogen_dioxide"
	var aq struct {
		Current map[string]float64 `json:"current"`
	}
	if err := getJSON(ctx, uaq, &aq); err == nil {
		board.AQI = aq.Current["european_aqi"]
		board.PM25 = aq.Current["pm2_5"]
		board.PM10 = aq.Current["pm10"]
		board.Ozone = aq.Current["ozone"]
		board.NO2 = aq.Current["nitrogen_dioxide"]
		board.AQILevel = aqiLevel(board.AQI)
	}
}

func atv(a []float64, i int) float64 {
	if i >= 0 && i < len(a) {
		return a[i]
	}
	return 0
}

var trWeekdays = []string{"Paz", "Pzt", "Sal", "Çar", "Per", "Cum", "Cmt"}

func dayLabel(t time.Time, i int) string {
	if i == 0 {
		return "Bugün"
	}
	if i == 1 {
		return "Yarın"
	}
	d := trWeekdays[int(t.Weekday())]
	return uiTK("gün:"+d, d)
}

func hhmm(s string) string {
	if t, err := time.ParseInLocation("2006-01-02T15:04", s, time.Local); err == nil {
		return t.Format("15:04")
	}
	return ""
}

// --- insights -------------------------------------------------------------

// computeRain returns the rain state and absolute onset/stop times (unix).
func computeRain(now time.Time, mt []time.Time, mp []float64, ht []time.Time, hp []float64) (raining bool, startTS, endTS int64) {
	const wet = 0.2
	cur := 0.0
	if len(mp) > 0 {
		cur = at(mp, nearestIdx(mt, now))
	} else if len(hp) > 0 {
		cur = at(hp, nearestIdx(ht, now))
	}
	if cur >= wet {
		raining = true
		if end, ok := firstBelow(mt, mp, now, wet); ok {
			endTS = end.Unix()
		} else if end, ok := firstBelow(ht, hp, now, wet); ok {
			endTS = end.Unix()
		}
		return
	}
	if start, ok := firstAbove(mt, mp, now, wet); ok {
		startTS = start.Unix()
		return
	}
	if start, ok := firstAbove(ht, hp, now, wet); ok {
		startTS = start.Unix()
	}
	return
}

func until(ts int64) time.Duration { return time.Until(time.Unix(ts, 0)) }

// refreshInsights rebuilds the time-relative rain/thunder texts from stored
// absolute times, so the "X dk içinde" countdown reflects the current moment
// instead of the value baked in when the (cached) board was built.
func aqiLevel(v float64) string {
	switch {
	case v <= 20:
		return "İyi"
	case v <= 40:
		return "Makul"
	case v <= 60:
		return "Orta"
	case v <= 80:
		return "Kötü"
	case v <= 100:
		return "Çok kötü"
	case v > 100:
		return "Aşırı kötü"
	default:
		return "—"
	}
}

func refreshInsights(b *weatherBoard) {
	if b.Raining {
		if b.RainEndTS > 0 {
			b.RainText = "Yağış sürüyor; tahminen " + human(until(b.RainEndTS)) + " içinde diniyor."
		} else {
			b.RainText = "Yağış sürüyor; önümüzdeki saatlerde devam ediyor."
		}
	} else if b.RainStartTS > 0 {
		t := "Yağış ~" + human(until(b.RainStartTS)) + " sonra başlıyor"
		if b.RainProb > 0 {
			t += " (%" + strconv.Itoa(b.RainProb) + " civarı)"
		}
		b.RainText = t + "."
	} else {
		b.RainText = "Önümüzdeki saatlerde kayda değer yağış görünmüyor."
	}

	if b.ThunderStart == 0 && b.ThunderEnd == 0 {
		b.ThunderTxt = "Gök gürültülü fırtına beklenmiyor."
		return
	}
	if b.ThunderNow {
		b.ThunderTxt = "Gök gürültülü fırtına riski şu an; tahminen " + human(until(b.ThunderEnd)) + " içinde diniyor. (tahmin)"
	} else {
		b.ThunderTxt = "Gök gürültülü fırtına riski ~" + human(until(b.ThunderStart)) + " sonra; tahminen " + human(until(b.ThunderEnd)) + " içinde diniyor. (tahmin)"
	}
}

func windInsight(now, later wbStep) string {
	s := "Rüzgâr " + now.DirName + "'den " + strconv.Itoa(int(math.Round(now.Wind))) + " km/s (→ " + now.ToName + " yönüne)."
	if later.DirName != now.DirName {
		s += " 2 saat sonra " + later.DirName + "'ye dönüyor."
	}
	if now.Gust >= now.Wind+15 {
		s += " Ani hamleler " + strconv.Itoa(int(math.Round(now.Gust))) + " km/s'e çıkabilir."
	}
	return s
}

// computeThunder returns the storm risk level and absolute start/end times.
func computeThunder(now time.Time, ht []time.Time, code []int, cape []float64, precip []float64) (lvl string, startTS, endTS int64, nowFlag bool) {
	isStorm := func(i int) bool {
		if iat(code, i) >= 95 {
			return true
		}
		return at(cape, i) >= 1200 && at(precip, i) >= 0.2
	}
	var startI, endI = -1, -1
	for i := 0; i < len(ht); i++ {
		if ht[i].Before(now.Add(-time.Hour)) {
			continue
		}
		if ht[i].After(now.Add(12 * time.Hour)) {
			break
		}
		if isStorm(i) {
			if startI < 0 {
				startI = i
			}
			endI = i
		} else if startI >= 0 {
			break
		}
	}
	if startI < 0 {
		return "yok", 0, 0, false
	}
	maxCape := 0.0
	for i := startI; i <= endI; i++ {
		if at(cape, i) > maxCape {
			maxCape = at(cape, i)
		}
	}
	lvl = "düşük"
	switch {
	case maxCape >= 2500:
		lvl = "yüksek"
	case maxCape >= 1200:
		lvl = "orta"
	}
	startT, endT := ht[startI], ht[endI].Add(time.Hour)
	return lvl, startT.Unix(), endT.Unix(), !startT.After(now)
}

func confidence(spread, mean []float64, idx int) string {
	// relative model disagreement over the next 6 hours of wind speed
	num, den := 0.0, 0.0
	for i := idx; i < idx+6 && i < len(spread); i++ {
		num += spread[i]
		den += math.Max(at(mean, i), 1)
	}
	if den == 0 {
		return "orta"
	}
	rel := num / den
	switch {
	case rel < 0.18:
		return "yüksek"
	case rel < 0.4:
		return "orta"
	default:
		return "düşük"
	}
}

// --- series math ----------------------------------------------------------

// avgSeries averages a variable across all requested models, returning the mean
// and the per-step spread (max-min) as an agreement signal.
func avgSeries(h map[string]json.RawMessage, base string) (mean, spread []float64) {
	var cols [][]float64
	for _, m := range wxModels {
		if raw, ok := h[base+"_"+m]; ok {
			cols = append(cols, parseFloatArr(raw))
		}
	}
	if len(cols) == 0 {
		if raw, ok := h[base]; ok {
			cols = append(cols, parseFloatArr(raw))
		}
	}
	n := 0
	for _, c := range cols {
		if len(c) > n {
			n = len(c)
		}
	}
	mean = make([]float64, n)
	spread = make([]float64, n)
	for i := 0; i < n; i++ {
		var vals []float64
		for _, c := range cols {
			if i < len(c) {
				vals = append(vals, c[i])
			}
		}
		if len(vals) == 0 {
			continue
		}
		sum, mn, mx := 0.0, vals[0], vals[0]
		for _, v := range vals {
			sum += v
			mn = math.Min(mn, v)
			mx = math.Max(mx, v)
		}
		mean[i] = sum / float64(len(vals))
		spread[i] = mx - mn
	}
	return mean, spread
}

// avgDirSeries averages wind direction across models using circular mean.
func avgDirSeries(h map[string]json.RawMessage, base string) []float64 {
	var cols [][]float64
	for _, m := range wxModels {
		if raw, ok := h[base+"_"+m]; ok {
			cols = append(cols, parseFloatArr(raw))
		}
	}
	if len(cols) == 0 {
		if raw, ok := h[base]; ok {
			cols = append(cols, parseFloatArr(raw))
		}
	}
	n := 0
	for _, c := range cols {
		if len(c) > n {
			n = len(c)
		}
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		var sx, sy float64
		cnt := 0
		for _, c := range cols {
			if i < len(c) {
				r := c[i] * math.Pi / 180
				sx += math.Cos(r)
				sy += math.Sin(r)
				cnt++
			}
		}
		if cnt == 0 {
			continue
		}
		d := math.Atan2(sy, sx) * 180 / math.Pi
		if d < 0 {
			d += 360
		}
		out[i] = d
	}
	return out
}

func firstModelInts(h map[string]json.RawMessage, base string) []int {
	for _, m := range wxModels {
		if raw, ok := h[base+"_"+m]; ok {
			return toInts(parseFloatArr(raw))
		}
	}
	if raw, ok := h[base]; ok {
		return toInts(parseFloatArr(raw))
	}
	return nil
}

// --- small helpers --------------------------------------------------------

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', 4, 64) }

func parseStrArr(raw json.RawMessage) []string {
	var s []string
	_ = json.Unmarshal(raw, &s)
	return s
}
func parseFloatArr(raw json.RawMessage) []float64 {
	var f []float64
	_ = json.Unmarshal(raw, &f)
	return f
}
func toInts(f []float64) []int {
	out := make([]int, len(f))
	for i, v := range f {
		out[i] = int(math.Round(v))
	}
	return out
}
func at(a []float64, i int) float64 {
	if i >= 0 && i < len(a) {
		return a[i]
	}
	return 0
}
func iat(a []int, i int) int {
	if i >= 0 && i < len(a) {
		return a[i]
	}
	return 0
}

func parseLocalTimes(ss []string) []time.Time {
	out := make([]time.Time, len(ss))
	for i, s := range ss {
		t, err := time.ParseInLocation("2006-01-02T15:04", s, time.Local)
		if err != nil {
			t, _ = time.ParseInLocation("2006-01-02T15:04:05", s, time.Local)
		}
		out[i] = t
	}
	return out
}

func nearestIdx(ts []time.Time, t time.Time) int {
	best, bestD := 0, time.Duration(math.MaxInt64)
	for i, x := range ts {
		d := x.Sub(t)
		if d < 0 {
			d = -d
		}
		if d < bestD {
			bestD, best = d, i
		}
	}
	return best
}

func firstAbove(ts []time.Time, v []float64, now time.Time, th float64) (time.Time, bool) {
	type pair struct {
		t time.Time
		v float64
	}
	var ps []pair
	for i := range ts {
		if !ts[i].Before(now) {
			ps = append(ps, pair{ts[i], at(v, i)})
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].t.Before(ps[j].t) })
	for _, p := range ps {
		if p.v >= th {
			return p.t, true
		}
	}
	return time.Time{}, false
}

func firstBelow(ts []time.Time, v []float64, now time.Time, th float64) (time.Time, bool) {
	type pair struct {
		t time.Time
		v float64
	}
	var ps []pair
	for i := range ts {
		if !ts[i].Before(now) {
			ps = append(ps, pair{ts[i], at(v, i)})
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].t.Before(ps[j].t) })
	for _, p := range ps {
		if p.v < th {
			return p.t, true
		}
	}
	return time.Time{}, false
}

func human(d time.Duration) string {
	m := int(math.Round(d.Minutes()))
	if m < 1 {
		m = 1
	}
	if m < 60 {
		return strconv.Itoa(m) + " dk"
	}
	h := m / 60
	rem := m % 60
	if rem >= 30 {
		return strconv.Itoa(h) + ",5 sa"
	}
	return strconv.Itoa(h) + " sa"
}

var compass = []string{"K", "KD", "D", "GD", "G", "GB", "B", "KB"}

func dirName(deg int) string {
	deg = ((deg % 360) + 360) % 360
	c := compass[int(math.Round(float64(deg)/45))%8]
	return uiTK("yön:"+c, c)
}

// weatherTextUI: weatherText'in arayüz dilindeki hâli ("Açık" burada gökyüzü).
func weatherTextUI(code int) (string, string) {
	ic, d := weatherText(code)
	return ic, uiTK("hava:"+d, d)
}

// weatherText maps a WMO weather code to an emoji icon and a Turkish label.
func weatherText(code int) (string, string) {
	switch {
	case code == 0:
		return "☀️", "Açık"
	case code == 1 || code == 2:
		return "🌤️", "Az bulutlu"
	case code == 3:
		return "☁️", "Bulutlu"
	case code == 45 || code == 48:
		return "🌫️", "Sisli"
	case code >= 51 && code <= 57:
		return "🌦️", "Çisenti"
	case code >= 61 && code <= 67:
		return "🌧️", "Yağmurlu"
	case code >= 71 && code <= 77:
		return "❄️", "Karlı"
	case code >= 80 && code <= 82:
		return "🌧️", "Sağanak"
	case code == 85 || code == 86:
		return "🌨️", "Kar sağanağı"
	case code >= 95:
		return "⛈️", "Gök gürültülü"
	default:
		return "🌡️", "—"
	}
}
