// Tests for tuicd: pipeline simulation, compact view, rendering and
// checkbox toggles.
package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/Vinetwigs/tuix"
)

func loadDash(t *testing.T) *tuix.App {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	app, err := tuix.LoadFile(filepath.Join(filepath.Dir(file), "dash.tuix"))
	if err != nil {
		t.Fatalf("load dash.tuix: %v", err)
	}
	return app
}

func TestPipelineStep(t *testing.T) {
	p := &pipeline{}
	p.step()
	if p.build <= 0 {
		t.Fatalf("build did not advance: %v", p.build)
	}
	// Drive a full cycle plus a bit: build, test, deploy, wrap.
	for i := 0; i < 40; i++ {
		p.step()
	}
	if p.done < 1 {
		t.Fatalf("no cycle completed: done=%d", p.done)
	}
	if len(p.history) == 0 {
		t.Fatal("history not recorded")
	}
	if ov := p.overall(); ov < 0 || ov > 100 {
		t.Fatalf("overall out of range: %v", ov)
	}
}

func TestHistoryText(t *testing.T) {
	p := &pipeline{history: []float64{12, 14, 13}}
	if got := p.historyText(); got != "12 14 13" {
		t.Fatalf("historyText = %q", got)
	}
}

func TestPublishCompact(t *testing.T) {
	app := loadDash(t)
	p := &pipeline{}
	for i := 0; i < 40; i++ {
		p.step()
	}
	app.Set("compact", "false")
	publish(app, p)
	if app.Get("history") == "" {
		t.Fatal("history should be present when not compact")
	}
	app.Set("compact", "true")
	publish(app, p)
	if app.Get("history") != "" {
		t.Fatalf("history should be hidden when compact, got %q", app.Get("history"))
	}
}

func TestDashRenders(t *testing.T) {
	app := loadDash(t)
	app.Set("build", "100")
	app.Set("test", "60")
	app.Set("deploy", "20")
	app.Set("overall", "60")
	app.Set("history", "12 14 13 15 17 16")
	app.Set("bars", "12 14 13 15 17 16")
	app.Set("runs", "3|ok|38s\n2|SLOW|52s")
	app.Set("threshold", "30")
	app.Set("clock", "12:00:00")
	rows := strings.Join(app.RenderCells(80, 40), "\n")
	for _, want := range []string{
		"TUICD", "stages", "build", "60%", "100%",
		"trend", "recent builds", "RUN", "STATUS", "SLOW",
		"alert threshold",
		"[ ] Pause", "[ ] Compact", "[ ] Notify",
	} {
		if !strings.Contains(rows, want) {
			t.Fatalf("missing %q:\n%s", want, rows)
		}
	}
	if !strings.ContainsAny(rows, "-=+*#") {
		t.Fatalf("no chart ramp drawn:\n%s", rows)
	}
}

func TestToggleViaKeyboard(t *testing.T) {
	app := loadDash(t)
	// Focus starts on the slider; move to the Pause checkbox.
	if f := app.Focused(); f == nil || f.ID != "thr" {
		t.Fatalf("first focusable = %+v", f)
	}
	app.SetFocus(1)
	if f := app.Focused(); f == nil || f.ID != "pause" {
		t.Fatalf("focused = %+v", f)
	}
	app.HandleKey(tcell.KeyEnter, 0, 0)
	if app.Get("pause") != "true" {
		t.Fatalf("pause = %q", app.Get("pause"))
	}
	app.HandleKey(tcell.KeyRune, ' ', 0)
	if app.Get("pause") != "false" {
		t.Fatalf("pause after space = %q", app.Get("pause"))
	}
}

func TestSliderAdjust(t *testing.T) {
	app := loadDash(t)
	app.SetFocus(0) // threshold slider
	app.HandleKey(tcell.KeyRight, 0, 0)
	if app.Get("threshold") != "35" {
		t.Fatalf("threshold after right = %q", app.Get("threshold"))
	}
	app.HandleKey(tcell.KeyHome, 0, 0)
	if app.Get("threshold") != "10" {
		t.Fatalf("threshold after home = %q", app.Get("threshold"))
	}
}
