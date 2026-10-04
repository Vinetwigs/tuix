// Command todo runs tutask, a persistent terminal todo list.
//
// Tasks autosave to tasks.json in the working directory on every
// change and reload on start.
//
// Usage (from the module root):
//
//	go run ./examples/todo
//
// Mouse: click a row to select it, click buttons to press them.
// Keyboard: Tab moves focus, Up/Down navigate the list, Enter on a
// row toggles it (or adds from the input), Esc quits.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Vinetwigs/tuix"
	"github.com/Vinetwigs/tuix/component"
	"github.com/Vinetwigs/tuix/event"
)

// storePath is the tasks file. Relative: resolved against the working
// directory, so each project folder can keep its own list.
var storePath = "tasks.json"

var seedTasks = []string{
	"[ ] Welcome to tutask",
	"[ ] Type above and press Enter to add",
	"[ ] Click a row or press Enter on it to toggle",
}

func findUI() (*tuix.App, []error) {
	cands := []string{"examples/todo/todo.tuix", "todo.tuix"}
	if _, file, _, ok := runtime.Caller(0); ok {
		cands = append(cands, filepath.Join(filepath.Dir(file), "todo.tuix"))
	}
	var errs []error
	for _, p := range cands {
		if app, err := tuix.LoadFile(p); err == nil {
			return app, nil
		} else {
			errs = append(errs, fmt.Errorf("%s: %w", p, err))
		}
	}
	return nil, errs
}

func getTasks(app *tuix.App) []string {
	raw := strings.TrimSpace(app.Get("tasks"))
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}

func setTasks(app *tuix.App, tasks []string) {
	app.Set("tasks", strings.Join(tasks, "\n"))
	saveTasks(tasks)
}

func selIndex(app *tuix.App) int {
	if l := component.FindByID(app.Tree(), "tasks"); l != nil {
		return l.Sel
	}
	return -1
}

func loadTasks(app *tuix.App) {
	data, err := os.ReadFile(storePath)
	if err != nil {
		app.Set("tasks", strings.Join(seedTasks, "\n"))
		app.Set("status", "new list - edits autosave to "+storePath)
		return
	}
	var tasks []string
	if err := json.Unmarshal(data, &tasks); err != nil || tasks == nil {
		app.Set("tasks", strings.Join(seedTasks, "\n"))
		app.Set("status", "unreadable file - started fresh")
		return
	}
	app.Set("tasks", strings.Join(tasks, "\n"))
	app.Set("status", "loaded "+strconv.Itoa(len(tasks))+" tasks")
}

func saveTasks(tasks []string) {
	if tasks == nil {
		tasks = []string{}
	}
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(storePath, append(data, '\n'), 0644)
}

func isDone(line string) bool { return strings.HasPrefix(line, "[x] ") }

func addTask(app *tuix.App, text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		app.Set("status", "empty task ignored")
		return false
	}
	setTasks(app, append(getTasks(app), "[ ] "+text))
	app.Set("newTask", "")
	app.Set("status", "added")
	return true
}

func toggleSelected(app *tuix.App) bool {
	tasks := getTasks(app)
	i := selIndex(app)
	if i < 0 || i >= len(tasks) {
		app.Set("status", "nothing selected")
		return false
	}
	if isDone(tasks[i]) {
		tasks[i] = "[ ] " + strings.TrimPrefix(tasks[i], "[x] ")
		app.Set("status", "reopened")
	} else {
		tasks[i] = "[x] " + strings.TrimPrefix(tasks[i], "[ ] ")
		app.Set("status", "done")
	}
	setTasks(app, tasks)
	return true
}

func deleteSelected(app *tuix.App) bool {
	tasks := getTasks(app)
	i := selIndex(app)
	if i < 0 || i >= len(tasks) {
		app.Set("status", "nothing selected")
		return false
	}
	setTasks(app, append(tasks[:i], tasks[i+1:]...))
	app.Set("status", "deleted")
	return true
}

func clearDone(app *tuix.App) int {
	var keep []string
	n := 0
	for _, t := range getTasks(app) {
		if isDone(t) {
			n++
			continue
		}
		keep = append(keep, t)
	}
	setTasks(app, keep)
	app.Set("status", "cleared "+strconv.Itoa(n)+" done")
	return n
}

func registerCounts(app *tuix.App) {
	count := func(done bool) func(func(string) string) string {
		return func(get func(string) string) string {
			n := 0
			for _, line := range strings.Split(get("tasks"), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if isDone(line) == done {
					n++
				}
			}
			return strconv.Itoa(n)
		}
	}
	app.Computed("total", func(get func(string) string) string {
		n := 0
		for _, line := range strings.Split(get("tasks"), "\n") {
			if strings.TrimSpace(line) != "" {
				n++
			}
		}
		return strconv.Itoa(n)
	})
	app.Computed("done", count(true))
}

func main() {
	app, errs := findUI()
	if app == nil {
		fmt.Fprintln(os.Stderr, "load: could not find a valid todo.tuix:")
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "  -", e)
		}
		os.Exit(1)
	}
	registerCounts(app)
	loadTasks(app)

	app.On("onAdd", func(ev event.Event) { addTask(app, app.Get("newTask")) })
	app.On("onToggle", func(ev event.Event) { toggleSelected(app) })
	app.On("onDelete", func(ev event.Event) { deleteSelected(app) })
	app.On("onClearDone", func(ev event.Event) { clearDone(app) })

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
	fmt.Println("Bye! open tasks =", app.Get("total"))
}
