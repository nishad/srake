package main

import (
	"strings"
	"testing"
	"time"
)

// TestRenderProgressLine verifies the index progress display formats each phase
// correctly — in particular that the FTS5/startup phase shows a live elapsed-time
// heartbeat (so a long phase does not appear frozen) and that the Bleve phase
// shows the document count and speed.
func TestRenderProgressLine(t *testing.T) {
	t.Run("fts5 phase shows step and elapsed heartbeat", func(t *testing.T) {
		line, phase := renderProgressLine("fts5", "FTS5: Building experiment index (38M records)...", 0, 0, 0, 4*time.Minute+12*time.Second)
		if !strings.Contains(line, "experiment index") {
			t.Errorf("expected FTS5 step in line, got %q", line)
		}
		if !strings.Contains(line, "4m12s elapsed") {
			t.Errorf("expected elapsed-time heartbeat in line, got %q", line)
		}
		if !strings.HasPrefix(phase, "fts5:") {
			t.Errorf("expected fts5 phase key, got %q", phase)
		}
	})

	t.Run("startup with zero docs shows initializing heartbeat", func(t *testing.T) {
		line, _ := renderProgressLine("", "", 0, 0, 0, 3*time.Second)
		if !strings.Contains(line, "Initializing") || !strings.Contains(line, "3s elapsed") {
			t.Errorf("expected initializing heartbeat with elapsed, got %q", line)
		}
	})

	t.Run("bleve phase shows count and speed", func(t *testing.T) {
		line, phase := renderProgressLine("studies", "Indexing studies into Bleve...", 600, 600, 2, 60*time.Second)
		if !strings.Contains(line, "600 / ~696K docs") {
			t.Errorf("expected doc count, got %q", line)
		}
		if !strings.Contains(line, "10 docs/s") { // 600 docs / 60s
			t.Errorf("expected speed, got %q", line)
		}
		if !strings.Contains(line, "2 failed") {
			t.Errorf("expected failed count, got %q", line)
		}
		if phase != "studies" {
			t.Errorf("expected studies phase key, got %q", phase)
		}
	})

	t.Run("phase key changes between fts5 and bleve", func(t *testing.T) {
		_, p1 := renderProgressLine("fts5", "FTS5: Building sample index...", 0, 0, 0, time.Second)
		_, p2 := renderProgressLine("studies", "Indexing studies into Bleve...", 100, 100, 0, time.Second)
		if p1 == p2 {
			t.Errorf("expected distinct phase keys for fts5 vs bleve, both were %q", p1)
		}
	})
}
