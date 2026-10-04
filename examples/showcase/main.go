// Command showcase runs a rich TUIX control panel demonstrating
// nested layout (Row/Column/Box/Grid), reactive state with bindings
// and computed values, keyboard navigation and mouse clicks.
//
// Usage (from the module root):
//
//	go run ./examples/showcase
//
// Mouse: left-click a button to press it, click an input to focus it.
// Keyboard: Tab/Shift+Tab and arrows move focus, Enter presses,
// typing edits the focused input, Esc quits.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Vinetwigs/tuix"
	"github.com/Vinetwigs/tuix/event"
)

func findUI() (*tuix.App, []error) {
	cands := []string{"examples/showcase/showcase.tuix", "showcase.tuix"}
	if _, file, _, ok := runtime.Caller(0); ok {
		cands = append(cands, filepath.Join(filepath.Dir(file), "showcase.tuix"))
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

func bump(app *tuix.App) {
	n, _ := strconv.Atoi(app.Get("clicks"))
	app.Set("clicks", strconv.Itoa(n+1))
}

func pressDigit(app *tuix.App, targetID string) {
	d := strings.TrimPrefix(targetID, "k")
	if len(d) != 1 || d < "0" || d > "9" {
		return
	}
	app.Set("display", app.Get("display")+d)
	bump(app)
	app.Set("status", "typed "+d)
}

func pressClear(app *tuix.App) {
	app.Set("display", "")
	app.Set("status", "cleared")
}

func pressConfirm(app *tuix.App) {
	v := app.Get("display")
	if v == "" {
		app.Set("status", "nothing to confirm")
		return
	}
	app.Set("confirmed", v)
	app.Set("display", "")
	bump(app)
	app.Set("status", "confirmed "+v)
}

func pressReset(app *tuix.App) {
	app.Set("user", "ada")
	app.Set("role", "operator")
	app.Set("display", "")
	app.Set("confirmed", "-")
	app.Set("clicks", "0")
	app.Set("status", "reset")
}

func main() {
	app, errs := findUI()
	if app == nil {
		fmt.Fprintln(os.Stderr, "load: could not find a valid showcase.tuix:")
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "  -", e)
		}
		os.Exit(1)
	}

	app.Computed("badge", func(get func(string) string) string {
		return "[" + get("role") + "] " + get("user")
	})

	app.On("onDigit", func(ev event.Event) { pressDigit(app, ev.TargetID) })
	app.On("onClear", func(ev event.Event) { pressClear(app) })
	app.On("onConfirm", func(ev event.Event) { pressConfirm(app) })
	app.On("onReset", func(ev event.Event) { pressReset(app) })
	app.On("onNew", func(ev event.Event) {
		bump(app)
		app.Set("status", "new session for "+app.Get("user"))
	})
	app.On("onSave", func(ev event.Event) {
		bump(app)
		app.Set("status", "saved "+app.Get("user")+" ("+app.Get("role")+")")
	})
	app.On("onLogin", func(ev event.Event) {
		bump(app)
		app.Set("status", "welcome, "+app.Get("user"))
	})
	// Global observer: narrate focus and edits on the status line.
	app.On("*", func(ev event.Event) {
		switch ev.Type {
		case event.FocusChange:
			if ev.TargetID != "" {
				app.Set("status", "focus: "+ev.TargetID)
			}
		case event.InputChange:
			app.Set("status", "editing "+ev.TargetID+" = "+ev.Value)
		}
	})

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
	fmt.Println("Bye! clicks =", app.Get("clicks"), "confirmed =", app.Get("confirmed"))
}
