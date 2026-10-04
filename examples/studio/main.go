// Command studio runs TUIX Studio, a snippet playground for .tuix
// authors. It exercises every recent framework feature in one
// coherent tool:
//
//   - List: snippet browser and AST outline (scroll, select, click)
//   - Multiline Input: the snippet editor (Enter = newline)
//   - Cursor: arrows/Home/End/click in every input
//   - Echo: password-gated publishing
//   - Text wrap + maxLines: validation results
//   - Parser line numbers: validation errors point at the offending line
//   - Notify: a live clock refreshed from a background goroutine
//
// Usage (from the module root):
//
//	go run ./examples/studio
//
// Mouse: click rows and buttons. Keyboard: Tab navigates, Up/Down move
// in lists, Enter opens/activates, Esc quits. Publishing needs the
// password "tuix".
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Vinetwigs/tuix"
	"github.com/Vinetwigs/tuix/ast"
	"github.com/Vinetwigs/tuix/component"
	"github.com/Vinetwigs/tuix/event"
	"github.com/Vinetwigs/tuix/parser"
)

const publishPassword = "tuix"

// snippetsDir holds the snippet library. Overridable in tests.
var snippetsDir = ""

// snippetSrc caches file contents by snippet name.
var snippetSrc = map[string]string{}

func findUI() (*tuix.App, []error) {
	cands := []string{"examples/studio/studio.tuix", "studio.tuix"}
	if _, file, _, ok := runtime.Caller(0); ok {
		cands = append(cands, filepath.Join(filepath.Dir(file), "studio.tuix"))
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

func resolveSnippetsDir() string {
	if snippetsDir != "" {
		return snippetsDir
	}
	for _, p := range []string{"examples/studio/snippets", "snippets"} {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		p := filepath.Join(filepath.Dir(file), "snippets")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return "snippets"
}

// refreshSnippets reloads the snippet library into state.
func refreshSnippets(app *tuix.App) {
	dir := resolveSnippetsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		app.Set("status", "no snippets dir: "+dir)
		return
	}
	snippetSrc = map[string]string{}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tuix") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".tuix")
		snippetSrc[name] = string(data)
		names = append(names, name)
	}
	sort.Strings(names)
	app.Set("snippets", strings.Join(names, "\n"))
	app.Set("status", fmt.Sprintf("%d snippets in %s", len(names), dir))
}

func selName(app *tuix.App) string {
	l := component.FindByID(app.Tree(), "snips")
	names := strings.Split(strings.TrimSpace(app.Get("snippets")), "\n")
	if l == nil || l.Sel < 0 || l.Sel >= len(names) {
		return ""
	}
	return strings.TrimSpace(names[l.Sel])
}

func openSelected(app *tuix.App) bool {
	name := selName(app)
	if name == "" {
		app.Set("status", "nothing selected")
		return false
	}
	src, ok := snippetSrc[name]
	if !ok {
		app.Set("status", "missing source for "+name)
		return false
	}
	app.Set("snippetName", name)
	app.Set("code", src)
	validateCode(app)
	app.Set("status", "opened "+name)
	return true
}

// validateCode parses AND builds the editor content so both syntax
// errors and unknown-component errors surface with line numbers.
// It also refreshes the AST outline list.
func validateCode(app *tuix.App) bool {
	code := app.Get("code")
	root, err := parser.ParseString(code)
	if err != nil {
		app.Set("result", "ERROR: "+err.Error())
		app.Set("outline", "")
		app.Set("status", "invalid markup")
		return false
	}
	if _, err := component.Build(root); err != nil {
		app.Set("result", "ERROR: "+err.Error())
		app.Set("outline", "")
		app.Set("status", "invalid components")
		return false
	}
	var lines []string
	var walk func(n *ast.Node, depth int)
	walk = func(n *ast.Node, depth int) {
		label := n.Name
		if id := n.Attrs["id"]; id != "" {
			label += " #" + id
		}
		lines = append(lines, strings.Repeat("  ", depth)+label)
		for _, c := range n.Children {
			walk(c, depth+1)
		}
	}
	walk(root, 0)
	app.Set("outline", strings.Join(lines, "\n"))
	app.Set("result", fmt.Sprintf("OK: %d nodes", len(lines)))
	app.Set("status", "valid")
	return true
}

func newSnippet(app *tuix.App) {
	app.Set("snippetName", "")
	app.Set("code", "")
	app.Set("result", "Fresh buffer - Validate checks it live as you type.")
	app.Set("outline", "")
	app.Set("status", "new snippet")
}

func publishSnippet(app *tuix.App) bool {
	if app.Get("password") != publishPassword {
		app.Set("status", "locked - wrong password")
		return false
	}
	name := strings.TrimSpace(app.Get("snippetName"))
	if name == "" {
		app.Set("status", "name the snippet first")
		return false
	}
	name = strings.TrimSuffix(name, ".tuix")
	dir := resolveSnippetsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		app.Set("status", "cannot create "+dir)
		return false
	}
	if err := os.WriteFile(filepath.Join(dir, name+".tuix"), []byte(app.Get("code")), 0644); err != nil {
		app.Set("status", "save failed: "+err.Error())
		return false
	}
	app.Set("password", "")
	refreshSnippets(app)
	app.Set("status", "published "+name)
	return true
}

// startClock refreshes the header clock every second from a background
// goroutine, waking the run loop with Notify after each update.
func startClock(app *tuix.App) {
	t := time.NewTicker(time.Second)
	go func() {
		for now := range t.C {
			app.Set("clock", now.Format("15:04:05"))
			app.Notify()
		}
	}()
}

func main() {
	app, errs := findUI()
	if app == nil {
		fmt.Fprintln(os.Stderr, "load: could not find a valid studio.tuix:")
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "  -", e)
		}
		os.Exit(1)
	}
	refreshSnippets(app)

	app.On("onOpen", func(ev event.Event) { openSelected(app) })
	app.On("onNew", func(ev event.Event) { newSnippet(app) })
	app.On("onValidate", func(ev event.Event) { validateCode(app) })
	app.On("onEdit", func(ev event.Event) { validateCode(app) })
	app.On("onSave", func(ev event.Event) { publishSnippet(app) })
	app.On("*", func(ev event.Event) {
		if ev.Type == event.FocusChange && ev.TargetID != "" {
			// Keep the last validation status; only narrate focus
			// while the buffer is untouched.
			if app.Get("code") == "" {
				app.Set("status", "focus: "+ev.TargetID)
			}
		}
	})

	startClock(app)
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
	fmt.Println("Bye! snippets in", resolveSnippetsDir())
}
