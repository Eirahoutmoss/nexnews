package core

import (
	"testing"
	"time"
)

func TestQuietHoursWrapMidnight(t *testing.T) {
	c := defaultConfig()
	c.QuietHours = true
	c.QuietFrom, c.QuietTo = "23:00", "07:00"
	at := func(h, m int) time.Time { return time.Date(2026, 10, 7, h, m, 0, 0, time.Local) }
	if !c.inQuietHours(at(23, 30)) || !c.inQuietHours(at(3, 0)) || c.inQuietHours(at(7, 0)) || c.inQuietHours(at(12, 0)) {
		t.Fatal("wrap-around window wrong")
	}
	c.QuietHours = false
	if c.inQuietHours(at(3, 0)) {
		t.Fatal("disabled quiet hours must not apply")
	}
}

func TestMigrateKeepsTranslateOffAndSetsV11Defaults(t *testing.T) {
	c := &Config{SchemaV: 10, Translate: false}
	c.migrate()
	if c.Translate {
		t.Fatal("v10→v11 migration must not force translation back on")
	}
	if c.WidgetMode != "cam" || !c.NetMeter || c.Density != "rahat" || c.SchemaV != currentSchema {
		t.Fatalf("v11 defaults missing: %+v", c.WidgetMode)
	}
}

func TestNormalizeV11Clamps(t *testing.T) {
	c := &Config{WidgetMode: "neon", WidgetTint: 400, Density: "x", QuietFrom: "25:00", RightTab: "?"}
	c.normalizeV11()
	if c.WidgetMode != "cam" || c.WidgetTint != 45 || c.Density != "rahat" || c.QuietFrom != "23:00" || c.RightTab != "ozet" {
		t.Fatalf("not normalised: %+v", c)
	}
}
