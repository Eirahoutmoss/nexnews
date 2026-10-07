package core

import "testing"

func TestAccessibilityDefaults(t *testing.T) {
	c := defaultConfig()
	if c.WidgetOpacity != 78 {
		t.Fatalf("widget opacity default: %d", c.WidgetOpacity)
	}
	if c.FontScale != 100 {
		t.Fatalf("font scale default: %d", c.FontScale)
	}
	if c.LineHeight != 1.75 {
		t.Fatalf("line height default: %v", c.LineHeight)
	}
	if c.ReaderWidth != 760 {
		t.Fatalf("reader width default: %d", c.ReaderWidth)
	}
	if c.SchemaV != currentSchema {
		t.Fatalf("schema: %d", c.SchemaV)
	}
}
