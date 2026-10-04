// Tests for the showcase UI: loading, keypad logic and rendering.
package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Vinetwigs/tuix"
)

func loadShowcase(t *testing.T) *tuix.App {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	app, err := tuix.LoadFile(filepath.Join(filepath.Dir(file), "showcase.tuix"))
	if err != nil {
		t.Fatalf("load showcase.tuix: %v", err)
	}
	app.Computed("badge", func(get func(string) string) string {
		return "[" + get("role") + "] " + get("user")
	})
	return app
}

func TestShowcaseKeypadFlow(t *testing.T) {
	app := loadShowcase(t)
	pressDigit(app, "k4")
	pressDigit(app, "k2")
	if got := app.Get("display"); got != "42" {
		t.Fatalf("display = %q", got)
	}
	pressDigit(app, "kC") // not a digit: ignored
	if got := app.Get("display"); got != "42" {
		t.Fatalf("display = %q", got)
	}
	pressConfirm(app)
	if got := app.Get("confirmed"); got != "42" {
		t.Fatalf("confirmed = %q", got)
	}
	if got := app.Get("display"); got != "" {
		t.Fatalf("display not cleared: %q", got)
	}
	pressReset(app)
	if got := app.Get("clicks"); got != "0" {
		t.Fatalf("clicks = %q", got)
	}
}

func TestShowcaseRenders(t *testing.T) {
	app := loadShowcase(t)
	pressDigit(app, "k7")
	joined := strings.Join(app.RenderCells(90, 44), "\n")
	for _, want := range []string{"SHOWCASE", "Menu", "Keypad", "Login", "7", "[operator] ada"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("render missing %q:\n%s", want, joined)
		}
	}
}
