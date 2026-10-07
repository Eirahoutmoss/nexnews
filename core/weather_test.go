package core

import (
	"encoding/json"
	"testing"
)

func TestWeatherText(t *testing.T) {
	if ic, d := weatherText(0); ic != "☀️" || d != "Açık" {
		t.Errorf("code0 => %s %s", ic, d)
	}
	if _, d := weatherText(63); d != "Yağmurlu" {
		t.Errorf("code63 => %s", d)
	}
	if _, d := weatherText(95); d != "Gök gürültülü" {
		t.Errorf("code95 => %s", d)
	}
}

func TestDirName(t *testing.T) {
	cases := map[int]string{0: "K", 45: "KD", 90: "D", 180: "G", 270: "B", 359: "K"}
	for deg, want := range cases {
		if got := dirName(deg); got != want {
			t.Errorf("dirName(%d)=%s want %s", deg, got, want)
		}
	}
}

func TestAvgDirCircular(t *testing.T) {
	h := map[string]json.RawMessage{
		"wind_direction_10m_icon_seamless": json.RawMessage(`[350]`),
		"wind_direction_10m_gfs_seamless":  json.RawMessage(`[10]`),
	}
	out := avgDirSeries(h, "wind_direction_10m")
	if len(out) != 1 || out[0] > 5 && out[0] < 355 {
		t.Errorf("circular mean of 350/10 should be ~0, got %v", out)
	}
}

func TestAvgSeriesSpread(t *testing.T) {
	h := map[string]json.RawMessage{
		"wind_speed_10m_icon_seamless": json.RawMessage(`[10,20]`),
		"wind_speed_10m_gfs_seamless":  json.RawMessage(`[20,20]`),
	}
	mean, spread := avgSeries(h, "wind_speed_10m")
	if mean[0] != 15 || spread[0] != 10 {
		t.Errorf("mean/spread[0]=%v/%v want 15/10", mean[0], spread[0])
	}
}
