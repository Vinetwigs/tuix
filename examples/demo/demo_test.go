// Regression test for the demo UI: it loads examples/demo/demo.tuix
// (including the <Include> of card.tuix) and checks the headless
// render. This catches markup errors that would otherwise surface
// only at runtime via "go run ./examples/demo".
package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Vinetwigs/tuix"
)

func loadDemo(t *testing.T) *tuix.App {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	app, err := tuix.LoadFile(filepath.Join(filepath.Dir(file), "demo.tuix"))
	if err != nil {
		t.Fatalf("load demo.tuix: %v", err)
	}
	return app
}

func TestDemoLoads(t *testing.T) {
	app := loadDemo(t)
	if app.Title() != "TUIX Demo" {
		t.Fatalf("title = %q", app.Title())
	}
}

func TestDemoRenders(t *testing.T) {
	app := loadDemo(t)
	joined := strings.Join(app.RenderCells(80, 30), "\n")
	for _, want := range []string{"Hello ada", "Clicks: 0", "Included card", "card.tuix"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("render missing %q:\n%s", want, joined)
		}
	}
}
