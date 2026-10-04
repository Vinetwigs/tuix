// Tests for tutask: task flows and JSON persistence round-trip.
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

func loadTodo(t *testing.T) *tuix.App {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	app, err := tuix.LoadFile(filepath.Join(filepath.Dir(file), "todo.tuix"))
	if err != nil {
		t.Fatalf("load todo.tuix: %v", err)
	}
	registerCounts(app)
	return app
}

func withTempStore(t *testing.T) {
	t.Helper()
	old := storePath
	storePath = filepath.Join(t.TempDir(), "tasks.json")
	t.Cleanup(func() { storePath = old })
}

func selectRow(app *tuix.App, i int) {
	if l := component.FindByID(app.Tree(), "tasks"); l != nil {
		l.Sel = i
	}
}

func TestTodoFlow(t *testing.T) {
	withTempStore(t)
	app := loadTodo(t)
	loadTasks(app) // missing file -> seeds

	if got := app.Get("total"); got != "3" {
		t.Fatalf("seeded total = %q", got)
	}
	if !addTask(app, "  ") {
		// empty ignored, good
	} else {
		t.Fatal("empty task should be ignored")
	}
	if !addTask(app, "Write tests") {
		t.Fatal("add failed")
	}
	if got := app.Get("total"); got != "4" {
		t.Fatalf("total = %q", got)
	}
	selectRow(app, 3)
	if !toggleSelected(app) {
		t.Fatal("toggle failed")
	}
	if got := app.Get("done"); got != "1" {
		t.Fatalf("done = %q", got)
	}
	rows := strings.Join(app.RenderCells(60, 24), "\n")
	if !strings.Contains(rows, "[x] Write tests") {
		t.Fatalf("toggled task not rendered:\n%s", rows)
	}
	if !deleteSelected(app) {
		t.Fatal("delete failed")
	}
	if got := app.Get("total"); got != "3" {
		t.Fatalf("total = %q", got)
	}
}

func TestTodoPersistence(t *testing.T) {
	withTempStore(t)
	app := loadTodo(t)
	loadTasks(app)
	addTask(app, "Persist me")
	selectRow(app, 0)
	toggleSelected(app)

	again := loadTodo(t)
	loadTasks(again)
	if again.Get("tasks") != app.Get("tasks") {
		t.Fatalf("round-trip mismatch:\n%q\n%q", app.Get("tasks"), again.Get("tasks"))
	}
	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Persist me") {
		t.Fatalf("file content: %s", data)
	}
}

func TestTodoClearDone(t *testing.T) {
	withTempStore(t)
	app := loadTodo(t)
	loadTasks(app)
	selectRow(app, 0)
	toggleSelected(app)
	if n := clearDone(app); n != 1 {
		t.Fatalf("cleared = %d", n)
	}
	if got := app.Get("total"); got != "2" {
		t.Fatalf("total = %q", got)
	}
}
