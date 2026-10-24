package core

import (
	"strings"
	"testing"
)

func TestIndexAssembled(t *testing.T) {
	h := assembleIndex()
	if strings.Contains(h, "\n@@") || !strings.Contains(h, "function loadOverview") || !strings.Contains(h, "window.NexI18n") || !strings.HasSuffix(strings.TrimSpace(h), "</html>") {
		t.Fatal("sayfa eksik birleştirildi")
	}
}
