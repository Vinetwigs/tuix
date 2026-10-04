// Tests for TUIX Studio: snippet flows, validation with line
// numbers, outline rendering and password-gated publishing.
package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Vinetwigs/tuix"
	"github.com/Vinetwigs/tuix/component"
)

func loadStudio(t *testing.T) *tuix.App {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	app, err := tuix.LoadFile(filepath.Join(filepath.Dir(file), "studio.tuix"))
	if err != nil {
		t.Fatalf("load studio.tuix: %v", err)
	}
	return app
}

func seedSnippets(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"good.tuix": "<App>\n<Text value=\"hi\"/>\n</App>",
		"bad.tuix":  "<App>\n<Text/>\n</Nope>",
		"myst.tuix": "<App>\n<Frobnicate/>\n</App>",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}
	old := snippetsDir
	snippetsDir = dir
	t.Cleanup(func() { snippetsDir = old })
	return dir
}

func selectSnippet(app *tuix.App, i int) {
	if l := component.FindByID(app.Tree(), "snips"); l != nil {
		l.Sel = i
	}
}

func TestStudioValidateGood(t *testing.T) {
	seedSnippets(t)
	app := loadStudio(t)
	refreshSnippets(app)
	// Sorted: bad, good, myst -> good is index 1.
	selectSnippet(app, 1)
	if !openSelected(app) {
		t.Fatal("open failed")
	}
	if got := app.Get("result"); !strings.HasPrefix(got, "OK:") {
		t.Fatalf("result = %q", got)
	}
	outline := app.Get("outline")
	if !strings.Contains(outline, "App") || !strings.Contains(outline, "Text") {
		t.Fatalf("outline = %q", outline)
	}
	rows := strings.Join(app.RenderCells(100, 34), "\n")
	if !strings.Contains(rows, "OK:") {
		t.Fatalf("result not rendered:\n%s", rows)
	}
}

func TestStudioValidateBadLineNumber(t *testing.T) {
	seedSnippets(t)
	app := loadStudio(t)
	refreshSnippets(app)
	selectSnippet(app, 0) // bad.tuix
	if !openSelected(app) {
		t.Fatal("open failed")
	}
	got := app.Get("result")
	if !strings.HasPrefix(got, "ERROR:") || !strings.Contains(got, "line 3") {
		t.Fatalf("result = %q", got)
	}
	// Unknown component type: also reported with its line.
	app.Set("code", "<App>\n<Frobnicate/>\n</App>")
	if validateCode(app) {
		t.Fatal("mystery type should fail")
	}
	got = app.Get("result")
	if !strings.Contains(got, "line 2") || !strings.Contains(got, "Frobnicate") {
		t.Fatalf("result = %q", got)
	}
}

func TestStudioPublishGate(t *testing.T) {
	dir := seedSnippets(t)
	app := loadStudio(t)
	refreshSnippets(app)
	app.Set("snippetName", "mine")
	app.Set("code", "<App><Text value=\"x\"/></App>")
	app.Set("password", "wrong")
	if publishSnippet(app) {
		t.Fatal("publish with wrong password should fail")
	}
	if _, err := os.Stat(filepath.Join(dir, "mine.tuix")); !os.IsNotExist(err) {
		t.Fatal("file should not exist")
	}
	app.Set("password", "tuix")
	if !publishSnippet(app) {
		t.Fatalf("publish failed: %s", app.Get("status"))
	}
	data, err := os.ReadFile(filepath.Join(dir, "mine.tuix"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Text") {
		t.Fatalf("saved content: %s", data)
	}
	if !strings.Contains(app.Get("snippets"), "mine") {
		t.Fatalf("list not refreshed: %q", app.Get("snippets"))
	}
}
